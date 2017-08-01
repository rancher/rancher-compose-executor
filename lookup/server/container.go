package server

import (
	"github.com/rancher/go-rancher/v3"
)

func (r *RancherServerLookup) Container(name string) (*client.Container, error) {
	name, stackId, err := resolveNameAndStackId(r.c, r.stackID, name)
	if err != nil {
		return nil, err
	}

	containers, err := r.c.Container.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"stackId":      stackId,
			"name":         name,
			"removed_null": nil,
		},
	})
	if err != nil {
		return nil, err
	}

	if len(containers.Data) == 0 {
		return nil, nil
	}

	return &containers.Data[0], nil
}
