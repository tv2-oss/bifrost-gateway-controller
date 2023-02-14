package controllers

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
  controllerName: "github.com/tv2/cloud-gateway-controller"
  parametersRef:
    group: v1
    kind: ConfigMap
    name: default-gateway-class
    namespace: default`

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

const configMapManifest string = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: default-gateway-class
  namespace: default
data:
    istio: |
      apiVersion: gateway.networking.k8s.io/v1beta1
      kind: Gateway
      metadata:
        name: {{ .Gateway.ObjectMeta.Name }}-istio
        namespace: {{ .Gateway.ObjectMeta.Namespace }}
        annotations:
          networking.istio.io/service-type: ClusterIP
      spec:
        gatewayClassName: istio
        listeners:
        {{- range .Gateway.Spec.Listeners }}
        - name: {{ .Name }}
          port: {{ .Port }}
          protocol: {{ .Protocol }}
          hostname: {{ .Hostname }}
        {{- end }}`

var _ = Describe("Gateway controller", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	var (
		gwc *gateway.GatewayClass
		cm  *corev1.ConfigMap
		ctx context.Context
	)

	BeforeEach(func() {
		gwc = &gateway.GatewayClass{}
		cm = &corev1.ConfigMap{}
		ctx = context.Background()
		Expect(yaml.Unmarshal([]byte(gatewayClassManifest), gwc)).To(Succeed())
		Expect(k8sClient.Create(ctx, gwc)).Should(Succeed())
		Expect(yaml.Unmarshal([]byte(configMapManifest), cm)).To(Succeed())
		Expect(k8sClient.Create(ctx, cm)).Should(Succeed())
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(ctx, gwc)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, cm)).Should(Succeed())
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
