package controllers

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	sigsyaml "sigs.k8s.io/yaml"
)

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
