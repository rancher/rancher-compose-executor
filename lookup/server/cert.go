package server

import (
	"github.com/rancher/go-rancher/v3"
)

func (r *RancherServerLookup) Cert(name string) (*client.Certificate, error) {
	certs, err := r.c.Certificate.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"removed_null": nil,
			"name":         name,
		},
	})

	if err != nil {
		return nil, err
	}

	if len(certs.Data) == 0 {
		return nil, nil
	}

	return &certs.Data[0], nil
}
