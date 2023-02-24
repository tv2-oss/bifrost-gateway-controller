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

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logger "sigs.k8s.io/controller-runtime/pkg/log"
	gatewayapi "sigs.k8s.io/gateway-api/apis/v1beta1"
)

// GatewayClassReconciler reconciles a GatewayClass object
type GatewayClassReconciler struct {
	client client.Client
	scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gatewayclasses,verbs=get;list;watch
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gatewayclasses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gatewayclasses/finalizers,verbs=update

//+kubebuilder:rbac:groups=gateway.tv2.dk,resources=gatewayclassparameters,verbs=get;list;watch
//+kubebuilder:rbac:groups=gateway.tv2.dk,resources=gatewayclassparameters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gateway.tv2.dk,resources=gatewayclassparameters/finalizers,verbs=update

func (r *GatewayClassReconciler) Client() client.Client {
	return r.client
}

func (r *GatewayClassReconciler) Scheme() *runtime.Scheme {
	return r.scheme
}

func NewGatewayClassController(mgr ctrl.Manager) *GatewayClassReconciler {
	r := &GatewayClassReconciler{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		//dynClient: dynamic.NewForConfigOrDie(ctrl.GetConfigOrDie()),
	}
	return r
}

func (r *GatewayClassReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gatewayapi.GatewayClass{}).
		Complete(r)
}

func (r *GatewayClassReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logger.FromContext(ctx)

	var valid = true
	var errWhyInvalid error

	gwc, err := lookupGatewayClass(ctx, r, gatewayapi.ObjectName(req.Name))
	if err != nil {
		return ctrl.Result{}, err
	}

	if !isOurGatewayClass(gwc) {
		return ctrl.Result{}, nil
	}

	_, err = lookupGatewayClassParameters(ctx, r, gwc)
	if err != nil {
		valid = false
		errWhyInvalid = fmt.Errorf("parameters for GatewayClass %q not found", gwc.ObjectMeta.Name)
	}

	if valid {
		log.Info("Accepted", "GatewayClass", req.Name)
		meta.SetStatusCondition(&gwc.Status.Conditions, metav1.Condition{
			Type:               string(gatewayapi.GatewayClassConditionStatusAccepted),
			Status:             "True",
			Reason:             string(gatewayapi.GatewayClassReasonAccepted),
			ObservedGeneration: gwc.ObjectMeta.Generation})
	} else {
		meta.SetStatusCondition(&gwc.Status.Conditions, metav1.Condition{
			Type:               string(gatewayapi.GatewayClassConditionStatusAccepted),
			Status:             "False",
			Reason:             string(gatewayapi.GatewayClassReasonInvalidParameters),
			ObservedGeneration: gwc.ObjectMeta.Generation})
	}

	err = r.Client().Status().Update(ctx, gwc)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update GatewayClass status condition: %w", err)
	}

	if !valid {
		return ctrl.Result{RequeueAfter: dependencyMissingRequeuePeriod}, errWhyInvalid
	}
	return ctrl.Result{}, nil
}
