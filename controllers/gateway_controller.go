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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	gateway "sigs.k8s.io/gateway-api/apis/v1beta1"
)

// GatewayReconciler reconciles a Gateway object
type GatewayReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=networking.k8s.io.ccs.tv2.dk,resources=gateways,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io.ccs.tv2.dk,resources=gateways/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=networking.k8s.io.ccs.tv2.dk,resources=gateways/finalizers,verbs=update

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

	var g gateway.Gateway
	if err := r.Get(ctx, req.NamespacedName, &g); err != nil {
		logger.Error(err, "Unable to fetch Gateway")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Gateway")

	if g.Spec.GatewayClassName != "istio" {
		logger.Info("Creating Istio Gateway")
		newGW := BuildGatewayResource(&g)
		if err := r.Create(ctx, newGW); err != nil {
			logger.Error(err, "Unable to create gateway")
			return ctrl.Result{}, err
		}
	}

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gateway.Gateway{}).
		//Owns(&gateway.Gateway{}).
		Complete(r)
}

func BuildGatewayResource(gateway *gateway.Gateway) *gateway.Gateway {
	name := fmt.Sprintf("%s-%s", gateway.ObjectMeta.Name, "istio")
	gw := gateway.DeepCopy()
	gw.ResourceVersion = ""
	gw.ObjectMeta.Name = name
	gw.Spec.GatewayClassName = "istio"
	if gw.ObjectMeta.Annotations == nil {
		gw.ObjectMeta.Annotations = map[string]string{}
	}
	gw.ObjectMeta.Annotations["networking.istio.io/service-type"] = "ClusterIP"

	return gw
}
