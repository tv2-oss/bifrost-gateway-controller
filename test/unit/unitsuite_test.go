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

package unitsuite

import (
	"embed"
	"strings"
	"testing"

	"github.com/tv2/cloud-gateway-controller/test/unit/tests"

	"sigs.k8s.io/gateway-api/apis/v1alpha2"
	"sigs.k8s.io/gateway-api/apis/v1beta1"
	"sigs.k8s.io/gateway-api/conformance"
	"sigs.k8s.io/gateway-api/conformance/utils/flags"
	"sigs.k8s.io/gateway-api/conformance/utils/suite"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

//go:embed tests/*.yaml base/*.yaml
var Manifests embed.FS

func TestUnit(t *testing.T) {
	cfg, err := config.GetConfig()
	if err != nil {
		t.Fatalf("Error loading Kubernetes config: %v", err)
	}
	client, err := client.New(cfg, client.Options{})
	if err != nil {
		t.Fatalf("Error initializing Kubernetes client: %v", err)
	}
	v1alpha2.AddToScheme(client.Scheme())
	v1beta1.AddToScheme(client.Scheme())

	supportedFeatures := parseSupportedFeatures(*flags.SupportedFeatures)
	exemptFeatures := parseSupportedFeatures(*flags.ExemptFeatures)
	for feature := range exemptFeatures {
		supportedFeatures[feature] = false
	}

	t.Logf("Running conformance tests with %s GatewayClass\n cleanup: %t\n debug: %t\n supported features: [%v]\n exempt features: [%v]\n num tests: %v",
		*flags.GatewayClassName, *flags.CleanupBaseResources, *flags.ShowDebug, *flags.SupportedFeatures, *flags.ExemptFeatures, len(tests.UnitTests))

	conformance.Manifests = Manifests
	suite := suite.New(suite.Options{
		Client:               client,
		GatewayClassName:     *flags.GatewayClassName,
		Debug:                *flags.ShowDebug,
		CleanupBaseResources: *flags.CleanupBaseResources,
		SupportedFeatures:    supportedFeatures,
	})
	suite.Setup(t)
	suite.Run(t, tests.UnitTests)
}

// parseSupportedFeatures parses flag arguments and converts the string to
// map[suite.SupportedFeature]bool
func parseSupportedFeatures(f string) map[suite.SupportedFeature]bool {
	res := map[suite.SupportedFeature]bool{}
	for _, value := range strings.Split(f, ",") {
		res[suite.SupportedFeature(value)] = true
	}
	return res
}
