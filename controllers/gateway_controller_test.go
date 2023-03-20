/*
Copyright 2023 TV 2 DANMARK A/S

Licensed under the Apache License, Version 2.0 (the "License") with the
following modification to section 6. Trademarks:

Section 6. Trademarks is deleted and replaced by the following wording:

6. Trademarks. This License does not grant permission to use the trademarks and
trade names of TV 2 DANMARK A/S, including but not limited to the TV 2® logo and
word mark, except (a) as required for reasonable and customary use in describing
the origin of the Work, e.g. as described in section 4(c) of the License, and
(b) to reproduce the content of the NOTICE file. Any reference to the Licensor
must be made by making a reference to "TV 2 DANMARK A/S", written in capitalized
letters as in this example, unless the format in which the reference is made,
requires lower case letters.

You may not use this software except in compliance with the License and the
modifications set out above.

You may obtain a copy of the license at:

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gwcapi "github.com/tv2-oss/gateway-controller/apis/gateway.tv2.dk/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	gatewayapi "sigs.k8s.io/gateway-api/apis/v1beta1"

	"sigs.k8s.io/gateway-api/conformance/utils/kubernetes"
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
    status:
      template: |
        addresses:
          {{ toYaml .Resources.childGateway.status.addresses | nindent 2}}
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
		gwc  *gatewayapi.GatewayClass
		gwcb *gwcapi.GatewayClassBlueprint
		ctx  context.Context
	)

	BeforeEach(func() {
		gwc = &gatewayapi.GatewayClass{}
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
		var childGateway, gw *gatewayapi.Gateway

		BeforeEach(func() {
			gw = &gatewayapi.Gateway{}
			childGateway = &gatewayapi.Gateway{}
			Expect(yaml.Unmarshal([]byte(gatewayManifest), gw)).To(Succeed())
		})

		It("Should lifecycle correctly", func() {

			By("Creating the gateway")
			Expect(k8sClient.Create(ctx, gw)).Should(Succeed())
			Expect(string(gw.Spec.GatewayClassName)).To(Equal("default"))

			gwNN := types.NamespacedName{Name: gw.ObjectMeta.Name, Namespace: gw.ObjectMeta.Namespace}
			gwChildNN := types.NamespacedName{Name: gw.ObjectMeta.Name + "-istio", Namespace: gw.ObjectMeta.Namespace}

			By("Creating the child gateway")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, gwChildNN, childGateway)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Setting the owner reference to enable garbage collection")
			t := true
			expectedOwnerReference := metav1.OwnerReference{
				Kind:               "Gateway",
				APIVersion:         "gateway.networking.k8s.io/v1beta1",
				UID:                gw.ObjectMeta.GetUID(),
				Name:               gw.ObjectMeta.Name,
				Controller:         &t,
				BlockOwnerDeletion: &t,
			}
			Expect(childGateway.ObjectMeta.OwnerReferences).To(ContainElement(expectedOwnerReference))

			By("Updating conditions")

			// Set child status to not ready
			Expect(setGatewayStatus(gwChildNN, &metav1.Condition{
				Type:   string(gatewayapi.GatewayConditionReady),
				Status: metav1.ConditionFalse,
				Reason: string(gatewayapi.GatewayReasonReady)}, nil)).Should(Succeed())
			time.Sleep(5 * time.Second) // Ensure that controllers cache is updated and we can use 'Consistently' below

			gwRead := &gatewayapi.Gateway{}
			Consistently(func() bool {
				err := k8sClient.Get(ctx, gwNN, gwRead)
				if err != nil {
					return false
				}
				GinkgoT().Logf("gwRead cond: %+v\n", gwRead.Status.Conditions)
				if kubernetes.ConditionsHaveLatestObservedGeneration(gwRead, gwRead.Status.Conditions) != nil {
					return false
				}
				if !conditionStateIs(gwRead, "Ready", metav1.ConditionFalse) ||
					!conditionStateIs(gwRead, "Programmed", metav1.ConditionTrue) {
					return false
				}
				return true
			}, 5*time.Second, interval).Should(BeTrue())

			// Set child status to ready
			addrType := gatewayapi.IPAddressType
			Expect(setGatewayStatus(gwChildNN, &metav1.Condition{
				Type:   string(gatewayapi.GatewayConditionReady),
				Status: metav1.ConditionTrue,
				Reason: string(gatewayapi.GatewayReasonReady)},
				&gatewayapi.GatewayAddress{Type: &addrType, Value: "4.5.6.7"})).Should(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, gwNN, gwRead)
				if err != nil {
					return false
				}
				GinkgoT().Logf("gwRead cond: %+v\n", gwRead.Status.Conditions)
				if kubernetes.ConditionsHaveLatestObservedGeneration(gwRead, gwRead.Status.Conditions) != nil {
					return false
				}
				if !conditionStateIs(gwRead, "Ready", metav1.ConditionTrue) ||
					!conditionStateIs(gwRead, "Programmed", metav1.ConditionTrue) {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

		})

		It("Should update inter resource-references", func() {

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

func conditionStateIs(gw *gatewayapi.Gateway, condType string, status metav1.ConditionStatus) bool {
	for _, cond := range gw.Status.Conditions {
		if cond.Type == condType && cond.Status == status {
			return true
		}
	}

	return false
}

func setGatewayStatus(nn types.NamespacedName, newCondition *metav1.Condition, address *gatewayapi.GatewayAddress) error {
	gw := &gatewayapi.Gateway{}

	if err := k8sClient.Get(context.TODO(), nn, gw); err != nil {
		return err
	}

	if newCondition != nil {
		newCondition.ObservedGeneration = gw.ObjectMeta.Generation
		meta.SetStatusCondition(&gw.Status.Conditions, *newCondition)
		GinkgoT().Logf("update gw: %+v conditions: %+v\n", gw, newCondition)
	}
	if address != nil {
		gw.Status.Addresses = []gatewayapi.GatewayAddress{}
		gw.Status.Addresses = append(gw.Status.Addresses, *address)
	}

	if err := k8sClient.Status().Update(context.TODO(), gw); err != nil {
		return err
	}
	return nil
}
