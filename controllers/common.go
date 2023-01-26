package controllers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"html/template"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayapi "sigs.k8s.io/gateway-api/apis/v1beta1"

	selfapi "github.com/tv2/cloud-gateway-controller/pkg/api"
)

type templateValues struct {
	Gateway *gatewayapi.Gateway
}

type Controller interface {
	GetClient() client.Client
	DynamicClient() dynamic.Interface
	Scheme() *runtime.Scheme
}

func isOurGatewayClass(gwc *gatewayapi.GatewayClass) bool {
	return gwc.Spec.ControllerName == selfapi.SelfControllerName
}

func lookupGatewayClass(ctx context.Context, r Controller, name gatewayapi.ObjectName) (*gatewayapi.GatewayClass, error) {
	var gwc gatewayapi.GatewayClass
	if err := r.GetClient().Get(ctx, types.NamespacedName{Name: string(name)}, &gwc); err != nil {
		return nil, err
	}

	return &gwc, nil
}

func lookupGatewayClassParameters(ctx context.Context, r Controller, gwc *gatewayapi.GatewayClass) (*corev1.ConfigMap, error) {
	if gwc.Spec.ParametersRef == nil {
		return nil, errors.New("GatewayClass without parameters")
	}

	// FIXME: More validation...
	if gwc.Spec.ParametersRef.Kind != "ConfigMap" {
		return nil, errors.New("parameter Kind is not a ConfigMap")
	}

	var cm corev1.ConfigMap
	if err := r.GetClient().Get(ctx, types.NamespacedName{Name: gwc.Spec.ParametersRef.Name, Namespace: string(*gwc.Spec.ParametersRef.Namespace)}, &cm); err != nil {
		return nil, err
	}

	// FIXME: Validate ConfigMap

	return &cm, nil
}

func lookupGateway(ctx context.Context, r Controller, name gatewayapi.ObjectName, namespace string) (*gatewayapi.Gateway, error) {
	var gw gatewayapi.Gateway
	if err := r.GetClient().Get(ctx, types.NamespacedName{Name: string(name), Namespace: namespace}, &gw); err != nil {
		return nil, err
	}
	return &gw, nil
}

func parseTemplate(r Controller, gwParent *gatewayapi.Gateway, configMap *corev1.ConfigMap, configMapKey string) (*unstructured.Unstructured, error) {
	var buffer bytes.Buffer
	templateData, found := configMap.Data[configMapKey]

	if !found {
		return nil, errors.New("key not found in ConfigMap")
	}

	tmpl, err := template.New("resourceTemplate").Parse(templateData)
	if err != nil {
		return nil, err
	}

	err = tmpl.Execute(io.Writer(&buffer), &templateValues{gwParent})
	if err != nil {
		return nil, err
	}

	rawResource := map[string]any{}
	err = yaml.Unmarshal(buffer.Bytes(), &rawResource)
	if err != nil {
		return nil, err
	}

	unstruct := &unstructured.Unstructured{Object: rawResource}

	if err := ctrl.SetControllerReference(gwParent, unstruct, r.Scheme()); err != nil {
		return nil, err
	}

	return unstruct, nil
}

// TODO This function needs explanation
func unstructuredToGVR(r Controller, u *unstructured.Unstructured) (*schema.GroupVersionResource, error) {
	// TODO: Why parse GroupVersion?
	gv, err := schema.ParseGroupVersion(u.GetAPIVersion())
	if err != nil {
		return nil, err
	}

	gk := schema.GroupKind{
		Group: gv.Group,
		Kind:  u.GetKind(),
	}

	mapping, err := r.GetClient().RESTMapper().RESTMapping(gk, gv.Version)
	if err != nil {
		return nil, err
	}

	return &schema.GroupVersionResource{
		Group:    gv.Group,
		Version:  gv.Version,
		Resource: mapping.Resource.Resource,
	}, nil
}

func patchUnstructured(ctx context.Context, r Controller, unstructured *unstructured.Unstructured, namespace string) error {
	gvr, err := unstructuredToGVR(r, unstructured)

	if err != nil {
		return fmt.Errorf("unable to convert unstructured to GVR %w", err)
	}

	jsonData, err := json.Marshal(unstructured.Object)
	if err != nil {
		return fmt.Errorf("unable to marshal unstructured to json %w", err)
	}

	r.DynamicClient().Resource(*gvr)

	return nil
}
