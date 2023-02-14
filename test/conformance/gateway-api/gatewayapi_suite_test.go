package conformance_test

import (
	"os"
	"strings"
	"testing"

	"sigs.k8s.io/gateway-api/apis/v1alpha2"
	"sigs.k8s.io/gateway-api/apis/v1beta1"
	"sigs.k8s.io/gateway-api/conformance/tests"
	"sigs.k8s.io/gateway-api/conformance/utils/flags"
	"sigs.k8s.io/gateway-api/conformance/utils/suite"

	"k8s.io/apimachinery/pkg/util/sets"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func TestConformance(t *testing.T) {
	if os.Getenv("USE_EXISTING_CLUSTER") != "true" {
		t.Skipf("Skipping conformance tests - requires an external cluster")
	}

	cfg, err := config.GetConfig()
	if err != nil {
		t.Fatalf("Error loading Kubernetes config: %v", err)
	}
	cl, err := client.New(cfg, client.Options{})
	if err != nil {
		t.Fatalf("Error initializing Kubernetes client: %v", err)
	}
	err = v1alpha2.AddToScheme(cl.Scheme())
	if err != nil {
		t.Fatalf("Error adding api v1alpha2: %v", err)
	}
	err = v1beta1.AddToScheme(cl.Scheme())
	if err != nil {
		t.Fatalf("Error adding api v1beta1: %v", err)
	}

	supportedFeatures := parseSupportedFeatures(*flags.SupportedFeatures)
	exemptFeatures := parseSupportedFeatures(*flags.ExemptFeatures)
	for feature := range exemptFeatures {
		supportedFeatures.Delete(feature)
	}

	t.Logf("Running conformance tests with %s GatewayClass\n cleanup: %t\n debug: %t\n supported features: [%v]\n exempt features: [%v]\n num tests: %v",
		*flags.GatewayClassName, *flags.CleanupBaseResources, *flags.ShowDebug, *flags.SupportedFeatures, *flags.ExemptFeatures, len(tests.ConformanceTests))

	cSuite := suite.New(suite.Options{
		Client:               cl,
		GatewayClassName:     *flags.GatewayClassName,
		Debug:                *flags.ShowDebug,
		CleanupBaseResources: *flags.CleanupBaseResources,
		SupportedFeatures:    supportedFeatures,
	})
	cSuite.Setup(t)
	cSuite.Run(t, tests.ConformanceTests)
}

func parseSupportedFeatures(f string) sets.Set[suite.SupportedFeature] {
	res := sets.Set[suite.SupportedFeature]{}
	for _, value := range strings.Split(f, ",") {
		res.Insert(suite.SupportedFeature(value))
	}
	return res
}
