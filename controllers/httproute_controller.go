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

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	gatewayapi "sigs.k8s.io/gateway-api/apis/v1beta1"

	selfapi "github.com/tv2-oss/gateway-controller/pkg/api"
)

type HTTPRouteReconciler struct {
	client    client.Client
	scheme    *runtime.Scheme
	dynClient dynamic.Interface
}

// Parameters used to render HTTPRoute templates
type httprouteTemplateValues struct {
	// Parent HTTPRoute
	HTTPRoute map[string]any

	// Parent Gateway references. Only Gateways managed by this controller by will be included
	ParentRef map[string]any
}

//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes/finalizers,verbs=update

func (r *HTTPRouteReconciler) Client() client.Client {
	return r.client
}

func (r *HTTPRouteReconciler) Scheme() *runtime.Scheme {
	return r.scheme
}

func (r *HTTPRouteReconciler) DynamicClient() dynamic.Interface {
	return r.dynClient
}

func NewHTTPRouteController(mgr ctrl.Manager, config *rest.Config) *HTTPRouteReconciler {
	r := &HTTPRouteReconciler{
		client:    mgr.GetClient(),
		scheme:    mgr.GetScheme(),
		dynClient: dynamic.NewForConfigOrDie(config),
	}
	return r
}

func (r *HTTPRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gatewayapi.HTTPRoute{}).
		Complete(r)
}

// Compare values referenced by pointers. Both a and b must be pointers to the same type
func derefCmp[T comparable](a, b *T) bool {
	if (a != nil && b == nil) || (a == nil && b != nil) {
		return false // Different since one has nil pointer while other has non-nil
	}
	if a != nil && b != nil && (*a != *b) {
		return false // Actual value is different
	}
	return true
}

// Compare parentRefs, return true if same parent is referenced
func parentRefCmp(a, b gatewayapi.ParentReference) bool {
	// Group, Kind and Namespace, SectionName and Port are pointers and optional (may be nil)
	if !derefCmp(a.Group, b.Group) || !derefCmp(a.Kind, b.Kind) || !derefCmp(a.Namespace, b.Namespace) ||
		!derefCmp(a.SectionName, b.SectionName) || !derefCmp(a.Port, b.Port) {
		return false
	}
	return a.Name == b.Name
}

// Lookup Gateway from parentRef
func lookupParent(ctx context.Context, r ControllerClient, rt *gatewayapi.HTTPRoute, p gatewayapi.ParentReference) (*gatewayapi.Gateway, error) {
	if p.Namespace == nil {
		return lookupGateway(ctx, r, p.Name, rt.ObjectMeta.Namespace)
	}
	return lookupGateway(ctx, r, p.Name, string(*p.Namespace))
}

// Find statuscondition for a specific parentRef
func findParentRouteStatus(rtStatus *gatewayapi.RouteStatus, parent gatewayapi.ParentReference) *gatewayapi.RouteParentStatus {
	for i := range rtStatus.Parents {
		pStat := &rtStatus.Parents[i]
		if parentRefCmp(pStat.ParentRef, parent) && pStat.ControllerName == selfapi.SelfControllerName {
			return pStat
		}
	}
	return nil
}

// Set status condition for a specific parentRef
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

	var doStatusUpdate = false
	var requeue = false
	var rt gatewayapi.HTTPRoute
	if err := r.Client().Get(ctx, req.NamespacedName, &rt); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("HTTPRoute")

	// Prepare HTTPRoute resource for use in templates by converting to map[string]any
	rtMap, err := objectToMap(&rt, r.Scheme())
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("cannot convert httproute to map: %w", err)
	}

	templateValues := httprouteTemplateValues{
		HTTPRoute: rtMap,
	}

	// Prepare for setting status in parentRef loop
	if rt.Status.Parents == nil {
		rt.Status.Parents = []gatewayapi.RouteParentStatus{}
	}

	// Loop through Gateway parents, render HTTPRoute using templates defined by associated GatewayClassParameters
	for _, parent := range rt.Spec.ParentRefs {
		if *parent.Kind != gatewayapi.Kind("Gateway") {
			continue
		}

		gw, err := lookupParent(ctx, r, &rt, parent)
		if err != nil {
			logger.Info("gateway for httproute not found", "httproute", rt.Name, "parent", parent)
			requeue = true
			continue
		}

		// FIXME check route matches gateway hostname,
		// listener sectionname, port etc. From this a list of
		// listeners should be matched and we should validate
		// that listener allows route to attach

		gwc, err := lookupGatewayClass(ctx, r, gw.Spec.GatewayClassName)
		if err != nil {
			logger.Info("gatewayClass not found", "gatewayclassname", gw.Spec.GatewayClassName)
			requeue = true
			continue
		}
		if !isOurGatewayClass(gwc) {
			continue
		}

		gwcp, err := lookupGatewayClassParameters(ctx, r, gwc)
		if err != nil {
			logger.Info("parameters for GatewayClass not found", "gatewayclassparameters", gwc.ObjectMeta.Name)
			requeue = true
			continue
		}

		// Prepare Gateway resource for use in templates by converting to map[string]any
		gatewayMap, err := objectToMap(gw, r.Scheme())
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("cannot convert gateway to map: %w", err)
		}

		templateValues.ParentRef = gatewayMap
		templates := gwcp.Spec.HTTPRouteTemplate.ResourceTemplates
		if err := applyTemplates(ctx, r, &rt, templates, templateValues); err != nil {
			logger.Info("unable to apply templates")
			requeue = true
			continue
		}

		// FIXME errors in templating and status of sub-resources in general should set status conditions

		// Update status for current parent Gateway
		doStatusUpdate = true
		setRouteStatusCondition(&rt.Status.RouteStatus, parent,
			&metav1.Condition{
				Type:   string(gatewayapi.RouteConditionAccepted),
				Status: "True",
				Reason: string(gatewayapi.RouteReasonAccepted),
			})
	}

	if doStatusUpdate {
		if err := r.Client().Status().Update(ctx, &rt); err != nil {
			logger.Error(err, "unable to update HTTPRoute status")
			return ctrl.Result{}, err
		}
	}

	if requeue {
		return ctrl.Result{RequeueAfter: dependencyMissingRequeuePeriod}, nil
	}
	return ctrl.Result{}, nil
}
