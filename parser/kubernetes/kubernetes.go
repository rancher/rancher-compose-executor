package kubernetes

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
)

type KubernetesResource struct {
	Kind             string             `yaml:"kind,omitempty"`
	Metadata         KubernetesMetadata `yaml:"metadata,omitempty"`
	CombinedName     string
	ResourceContents map[string]interface{}
}

type KubernetesMetadata struct {
	Name string `yaml:"name,omitempty"`
}

func GetResources(contents []byte) ([]*KubernetesResource, error) {
	documents := splitMultiDocument(contents)
	var resources []*KubernetesResource
	for _, contents := range documents {
		resource, err := getResource(contents)
		if err != nil {
			return nil, err
		}
		if resource != nil {
			resources = append(resources, resource)
		}
	}
	return resources, nil
}

func splitMultiDocument(contents []byte) [][]byte {
	var documents [][]byte
	var currentResource []string
	scanner := bufio.NewScanner(bytes.NewReader(contents))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Trim(line, " \t") == "---" {
			if len(currentResource) > 0 {
				documents = append(documents, []byte(strings.Join(currentResource, "\n")))
			}
			currentResource = nil
		}
		currentResource = append(currentResource, line)
	}
	if len(currentResource) > 0 {
		documents = append(documents, []byte(strings.Join(currentResource, "\n")))
	}
	return documents
}

func getResource(contents []byte) (*KubernetesResource, error) {
	var resource KubernetesResource
	if err := yaml.Unmarshal(contents, &resource); err != nil {
		return nil, err
	}
	if resource.Kind == "" || resource.Metadata.Name == "" {
		return nil, nil
	}
	if err := yaml.Unmarshal(contents, &resource.ResourceContents); err != nil {
		return nil, err
	}
	resource.CombinedName = fmt.Sprintf("%s/%s", resource.Kind, resource.Metadata.Name)
	return &resource, nil
}
