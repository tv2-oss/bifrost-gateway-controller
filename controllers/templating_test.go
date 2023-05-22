package controllers

import (
	"k8s.io/apimachinery/pkg/util/yaml"
	"testing"
)

func TestParseSingleTemplate(t *testing.T) {
	template := "foo"
	tmpl, err := parseSingleTemplate("foo", template)
	if tmpl == nil || err != nil {
		t.Fatalf("Error parsing template %v", err)
	}
}

var textTemplate = `
t1: |
    name: {{ .Values.name1 }}
t2: |
    {{ if .Values.t2enable }}
    name: {{ .Values.name2 }}
    {{ end }}
t3: |
    {{ range .Values.t3data }}
    name: {{ $.Values.name3 }}-{{ . }}
    ---
    {{ end }}
`

var textValues = `
name1: t1name
name2: t2name
name3: t3name
t2enable: false
t3data:
- foo1
- foo2
- foo3
`

func helperGetResourceState() ([]*ResourceTemplateState, error) {
	templates := map[string]string{}
	_ = yaml.Unmarshal([]byte(textTemplate), &templates)
	return parseTemplates(templates)
}

func helperGetValues() *TemplateValues {
	values := map[string]any{}
	_ = yaml.Unmarshal([]byte(textValues), &values)
	templateValues := TemplateValues{
		Values: values,
	}
	return &templateValues
}

func TestParseTemplate(t *testing.T) {
	tmpl, err := helperGetResourceState()
	if tmpl == nil || err != nil {
		t.Fatalf("Error parsing templates %v", err)
	}
	if len(tmpl) != 3 {
		t.Fatalf("Template slice lenght mismatch, got %v, expected 3", len(tmpl))
	}
	if tmpl[0].TemplateName != "t1" {
		t.Fatalf("Template[0] name, got %v, expected t1", tmpl[0].TemplateName)
	}
}

func TestTemplate2map(t *testing.T) {
	tmpl, err := helperGetResourceState()
	tmplValues := helperGetValues()
	rawResources, err := template2maps(tmpl[0].Template, tmplValues)
	if rawResources == nil {
		t.Fatalf("Cannot render template to map: %v", err)
	}
	if len(rawResources) != 1 {
		t.Fatalf("Error rendering resource, got len %v, expected 1", len(rawResources))
	}
	if rawResources[0]["name"] != "t1name" {
		t.Fatalf("Rendered template error, got %v, expected 't1name'", rawResources[0]["name"])
	}
	rawResources, err = template2maps(tmpl[1].Template, tmplValues)
	if err != nil {
		t.Fatalf("Error rendering empty resource, got err %v", err)
	}
	if len(rawResources) != 0 {
		t.Fatalf("Error rendering empty resource, got len %v, expected 0", len(rawResources))
	}
	rawResources, err = template2maps(tmpl[2].Template, tmplValues)
	if err != nil {
		t.Fatalf("Error rendering multi-resource, got err %v", err)
	}
	if len(rawResources) != 3 {
		t.Fatalf("Error rendering multi-resource, got len %v, expected 3", len(rawResources))
	}
}
