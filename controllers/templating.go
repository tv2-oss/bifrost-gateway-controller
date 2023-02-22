package controllers

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/yaml"
	sigsyaml "sigs.k8s.io/yaml"
)

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
