package server

import (
	log "github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v3"
)

func (r *RancherServerLookup) Service(name string) (*client.Service, error) {
	log.Debugf("Finding service %s", name)

	name, stackId, err := resolveNameAndStackId(r.c, r.stackID, name)
	if err != nil {
		return nil, err
	}

	services, err := r.c.Service.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"stackId":      stackId,
			"name":         name,
			"removed_null": nil,
		},
	})

	if err != nil {
		return nil, err
	}

	if len(services.Data) == 0 {
		return nil, nil
	}

	log.Debugf("Found service %s", name)
	return &services.Data[0], nil
}
