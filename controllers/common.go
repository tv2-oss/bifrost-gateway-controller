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
	"errors"
	"fmt"

	"golang.org/x/exp/maps"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayapi "sigs.k8s.io/gateway-api/apis/v1beta1"

	gwcapi "github.com/tv2-oss/gateway-controller/apis/gateway.tv2.dk/v1alpha1"
	selfapi "github.com/tv2-oss/gateway-controller/pkg/api"
)

var (
	// The namespace the controller will watch for global policies
	ControllerNamespace string
)

type ControllerClient interface {
	Client() client.Client
	Scheme() *runtime.Scheme
}

type ControllerDynClient interface {
	ControllerClient
	DynamicClient() dynamic.Interface
}

func isOurGatewayClass(gwc *gatewayapi.GatewayClass) bool {
	return gwc.Spec.ControllerName == selfapi.SelfControllerName
}

func lookupGatewayClass(ctx context.Context, r ControllerClient, name gatewayapi.ObjectName) (*gatewayapi.GatewayClass, error) {
	var gwc gatewayapi.GatewayClass
	if err := r.Client().Get(ctx, types.NamespacedName{Name: string(name)}, &gwc); err != nil {
		return nil, err
	}

	return &gwc, nil
}

func lookupGatewayClassBlueprint(ctx context.Context, r ControllerClient, gwc *gatewayapi.GatewayClass) (*gwcapi.GatewayClassBlueprint, error) {
	if gwc.Spec.ParametersRef == nil {
		return nil, errors.New("GatewayClass without parameters")
	}

	if gwc.Spec.ParametersRef.Kind != "GatewayClassBlueprint" || gwc.Spec.ParametersRef.Group != "gateway.tv2.dk" {
		return nil, errors.New("parameter kind/group is not a valid GatewayClassBlueprint")
	}

	var gwcb gwcapi.GatewayClassBlueprint
	if err := r.Client().Get(ctx, types.NamespacedName{Name: gwc.Spec.ParametersRef.Name}, &gwcb); err != nil {
		return nil, err
	}

	return &gwcb, nil
}

// Lookup values from GatewayClassConfig/GatewayConfig CRDs (eventually) and combine using precedence rules:
// - Values from GatewayClassBlueprint (highest precedence)
// - Values from GatewayClassConfig in controller namespace
// - Values from GatewayClassConfig in Gateway/HTTPRoute local namespace (lowest precedence)
// - Values from GatewayConfig in Gateway/HTTPRoute local namespace (lowest precedence)
// Note, defaults are processed top-to-bottom, while overrides are bottom-to-top (see GEP-713)
func lookupValues(ctx context.Context, r ControllerClient, gatewayClassName string, gwcb *gwcapi.GatewayClassBlueprint, _ /*namespace*/ string) (map[string]any, error) {
	values := map[string]any{}
	var err error

	// Helper to parse and merge-overwrite values
	mergeValues := func(src *apiextensionsv1.JSON, existing map[string]any) (map[string]any, error) {
		if src != nil {
			newvals := map[string]any{}
			if err = json.Unmarshal(src.Raw, &newvals); err != nil {
				return nil, fmt.Errorf("cannot unmarshal values: %w", err)
			}
			maps.Copy(existing, newvals) // FIXME: This overwrites at top-level. GEP-713 specifies deep merge
		}
		return existing, nil
	}

	var gwccl gwcapi.GatewayClassConfigList
	err = r.Client().List(ctx, &gwccl, client.InNamespace(ControllerNamespace))
	if err != nil {
		return nil, err
	}

	// Blueprint default values are first
	if values, err = mergeValues(gwcb.Spec.Values.Default, values); err != nil {
		return nil, fmt.Errorf("while processing blueprint default values for gatewayclass %s: %w", gatewayClassName, err)
	}
	// FIXME: Process other defaults

	// Process overrides in increasing order of precedence

	// FIXME: Use GatewayConfig and GatewayClassconfig from parent resource namespace (lowest override precedence)

	// GatewayClassConfig from controller namesapce
	for idx := range gwccl.Items {
		gwcc := &gwccl.Items[idx]
		if gwcc.Spec.TargetRef.Kind == "GatewayClass" &&
			gwcc.Spec.TargetRef.Group == gatewayapi.GroupName &&
			string(gwcc.Spec.TargetRef.Name) == gatewayClassName {
			// gwcc targets gatewayclass
			if values, err = mergeValues(gwcc.Spec.Override, values); err != nil {
				return nil, fmt.Errorf("while processing %s: %w", gwcc.Name, err)
			}
		}
	}

	// Blueprint override values are last since they have highest precedence
	if values, err = mergeValues(gwcb.Spec.Values.Override, values); err != nil {
		return nil, fmt.Errorf("while processing blueprint override values for gatewayclass %s: %w", gatewayClassName, err)
	}

	return values, nil
}

func lookupGateway(ctx context.Context, r ControllerClient, name gatewayapi.ObjectName, namespace string) (*gatewayapi.Gateway, error) {
	var gw gatewayapi.Gateway
	if err := r.Client().Get(ctx, types.NamespacedName{Name: string(name), Namespace: namespace}, &gw); err != nil {
		return nil, err
	}
	return &gw, nil
}

// From an unstructured object, lookup GVR and whether resource is namespaced
func unstructuredToGVR(r ControllerClient, u *unstructured.Unstructured) (*schema.GroupVersionResource, bool, error) {
	gv, err := schema.ParseGroupVersion(u.GetAPIVersion())
	if err != nil {
		return nil, false, err
	}

	gk := schema.GroupKind{
		Group: gv.Group,
		Kind:  u.GetKind(),
	}

	mapping, err := r.Client().RESTMapper().RESTMapping(gk, gv.Version)
	if err != nil {
		return nil, false, err
	}

	isNamespaced := false
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		isNamespaced = true
	}

	return &schema.GroupVersionResource{
		Group:    gv.Group,
		Version:  gv.Version,
		Resource: mapping.Resource.Resource,
	}, isNamespaced, nil
}

// Apply an unstructured object using server-side apply
func patchUnstructured(ctx context.Context, r ControllerDynClient, us *unstructured.Unstructured,
	gvr *schema.GroupVersionResource, namespace *string) error {
	jsonData, err := json.Marshal(us.Object)
	if err != nil {
		return fmt.Errorf("unable to marshal unstructured to json %w", err)
	}

	force := true

	if namespace != nil {
		dynamicClient := r.DynamicClient().Resource(*gvr).Namespace(*namespace)
		_, err = dynamicClient.Patch(ctx, us.GetName(), types.ApplyPatchType, jsonData, metav1.PatchOptions{
			Force:        &force,
			FieldManager: string(selfapi.SelfControllerName),
		})
	} else {
		dynamicClient := r.DynamicClient().Resource(*gvr)
		_, err = dynamicClient.Patch(ctx, us.GetName(), types.ApplyPatchType, jsonData, metav1.PatchOptions{
			Force:        &force,
			FieldManager: string(selfapi.SelfControllerName),
		})
	}

	return err
}
