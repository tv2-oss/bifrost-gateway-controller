package controllers

import (
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gateway "sigs.k8s.io/gateway-api/apis/v1beta1"
)

const (
	SelfControllerName gateway.GatewayController = "github.com/tv2/cloud-gateway-controller"
)

type Controller interface {
	GetClient() client.Client
}

func isOurGatewayClass(gwc *gateway.GatewayClass) bool {
	return gwc.Spec.ControllerName == SelfControllerName
}

func lookupGatewayClass(ctx context.Context, r Controller, name gateway.ObjectName) (*gateway.GatewayClass, error) {
	var gwc gateway.GatewayClass
	if err := r.GetClient().Get(ctx, types.NamespacedName{Name: string(name)}, &gwc); err != nil {
		return nil, err
	}

	return &gwc, nil
}

func lookupGatewayClassParameters(ctx context.Context, r Controller, gwc *gateway.GatewayClass) (*corev1.ConfigMap, error) {
	if gwc.Spec.ParametersRef == nil {
		return nil, errors.New("GatewayClass without parameters")
	}

	// FIXME: More validation...
	if gwc.Spec.ParametersRef.Kind != "ConfigMap" {
		return nil, errors.New("parameter Kind is not a ConfigMap")
	}

	var cm corev1.ConfigMap
	if err := r.GetClient().Get(ctx, types.NamespacedName{Name: gwc.Spec.ParametersRef.Name, Namespace: string(*gwc.Spec.ParametersRef.Namespace)}, &cm); err != nil {
		return nil, err
	}

	// FIXME: Validate ConfigMap

	return &cm, nil
}
