package controllers

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cgcapi "github.com/tv2/cloud-gateway-controller/apis/cgc.tv2.dk/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	gateway "sigs.k8s.io/gateway-api/apis/v1beta1"
)

const gatewayClassManifest string = `
apiVersion: gateway.networking.k8s.io/v1beta1
kind: GatewayClass
metadata:
  name: default
spec:
  controllerName: "github.com/tv2/cloud-gateway-controller"
  parametersRef:
    group: v1alpha1
    kind: GatewayClassParameters
    name: default-gateway-class`

const gatewayManifest string = `
apiVersion: gateway.networking.k8s.io/v1beta1
kind: Gateway
metadata:
  name: foo-gateway
  namespace: default
spec:
  gatewayClassName: default
  listeners:
  - name: prod-web
    port: 80
    protocol: HTTP
    hostname: example.com
`

const gatewayClassParametersManifest string = `
apiVersion: cgc.tv2.dk/v1alpha1
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
          name: {{ .Gateway.metadata.name }}-istio
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
          name: {{ .HTTPRoute.metadata.name }}-istio
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

var _ = Describe("Gateway controller", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	var (
		gwc *gateway.GatewayClass
		gcp *cgcapi.GatewayClassParameters
		ctx context.Context
	)

	BeforeEach(func() {
		gwc = &gateway.GatewayClass{}
		gcp = &cgcapi.GatewayClassParameters{}
		ctx = context.Background()
		Expect(yaml.Unmarshal([]byte(gatewayClassManifest), gwc)).To(Succeed())
		Expect(k8sClient.Create(ctx, gwc)).Should(Succeed())
		Expect(yaml.Unmarshal([]byte(gatewayClassParametersManifest), gcp)).To(Succeed())
		Expect(k8sClient.Create(ctx, gcp)).Should(Succeed())
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(ctx, gwc)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, gcp)).Should(Succeed())
	})

	When("Reconciling a parent Gateway", func() {
		var childGateway, gw *gateway.Gateway

		BeforeEach(func() {
			gw = &gateway.Gateway{}
			childGateway = &gateway.Gateway{}
			Expect(yaml.Unmarshal([]byte(gatewayManifest), gw)).To(Succeed())
		})

		It("Should lifecycle correctly", func() {

			By("Creating the gateway")
			Expect(k8sClient.Create(ctx, gw)).Should(Succeed())
			Expect(string(gw.Spec.GatewayClassName)).To(Equal("default"))

			By("Creating the child gateway")
			name := fmt.Sprintf("%s-%s", gw.ObjectMeta.Name, "istio")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: "default"}, childGateway)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Setting the owner reference to enable garbage collection")
			t := true
			expectedOwnerReference := v1.OwnerReference{
				Kind:               "Gateway",
				APIVersion:         "gateway.networking.k8s.io/v1beta1",
				UID:                gw.ObjectMeta.GetUID(),
				Name:               gw.ObjectMeta.Name,
				Controller:         &t,
				BlockOwnerDeletion: &t,
			}
			Expect(childGateway.ObjectMeta.OwnerReferences).To(ContainElement(expectedOwnerReference))
		})
	})
})
