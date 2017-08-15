package lookup

import "github.com/rancher/go-rancher/v3"

type ServerResourceLookup interface {
	Service(name string) (*client.Service, error)
	Container(name string) (*client.Container, error)
	Cert(name string) (*client.Certificate, error)
}
