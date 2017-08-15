package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"github.com/docker/docker/pkg/urlutil"
	"github.com/docker/libcompose/utils"
	"github.com/fatih/structs"
	"github.com/rancher/go-rancher/catalog"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/rancher-compose-executor/config"
	"github.com/rancher/rancher-compose-executor/lookup"
	"github.com/rancher/rancher-compose-executor/template"
	composeYaml "github.com/rancher/rancher-compose-executor/yaml"
	"gopkg.in/yaml.v2"
)

var (
	noMerge = []string{
		"links",
		"volumes_from",
	}
)

func transferFields(from, to config.RawService, prefixField string, instance interface{}) {
	s := structs.New(instance)
	for _, f := range s.Fields() {
		field := strings.SplitN(f.Tag("yaml"), ",", 2)[0]
		if fieldValue, ok := from[field]; ok {
			if _, ok = to[prefixField]; !ok {
				to[prefixField] = map[interface{}]interface{}{}
			}
			to[prefixField].(map[interface{}]interface{})[field] = fieldValue
		}
	}
}

// Createconfig.RawConfig unmarshals contents to config and creates config based on version
func CreateRawConfig(contents []byte) (*config.RawConfig, error) {
	var rawConfig config.RawConfig
	if err := yaml.Unmarshal(contents, &rawConfig); err != nil {
		return nil, err
	}

	if rawConfig.Version != "2" {
		var baseRawServices config.RawServiceMap
		if err := yaml.Unmarshal(contents, &baseRawServices); err != nil {
			return nil, err
		}
		if _, ok := baseRawServices[".catalog"]; ok {
			delete(baseRawServices, ".catalog")
		}
		rawConfig.Services = baseRawServices
	}

	if rawConfig.Services == nil {
		rawConfig.Services = make(config.RawServiceMap)
	}
	if rawConfig.Volumes == nil {
		rawConfig.Volumes = make(map[string]interface{})
	}
	if rawConfig.Networks == nil {
		rawConfig.Networks = make(map[string]interface{})
	}
	if rawConfig.Hosts == nil {
		rawConfig.Hosts = make(map[string]interface{})
	}
	if rawConfig.Secrets == nil {
		rawConfig.Secrets = make(map[string]interface{})
	}

	// Merge other service types into primary service map
	for name, baseRawLoadBalancer := range rawConfig.LoadBalancers {
		rawConfig.Services[name] = baseRawLoadBalancer
		transferFields(baseRawLoadBalancer, rawConfig.Services[name], "lb_config", config.LBConfig{})
	}
	// TODO: validation will throw errors for fields directly under service
	for name, baseRawStorageDriver := range rawConfig.StorageDrivers {
		rawConfig.Services[name] = baseRawStorageDriver
		transferFields(baseRawStorageDriver, rawConfig.Services[name], "storage_driver", client.StorageDriver{})
	}
	// TODO: validation will throw errors for fields directly under service
	for name, baseRawNetworkDriver := range rawConfig.NetworkDrivers {
		rawConfig.Services[name] = baseRawNetworkDriver
		transferFields(baseRawNetworkDriver, rawConfig.Services[name], "network_driver", client.NetworkDriver{})
	}
	for name, baseRawVirtualMachine := range rawConfig.VirtualMachines {
		rawConfig.Services[name] = baseRawVirtualMachine
	}
	for name, baseRawExternalService := range rawConfig.ExternalServices {
		rawConfig.Services[name] = baseRawExternalService
		rawConfig.Services[name]["image"] = "rancher/external-service"
	}
	// TODO: container aliases
	for name, baseRawAlias := range rawConfig.Aliases {
		if serviceAliases, ok := baseRawAlias["services"]; ok {
			rawConfig.Services[name] = baseRawAlias
			rawConfig.Services[name]["image"] = "rancher/dns-service"
			rawConfig.Services[name]["links"] = serviceAliases
			delete(rawConfig.Services[name], "services")
		}
	}

	return &rawConfig, nil
}

