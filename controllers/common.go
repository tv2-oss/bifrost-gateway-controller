package controllers

import (
	"context"

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

func lookupOurGatewayClass(r Controller, ctx context.Context, name gateway.ObjectName) (*gateway.GatewayClass, error) {
	var gwc gateway.GatewayClass
	if err := r.GetClient().Get(ctx, types.NamespacedName{Name: string(name)}, &gwc); err != nil {
		return nil, err
	}
	if gwc.Spec.ControllerName != SelfControllerName {
		// Silent error if not implemented by us
		return nil, nil
	}
	return &gwc, nil
}
