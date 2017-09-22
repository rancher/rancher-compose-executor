package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetResource(t *testing.T) {
	resourceName, resource, err := GetResource([]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: test
`))
	assert.NoError(t, err)
	assert.Equal(t, "Pod/test", resourceName)
	assert.Equal(t, map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[interface{}]interface{}{
			"name": "test",
		},
	}, resource)

	resourceName, resource, err = GetResource([]byte(`
s1:
  image: nginx
`))
	assert.NoError(t, err)
	assert.Empty(t, resourceName)
	assert.Nil(t, resource)

	resourceName, resource, err = GetResource([]byte(`
services:
  s1:
    image: nginx
`))
	assert.NoError(t, err)
	assert.Empty(t, resourceName)
	assert.Nil(t, resource)
}
