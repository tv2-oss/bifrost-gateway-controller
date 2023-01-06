package controllers

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	gateway "sigs.k8s.io/gateway-api/apis/v1beta1"
)

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

var _ = Describe("Gateway controller", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When building Gateway resource from input Gateway", func() {
		It("Should return a new Gateway", func() {
			gateway := &gateway.Gateway{}
			gw_out := BuildGatewayResource(gateway)
			Expect(gw_out).NotTo(BeNil())
		})
	})

	Context("When reconciling a parent Gateway", func() {
		ctx := context.Background()
		gw := &gateway.Gateway{}
		_ = yaml.Unmarshal([]byte(gatewayManifest), gw)

		It("should create gateway", func() {
			Expect(k8sClient.Create(ctx, gw)).Should(Succeed())
			Expect(string(gw.Spec.GatewayClassName)).To(Equal("default"))
		})

		childGateway := &gateway.Gateway{}
		It("Should create a child gateway", func() {
			name := fmt.Sprintf("%s-%s", gw.ObjectMeta.Name, "istio")

			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: "default"}, childGateway)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
		})

		It("Should set owner reference to enable garbage collection", func() {
			var t bool = true
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
