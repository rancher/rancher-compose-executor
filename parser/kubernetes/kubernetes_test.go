package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetResources(t *testing.T) {
	resources, err := GetResources([]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: test
`))
	assert.NoError(t, err)
	assert.Len(t, resources, 1)
	assert.Equal(t, "test", resources[0].Metadata.Name)
	assert.Equal(t, map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[interface{}]interface{}{
			"name": "test",
		},
	}, resources[0].ResourceContents)

	resources, err = GetResources([]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: test
---
apiVersion: v1
kind: Pod
metadata:
  name: test2`))
	assert.NoError(t, err)
	assert.Len(t, resources, 2)
	assert.Equal(t, "test", resources[0].Metadata.Name)
	assert.Equal(t, "test2", resources[1].Metadata.Name)
	assert.Equal(t, map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[interface{}]interface{}{
			"name": "test",
		},
	}, resources[0].ResourceContents)
	assert.Equal(t, map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[interface{}]interface{}{
			"name": "test2",
		},
	}, resources[1].ResourceContents)

	resources, err = GetResources([]byte(`
s1:
  image: nginx`))
	assert.NoError(t, err)
	assert.Len(t, resources, 0)

	resources, err = GetResources([]byte(`
services:
  s1:
    image: nginx`))
	assert.NoError(t, err)
	assert.Len(t, resources, 0)
}
