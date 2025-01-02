/*
Copyright 2023 TV 2 DANMARK A/S

Licensed under the Apache License, Version 2.0 (the "License") with the
following modification to section 6. Trademarks:

Section 6. Trademarks is deleted and replaced by the following wording:

6. Trademarks. This License does not grant permission to use the trademarks and
trade names of TV 2 DANMARK A/S, including but not limited to the TV 2® logo and
word mark, except (a) as required for reasonable and customary use in describing
the origin of the Work, e.g. as described in section 4(c) of the License, and
(b) to reproduce the content of the NOTICE file. Any reference to the Licensor
must be made by making a reference to "TV 2 DANMARK A/S", written in capitalized
letters as in this example, unless the format in which the reference is made,
requires lower case letters.

You may not use this software except in compliance with the License and the
modifications set out above.

You may obtain a copy of the license at:

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tests

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	gatewayapi "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/gateway-api/conformance/utils/kubernetes"
	"sigs.k8s.io/gateway-api/conformance/utils/suite"
)

func init() {
	UnitTests = append(UnitTests, CrossNamespaceRouteHostname)
}

var CrossNamespaceRouteHostname = suite.ConformanceTest{
	ShortName:   "CrossNamespaceRouteHostname",
	Description: "A Gateway with multiple HTTPRoutes should normalize listeners and routes.",
	Manifests:   []string{"tests/cross_namespace_route_hostname.yaml"},
	Test: func(t *testing.T, s *suite.ConformanceTestSuite) {
		t.Run("Parent Gateway listener should have a true ListenerConditionAccepted with 2 of 3 attached routes", func(t *testing.T) {
			gwNN := types.NamespacedName{Name: "example", Namespace: "ns0"}
			listeners := []gatewayapi.ListenerStatus{{
				Name: gatewayapi.SectionName("http"),
				SupportedKinds: []gatewayapi.RouteGroupKind{{
					Group: (*gatewayapi.Group)(&gatewayapi.GroupVersion.Group),
					Kind:  gatewayapi.Kind("HTTPRoute"),
				}},
				AttachedRoutes: 2,
				Conditions: []metav1.Condition{{
					Type:   string(gatewayapi.ListenerConditionAccepted),
					Status: metav1.ConditionTrue,
					Reason: string(gatewayapi.ListenerReasonAccepted),
				}},
			}}

			kubernetes.GatewayStatusMustHaveListeners(t, s.Client, s.TimeoutConfig, gwNN, listeners)
		})
		t.Run("Shadow Gateway listeners (x2) should have a true ListenerConditionAccepted", func(t *testing.T) {
			gwNN := types.NamespacedName{Name: "example", Namespace: "ns0"}
			listeners := []gatewayapi.ListenerStatus{{
				Name: gatewayapi.SectionName("http"),
				SupportedKinds: []gatewayapi.RouteGroupKind{{
					Group: (*gatewayapi.Group)(&gatewayapi.GroupVersion.Group),
					Kind:  gatewayapi.Kind("HTTPRoute"),
				}},
				AttachedRoutes: 2,
				Conditions: []metav1.Condition{{
					Type:   string(gatewayapi.ListenerConditionAccepted),
					Status: metav1.ConditionTrue,
					Reason: string(gatewayapi.ListenerReasonAccepted),
				}},
			}}

			kubernetes.GatewayStatusMustHaveListeners(t, s.Client, s.TimeoutConfig, gwNN, listeners)
		})
	},
}
