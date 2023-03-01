package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// Template values - values that will be made available for templates defined in GatewayClassBlueprint.
//
// Follows the hierarchy model defined in https://gateway-api.sigs.k8s.io/geps/gep-713/#hierarchy
type TemplateValues struct {
	// Overrides have precedence from GatewayClassBlueprint
	// (highest) through GatewayClassConfig to GatewayConfig
	// (lowest)
	//
	// +optional
	Override *apiextensionsv1.JSON `json:"override,omitempty"`

	// Defaults have precedence from GatewayConfig (highest)
	// through GatewayClassConfig to GatewayClassBlueprint
	// (lowest)
	//
	// +optional
	Default *apiextensionsv1.JSON `json:"default,omitempty"`
}
