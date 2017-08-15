package convert

import (
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/rancher-compose-executor/project"
	"github.com/rancher/rancher-compose-executor/utils"
)

func Service(p *project.Project, name string) (*client.Service, error) {
	launchConfig, secondaryLaunchConfigs, err := createLaunchConfigs(p, name)
	if err != nil {
		return nil, err
	}

	serviceConfig := p.Config.Services[name]

	service := client.Service{
		LaunchConfig:           &launchConfig,
		Name:                   name,
		Metadata:               utils.NestedMapsToMapInterface(serviceConfig.Metadata),
		Scale:                  max(1, int64(serviceConfig.Scale)),
		StackId:                p.Stack.Id,
		Selector:               serviceConfig.Labels["io.rancher.service.selector.container"],
		ExternalIpAddresses:    serviceConfig.ExternalIps,
		Hostname:               serviceConfig.Hostname,
		HealthCheck:            serviceConfig.HealthCheck,
		StorageDriver:          serviceConfig.StorageDriver,
		NetworkDriver:          serviceConfig.NetworkDriver,
		SecondaryLaunchConfigs: secondaryLaunchConfigs,
	}

	if service.NetworkDriver != nil {
		service.NetworkDriver.CniConfig = utils.NestedMapsToMapInterface(service.NetworkDriver.CniConfig)
	}

	if err := populateLb(p.ServerResourceLookup, *serviceConfig, &launchConfig, &service); err != nil {
		return nil, err
	}

	return &service, nil
}

func max(i, j int64) int64 {
	if i > j {
		return i
	}
	return j
}
