package controllers

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gwcapi "github.com/tv2-oss/gateway-controller/apis/gateway.tv2.dk/v1alpha1"
	corev1 "k8s.io/api/core/v1"
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
  controllerName: "github.com/tv2-oss/gateway-controller"
  parametersRef:
    group: gateway.tv2.dk
    kind: GatewayClassBlueprint
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

const gatewayClassBlueprintManifest string = `
apiVersion: gateway.tv2.dk/v1alpha1
kind: GatewayClassBlueprint
metadata:
  name: default-gateway-class
spec:
  gatewayTemplate:
    resourceTemplates:
      childGateway: |
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
      # The following three configmaps tests referencing between resources
      configMapTestSource: |
        apiVersion: v1
        kind: ConfigMap
        metadata:
          name: source-configmap
          namespace: {{ .Gateway.metadata.namespace }}
        data:
          valueToRead1: Hello
          valueToRead2: World
      configMapTestIntermediate: |
        apiVersion: v1
        kind: ConfigMap
        metadata:
          name: intermediate-configmap
          namespace: {{ .Gateway.metadata.namespace }}
        data:
          valueIntermediate: {{ .Resources.configMapTestSource.data.valueToRead1 }}
      # Use references to multiple resources coupled with template pipeline and functions
      configMapTestDestination: |
        apiVersion: v1
        kind: ConfigMap
        metadata:
          name: dst-configmap
          namespace: {{ .Gateway.metadata.namespace }}
        data:
          valueRead: {{ printf "%s, %s" .Resources.configMapTestIntermediate.data.valueIntermediate .Resources.configMapTestSource.data.valueToRead2 | upper }}
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
		gwc  *gateway.GatewayClass
		gwcb *gwcapi.GatewayClassBlueprint
		ctx  context.Context
	)

	BeforeEach(func() {
		gwc = &gateway.GatewayClass{}
		gwcb = &gwcapi.GatewayClassBlueprint{}
		ctx = context.Background()
		Expect(yaml.Unmarshal([]byte(gatewayClassManifest), gwc)).To(Succeed())
		Expect(k8sClient.Create(ctx, gwc)).Should(Succeed())
		Expect(yaml.Unmarshal([]byte(gatewayClassBlueprintManifest), gwcb)).To(Succeed())
		Expect(k8sClient.Create(ctx, gwcb)).Should(Succeed())
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(ctx, gwc)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, gwcb)).Should(Succeed())
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

		It("Should update intra resource-references", func() {

			cm := corev1.ConfigMap{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: "dst-configmap", Namespace: "default"}, &cm)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Setting the content of the destination configmap")
			Expect(cm.Data["valueRead"]).To(Equal("HELLO, WORLD"))
		})
	})
})
