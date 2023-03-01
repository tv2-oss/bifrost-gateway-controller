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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ResourceTemplate struct {
	ResourceTemplates map[string]string `json:"resourceTemplates,omitempty"`
}

type GatewayClassBlueprintSpec struct {
	// Template for hardcoded values
	//
	// +optional
	Values TemplateValues `json:"values,omitempty"`

	// Template for child resources created from Gateways
	//
	// +optional
	GatewayTemplate ResourceTemplate `json:"gatewayTemplate,omitempty"`

	// Template for child resources created from HTTPRoutes
	//
	// +optional
	HTTPRouteTemplate ResourceTemplate `json:"httpRouteTemplate,omitempty"`
}

type GatewayClassBlueprintStatus struct {
	// Conditions is the current status from the controller for
	// this GatewayClassParameter. Updates follow the same
	// specification as conditions for GatewayClass.
	//
	// +optional
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MaxItems=8
	// +kubebuilder:default={{type: "Accepted", status: "Unknown", message: "Waiting for controller", reason: "Pending", lastTransitionTime: "1970-01-01T00:00:00Z"}}
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:resource:scope=Cluster
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// GatewayClassBlueprint represents parameters and settings for a specific GatewayClass.
type GatewayClassBlueprint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GatewayClassBlueprintSpec   `json:"spec,omitempty"`
	Status GatewayClassBlueprintStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

type GatewayClassBlueprintList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GatewayClassBlueprint `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GatewayClassBlueprint{}, &GatewayClassBlueprintList{})
}
