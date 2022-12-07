package controllers

import (
	//"context"
	//"reflect"
	//"time"

	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/yaml"

	//"k8s.io/apimachinery/pkg/types"
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

	Context("When building Gateway resource from input Gateway", func() {
		It("Should return a new Gateway", func() {
			//By("By creating a new CronJob")
			//ctx := context.Background()
			gateway := &gateway.Gateway{}
			gw_out := BuildGatewayResource(gateway)
			Expect(gw_out).NotTo(BeNil())
		})
	})

	Context("When applying a parent Gateway", func() {
		It("Should create a child gateway", func() {
			ctx := context.Background()
			gateway := &gateway.Gateway{}
			_ = yaml.Unmarshal([]byte(gatewayManifest), gateway)
			Expect(k8sClient.Create(ctx, gateway)).Should(Succeed())
			Expect(string(gateway.Spec.GatewayClassName)).To(Equal("default"))
		})
	})
})
