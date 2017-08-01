package convert

import (
	"fmt"
	"strings"

	"github.com/docker/libcompose/utils"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/rancher-compose-executor/config"
	"github.com/rancher/rancher-compose-executor/project"
	"github.com/rancher/rancher-compose-executor/yaml"
)

const (
	LegacyLBImage       = "rancher/load-balancer-service"
)

func createLaunchConfigs(project *project.Project, name string) (client.LaunchConfig, []client.LaunchConfig, error) {
	serviceConfig, ok := project.Config.Services[name]
	if !ok {
		return client.LaunchConfig{}, nil, fmt.Errorf("Failed to find service config for %s", name)
	}
	secondaryLaunchConfigs := []client.LaunchConfig{}
	launchConfig, err := createLaunchConfig(project, *serviceConfig)
	if err != nil {
		return launchConfig, nil, err
	}

	if secondaries, ok := project.Config.SidekickInfo.PrimariesToSidekicks[name]; ok {
		for _, secondaryName := range secondaries {
			serviceConfig, ok := project.Config.Services[secondaryName]
			if !ok {
				return launchConfig, nil, fmt.Errorf("Failed to find sidekick: %s", secondaryName)
			}

			launchConfig, err := createLaunchConfig(project, *serviceConfig)
			if err != nil {
				return launchConfig, nil, err
			}

			var secondaryLaunchConfig client.LaunchConfig
			utils.Convert(launchConfig, &secondaryLaunchConfig)
			secondaryLaunchConfig.Name = secondaryName

			if secondaryLaunchConfig.Labels == nil {
				secondaryLaunchConfig.Labels = map[string]interface{}{}
			}
			secondaryLaunchConfigs = append(secondaryLaunchConfigs, secondaryLaunchConfig)
		}
	}

	return launchConfig, secondaryLaunchConfigs, nil
}

func createLaunchConfig(p *project.Project, serviceConfig config.ServiceConfig) (client.LaunchConfig, error) {
	newLabels := yaml.SliceorMap{}
	if serviceConfig.Image == "rancher/load-balancer-service" {
		// Lookup default load balancer image
		lbImageSetting, err := p.Client.Setting.ById("lb.instance.image")
		if err != nil {
			return client.LaunchConfig{}, err
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

	result, err := serviceConfigToLaunchConfig(serviceConfig, p)
	if err != nil {
		return result, err
	}

	result.Secrets, err = setupSecrets(p.Client, serviceConfig)
	if err != nil {
		return result, err
	}

	result.Image, err = modifyLbImage(p, result.Image)
	return result, err
}

func modifyLbImage(p *project.Project, image string) (string, error) {
	if image != LegacyLBImage {
		return image, nil
	}

	// Lookup default load balancer image
	lbImageSetting, err := p.Client.Setting.ById("lb.instance.image")
	if err != nil {
		return "", err
	}
	return lbImageSetting.Value, nil
}

func setupSecrets(c *client.RancherClient, serviceConfig config.ServiceConfig) ([]client.SecretReference, error) {
	var result []client.SecretReference
	for _, secret := range serviceConfig.Secrets {
		existingSecrets, err := c.Secret.List(&client.ListOpts{
			Filters: map[string]interface{}{
				"name": secret.Source,
			},
		})
		if err != nil {
			return nil, err
		}
		if len(existingSecrets.Data) == 0 {
			return nil, fmt.Errorf("Failed to find secret %s", secret.Source)
		}
		result = append(result, client.SecretReference{
			SecretId: existingSecrets.Data[0].Id,
			Name:     secret.Target,
			Uid:      secret.Uid,
			Gid:      secret.Gid,
			Mode:     secret.Mode,
		})
	}
	return result, nil
}
