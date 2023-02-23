// Package api provide an exported API for the controller.
package api

import (
	gatewayapi "sigs.k8s.io/gateway-api/apis/v1beta1"
)

const (
	SelfControllerName gatewayapi.GatewayController = "github.com/tv2-oss/gateway-controller"
)
