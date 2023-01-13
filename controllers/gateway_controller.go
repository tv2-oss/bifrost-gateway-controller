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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	gatewayapi "sigs.k8s.io/gateway-api/apis/v1beta1"
)

// GatewayReconciler reconciles a Gateway object
type GatewayReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=networking.k8s.io,resources=gateways,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=gateways/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=networking.k8s.io,resources=gateways/finalizers,verbs=update

func (r *GatewayReconciler) GetClient() client.Client {
	return r.Client
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Gateway object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *GatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconcile")

	var g gatewayapi.Gateway
	if err := r.Client.Get(ctx, req.NamespacedName, &g); err != nil {
		logger.Error(err, "Unable to fetch Gateway")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Gateway")

	gwc, err := lookupGatewayClass(r, ctx, g.Spec.GatewayClassName)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !isOurGatewayClass(gwc) {
		return ctrl.Result{}, nil
	}

	cm, err := lookupGatewayClassParameters(r, ctx, gwc)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("Parameters for GatewayClass %q not found: %w", gwc.ObjectMeta.Name, err)
	}

	logger.Info("Creating Istio Gateway")
	newGW := BuildGatewayResource(&g, cm)

	if err := ctrl.SetControllerReference(&g, newGW, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.Client.Create(ctx, newGW); err != nil {
		logger.Error(err, "Unable to create gateway")
		return ctrl.Result{}, err
	}

	addrType := gatewayapi.IPAddressType
	g.Status.Addresses = []gatewayapi.GatewayAddress{gatewayapi.GatewayAddress{Type: &addrType, Value: "1.2.3.4"}}

	if err := r.Status().Update(ctx, &g); err != nil {
		logger.Error(err, "unable to update Gateway status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gatewayapi.Gateway{}).
		Complete(r)
}

func BuildGatewayResource(gateway *gatewayapi.Gateway, cm *corev1.ConfigMap) *gatewayapi.Gateway {
	gatewayClassName := gatewayapi.ObjectName(cm.Data["tier2GatewayClass"])
	name := fmt.Sprintf("%s-%s", gateway.ObjectMeta.Name, gatewayClassName)
	gw := gateway.DeepCopy()
	gw.ResourceVersion = ""
	gw.ObjectMeta.Name = name
	gw.Spec.GatewayClassName = gatewayClassName

	if gw.ObjectMeta.Annotations == nil {
		gw.ObjectMeta.Annotations = map[string]string{}
	}

	gw.ObjectMeta.Annotations["networking.istio.io/service-type"] = "ClusterIP"

	return gw
}
