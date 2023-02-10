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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	gatewayapi "sigs.k8s.io/gateway-api/apis/v1beta1"
)

// GatewayReconciler reconciles a Gateway object
type GatewayReconciler struct {
	client    client.Client
	dynClient dynamic.Interface
	scheme    *runtime.Scheme
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

func NewGatewayController(mgr ctrl.Manager) *GatewayReconciler {
	r := &GatewayReconciler{
		client:    mgr.GetClient(),
		scheme:    mgr.GetScheme(),
		dynClient: dynamic.NewForConfigOrDie(ctrl.GetConfigOrDie()),
	}
	return r
}

func (r *GatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gatewayapi.Gateway{}).
		Complete(r)
}

func (r *GatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconcile")

	var g gatewayapi.Gateway
	if err := r.Client().Get(ctx, req.NamespacedName, &g); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Gateway")

	gwc, err := lookupGatewayClass(ctx, r, g.Spec.GatewayClassName)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !isOurGatewayClass(gwc) {
		return ctrl.Result{}, nil
	}

	cm, err := lookupGatewayClassParameters(ctx, r, gwc)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("parameters for GatewayClass %q not found: %w", gwc.ObjectMeta.Name, err)
	}

	if err := applyTemplates(ctx, r, &g, cm); err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to apply templates: %w", err)
	}

	addrType := gatewayapi.IPAddressType
	g.Status.Addresses = []gatewayapi.GatewayAddress{gatewayapi.GatewayAddress{Type: &addrType, Value: "1.2.3.4"}}

	meta.SetStatusCondition(&g.Status.Conditions, metav1.Condition{
		Type:               string(gatewayapi.GatewayConditionAccepted),
		Status:             "True",
		Reason:             string(gatewayapi.GatewayReasonAccepted),
		ObservedGeneration: g.ObjectMeta.Generation})

	if err := r.Client().Status().Update(ctx, &g); err != nil {
		logger.Error(err, "unable to update Gateway status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func applyTemplates(ctx context.Context, r ControllerDynClient, gwParent *gatewayapi.Gateway, configmap *corev1.ConfigMap) error {
	for tmplKey, tmpl := range configmap.Data {
		u, err := template2Unstructured(gwParent, tmpl)
		if err != nil {
			return fmt.Errorf("cannot render template %q: %w", tmplKey, err)
		}

		if err := ctrl.SetControllerReference(gwParent, u, r.Scheme()); err != nil {
			return fmt.Errorf("cannot set owner for resource created from template %q: %w", tmplKey, err)
		}

		if err := patchUnstructured(ctx, r, u, gwParent.ObjectMeta.Namespace); err != nil {
			return fmt.Errorf("cannot apply template %q: %w", tmplKey, err)
		}
	}
	return nil
}
