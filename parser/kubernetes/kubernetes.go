package kubernetes

import (
	"gopkg.in/yaml.v2"
)

type kubernetesResource struct {
	Metadata         kubernetesMetadata `yaml:"metadata,omitempty"`
	ResourceContents map[string]interface{}
}

type kubernetesMetadata struct {
	Name string `yaml:"name,omitempty"`
}

func GetResource(contents []byte) (string, map[string]interface{}, error) {
	var resource kubernetesResource
	if err := yaml.Unmarshal(contents, &resource); err != nil {
		return "", nil, err
	}
	if resource.Metadata.Name == "" {
		return "", nil, nil
	}
	if err := yaml.Unmarshal(contents, &resource.ResourceContents); err != nil {
		return "", nil, err
	}
	return resource.Metadata.Name, resource.ResourceContents, nil
}
