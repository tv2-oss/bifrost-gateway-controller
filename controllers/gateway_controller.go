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
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	gatewayapi "sigs.k8s.io/gateway-api/apis/v1beta1"
)

// Used to requeue when a resource is missing a dependency
var dependencyMissingRequeuePeriod = 5 * time.Second

// GatewayReconciler reconciles a Gateway object
type GatewayReconciler struct {
	client    client.Client
	scheme    *runtime.Scheme
	dynClient dynamic.Interface
}

//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways/finalizers,verbs=update

func (r *GatewayReconciler) Client() client.Client {
	return r.client
}

func (r *GatewayReconciler) Scheme() *runtime.Scheme {
	return r.scheme
}

func (r *GatewayReconciler) DynamicClient() dynamic.Interface {
	return r.dynClient
}

func NewGatewayController(mgr ctrl.Manager, config *rest.Config) *GatewayReconciler {
	r := &GatewayReconciler{
		client:    mgr.GetClient(),
		scheme:    mgr.GetScheme(),
		dynClient: dynamic.NewForConfigOrDie(config),
	}
	return r
}

func (r *GatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gatewayapi.Gateway{}).
		Complete(r)
}

func (r *GatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var requeue bool

	logger := log.FromContext(ctx)
	logger.Info("Reconcile")

	var gw gatewayapi.Gateway
	if err := r.Client().Get(ctx, req.NamespacedName, &gw); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Gateway")

	gwc, err := lookupGatewayClass(ctx, r, gw.Spec.GatewayClassName)
	if err != nil {
		return ctrl.Result{RequeueAfter: dependencyMissingRequeuePeriod}, client.IgnoreNotFound(err)
	}

	if !isOurGatewayClass(gwc) {
		return ctrl.Result{}, nil
	}

	gwcb, err := lookupGatewayClassBlueprint(ctx, r, gwc)
	if err != nil {
		return ctrl.Result{RequeueAfter: dependencyMissingRequeuePeriod}, fmt.Errorf("parameters for GatewayClass %q not found: %w", gwc.ObjectMeta.Name, err)
	}

	routes, err := lookupHTTPRoutes(ctx, r)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("cannot look up routes: %w", err)
	}
	gwRoutes := filterHTTPRoutesForGateway(&gw, routes)
	union, isect := combineHostnames(&gw, gwRoutes)

	// Prepare Gateway resource for use in templates by converting to map[string]any
	gatewayMap, err := objectToMap(&gw, r.Scheme())
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("cannot convert gateway to map: %w", err)
	}

	values, err := lookupValues(ctx, r, gwc.Name, gwcb, gw.ObjectMeta.Namespace)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("cannot lookup values: %w", err)
	}

	// Setup template variables context
	templateValues := TemplateValues{
		Gateway: &gatewayMap,
		Values:  values,
		Hostnames: TemplateHostnameValues{
			Union:        union,
			Intersection: isect,
		},
	}

	templates, err := parseTemplates(gwcb.Spec.GatewayTemplate.ResourceTemplates)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Resource templates may reference each other, with the
	// worst-case being a strictly linear DAG. This means that we
	// may have to loop N-1 times, with N being the number of
	// resources. We break the loop when we no longer make
	// progress.
	var lastRenderedNum, renderedNum, existsNum int
	lastRenderedNum = -1
	requeue = true
	for attempt := 0; attempt < len(templates); attempt++ {
		isFinalAttempt := attempt < len(templates)-1
		templateValues.Resources, err = buildResourceValues(r, templates)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("unable to build values from current resources: %w", err)
		}

		renderedNum, existsNum = renderTemplates(ctx, r, &gw, templates, &templateValues, isFinalAttempt)
		logger.Info("Rendered", "rendered", renderedNum, "exists", existsNum)
		if renderedNum == lastRenderedNum {
			logger.Info("breaking render/apply loop", "renderedNum", renderedNum, "totalNum", len(templates))
			requeue = false
			break
		}

		if err := applyTemplates(ctx, r, &gw, templates); err != nil {
			return ctrl.Result{}, fmt.Errorf("unable to apply templates: %w", err)
		}
		lastRenderedNum = renderedNum
	}

	// FIXME, this is not a valid address
	addrType := gatewayapi.IPAddressType
	gw.Status.Addresses = []gatewayapi.GatewayAddress{gatewayapi.GatewayAddress{Type: &addrType, Value: "1.2.3.4"}}

	// FIXME: Set real status conditions calculated from child resources
	lStatus := make([]gatewayapi.ListenerStatus, 0, len(gw.Spec.Listeners))
	for _, listener := range gw.Spec.Listeners {
		status := gatewayapi.ListenerStatus{
			Name: listener.Name,
			SupportedKinds: []gatewayapi.RouteGroupKind{{
				Group: (*gatewayapi.Group)(&gatewayapi.GroupVersion.Group),
				Kind:  gatewayapi.Kind("HTTPRoute"),
			}},
			AttachedRoutes: int32(len(gwRoutes)),
		}
		meta.SetStatusCondition(&status.Conditions, metav1.Condition{
			Type:               string(gatewayapi.ListenerConditionAccepted),
			Status:             metav1.ConditionTrue,
			Reason:             string(gatewayapi.ListenerReasonAccepted),
			ObservedGeneration: gw.ObjectMeta.Generation})
		lStatus = append(lStatus, status)
	}
	gw.Status.Listeners = lStatus

	meta.SetStatusCondition(&gw.Status.Conditions, metav1.Condition{
		Type:               string(gatewayapi.GatewayConditionAccepted),
		Status:             metav1.ConditionTrue,
		Reason:             string(gatewayapi.GatewayReasonAccepted),
		ObservedGeneration: gw.ObjectMeta.Generation})

	if err := r.Client().Status().Update(ctx, &gw); err != nil {
		logger.Error(err, "unable to update Gateway status")
		return ctrl.Result{}, err
	}

	if requeue {
		return ctrl.Result{RequeueAfter: dependencyMissingRequeuePeriod}, nil
	}
	return ctrl.Result{}, nil
}

