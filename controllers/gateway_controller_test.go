package controllers

import (
	//"context"
	//"reflect"
	//"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	//v1 "k8s.io/api/core/v1"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/apimachinery/pkg/types"
	gateway "sigs.k8s.io/gateway-api/apis/v1beta1"
)

var _ = Describe("CronJob controller", func() {

	Context("When building Gateway resource from input Gateway", func() {
		It("Should return a new Gateway", func() {
			//By("By creating a new CronJob")
			//ctx := context.Background()
			gateway := &gateway.Gateway{}
			gw_out := BuildGatewayResource(gateway)
			Expect(gw_out).NotTo(BeNil())
		})
	})
})
