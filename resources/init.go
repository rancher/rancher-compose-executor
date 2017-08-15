package resources

import "github.com/rancher/rancher-compose-executor/project"

func init() {
	project.SetResourceFactories(
		DependenciesCreate,
		HostsCreate,
		SecretsCreate,
		VolumesCreate,
		ServicesCreate,
	)
}
