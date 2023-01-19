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
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	gatewayapi "sigs.k8s.io/gateway-api/apis/v1beta1"
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
		if pStat.ParentRef == parent && pStat.ControllerName == SelfControllerName {
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
			ControllerName: SelfControllerName,
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

	doStatusUpdate := false
	rt.Status.Parents = []gatewayapi.RouteParentStatus{}
	for _, parent := range rt.Spec.ParentRefs {
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
