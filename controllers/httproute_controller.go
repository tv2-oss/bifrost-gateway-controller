/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	gatewayapi "sigs.k8s.io/gateway-api/apis/v1beta1"

	selfapi "github.com/tv2/cloud-gateway-controller/pkg/api"
)

type HTTPRouteReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes/finalizers,verbs=update

func (r *HTTPRouteReconciler) GetClient() client.Client {
	return r.Client
}

func lookupParent(ctx context.Context, r Controller, rt *gatewayapi.HTTPRoute, p gatewayapi.ParentReference) (*gatewayapi.Gateway, error) {
	if p.Namespace == nil {
		return lookupGateway(ctx, r, p.Name, rt.ObjectMeta.Namespace)
	}
	return lookupGateway(ctx, r, p.Name, string(*p.Namespace))
}

func findParentRouteStatus(rtStatus *gatewayapi.RouteStatus, parent gatewayapi.ParentReference) *gatewayapi.RouteParentStatus {
	for i := range rtStatus.Parents {
		pStat := &rtStatus.Parents[i]
		if pStat.ParentRef == parent && pStat.ControllerName == selfapi.SelfControllerName {
			return pStat
		}
	}
	return nil
}

func setRouteStatusCondition(rtStatus *gatewayapi.RouteStatus, parent gatewayapi.ParentReference, newCondition *metav1.Condition) {
	if newCondition.LastTransitionTime.IsZero() {
		newCondition.LastTransitionTime = metav1.NewTime(time.Now())
	}

	existingParentRouteStat := findParentRouteStatus(rtStatus, parent)
	if existingParentRouteStat == nil {
		newStatus := gatewayapi.RouteParentStatus{
			ParentRef:      parent,
			ControllerName: selfapi.SelfControllerName,
			Conditions:     []metav1.Condition{*newCondition},
		}
		rtStatus.Parents = append(rtStatus.Parents, newStatus)
		return
	}

	meta.SetStatusCondition(&existingParentRouteStat.Conditions, *newCondition)
}

func (r *HTTPRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var rt gatewayapi.HTTPRoute
	if err := r.Client.Get(ctx, req.NamespacedName, &rt); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("HTTPRoute")

	prefs := rt.Spec.CommonRouteSpec.ParentRefs
	// FIXME check kind of parent ref is Gateway and missing parentRef. Accepts more than one parent ref
	pref := prefs[0]

	gw := &gatewayapi.Gateway{}
	err := r.Get(ctx, types.NamespacedName{Name: string(pref.Name), Namespace: string(*pref.Namespace)}, gw)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	logger.Info("reconcile", "gateway", gw)

	gwc, err := lookupGatewayClass(ctx, r, gw.Spec.GatewayClassName)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !isOurGatewayClass(gwc) {
		return ctrl.Result{}, nil
	}

	cm, err := lookupGatewayClassParameters(ctx, r, gwc)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("parameters for GatewayClass %q not found: %w", gwc.ObjectMeta.Name, err)
	}

	// Create HTTPRoute resource
	rtOut, err := r.constructHTTPRoute(&rt, cm)
	if err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("create httproute", "rtOut", rtOut)

	if err := ctrl.SetControllerReference(&rt, rtOut, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	rtFound := &gatewayapi.HTTPRoute{}
	err = r.Get(ctx, types.NamespacedName{Name: rtOut.Name, Namespace: rtOut.Namespace}, rtFound)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("create gateway")
		if err := r.Create(ctx, rtOut); err != nil {
			return ctrl.Result{}, err
		}
	} else if err == nil {
		rtFound.Spec = rtOut.Spec
		logger.Info("update httproute", "rt", rtFound)
		if err := r.Update(ctx, rtFound); err != nil {
			return ctrl.Result{}, err
		}
	}

	doStatusUpdate := false
	rt.Status.Parents = []gatewayapi.RouteParentStatus{}
	for _, parent := range rt.Spec.ParentRefs {
		if parent.Namespace == nil {
			// Route parents default to same namespace as route
			parent.Namespace = (*gatewayapi.Namespace)(&rt.ObjectMeta.Namespace)
		}
		gw, err := lookupParent(ctx, r, &rt, parent)
		if err != nil {
			continue
		}
		gwc, err := lookupGatewayClass(ctx, r, gw.Spec.GatewayClassName)
		if err != nil || !isOurGatewayClass(gwc) {
			continue
		}
		doStatusUpdate = true

		setRouteStatusCondition(&rt.Status.RouteStatus, parent,
			&metav1.Condition{
				Type:   string(gatewayapi.RouteConditionAccepted),
				Status: "True",
				Reason: string(gatewayapi.RouteReasonAccepted),
			})
	}

	if doStatusUpdate {
		if err := r.Status().Update(ctx, &rt); err != nil {
			logger.Error(err, "unable to update HTTPRoute status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *HTTPRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gatewayapi.HTTPRoute{}).
		Complete(r)
}

func (r *HTTPRouteReconciler) constructHTTPRoute(rtIn *gatewayapi.HTTPRoute, configmap *corev1.ConfigMap) (*gatewayapi.HTTPRoute, error) {
	name := fmt.Sprintf("%s-%s", rtIn.ObjectMeta.Name, configmap.Data["tier2GatewayClass"])
	rtOut := rtIn.DeepCopy()
	rtOut.ResourceVersion = ""
	rtOut.ObjectMeta.Name = name
	// FIXME, should follow pattern in gateway-controller and remap parents of type gateway similarly
	rtOut.Spec.CommonRouteSpec.ParentRefs[0].Name = "foo-gateway-istio"

	return rtOut, nil
}
