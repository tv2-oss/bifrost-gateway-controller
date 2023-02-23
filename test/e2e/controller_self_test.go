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
	"fmt"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayapi "sigs.k8s.io/gateway-api/apis/v1beta1"

	cgcapi "github.com/tv2/cloud-gateway-controller/apis/cgc.tv2.dk/v1alpha1"
	cgwctlapi "github.com/tv2/cloud-gateway-controller/pkg/api"
)

const gatewayclassManifest string = `
apiVersion: gateway.networking.k8s.io/v1beta1
kind: GatewayClass
metadata:
  name: cloud-gw
spec:
  controllerName: "github.com/tv2/cloud-gateway-controller"
  parametersRef:
    group: v1alpha1
    kind: GatewayClassParameters
    name: default-gateway-class`

const gwClassParametersManifest string = `
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
            {{ if .namespace }}
            namespace: {{ .namespace }}
            {{ end }}
          {{ end }}
          rules:
          {{ toYaml .HTTPRoute.spec.rules | nindent 4 }}`

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
		fixmeExtendedTimeout = time.Second * 20 // This should go away when the normalization refactoring is implemented
		interval             = time.Millisecond * 250
		timeout              = time.Second * 10
	)

	Context("When a GatewayClass we own is created", func() {
		It("Should be marked as accepted", func() {
			By("Setting a condition")
			ctx := context.Background()

			gwc := &gatewayapi.GatewayClass{}
			err := yaml.Unmarshal([]byte(gatewayclassManifest), gwc)
			Expect(err).To(Succeed())
			Expect(k8sClient.Create(ctx, gwc)).To(Succeed())

			// We deliberately sleep here to make the controller initially see the GatewayClass without its corresponding parameters
			time.Sleep(5 * time.Second)

			gcp := &cgcapi.GatewayClassParameters{}
			Expect(yaml.Unmarshal([]byte(gwClassParametersManifest), gcp)).To(Succeed())
			Expect(k8sClient.Create(ctx, gcp)).Should(Succeed())

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
			}, fixmeExtendedTimeout, interval).Should(BeTrue())

			Expect(k8sClient.Delete(ctx, gcp)).To(Succeed())
			Expect(k8sClient.Delete(ctx, gwc)).To(Succeed())
		})
	})
})

var _ = Describe("Gateway addresses", func() {

	const (
		externalDNSTimeout   = time.Second * 120
		interval             = time.Millisecond * 250
		timeout              = time.Second * 10
		fixmeExtendedTimeout = time.Second * 20 // This should go away when the normalization refactoring is implemented
	)
	var (
		ip4AddressRe *regexp.Regexp
		hostnameRe   *regexp.Regexp
	)

	BeforeEach(func() {
		ip4AddressRe = regexp.MustCompile(`^((25[0-5]|(2[0-4]|1\d|[1-9]|)\d)\.?\b){4}$`)
		hostnameRe = regexp.MustCompile(`^([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])(\.([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9]))*$`)
		gcp := &cgcapi.GatewayClassParameters{}
		Expect(yaml.Unmarshal([]byte(gwClassParametersManifest), gcp)).To(Succeed())
		Expect(k8sClient.Create(ctx, gcp)).Should(Succeed())

		gwc := &gatewayapi.GatewayClass{}
		err := yaml.Unmarshal([]byte(gatewayclassManifest), gwc)
		ctx = context.Background()
		Expect(err).To(Succeed())
		Expect(k8sClient.Create(ctx, gwc)).To(Succeed())

		DeferCleanup(func() {
			Expect(k8sClient.Delete(ctx, gwc)).To(Succeed())
			Expect(k8sClient.Delete(ctx, gcp)).To(Succeed())
		})
	})

	Context("When a Gateway/HTTPRoute is created", func() {
		It("Should be accepted by the controller", func() {
			By("Assigning an address to the Gateway")

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
				GinkgoT().Logf("gwRead: %+v", gwRead)
				if err != nil ||
					len(gwRead.Status.Addresses) != 1 ||
					(*gwRead.Status.Addresses[0].Type == gatewayapi.IPAddressType && !ip4AddressRe.MatchString(gwRead.Status.Addresses[0].Value)) ||
					(*gwRead.Status.Addresses[0].Type == gatewayapi.HostnameAddressType && !hostnameRe.MatchString(gwRead.Status.Addresses[0].Value)) {
					return false
				}
				return true
			}, fixmeExtendedTimeout, interval).Should(BeTrue())

			By("Assigning status and address such that external-dns accepts and propagates the address")
			Eventually(func() bool {
				stdout := new(bytes.Buffer)
				err := ExecCmdInPodBySelector(k8sClient, restClient, cfg, client.MatchingLabels{"app": "multitool"}, "default",
					fmt.Sprintf("dig @coredns-test-only-coredns %s +short", *gw.Spec.Listeners[0].Hostname),
					nil, stdout, nil)
				if err != nil {
					return false
				}
				foundDNSLookup := strings.TrimRight(stdout.String(), "\n")
				GinkgoT().Logf("foundDNSLookup: %s, gateway has %s", foundDNSLookup, gwRead.Status.Addresses[0].Value)
				return gwRead.Status.Addresses[0].Value == foundDNSLookup
			}, externalDNSTimeout, interval).Should(BeTrue())

			By("Setting a status condition on the HTTPRoute")
			lookupKey = types.NamespacedName{Name: rt.ObjectMeta.Name, Namespace: rt.ObjectMeta.Namespace}
			rtRead := &gatewayapi.HTTPRoute{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, rtRead)
				if err != nil ||
					len(rtRead.Status.RouteStatus.Parents) != 1 ||
					len(rtRead.Status.RouteStatus.Parents[0].Conditions) != 1 ||
					string(rtRead.Status.RouteStatus.Parents[0].ParentRef.Name) != gw.ObjectMeta.Name ||
					// Namespace is optional, if defined it must match HTTPRoute namespace
					(rtRead.Status.RouteStatus.Parents[0].ParentRef.Namespace != nil && string(*rtRead.Status.RouteStatus.Parents[0].ParentRef.Namespace) != gw.ObjectMeta.Namespace) ||
					rtRead.Status.RouteStatus.Parents[0].ControllerName != cgwctlapi.SelfControllerName ||
					rtRead.Status.RouteStatus.Parents[0].Conditions[0].Type != string(gatewayapi.RouteConditionAccepted) ||
					rtRead.Status.RouteStatus.Parents[0].Conditions[0].Status != "True" {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			Expect(k8sClient.Delete(ctx, rt)).To(Succeed())
			Expect(k8sClient.Delete(ctx, gw)).To(Succeed())
		})
	})
})
