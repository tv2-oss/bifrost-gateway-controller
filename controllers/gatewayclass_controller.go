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

	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logger "sigs.k8s.io/controller-runtime/pkg/log"
	gateway "sigs.k8s.io/gateway-api/apis/v1beta1"
)

// GatewayClassReconciler reconciles a GatewayClass object
type GatewayClassReconciler struct {
	Client client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gatewayclasses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gatewayclasses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gatewayclasses/finalizers,verbs=update

func (r *GatewayClassReconciler) GetClient() client.Client {
	return r.Client
}

func (r *GatewayClassReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logger.FromContext(ctx)

	gwc, cm, err := lookupOurGatewayClass(r, ctx, gateway.ObjectName(req.Name))
	if err != nil || gwc == nil {
		if gwc == nil { // Also covers not implemented by us
			return ctrl.Result{}, client.IgnoreNotFound(err)
		} else {
			return ctrl.Result{}, fmt.Errorf("gatewayclass parameters not found")
		}
	}

	var valid bool = true
	if cm == nil {
		valid = false
	}

	if valid {
		log.Info("Accepted", "GatewayClass", req.Name)
		meta.SetStatusCondition(&gwc.Status.Conditions, metav1.Condition{
			Type:   string(gateway.GatewayClassConditionStatusAccepted),
			Status: "True",
			Reason: string(gateway.GatewayClassReasonAccepted)})
	} else {
		meta.SetStatusCondition(&gwc.Status.Conditions, metav1.Condition{
			Type:   string(gateway.GatewayClassConditionStatusAccepted),
			Status: "False",
			Reason: string(gateway.GatewayClassReasonInvalidParameters)})
	}

	err = r.Client.Status().Update(ctx, gwc)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("Failed to update GatewayClass status condition: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *GatewayClassReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gateway.GatewayClass{}).
		Complete(r)
}
