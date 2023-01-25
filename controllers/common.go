package controllers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"html/template"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayapi "sigs.k8s.io/gateway-api/apis/v1beta1"

	selfapi "github.com/tv2/cloud-gateway-controller/pkg/api"
)

type templateValues struct {
	Gateway *gatewayapi.Gateway
}

type Controller interface {
	GetClient() client.Client
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

func renderTemplate(gwParent *gatewayapi.Gateway, configMap *corev1.ConfigMap, configMapKey string) (*unstructured.Unstructured, error) {
	var buffer bytes.Buffer
	templateData, found := configMap.Data[configMapKey]

	if !found {
		return nil, errors.New("key not found in ConfigMap")
	}

	template, err := template.New("resourceTemplate").Parse(templateData)
	if err != nil {
		return nil, err
	}

	err = template.Execute(io.Writer(&buffer), &templateValues{gwParent})
	if err != nil {
		return nil, err
	}
	fmt.Printf("xxxxxxx buffer: %s\n", buffer.String())

	rawResource := map[string]any{}
	err = yaml.Unmarshal(buffer.Bytes(), &rawResource)
	if err != nil {
		return nil, err
	}

	unstruct := unstructured.Unstructured{Object: rawResource}

	return &unstruct, nil
}
