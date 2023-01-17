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

package e2e_suite

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	gatewayapi "sigs.k8s.io/gateway-api/apis/v1beta1"
)

const gatewayclass_manifest string = `
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

var _ = Describe("GatewayClass controller", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When a gatewayclass we own is created", func() {
		It("Should be marked as accepted", func() {
			By("Setting a condition")
			ctx := context.Background()

			gwc_in := &gatewayapi.GatewayClass{}
			err := yaml.Unmarshal([]byte(gatewayclass_manifest), gwc_in)
			Expect(err).Should(Succeed())
			Expect(k8sClient.Create(ctx, gwc_in)).Should(Succeed())

			cm := &corev1.ConfigMap{}
			Expect(yaml.Unmarshal([]byte(gwClassConfigMapManifest), cm)).To(Succeed())
			Expect(k8sClient.Create(ctx, cm)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: gwc_in.ObjectMeta.Name, Namespace: ""}
			gwc := &gatewayapi.GatewayClass{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, gwc)
				if err != nil ||
					gwc.Status.Conditions[0].Type != string(gatewayapi.GatewayClassConditionStatusAccepted) ||
					gwc.Status.Conditions[0].Status != "True" {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			Expect(k8sClient.Delete(ctx, cm)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, gwc_in)).Should(Succeed())
		})
	})
})
