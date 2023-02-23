package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	cgcapi "github.com/tv2-oss/gateway-controller/apis/gateway.tv2.dk/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	gateway "sigs.k8s.io/gateway-api/apis/v1beta1"
)

const gatewayclassManifest string = `
apiVersion: gateway.networking.k8s.io/v1beta1
kind: GatewayClass
metadata:
  name: cloud-gw
spec:
  controllerName: "github.com/tv2-oss/gateway-controller"
  parametersRef:
    group: gateway.tv2.dk
    kind: GatewayClassParameters
    name: default-gateway-class`

const gatewayclassManifestInvalid string = `
apiVersion: gateway.networking.k8s.io/v1beta1
kind: GatewayClass
metadata:
  name: cloud-gw-invalid
spec:
  controllerName: "github.com/tv2-oss/gateway-controller"`

const gatewayclassManifestNotOurs string = `
apiVersion: gateway.networking.k8s.io/v1beta1
kind: GatewayClass
metadata:
  name: not-our-gatewayclass
spec:
  controllerName: "github.com/acme/gateway-controller"`

const gwClassParametersManifest string = `
apiVersion: gateway.tv2.dk/v1alpha1
kind: GatewayClassParameters
metadata:
  name: default-gateway-class
spec:
  gatewayTemplate:
    resourceTemplates:
      istioShadowGw: |
        apiVersion: gateway.networking.k8s.io/v1beta1
        kind: Gateway
        metadata:
          name: {{ .Gateway.ObjectMeta.Name }}-istio
          namespace: {{ .Gateway.metadata.namespace }}
          annotations:
            networking.istio.io/service-type: ClusterIP
        spec:
          gatewayClassName: istio
          listeners:
            {{- toYaml .Gateway.spec.listeners | nindent 6 }}
  httpRouteTemplate:
    resourceTemplates:
      shadowHttproute: |
        apiVersion: gateway.networking.k8s.io/v1beta1
        kind: HTTPRoute
        metadata:
          name: {{ .HTTPRoute.ObjectMeta.Name }}-istio
          namespace: {{ .HTTPRoute.metadata.namespace }}
        spec:
          parentRefs:
          {{ range .HTTPRoute.spec.parentRefs }}
          - kind: {{ .kind }}
            name: {{ .name }}-istio
            namespace: {{ .namespace }}
          {{ end }}
          rules:
          {{ toYaml .HTTPRoute.spec.rules | nindent 4 }}`

var _ = Describe("GatewayClass controller", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	var (
		gwcIn, gwc *gateway.GatewayClass
		gcp        *cgcapi.GatewayClassParameters
		ctx        context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		gwcIn = &gateway.GatewayClass{}
		gwc = &gateway.GatewayClass{}
		gcp = &cgcapi.GatewayClassParameters{}
	})

	When("A gatewayclass we own is created", func() {

		It("Should be marked as accepted", func() {

			err := yaml.Unmarshal([]byte(gatewayclassManifest), gwcIn)
			Expect(err).Should(Succeed())
			Expect(k8sClient.Create(ctx, gwcIn)).Should(Succeed())

			Expect(yaml.Unmarshal([]byte(gwClassParametersManifest), gcp)).To(Succeed())
			Expect(k8sClient.Create(ctx, gcp)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: gwcIn.ObjectMeta.Name, Namespace: ""}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, gwc)
				if err != nil ||
					gwc.Status.Conditions[0].Type != string(gateway.GatewayClassConditionStatusAccepted) ||
					gwc.Status.Conditions[0].Status != "True" {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			Expect(k8sClient.Delete(ctx, gwcIn)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, gcp)).Should(Succeed())
		})
	})

	When("An invalid gatewayclass we own is created", func() {
		It("Should be marked as invalid", func() {

			err := yaml.Unmarshal([]byte(gatewayclassManifestInvalid), gwcIn)
			Expect(err).Should(Succeed())
			Expect(k8sClient.Create(ctx, gwcIn)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: gwcIn.ObjectMeta.Name, Namespace: ""}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, gwc)
				if err != nil ||
					gwc.Status.Conditions[0].Type != string(gateway.GatewayClassConditionStatusAccepted) ||
					gwc.Status.Conditions[0].Status != "False" ||
					gwc.Status.Conditions[0].Reason != string(gateway.GatewayClassReasonInvalidParameters) {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			Expect(k8sClient.Delete(ctx, gwcIn)).Should(Succeed())
		})
	})

	When("A gatewayclass we do not own is created", func() {
		It("Should not be marked as accepted", func() {

			err := yaml.Unmarshal([]byte(gatewayclassManifestNotOurs), gwcIn)
			Expect(err).Should(Succeed())
			Expect(k8sClient.Create(ctx, gwcIn)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: gwcIn.ObjectMeta.Name, Namespace: ""}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, gwc)
				if err != nil ||
					gwc.Status.Conditions[0].Status != "Unknown" {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			Expect(k8sClient.Delete(ctx, gwcIn)).Should(Succeed())
		})
	})
})
