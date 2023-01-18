/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// This file contain tests that are not e2e per se, however the e2e
// tests assume an external cluster with the controller deployed using
// 'production' means, e.g. a Helm chart and thus we have some basic
// tests here to validate the deployment of the controller on the
// external cluster.

package e2esuite

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayapi "sigs.k8s.io/gateway-api/apis/v1beta1"
)

const gatewayclassManifest string = `
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

const gwClassConfigMapManifest string = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: cloud-gw-params
  namespace: default
data:
  tier2GatewayClass: istio`

// example.com does resolve to an IP address so it is not ideal for testing
const gatewayManifest string = `
apiVersion: gateway.networking.k8s.io/v1beta1
kind: Gateway
metadata:
  name: foo-gateway
  namespace: default
spec:
  gatewayClassName: cloud-gw
  listeners:
  - name: prod-web
    port: 80
    protocol: HTTP
    hostname: example-foo4567.com`

const httprouteManifest string = `
apiVersion: gateway.networking.k8s.io/v1beta1
kind: HTTPRoute
metadata:
  name: foo-site
  namespace: default
spec:
  parentRefs:
  - kind: Gateway
    name: foo-gateway
  rules:
  - backendRefs:
    - name: foo-site
      port: 80`

var _ = Describe("GatewayClass", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When a gatewayclass we own is created", func() {
		It("Should be marked as accepted", func() {
			By("Setting a condition")
			ctx := context.Background()

			gwc := &gatewayapi.GatewayClass{}
			err := yaml.Unmarshal([]byte(gatewayclassManifest), gwc)
			Expect(err).To(Succeed())
			Expect(k8sClient.Create(ctx, gwc)).To(Succeed())

			cm := &corev1.ConfigMap{}
			Expect(yaml.Unmarshal([]byte(gwClassConfigMapManifest), cm)).To(Succeed())
			Expect(k8sClient.Create(ctx, cm)).To(Succeed())

			lookupKey := types.NamespacedName{Name: gwc.ObjectMeta.Name, Namespace: ""}
			gwcRead := &gatewayapi.GatewayClass{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, gwcRead)
				if err != nil ||
					len(gwcRead.Status.Conditions) == 0 ||
					gwcRead.Status.Conditions[0].Type != string(gatewayapi.GatewayClassConditionStatusAccepted) ||
					gwcRead.Status.Conditions[0].Status != "True" {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			Expect(k8sClient.Delete(ctx, cm)).To(Succeed())
			Expect(k8sClient.Delete(ctx, gwc)).To(Succeed())
		})
	})
})

var _ = Describe("Gateway addresses", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)
	var (
		gwc          *gatewayapi.GatewayClass
		cm           *corev1.ConfigMap
		ip4AddressRe *regexp.Regexp
		hostnameRe   *regexp.Regexp
	)

	BeforeEach(func() {
		ip4AddressRe, _ = regexp.Compile(`^((25[0-5]|(2[0-4]|1\d|[1-9]|)\d)\.?\b){4}$`)
		hostnameRe, _ = regexp.Compile(`^([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])(\.([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9]))*$`)
		gwc = &gatewayapi.GatewayClass{}
		err := yaml.Unmarshal([]byte(gatewayclassManifest), gwc)
		ctx := context.Background()
		Expect(err).To(Succeed())
		Expect(k8sClient.Create(ctx, gwc)).To(Succeed())

		cm = &corev1.ConfigMap{}
		Expect(yaml.Unmarshal([]byte(gwClassConfigMapManifest), cm)).To(Succeed())
		Expect(k8sClient.Create(ctx, cm)).To(Succeed())
	})

	AfterEach(func() {
		ctx := context.Background()
		Expect(k8sClient.Delete(ctx, gwc)).To(Succeed())
		Expect(k8sClient.Delete(ctx, cm)).To(Succeed())
	})

	Context("When a gateway/httproute is created", func() {
		It("They be marked as ready/accepted", func() {
			By("Setting a condition")

			ctx := context.Background()
			gw := &gatewayapi.Gateway{}
			Expect(yaml.Unmarshal([]byte(gatewayManifest), gw)).To(Succeed())
			Expect(k8sClient.Create(ctx, gw)).To(Succeed())

			rt := &gatewayapi.HTTPRoute{}
			Expect(yaml.Unmarshal([]byte(httprouteManifest), rt)).To(Succeed())
			Expect(k8sClient.Create(ctx, rt)).To(Succeed())

			lookupKey := types.NamespacedName{Name: gw.ObjectMeta.Name, Namespace: gw.ObjectMeta.Namespace}
			gwRead := &gatewayapi.Gateway{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, gwRead)
				if err != nil ||
					len(gwRead.Status.Addresses) != 1 ||
					(*gwRead.Status.Addresses[0].Type == gatewayapi.IPAddressType && !ip4AddressRe.MatchString(gwRead.Status.Addresses[0].Value)) ||
					(*gwRead.Status.Addresses[0].Type == gatewayapi.HostnameAddressType && !hostnameRe.MatchString(gwRead.Status.Addresses[0].Value)) {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			lookupKey = types.NamespacedName{Name: rt.ObjectMeta.Name, Namespace: rt.ObjectMeta.Namespace}
			rtRead := &gatewayapi.HTTPRoute{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, rtRead)
				if err != nil ||
					len(rtRead.Status.RouteStatus.Parents) != 1 ||
					len(rtRead.Status.RouteStatus.Parents[0].Conditions) != 1 ||
					string(rtRead.Status.RouteStatus.Parents[0].ParentRef.Name) != gw.ObjectMeta.Name ||
					string(*rtRead.Status.RouteStatus.Parents[0].ParentRef.Namespace) != gw.ObjectMeta.Namespace ||
					//rtRead.Status.RouteStatus.Parents[0].ControllerName != "xxx"
					rtRead.Status.RouteStatus.Parents[0].Conditions[0].Type != string(gatewayapi.RouteConditionAccepted) ||
					rtRead.Status.RouteStatus.Parents[0].Conditions[0].Status != "True" {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			stdout := new(bytes.Buffer)
			ExecCmdInPodBySelector(k8sClient, restClient, cfg, client.MatchingLabels{"app": "multitool"}, "default",
				"dig @coredns-test-only-coredns example-foo4567.com +short",
				nil, stdout, nil)

			foundDnsLookup := strings.TrimRight(string(stdout.Bytes()), "\n")
			Expect(gwRead.Status.Addresses[0].Value).To(Equal(foundDnsLookup))

			Expect(k8sClient.Delete(ctx, rt)).To(Succeed())
			Expect(k8sClient.Delete(ctx, gw)).To(Succeed())
		})
	})
})
