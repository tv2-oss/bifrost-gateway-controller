package controllers

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	gatewayapi "sigs.k8s.io/gateway-api/apis/v1beta1"

	gwcapi "github.com/tv2-oss/gateway-controller/apis/gateway.tv2.dk/v1alpha1"
	selfapi "github.com/tv2-oss/gateway-controller/pkg/api"
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

func lookupGatewayClassParameters(ctx context.Context, r ControllerClient, gwc *gatewayapi.GatewayClass) (*gwcapi.GatewayClassParameters, error) {
	if gwc.Spec.ParametersRef == nil {
		return nil, errors.New("GatewayClass without parameters")
	}

	// FIXME: More validation...
	if gwc.Spec.ParametersRef.Kind != "GatewayClassParameters" || gwc.Spec.ParametersRef.Group != "gateway.tv2.dk" {
		return nil, errors.New("parameter Kind is not a valid GatewayClassParameters")
	}

	var gwcp gwcapi.GatewayClassParameters
	if err := r.Client().Get(ctx, types.NamespacedName{Name: gwc.Spec.ParametersRef.Name}, &gwcp); err != nil {
		return nil, err
	}

	return &gwcp, nil
}

func lookupGateway(ctx context.Context, r ControllerClient, name gatewayapi.ObjectName, namespace string) (*gatewayapi.Gateway, error) {
	var gw gatewayapi.Gateway
	if err := r.Client().Get(ctx, types.NamespacedName{Name: string(name), Namespace: namespace}, &gw); err != nil {
		return nil, err
	}
	return &gw, nil
}

func template2Unstructured(templateData string, templateValues any) (*unstructured.Unstructured, error) {
	renderBuffer, err := templateRender(templateData, templateValues)
	if err != nil {
		fmt.Printf("Template:\n%s\n", templateData)
		fmt.Printf("Template values:\n%s\n", templateValues)
		return nil, err
	}

	rawResource := map[string]any{}
	err = yaml.Unmarshal(renderBuffer.Bytes(), &rawResource)
	if err != nil {
		return nil, err
	}

	unstruct := &unstructured.Unstructured{Object: rawResource}

	return unstruct, nil
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

// Apply an unstructured object using service-side apply
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

// Apply a list of templates. If errors are found, only the first detected error is returned.
func applyTemplates(ctx context.Context, r ControllerDynClient, parent metav1.Object,
	templates map[string]string, templateValues any) error {
	var err error
	var errorCnt = 0
	var u *unstructured.Unstructured

	logger := log.FromContext(ctx)

	for tmplKey, tmpl := range templates {
		u, err = template2Unstructured(tmpl, &templateValues)
		if err != nil {
			logger.Error(err, "cannot render template", "templateKey", tmplKey)
			errorCnt++
		}

		gvr, isNamespaced, err := unstructuredToGVR(r, u)
		if err != nil {
			logger.Error(err, "cannot detect GVR for resource", "templateKey", tmplKey)
			errorCnt++
		}

		if isNamespaced {
			// Only namespaced objects can have namespaced object as owner
			err = ctrl.SetControllerReference(parent, u, r.Scheme())
			if err != nil {
				logger.Error(err, "cannot set owner for namespaced template", "templateKey", tmplKey)
				errorCnt++
			} else {
				ns := parent.GetNamespace()
				err = patchUnstructured(ctx, r, u, gvr, &ns)
				if err != nil {
					logger.Error(err, "cannot apply namespaced template", "templateKey", tmplKey)
					errorCnt++
				}
			}
		} else {
			err = patchUnstructured(ctx, r, u, gvr, nil)
			if err != nil {
				logger.Error(err, "cannot apply cluster-scoped template", "templateKey", tmplKey)
				errorCnt++
			}
		}
	}

	if errorCnt > 0 {
		return fmt.Errorf("found %v problems while applying %v templates", errorCnt, len(templates))
	}
	return nil
}
