package convert

import (
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/rancher-compose-executor/config"
	"github.com/rancher/rancher-compose-executor/convert"
	"github.com/rancher/rancher-compose-executor/project"
	"github.com/rancher/rancher-compose-executor/yaml"
)

type ContainerInspect struct {
	Name       string
	Config     *container.Config
	HostConfig *container.HostConfig
}

func CreateLaunchConfig(name string, serviceConfig *config.ServiceConfig, c *client.RancherClient, context project.Context) (client.LaunchConfig, error) {
	var result client.LaunchConfig

	schemasUrl := strings.SplitN(c.GetSchemas().Links["self"], "/schemas", 2)[0]
	scriptsUrl := schemasUrl + "/scripts/transform"

	tempImage := serviceConfig.Image
	tempLabels := serviceConfig.Labels
	newLabels := yaml.SliceorMap{}
	if serviceConfig.Image == "rancher/load-balancer-service" {
		// Lookup default load balancer image
		lbImageSetting, err := c.Setting.ById("lb.instance.image")
		if err != nil {
			return result, err
		}
		serviceConfig.Image = lbImageSetting.Value

		// Strip off legacy load balancer labels
		for k, v := range serviceConfig.Labels {
			if !strings.HasPrefix(k, "io.rancher.loadbalancer") && !strings.HasPrefix(k, "io.rancher.service.selector") {
				newLabels[k] = v
			}
		}
		serviceConfig.Labels = newLabels
	}

	config, hostConfig, err := convert.Convert(serviceConfig, context)
	if err != nil {
		return result, err
	}

	serviceConfig.Image = tempImage
	serviceConfig.Labels = tempLabels

	dockerContainer := &ContainerInspect{
		Config:     config,
		HostConfig: hostConfig,
	}

	dockerContainer.HostConfig.NetworkMode = container.NetworkMode("")
	dockerContainer.Name = "/" + name

	if c.Post(scriptsUrl, dockerContainer, &result); err != nil {
		return result, err
	}

	result.VolumeDriver = hostConfig.VolumeDriver

	setupNetworking(serviceConfig.NetworkMode, &result)
	setupVolumesFrom(serviceConfig.VolumesFrom, &result)

	// TODO
	/*if err = setupBuild(r, name, &result, serviceConfig); err != nil {
		return result, err
	}
	if err = setupSecrets(r, name, &result, serviceConfig); err != nil {
		return result, err
	}*/

	if result.Labels == nil {
		result.Labels = map[string]interface{}{}
	}
	if result.LogConfig.Config == nil {
		result.LogConfig.Config = map[string]interface{}{}
	}

	result.Kind = serviceConfig.Type
	result.Vcpu = int64(serviceConfig.Vcpu)
	result.Userdata = serviceConfig.Userdata
	result.MemoryMb = int64(serviceConfig.Memory)
	result.Disks = serviceConfig.Disks

	if strings.EqualFold(result.Kind, "virtual_machine") || strings.EqualFold(result.Kind, "virtualmachine") {
		result.Kind = "virtualMachine"
	}

	return result, err
}

func setupNetworking(netMode string, launchConfig *client.LaunchConfig) {
	if netMode == "" {
		launchConfig.NetworkMode = "managed"
	} else if container.IpcMode(netMode).IsContainer() {
		// For some reason NetworkMode object is gone runconfig, but IpcMode works the same for this
		launchConfig.NetworkMode = "container"
		launchConfig.NetworkLaunchConfig = strings.TrimPrefix(netMode, "container:")
	} else {
		launchConfig.NetworkMode = netMode
	}
}

func setupVolumesFrom(volumesFrom []string, launchConfig *client.LaunchConfig) {
	launchConfig.DataVolumesFromLaunchConfigs = volumesFrom
}
