package tests

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	gatewayapi "sigs.k8s.io/gateway-api/apis/v1beta1"
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
		t.Run("Gateway listener should have a true ListenerConditionAccepted", func(t *testing.T) {
			gwNN := types.NamespacedName{Name: "example", Namespace: "ns0"}
			listeners := []gatewayapi.ListenerStatus{{
				Name:           gatewayapi.SectionName("http"),
				SupportedKinds: []gatewayapi.RouteGroupKind{{
					Group: (*gatewayapi.Group)(&gatewayapi.GroupVersion.Group),
					Kind:  gatewayapi.Kind("HTTPRoute"),
				}},
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
