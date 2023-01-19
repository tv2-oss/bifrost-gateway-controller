package controllers

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	gateway "sigs.k8s.io/gateway-api/apis/v1beta1"
)

const cloudGwGatewayClassManifest string = `
apiVersion: gateway.networking.k8s.io/v1beta1
kind: GatewayClass
metadata:
  name: cloud-gw
spec:
  controllerName: "github.com/tv2/cloud-gateway-controller"
  parametersRef:
    group: v1
    kind: ConfigMap
    name: cloud-gw-params
    namespace: default`

const gatewayClassManifestInvalid string = `
apiVersion: gateway.networking.k8s.io/v1beta1
kind: GatewayClass
metadata:
  name: cloud-gw-invalid
spec:
  controllerName: "github.com/tv2/cloud-gateway-controller"`

const gatewayClassManifestNotOur string = `
apiVersion: gateway.networking.k8s.io/v1beta1
kind: GatewayClass
metadata:
  name: not-our-gatewayclass
spec:
  controllerName: "github.com/acme/cloud-gateway-controller"`

const gwClassConfigMapManifest string = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: cloud-gw-params
  namespace: default
data:
  tier2GatewayClass: istio`

var _ = Describe("GatewayClass controller", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	var (
		gwcIn, gwc *gateway.GatewayClass
		ctx        context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		gwcIn = &gateway.GatewayClass{}
		gwc = &gateway.GatewayClass{}
	})

	When("A gatewayclass we own is created", func() {

		It("Should be marked as accepted", func() {

			err := yaml.Unmarshal([]byte(gatewayclass_manifest), gwcIn)
			Expect(err).Should(Succeed())
			Expect(k8sClient.Create(ctx, gwcIn)).Should(Succeed())

			cm := &corev1.ConfigMap{}
			Expect(yaml.Unmarshal([]byte(gwClassConfigMapManifest), cm)).To(Succeed())
			Expect(k8sClient.Create(ctx, cm)).Should(Succeed())

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
		})
	})

	When("An invalid gatewayclass we own is created", func() {
		It("Should be marked as invalid", func() {

			err := yaml.Unmarshal([]byte(gatewayclass_manifest_invalid), gwcIn)
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
		})
	})

	When("A gatewayclass we do not own is created", func() {
		It("Should not be marked as accepted", func() {

			err := yaml.Unmarshal([]byte(gatewayclass_manifest_not_our), gwcIn)
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
		})
	})
})
