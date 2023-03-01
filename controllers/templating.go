package controllers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/yaml"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	sigsyaml "sigs.k8s.io/yaml"
)

// Rendering and applying templates is a two-stage process. This
// structure holds information about a rendered template between these
// two stages
type RenderedTemplate struct {
	// Name of rendered resource (from template key in GatewayClassBlueprint, not Kubernetes resource name)
	Name string

	// The rendered resource
	Resource *unstructured.Unstructured

	// Whether resource is namespaced or not
	IsNamespaced bool

	// GVR for resource
	GVR *schema.GroupVersionResource

	// Current resource fetch from API-server (or as close as our local caching allows)
	Current *unstructured.Unstructured
}

// Render a list of templates
func renderTemplates(ctx context.Context, r ControllerDynClient, parent metav1.Object,
	templates map[string]string, templateValues any) ([]RenderedTemplate, error) {
	var err error
	var u *unstructured.Unstructured
	var rendered []RenderedTemplate

	logger := log.FromContext(ctx)
	ns := parent.GetNamespace()

	for tmplKey, tmpl := range templates {
		u, err = template2Unstructured(tmpl, &templateValues)
		if err != nil {
			logger.Error(err, "cannot render template", "templateKey", tmplKey)
			continue
		}

		gvr, isNamespaced, err := unstructuredToGVR(r, u)
		if err != nil {
			logger.Error(err, "cannot detect GVR for resource", "templateKey", tmplKey)
			continue
		}
		// Fetch the current version of the resource from API server cache
		dynamicClient := r.DynamicClient().Resource(*gvr).Namespace(ns)
		current, err := dynamicClient.Get(ctx, u.GetName(), metav1.GetOptions{})
		rendered = append(rendered, RenderedTemplate{tmplKey, u, isNamespaced, gvr, current})
	}

	if len(rendered) != len(templates) {
		return nil, fmt.Errorf("found %v problems while applying %v templates", len(templates)-len(rendered), len(templates))
	}
	return rendered, nil
}

// Build a map of values from current resources. Useful for
// referencing values between resources, e.g.
//
//	resource1.status.foo ---templated-into---> resource2.spec.bar)
func buildResourceValues(r ControllerDynClient, renderedTemplates []RenderedTemplate) (map[string]any, error) {
	var resources map[string]any

	for _, rendered := range renderedTemplates {
		objMap, err := objectToMap(rendered.Current, r.Scheme())
		if err != nil {
			return nil, err
		}
		resources[rendered.Name] = objMap
	}
	return resources, nil
}

// Apply a list of pre-rendered templates
func applyTemplates(ctx context.Context, r ControllerDynClient, parent metav1.Object, renderedTemplates []RenderedTemplate) error {
	var err error
	var errorCnt = 0

	logger := log.FromContext(ctx)

	for _, rendered := range renderedTemplates {
		if rendered.IsNamespaced {
			// Only namespaced objects can have namespaced object as owner
			err = ctrl.SetControllerReference(parent, rendered.Resource, r.Scheme())
			if err != nil {
				logger.Error(err, "cannot set owner for namespaced template", "templateKey", rendered.Name)
				errorCnt++
			} else {
				ns := parent.GetNamespace()
				err = patchUnstructured(ctx, r, rendered.Resource, rendered.GVR, &ns)
				if err != nil {
					logger.Error(err, "cannot apply namespaced template", "templateKey", rendered.Name)
					errorCnt++
				}
			}
		} else {
			err = patchUnstructured(ctx, r, rendered.Resource, rendered.GVR, nil)
			if err != nil {
				logger.Error(err, "cannot apply cluster-scoped template", "templateKey", rendered.Name)
				errorCnt++
			}
		}
	}

	if errorCnt > 0 {
		return fmt.Errorf("found %v problems while applying %v templates", errorCnt, len(renderedTemplates))
	}
	return nil
}

// This function is made available to templates as 'toYaml'
func helperToYaml(v interface{}) string {
	data, err := sigsyaml.Marshal(v)
	if err != nil {
		return ""
	}
	return strings.TrimSuffix(string(data), "\n")
}

func templateRender(templateData string, templateValues any) (*bytes.Buffer, error) {
	var buffer bytes.Buffer

	var funcs = template.FuncMap{
		"toYaml": helperToYaml,
	}

	tmpl, err := template.New("resourceTemplate").Funcs(sprig.FuncMap()).Funcs(funcs).Parse(templateData)
	if err != nil {
		return nil, err
	}

	err = tmpl.Execute(io.Writer(&buffer), templateValues)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Rendered:\n%s\n", buffer.Bytes())

	return &buffer, nil
}

// Prepare a resource like Gateway or HTTPRoute for use in templates
// by converting to map[string]any
func objectToMap(obj runtime.Object, scheme *runtime.Scheme) (map[string]any, error) {
	ser := json.NewSerializerWithOptions(json.DefaultMetaFactory, scheme, scheme,
		json.SerializerOptions{Yaml: true, Pretty: false, Strict: true})
	if ser == nil {
		return nil, fmt.Errorf("cannot create object serializer")
	}
	buffer, err := runtime.Encode(ser, obj)
	if err != nil {
		return nil, fmt.Errorf("cannot serialize oject: %w", err)
	}
	mapObj := map[string]any{}
	err = yaml.Unmarshal(buffer, &mapObj)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal object: %w", err)
	}
	return mapObj, nil
}
