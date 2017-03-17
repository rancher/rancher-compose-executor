package service

import (
	"github.com/docker/docker/api/types/container"
	"github.com/rancher/rancher-compose-executor/config"
	"github.com/rancher/rancher-compose-executor/project"
)

// DefaultDependentServices return the dependent services (as an array of ServiceRelationship)
// for the specified project and service. It looks for : links, volumesFrom, net and ipc configuration.
// It uses default project implementation and append some docker specific ones.
func DefaultDependentServices(configs *config.ServiceConfigs, s project.Service) []project.ServiceRelationship {
	result := project.DefaultDependentServices(configs, s)

	result = appendNs(configs, result, s.Config().NetworkMode, project.RelTypeNetNamespace)
	result = appendNs(configs, result, s.Config().Ipc, project.RelTypeIpcNamespace)

	return result
}

func appendNs(configs *config.ServiceConfigs, rels []project.ServiceRelationship, conf string, relType project.ServiceRelationshipType) []project.ServiceRelationship {
	service := GetContainerFromIpcLikeConfig(configs, conf)
	if service != "" {
		rels = append(rels, project.NewServiceRelationship(service, relType))
	}
	return rels
}

// GetContainerFromIpcLikeConfig returns name of the service that shares the IPC
// namespace with the specified service.
func GetContainerFromIpcLikeConfig(configs *config.ServiceConfigs, conf string) string {
	ipc := container.IpcMode(conf)
	if !ipc.IsContainer() {
		return ""
	}

	name := ipc.Container()
	if name == "" {
		return ""
	}

	if configs.Has(name) {
		return name
	}
	return ""
}
