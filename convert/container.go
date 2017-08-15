package convert

import (
	"github.com/docker/libcompose/utils"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/rancher-compose-executor/project"
)

func Container(p *project.Project, name string) (*client.Container, error) {
	var err error

	launchConfig, _, err := createLaunchConfigs(p, name)
	if err != nil {
		return nil, err
	}

	container := client.Container{}
	if err := utils.Convert(launchConfig, &container); err != nil {
		return nil, err
	}

	container.PidContainerId, err = resolveContainerReference(p, container.PidMode, container.PidContainerId)
	if err != nil {
		return nil, err
	}

	container.NetworkContainerId, err = resolveContainerReference(p, container.NetworkMode, container.NetworkContainerId)
	if err != nil {
		return nil, err
	}

	container.IpcContainerId, err = resolveContainerReference(p, container.IpcMode, container.IpcContainerId)
	if err != nil {
		return nil, err
	}

	for i := range container.DataVolumesFrom {
		container.DataVolumesFrom[i], err = resolveContainerReference(p, "container", container.DataVolumesFrom[i])
		if err != nil {
			return nil, err
		}
	}

	return &container, nil
}

func ContainerConfig(p *project.Project, name string) (*client.ContainerConfig, error) {
	container, err := Container(p, name)
	if err != nil {
		return nil, err
	}

	config := client.ContainerConfig{}
	err = utils.Convert(container, &config)
	return &config, err
}

func resolveContainerReference(p *project.Project, mode, ref string) (string, error) {
	if mode != "container" {
		return ref, nil
	}

	container, err := p.ServerResourceLookup.Container(ref)
	if err != nil {
		return "", err
	}
	return container.Id, nil
}