// Calculate union and intersection of Hostnames for use in templates.
// Union is useful to reduce the number of child resources
// changes. I.e. imagine a Gateway with hostname '*.example.com' and a
// HTTPRoute with 'foo.example.com'. We may want to create a TLS
// certificate using '*.example.com' (intersection) and not
// 'foo.example.com' (union). Calculating both allows template authors
// to choose which to use.
func combineHostnames(gw *gatewayapi.Gateway, rtList []*gatewayapi.HTTPRoute) (union, isect []string) {
	wildcards := sets.New[string]() // Wildcard hostnames without '*.' prefix
	hostnames := sets.New[string]() // Non-wildcard hostnames

	// Add 'hostname' to either 'wildcards' or 'hostnames' depending on presence of '*.' prefix
	addHostname := func(hostname string) {
		if strings.HasPrefix(hostname, "*.") {
			domain := strings.TrimPrefix(hostname, "*.")
			if !wildcards.Has(domain) {
				wildcards = sets.Insert(wildcards, domain)
			}
		} else if !hostnames.Has(hostname) {
			hostnames = sets.Insert(hostnames, hostname)
		}
	}

	for _, l := range gw.Spec.Listeners {
		if l.Hostname != nil {
			addHostname(string(*l.Hostname))
		}
	}
	for _, rt := range rtList {
		for _, rtHostname := range rt.Spec.Hostnames {
			addHostname(string(rtHostname))
		}
	}

	for hostname := range wildcards { // Unique wildcards goes in both union and intersection
		hostname = "*." + hostname
		union = append(union, hostname)
		isect = append(isect, hostname)
	}
	for hostname := range hostnames {
		union = append(union, hostname) // Unique hostnames goes in union
		if !wildcards.Has(hostname) {   // Unique hostnames goes in intersection if not covered by wildcard
			isect = append(isect, hostname)
		}
	}
	return union, isect
}

// Match HTTPRoutes against Gateway listeners, return valid HTTPRoute matches
func filterHTTPRoutesForGateway(gw *gatewayapi.Gateway, rtList []*gatewayapi.HTTPRoute) []*gatewayapi.HTTPRoute {
	rtOut := make([]*gatewayapi.HTTPRoute, 0, len(rtList))
	for _, rt := range rtList {
		for _, pRef := range rt.Spec.ParentRefs {
			if (pRef.Group != nil && *pRef.Group != gatewayapi.Group(gatewayapi.GroupName)) ||
				(pRef.Kind != nil && *pRef.Kind != gatewayapi.Kind("Gateway")) ||
				(pRef.Namespace != nil && *pRef.Namespace != gatewayapi.Namespace(gw.ObjectMeta.Namespace)) ||
				// Unspecified namespace means use HTTPRoute namespace
				(pRef.Namespace == nil && rt.ObjectMeta.Namespace != gw.ObjectMeta.Namespace) ||
				(pRef.Name != gatewayapi.ObjectName(gw.ObjectMeta.Name)) {
				// Skip as ParentRef does not refer to Gateway
				continue
			}
			// FIXME: Also include SectionName and Port in match
			// for _, l := range gw.Spec.Listeners {
			// 	if xxx
			// }
			rtOut = append(rtOut, rt)
		}
	}
	return rtOut
}

// Lookup all HTTPRoutes
func lookupHTTPRoutes(ctx context.Context, r ControllerClient) ([]*gatewayapi.HTTPRoute, error) {
	var rtList gatewayapi.HTTPRouteList

	if err := r.Client().List(ctx, &rtList); err != nil {
		return nil, err
	}

	// Return pointer slice
	rtOut := make([]*gatewayapi.HTTPRoute, 0, len(rtList.Items))
	for idx := range rtList.Items {
		rtOut = append(rtOut, &rtList.Items[idx])
	}
	return rtOut, nil
}
