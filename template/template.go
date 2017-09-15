package template

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/rancher/go-rancher/catalog"
	"github.com/rancher/rancher-compose-executor/template/funcs"
)

func Apply(contents []byte, templateVersion *catalog.TemplateVersion, variables map[string]string) ([]byte, error) {
	// Skip templating if contents begin with '# notemplating'
	trimmedContents := strings.TrimSpace(string(contents))
	if strings.HasPrefix(trimmedContents, "#notemplating") || strings.HasPrefix(trimmedContents, "# notemplating") {
		return contents, nil
	}

	t, err := template.New("template").Funcs(funcs.Funcs).Parse(string(contents))
	if err != nil {
		return nil, err
	}

	buf := bytes.Buffer{}
	t.Execute(&buf, map[string]interface{}{
		"Values":  variables,
		"Release": templateVersion,
		"Stack":   templateVersion,
	})
	return buf.Bytes(), nil
}