// Merge merges a compose file into an existing set of service configs
func Merge(existingServices map[string]*config.ServiceConfig, vars map[string]string, resourceLookup lookup.ResourceLookup, templateVersion *catalog.TemplateVersion, file string, contents []byte) (*config.Config, error) {
	var err error
	contents, err = template.Apply(contents, templateVersion, vars)
	if err != nil {
		return nil, err
	}

	rawConfig, err := CreateRawConfig(contents)
	if err != nil {
		return nil, err
	}

	baseRawServices := rawConfig.Services
	baseRawContainers := rawConfig.Containers

	// TODO: just interpolate at the map level earlier
	if err := InterpolateRawServiceMap(&baseRawServices, vars); err != nil {
		return nil, err
	}
	if err := InterpolateRawServiceMap(&baseRawContainers, vars); err != nil {
		return nil, err
	}

	for k, v := range rawConfig.Volumes {
		if err := Interpolate(k, &v, vars); err != nil {
			return nil, err
		}
		rawConfig.Volumes[k] = v
	}

	for k, v := range rawConfig.Networks {
		if err := Interpolate(k, &v, vars); err != nil {
			return nil, err
		}
		rawConfig.Networks[k] = v
	}

	baseRawServices, err = preProcessServiceMap(baseRawServices)
	if err != nil {
		return nil, err
	}
	baseRawContainers, err = preProcessServiceMap(baseRawContainers)
	if err != nil {
		return nil, err
	}

	baseRawServices, err = TryConvertStringsToInts(baseRawServices, getRancherConfigObjects())
	if err != nil {
		return nil, err
	}
	baseRawContainers, err = TryConvertStringsToInts(baseRawContainers, getRancherConfigObjects())
	if err != nil {
		return nil, err
	}

	var serviceConfigs map[string]*config.ServiceConfig
	if rawConfig.Version == "2" {
		var err error
		serviceConfigs, err = MergeServicesV2(vars, resourceLookup, file, baseRawServices)
		if err != nil {
			return nil, err
		}
	} else {
		serviceConfigsV1, err := MergeServicesV1(vars, resourceLookup, file, baseRawServices)
		if err != nil {
			return nil, err
		}
		serviceConfigs, err = convertServices(serviceConfigsV1)
		if err != nil {
			return nil, err
		}
	}

	for name, serviceConfig := range serviceConfigs {
		if existingServiceConfig, ok := existingServices[name]; ok {
			var rawService config.RawService
			if err := utils.Convert(serviceConfig, &rawService); err != nil {
				return nil, err
			}
			var rawExistingService config.RawService
			if err := utils.Convert(existingServiceConfig, &rawExistingService); err != nil {
				return nil, err
			}

			rawService = mergeConfig(rawExistingService, rawService)
			if err := utils.Convert(rawService, &serviceConfig); err != nil {
				return nil, err
			}
		}
	}

	var containerConfigs map[string]*config.ServiceConfig
	if rawConfig.Version == "2" {
		var err error
		containerConfigs, err = MergeServicesV2(vars, resourceLookup, file, baseRawContainers)
		if err != nil {
			return nil, err
		}
	}

	adjustValues(serviceConfigs)
	adjustValues(containerConfigs)

	var dependencies map[string]*config.DependencyConfig
	var volumes map[string]*config.VolumeConfig
	var networks map[string]*config.NetworkConfig
	var secrets map[string]*config.SecretConfig
	var hosts map[string]*config.HostConfig
	if err := utils.Convert(rawConfig.Dependencies, &dependencies); err != nil {
		return nil, err
	}
	if err := utils.Convert(rawConfig.Volumes, &volumes); err != nil {
		return nil, err
	}
	for i, volume := range volumes {
		if volume == nil {
			volumes[i] = &config.VolumeConfig{}
		}
	}
	if err := utils.Convert(rawConfig.Networks, &networks); err != nil {
		return nil, err
	}
	if err := utils.Convert(rawConfig.Hosts, &hosts); err != nil {
		return nil, err
	}
	if err := utils.Convert(rawConfig.Secrets, &secrets); err != nil {
		return nil, err
	}

	return &config.Config{
		Services:     serviceConfigs,
		Containers:   containerConfigs,
		Dependencies: dependencies,
		Volumes:      volumes,
		Networks:     networks,
		Secrets:      secrets,
		Hosts:        hosts,
	}, nil
}

func InterpolateRawServiceMap(baseRawServices *config.RawServiceMap, vars map[string]string) error {
	for k, v := range *baseRawServices {
		for k2, v2 := range v {
			if err := Interpolate(k2, &v2, vars); err != nil {
				return err
			}
			(*baseRawServices)[k][k2] = v2
		}
	}
	return nil
}

func adjustValues(configs map[string]*config.ServiceConfig) {
	// yaml parser turns "no" into "false" but that is not valid for a restart policy
	for _, v := range configs {
		if v.Restart == "false" {
			v.Restart = "no"
		}
	}
}

func readEnvFile(resourceLookup lookup.ResourceLookup, inFile string, serviceData config.RawService) (config.RawService, error) {
	if _, ok := serviceData["env_file"]; !ok {
		return serviceData, nil
	}

	var envFiles composeYaml.Stringorslice

	if err := utils.Convert(serviceData["env_file"], &envFiles); err != nil {
		return nil, err
	}

	if len(envFiles) == 0 {
		return serviceData, nil
	}

	if resourceLookup == nil {
		return nil, fmt.Errorf("Can not use env_file in file %s no mechanism provided to load files", inFile)
	}

	var vars composeYaml.MaporEqualSlice

	if _, ok := serviceData["environment"]; ok {
		if err := utils.Convert(serviceData["environment"], &vars); err != nil {
			return nil, err
		}
	}

	for i := len(envFiles) - 1; i >= 0; i-- {
		envFile := envFiles[i]
		content, _, err := resourceLookup.Lookup(envFile, inFile)
		if err != nil {
			return nil, err
		}

		if err != nil {
			return nil, err
		}

		scanner := bufio.NewScanner(bytes.NewBuffer(content))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())

			if len(line) > 0 && !strings.HasPrefix(line, "#") {
				key := strings.SplitAfter(line, "=")[0]

				found := false
				for _, v := range vars {
					if strings.HasPrefix(v, key) {
						found = true
						break
					}
				}

				if !found {
					vars = append(vars, line)
				}
			}
		}

		if scanner.Err() != nil {
			return nil, scanner.Err()
		}
	}

	serviceData["environment"] = vars

	delete(serviceData, "env_file")

	return serviceData, nil
}

func mergeConfig(baseService, serviceData config.RawService) config.RawService {
	for k, v := range serviceData {
		// Image and build are mutually exclusive in merge
		if k == "image" {
			delete(baseService, "build")
		} else if k == "build" {
			delete(baseService, "image")
		}
		existing, ok := baseService[k]
		if ok {
			baseService[k] = merge(existing, v)
		} else {
			baseService[k] = v
		}
	}

	return baseService
}

// IsValidRemote checks if the specified string is a valid remote (for builds)
func IsValidRemote(remote string) bool {
	return urlutil.IsGitURL(remote) || urlutil.IsURL(remote)
}
