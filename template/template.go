package template

import (
	"bytes"
	"strings"
	"text/template"

	"fmt"
	"github.com/rancher/go-rancher/catalog"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/rancher-compose-executor/template/funcs"
)

type ClusterInfo struct {
	Embedded      string
	Orchestration string
}

func Apply(contents []byte, templateVersion *catalog.TemplateVersion, cluster *client.Cluster, variables map[string]string) ([]byte, error) {
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
		"Cluster": ClusterInfo{
			Embedded:      fmt.Sprint(cluster.Embedded),
			Orchestration: cluster.Orchestration,
		},
	})
	return buf.Bytes(), nil
}
