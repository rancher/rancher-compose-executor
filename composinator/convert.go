package composinator

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"encoding/json"

	"github.com/pkg/errors"
	v3 "github.com/rancher/go-rancher/v3"
	"github.com/rancher/rancher-compose-executor/config"
	"github.com/rancher/rancher-compose-executor/yaml"
	yml "gopkg.in/yaml.v2"
)

const (
	managedNetwork         = "managed"
	containerNetwork       = "container"
	labelSelectorContainer = "io.rancher.service.selector.container"
	labelServiceGlobal     = "io.rancher.scheduler.global"
	virtualMachineKind     = "virtualMachine"
	hashLabel              = "io.rancher.service.hash"
	blkioWeight            = "weight"
	blkioReadIops          = "readIops"
	blkioReadBps           = "readBps"
	blkioWriteIops         = "writeIops"
	blkioWriteBps          = "writeBps"
	virtualMachine         = "virtualMachine"
)

func convert(w http.ResponseWriter, client *v3.RancherClient, input convertOptions) {
	stackData, err := GetStackData(client, input.StackID)
	if err != nil {
		http.Error(w, "can't obtain exported data", http.StatusInternalServerError)
	}
	dockerCompose, rancherCompose, compose, err := createComposeData(stackData, input.Format)
	if err != nil {
		http.Error(w, "can't create compose file", http.StatusInternalServerError)
	}
	result := map[string]string{}
	if input.Format == "combined" {
		result["compose"] = compose
	} else {
		result["dockerCompose"] = dockerCompose
		result["rancherCompose"] = rancherCompose
	}
	data, err := json.Marshal(result)
	if err != nil {
		http.Error(w, "can't marshall result", http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, string(data))
}

// StackData is the metadata for exporting compose file for a stack. All maps use resource.id as its ley except volumeTemplates,
// which uses its name.
type StackData struct {
	Services             map[string]v3.Service
	StandaloneContainers map[string]v3.Container
	VolumeTemplates      map[string]v3.VolumeTemplate
	Certificates         map[string]v3.Certificate
	PortRuleServices     map[string]v3.Service
	PortRuleContainers   map[string]v3.Container
	Secrets              map[string]v3.Secret
}

func GetStackData(client *v3.RancherClient, stackID string) (StackData, error) {
	stackData := StackData{
		Services:             map[string]v3.Service{},
		StandaloneContainers: map[string]v3.Container{},
		VolumeTemplates:      map[string]v3.VolumeTemplate{},
		Certificates:         map[string]v3.Certificate{},
		PortRuleServices:     map[string]v3.Service{},
		PortRuleContainers:   map[string]v3.Container{},
		Secrets:              map[string]v3.Secret{},
	}

	// services
	services, err := client.Service.List(&v3.ListOpts{
		Filters: map[string]interface{}{
			"stackId": stackID,
		},
	})
	if err != nil {
		return StackData{}, errors.Wrap(err, "can't list services")
	}
	for _, service := range services.Data {
		stackData.Services[service.Id] = service
	}

	// standalone containers
	containers, err := client.Container.List(&v3.ListOpts{
		Filters: map[string]interface{}{
			"stackId": stackID,
		},
	})
	for _, container := range containers.Data {
		if container.ServiceId == "" {
			stackData.StandaloneContainers[container.Id] = container
		}
	}

	// volumeTemplates
	volumeTemplates, err := client.VolumeTemplate.List(&v3.ListOpts{
		Filters: map[string]interface{}{
			"stackId":      stackID,
			"removed_null": "true",
			"limit":        "-1",
		},
	})
	if err != nil {
		return StackData{}, errors.Wrap(err, "can't list volumeTemplates")
	}
	for _, volumeTemplate := range volumeTemplates.Data {
		stackData.VolumeTemplates[volumeTemplate.Name] = volumeTemplate
	}

	// certificates, portRuleServices, portRuleContainers, secrets
	for _, service := range stackData.Services {
		if service.LbConfig != nil {
			// certificates
			if service.LbConfig.DefaultCertificateId != "" {
				cert, err := client.Certificate.ById(service.LbConfig.DefaultCertificateId)
				if err != nil {
					return StackData{}, errors.Wrap(err, "can't get lbconfig")
				}
				stackData.Certificates[cert.Id] = *cert
			}
			if len(service.LbConfig.CertificateIds) > 0 {
				for _, certID := range service.LbConfig.CertificateIds {
					cert, err := client.Certificate.ById(certID)
					if err != nil {
						return StackData{}, errors.Wrap(err, "can't get lbconfig")
					}
					stackData.Certificates[cert.Id] = *cert
				}
			}
			// portRuleServices, portRuleContainers
			for _, portRule := range service.LbConfig.PortRules {
				if portRule.ServiceId != "" {
					if s, ok := stackData.Services[portRule.ServiceId]; ok {
						stackData.PortRuleServices[s.Id] = s
					} else {
						service, err := client.Service.ById(portRule.ServiceId)
						if err != nil {
							return StackData{}, errors.Wrap(err, "can't get service")
						}
						stackData.PortRuleServices[service.Id] = *service
					}
				}
				if portRule.InstanceId != "" {
					if c, ok := stackData.StandaloneContainers[portRule.InstanceId]; ok {
						stackData.PortRuleContainers[c.Id] = c
					} else {
						container, err := client.Container.ById(portRule.InstanceId)
						if err != nil {
							return StackData{}, errors.Wrap(err, "can't get instance")
						}
						stackData.PortRuleContainers[container.Id] = *container
					}
				}
			}
		}
		// prepare secrets
		for _, launchConfig := range append([]v3.LaunchConfig{*service.LaunchConfig}, service.SecondaryLaunchConfigs...) {
			for _, sr := range launchConfig.Secrets {
				secret, err := client.Secret.ById(sr.SecretId)
				if err != nil {
					return StackData{}, errors.Wrap(err, "can't get secret")
				}
				stackData.Secrets[secret.Id] = *secret
			}
		}
	}
	return stackData, nil
}

// createCompose Data takes stackData and returns dockerCompose, rancherCompose, or combinedCompose depends on the format
func createComposeData(stackData StackData, format string) (string, string, string, error) {
	if format == "combined" {
		compose, err := createCombinedComposeData(stackData)
		return "", "", compose, err
	} else if format == "split" {
		dockerCompose, rancherCompose, err := createSplitComposeData(stackData)
		return dockerCompose, rancherCompose, "", err
	}
	return "", "", "", nil
}

func createCombinedComposeData(stackData StackData) (string, error) {
	compose := config.NewConfig()
	volumeConfig := map[string]*config.VolumeConfig{}
	secretConfig := map[string]*config.SecretConfig{}
	// for service export
	for _, service := range stackData.Services {
		launchConfigs := append([]v3.LaunchConfig{*service.LaunchConfig}, service.SecondaryLaunchConfigs...)
		for _, launchConfig := range launchConfigs {
			serviceConfig := &config.ServiceConfig{}
			// volume convert
			convertVolume(serviceConfig, volumeConfig, launchConfig.DataVolumes, launchConfig.VolumeDriver, stackData.VolumeTemplates)

			//secret convert
			convertSecret(serviceConfig, secretConfig, launchConfig.Secrets, stackData.Secrets)

			// docker-compose
			mergeDockerCompose(serviceConfig, launchConfig, service)

			//rancher-compose
			mergeRancherCompose(serviceConfig, service, launchConfig, stackData.Certificates, stackData.PortRuleServices, stackData.PortRuleContainers)

			if launchConfig.Name == "" {
				compose.Services[service.Name] = serviceConfig
			} else {
				compose.Services[launchConfig.Name] = serviceConfig
			}
		}
	}

	// for standalone container export
	for _, container := range stackData.StandaloneContainers {
		containerConfig := &config.ServiceConfig{}
		// volume convert
		convertVolume(containerConfig, volumeConfig, container.DataVolumes, container.VolumeDriver, stackData.VolumeTemplates)

		//secret convert
		convertSecret(containerConfig, secretConfig, container.Secrets, stackData.Secrets)

		// container export
		mergeDockerComposeStandalone(containerConfig, container)
		mergeRancherComposeStandalone(containerConfig, container)
		compose.Containers[container.Name] = containerConfig
	}

	compose.Volumes = volumeConfig
	compose.Secrets = secretConfig
	compose.Version = "2"
	result, err := yml.Marshal(compose)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

func createSplitComposeData(stackData StackData) (string, string, error) {
	dockerCompose := config.NewConfig()
	rancherCompose := config.NewConfig()
	volumeConfig := map[string]*config.VolumeConfig{}
	secretConfig := map[string]*config.SecretConfig{}
	for _, service := range stackData.Services {
		launchConfigs := append([]v3.LaunchConfig{*service.LaunchConfig}, service.SecondaryLaunchConfigs...)
		for _, launchConfig := range launchConfigs {
			serviceDockerConfig := &config.ServiceConfig{}
			serviceRancherConfig := &config.ServiceConfig{}
			// volume convert
			convertVolume(serviceDockerConfig, volumeConfig, launchConfig.DataVolumes, launchConfig.VolumeDriver, stackData.VolumeTemplates)

			//secret convert
			convertSecret(serviceDockerConfig, secretConfig, launchConfig.Secrets, stackData.Secrets)

			// docker-compose
			mergeDockerCompose(serviceDockerConfig, launchConfig, service)

			//rancher-compose
			mergeRancherCompose(serviceRancherConfig, service, launchConfig, stackData.Certificates, stackData.PortRuleServices, stackData.PortRuleContainers)

			serviceName := ""
			if launchConfig.Name == "" {
				serviceName = service.Name
			} else {
				serviceName = launchConfig.Name
			}
			dockerCompose.Services[serviceName] = serviceDockerConfig
			rancherCompose.Services[serviceName] = serviceRancherConfig
		}
	}

	// for standalone container export
	for _, container := range stackData.StandaloneContainers {
		containerDockerConfig := &config.ServiceConfig{}
		containerRancherConfig := &config.ServiceConfig{}
		// volume convert
		convertVolume(containerDockerConfig, volumeConfig, container.DataVolumes, container.VolumeDriver, stackData.VolumeTemplates)

		//secret convert
		convertSecret(containerDockerConfig, secretConfig, container.Secrets, stackData.Secrets)

		// container export
		mergeDockerComposeStandalone(containerDockerConfig, container)
		mergeRancherComposeStandalone(containerRancherConfig, container)
		dockerCompose.Containers[container.Name] = containerDockerConfig
		rancherCompose.Containers[container.Name] = containerRancherConfig
	}
	dockerCompose.Version = "2"
	rancherCompose.Version = "2"
	dockerCompose.Secrets = secretConfig
	dockerCompose.Volumes = volumeConfig
	d, err := yml.Marshal(dockerCompose)
	if err != nil {
		return "", "", err
	}
	r, err := yml.Marshal(rancherCompose)
	if err != nil {
		return "", "", err
	}
	return string(d), string(r), nil
}

func mergeDockerCompose(serviceConfig *config.ServiceConfig, launchConfig v3.LaunchConfig, service v3.Service) {
	serviceConfig.Image = launchConfig.Image
	serviceConfig.Command = launchConfig.Command
	serviceConfig.Ports = launchConfig.Ports
	// todo: check volumeFrom
	serviceConfig.VolumesFrom = launchConfig.DataVolumesFrom
	serviceConfig.DNS = launchConfig.Dns
	serviceConfig.CapAdd = launchConfig.CapAdd
	serviceConfig.CapDrop = launchConfig.CapDrop
	serviceConfig.DNSSearch = launchConfig.DnsSearch
	serviceConfig.WorkingDir = launchConfig.WorkingDir
	serviceConfig.Entrypoint = launchConfig.EntryPoint
	serviceConfig.User = launchConfig.User
	serviceConfig.Hostname = launchConfig.Hostname
	serviceConfig.DomainName = launchConfig.DomainName
	serviceConfig.MemLimit = yaml.MemStringorInt(launchConfig.Memory)
	serviceConfig.MemReservation = yaml.MemStringorInt(launchConfig.MemoryReservation)
	serviceConfig.Privileged = launchConfig.Privileged
	serviceConfig.StdinOpen = launchConfig.StdinOpen
	serviceConfig.Sysctls = launchConfig.Sysctls
	serviceConfig.Tty = launchConfig.Tty
	serviceConfig.CPUShares = yaml.StringorInt(launchConfig.CpuShares)
	serviceConfig.BlkioWeight = yaml.StringorInt(launchConfig.BlkioWeight)
	serviceConfig.CgroupParent = launchConfig.CgroupParent
	serviceConfig.CPUPeriod = yaml.StringorInt(launchConfig.CpuPeriod)
	serviceConfig.CPUQuota = yaml.StringorInt(launchConfig.CpuQuota)
	serviceConfig.DNSOpt = launchConfig.DnsOpt
	serviceConfig.GroupAdd = launchConfig.GroupAdd
	serviceConfig.ExtraHosts = launchConfig.ExtraHosts
	serviceConfig.SecurityOpt = launchConfig.SecurityOpt
	serviceConfig.ReadOnly = launchConfig.ReadOnly
	serviceConfig.MemSwappiness = yaml.StringorInt(launchConfig.MemorySwappiness)
	serviceConfig.MemSwapLimit = yaml.MemStringorInt(launchConfig.MemorySwap)
	serviceConfig.OomKillDisable = launchConfig.OomKillDisable
	serviceConfig.ShmSize = yaml.MemStringorInt(launchConfig.ShmSize)
	serviceConfig.Uts = launchConfig.Uts
	serviceConfig.StopSignal = launchConfig.StopSignal
	serviceConfig.OomScoreAdj = yaml.StringorInt(launchConfig.OomScoreAdj)
	serviceConfig.Ipc = launchConfig.IpcMode
	serviceConfig.Isolation = launchConfig.Isolation
	serviceConfig.VolumeDriver = launchConfig.VolumeDriver
	serviceConfig.Expose = launchConfig.Expose
	convertNetworkMode(serviceConfig, launchConfig)
	serviceConfig.CPUSet = launchConfig.CpuSetCpu
	serviceConfig.Labels = launchConfig.Labels
	delete(serviceConfig.Labels, hashLabel)
	serviceConfig.Pid = launchConfig.PidMode
	serviceConfig.Devices = launchConfig.Devices
	convertEnvironmentVariable(serviceConfig, launchConfig.Environment)
	convertSelectorLabel(serviceConfig, service)
	convertLogOptions(serviceConfig, launchConfig)
	convertTmpfs(serviceConfig, launchConfig)
	convertUlimit(serviceConfig, launchConfig)
	convertRestartPolicy(serviceConfig, launchConfig)
	convertBlkioOptions(serviceConfig, launchConfig)
}

func mergeDockerComposeStandalone(serviceConfig *config.ServiceConfig, container v3.Container) {
	serviceConfig.Image = container.Image
	serviceConfig.Command = container.Command
	serviceConfig.Ports = container.Ports
	// todo: check volumeFrom
	serviceConfig.VolumesFrom = container.DataVolumesFrom
	serviceConfig.DNS = container.Dns
	serviceConfig.CapAdd = container.CapAdd
	serviceConfig.CapDrop = container.CapDrop
	serviceConfig.DNSSearch = container.DnsSearch
	serviceConfig.WorkingDir = container.WorkingDir
	serviceConfig.Entrypoint = container.EntryPoint
	serviceConfig.User = container.User
	serviceConfig.Hostname = container.Hostname
	serviceConfig.DomainName = container.DomainName
	serviceConfig.MemLimit = yaml.MemStringorInt(container.Memory)
	serviceConfig.MemReservation = yaml.MemStringorInt(container.MemoryReservation)
	serviceConfig.Privileged = container.Privileged
	serviceConfig.StdinOpen = container.StdinOpen
	serviceConfig.Sysctls = container.Sysctls
	serviceConfig.Tty = container.Tty
	serviceConfig.CPUShares = yaml.StringorInt(container.CpuShares)
	serviceConfig.BlkioWeight = yaml.StringorInt(container.BlkioWeight)
	serviceConfig.CgroupParent = container.CgroupParent
	serviceConfig.CPUPeriod = yaml.StringorInt(container.CpuPeriod)
	serviceConfig.CPUQuota = yaml.StringorInt(container.CpuQuota)
	serviceConfig.DNSOpt = container.DnsOpt
	serviceConfig.GroupAdd = container.GroupAdd
	serviceConfig.ExtraHosts = container.ExtraHosts
	serviceConfig.SecurityOpt = container.SecurityOpt
	serviceConfig.ReadOnly = container.ReadOnly
	serviceConfig.MemSwappiness = yaml.StringorInt(container.MemorySwappiness)
	serviceConfig.MemSwapLimit = yaml.MemStringorInt(container.MemorySwap)
	serviceConfig.OomKillDisable = container.OomKillDisable
	serviceConfig.ShmSize = yaml.MemStringorInt(container.ShmSize)
	serviceConfig.Uts = container.Uts
	serviceConfig.StopSignal = container.StopSignal
	serviceConfig.OomScoreAdj = yaml.StringorInt(container.OomScoreAdj)
	serviceConfig.Ipc = container.IpcMode
	serviceConfig.Isolation = container.Isolation
	serviceConfig.VolumeDriver = container.VolumeDriver
	serviceConfig.Expose = container.Expose
	convertNetworkModeStandalone(serviceConfig, container)
	serviceConfig.CPUSet = container.CpuSetCpu
	serviceConfig.Labels = container.Labels
	delete(serviceConfig.Labels, hashLabel)
	serviceConfig.Pid = container.PidMode
	serviceConfig.Devices = container.Devices
	convertEnvironmentVariable(serviceConfig, container.Environment)
	convertLogOptionsStandalone(serviceConfig, container)
	convertTmpfsStandalone(serviceConfig, container)
	convertUlimitStandalone(serviceConfig, container)
	convertRestartPolicyStandalone(serviceConfig, container)
	convertBlkioOptionsStandalone(serviceConfig, container)
}

func mergeRancherCompose(serviceConfig *config.ServiceConfig, service v3.Service, launchConfig v3.LaunchConfig, certMap map[string]v3.Certificate, serviceMap map[string]v3.Service, containerMap map[string]v3.Container) {
	serviceConfig.HealthCheck = service.HealthCheck
	serviceConfig.ExternalIps = service.ExternalIpAddresses
	if service.Kind == virtualMachine {
		serviceConfig.Type = service.Kind
	}
	serviceConfig.Metadata = service.Metadata
	delete(serviceConfig.Metadata, hashLabel)
	serviceConfig.RetainIp = launchConfig.RetainIp
	serviceConfig.NetworkDriver = service.NetworkDriver
	serviceConfig.StorageDriver = service.StorageDriver
	serviceConfig.MilliCpuReservation = yaml.StringorInt(launchConfig.MilliCpuReservation)
	convertDefaultCerts(serviceConfig, service, certMap)
	convertCerts(serviceConfig, service, certMap)
	convertLBConfig(serviceConfig, service, serviceMap, containerMap)
	convertScale(serviceConfig, service, launchConfig)
	convertServiceType(serviceConfig, service)
}

func mergeRancherComposeStandalone(serviceConfig *config.ServiceConfig, container v3.Container) {
	serviceConfig.HealthCheck = container.HealthCheck
	serviceConfig.Metadata = container.Metadata
	delete(serviceConfig.Metadata, hashLabel)
	serviceConfig.RetainIp = container.RetainIp
	serviceConfig.MilliCpuReservation = yaml.StringorInt(container.MilliCpuReservation)
}

func convertSecret(serviceConfig *config.ServiceConfig, secretConfig map[string]*config.SecretConfig, secrets []v3.SecretReference, secretMap map[string]v3.Secret) {
	serviceConfig.Secrets = []config.SecretReference{}
	for _, secretReference := range secrets {
		if secret, ok := secretMap[secretReference.SecretId]; ok {
			serviceConfig.Secrets = append(serviceConfig.Secrets, config.SecretReference{
				Source: secret.Name,
				Target: secretReference.Name,
				Uid:    secretReference.Uid,
				Gid:    secretReference.Gid,
				Mode:   secretReference.Mode,
			})
			secretConfig[secret.Name] = &config.SecretConfig{
				External: "true",
			}
		}

	}
}

func convertRestartPolicy(serviceConfig *config.ServiceConfig, launchConfig v3.LaunchConfig) {
	if launchConfig.RestartPolicy != nil {
		serviceConfig.Restart = launchConfig.RestartPolicy.Name
	}
}

func convertRestartPolicyStandalone(serviceConfig *config.ServiceConfig, container v3.Container) {
	if container.RestartPolicy != nil {
		serviceConfig.Restart = container.RestartPolicy.Name
	}
}

func convertNetworkMode(serviceConfig *config.ServiceConfig, launchConfig v3.LaunchConfig) {
	if launchConfig.NetworkMode != managedNetwork {
		if launchConfig.NetworkMode == containerNetwork {
			serviceConfig.NetworkMode = fmt.Sprintf("%s:%s", containerNetwork, launchConfig.NetworkContainerId)
		} else {
			serviceConfig.NetworkMode = launchConfig.NetworkMode
		}
	}
}

func convertNetworkModeStandalone(serviceConfig *config.ServiceConfig, container v3.Container) {
	if container.NetworkMode != managedNetwork {
		if container.NetworkMode == containerNetwork {
			serviceConfig.NetworkMode = fmt.Sprintf("%s:%s", containerNetwork, container.NetworkContainerId)
		} else {
			serviceConfig.NetworkMode = container.NetworkMode
		}
	}
}

func convertEnvironmentVariable(serviceConfig *config.ServiceConfig, envs map[string]string) {
	r := []string{}
	for k, v := range envs {
		r = append(r, fmt.Sprintf("%s=%s", k, v))
	}
	serviceConfig.Environment = r
}

func convertVolume(serviceConfig *config.ServiceConfig, volumeConfig map[string]*config.VolumeConfig, dataVolume []string, volumeDriver string, volumeTemplates map[string]v3.VolumeTemplate) {
	volumes := yaml.Volumes{}
	for _, dataVolume := range dataVolume {
		parts := strings.Split(dataVolume, ":")
		if len(parts) < 2 {
			continue
		}
		volumeName := parts[0]
		if path.IsAbs(volumeName) {
			volume := yaml.Volume{}
			if len(parts) == 2 {
				volume.Source = parts[0]
				volume.Destination = parts[1]
			} else if len(parts) == 3 {
				volume.Source = parts[0]
				volume.Destination = parts[1]
				volume.AccessMode = parts[2]
			}
			volumes.Volumes = append(volumes.Volumes, &volume)
		} else {
			if vt, ok := volumeTemplates[parts[0]]; ok {
				volumeConfig[vt.Name] = &config.VolumeConfig{
					Driver:       vt.Driver,
					DriverOpts:   vt.DriverOpts,
					PerContainer: vt.PerContainer,
					External: yaml.External{
						External: vt.External,
					},
				}
			} else {
				volumeConfig[volumeName] = &config.VolumeConfig{
					Driver: volumeDriver,
					External: yaml.External{
						External: true,
					},
				}
			}
		}
	}
	if len(volumes.Volumes) > 0 {
		serviceConfig.Volumes = &volumes
	}
}

func convertScale(serviceConfig *config.ServiceConfig, service v3.Service, launchConfig v3.LaunchConfig) {
	if launchConfig.Labels != nil {
		if _, ok := launchConfig.Labels[labelServiceGlobal]; ok {
			serviceConfig.ScaleMin = yaml.StringorInt(service.ScaleMin)
			serviceConfig.ScaleMax = yaml.StringorInt(service.ScaleMax)
			serviceConfig.ScaleIncrement = yaml.StringorInt(service.ScaleIncrement)
		} else {
			serviceConfig.Scale = yaml.StringorInt(service.Scale)
		}
	}
}

func convertServiceType(serviceConfig *config.ServiceConfig, service v3.Service) {
	if service.Type == virtualMachineKind {
		serviceConfig.Type = service.Kind
	}
}

func convertDefaultCerts(serviceConfig *config.ServiceConfig, service v3.Service, certMap map[string]v3.Certificate) {
	if service.LbConfig != nil {
		serviceConfig.DefaultCert = certMap[service.LbConfig.DefaultCertificateId].Name
	}
}

func convertCerts(serviceConfig *config.ServiceConfig, service v3.Service, certMap map[string]v3.Certificate) {
	if service.LbConfig != nil {
		r := []string{}
		for _, certID := range service.LbConfig.CertificateIds {
			r = append(r, certMap[certID].Name)
		}
		serviceConfig.Certs = r
	}
}

func convertLBConfig(serviceConfig *config.ServiceConfig, service v3.Service, serviceMap map[string]v3.Service, containerMap map[string]v3.Container) {
	if service.LbConfig != nil {
		serviceConfig.LbConfig = &config.LBConfig{
			Certs:            serviceConfig.Certs,
			DefaultCert:      serviceConfig.DefaultCert,
			PortRules:        convertPortRules(service.LbConfig.PortRules, serviceMap, containerMap),
			Config:           service.LbConfig.Config,
			StickinessPolicy: convertStickinessPolicy(*service.LbConfig.StickinessPolicy),
		}
	}
}

func convertPortRules(portRules []v3.PortRule, serviceMap map[string]v3.Service, containerMap map[string]v3.Container) []config.PortRule {
	r := []config.PortRule{}
	for _, portRule := range portRules {
		rule := config.PortRule{}
		rule.Service = serviceMap[portRule.ServiceId].Name
		rule.Hostname = portRule.Hostname
		rule.Path = portRule.Path
		rule.Container = containerMap[portRule.InstanceId].Name
		rule.Protocol = portRule.Protocol
		rule.BackendName = portRule.BackendName
		rule.Selector = portRule.Selector
		rule.Priority = int(portRule.Priority)
		rule.SourcePort = int(portRule.SourcePort)
		rule.TargetPort = int(portRule.TargetPort)
		r = append(r, rule)
	}
	return r
}

func convertStickinessPolicy(policy v3.LoadBalancerCookieStickinessPolicy) *config.LBStickinessPolicy {
	r := config.LBStickinessPolicy{}
	r.Name = policy.Name
	r.Domain = policy.Domain
	r.Cookie = policy.Cookie
	r.Indirect = policy.Indirect
	r.Mode = policy.Mode
	r.Nocache = policy.Nocache
	r.Postonly = policy.Postonly
	return &r
}

func convertSelectorLabel(serviceConfig *config.ServiceConfig, service v3.Service) {
	if service.Selector != "" {
		serviceConfig.Labels[labelSelectorContainer] = service.Selector
	}
}

func convertLogOptions(serviceConfig *config.ServiceConfig, launchConfig v3.LaunchConfig) {
	if launchConfig.LogConfig != nil {
		serviceConfig.Logging.Driver = launchConfig.LogConfig.Driver
		serviceConfig.Logging.Options = launchConfig.LogConfig.Config
	}
}

func convertLogOptionsStandalone(serviceConfig *config.ServiceConfig, container v3.Container) {
	if container.LogConfig != nil {
		serviceConfig.Logging.Driver = container.LogConfig.Driver
		serviceConfig.Logging.Options = container.LogConfig.Config
	}
}

func convertTmpfs(serviceConfig *config.ServiceConfig, launchConfig v3.LaunchConfig) {
	serviceConfig.Tmpfs = []string{}
	m := launchConfig.Tmpfs
	for k, v := range m {
		if v != "" {
			serviceConfig.Tmpfs = append(serviceConfig.Tmpfs, fmt.Sprintf("%s:%s", k, v))
		} else {
			serviceConfig.Tmpfs = append(serviceConfig.Tmpfs, k)
		}
	}
}

func convertTmpfsStandalone(serviceConfig *config.ServiceConfig, container v3.Container) {
	serviceConfig.Tmpfs = []string{}
	m := container.Tmpfs
	for k, v := range m {
		if v != "" {
			serviceConfig.Tmpfs = append(serviceConfig.Tmpfs, fmt.Sprintf("%s:%s", k, v))
		} else {
			serviceConfig.Tmpfs = append(serviceConfig.Tmpfs, k)
		}
	}
}

func convertUlimit(serviceConfig *config.ServiceConfig, launchConfig v3.LaunchConfig) {
	if len(launchConfig.Ulimits) > 0 {
		serviceConfig.Ulimits.Elements = []yaml.Ulimit{}
		for _, ulimit := range launchConfig.Ulimits {
			serviceConfig.Ulimits.Elements = append(serviceConfig.Ulimits.Elements, yaml.NewUlimit(ulimit.Name, ulimit.Soft, ulimit.Hard))
		}
	}
}

func convertUlimitStandalone(serviceConfig *config.ServiceConfig, container v3.Container) {
	if len(container.Ulimits) > 0 {
		serviceConfig.Ulimits.Elements = []yaml.Ulimit{}
		for _, ulimit := range container.Ulimits {
			serviceConfig.Ulimits.Elements = append(serviceConfig.Ulimits.Elements, yaml.NewUlimit(ulimit.Name, ulimit.Soft, ulimit.Hard))
		}
	}
}

func convertBlkioOptions(serviceConfig *config.ServiceConfig, launchConfig v3.LaunchConfig) {
	for device, option := range launchConfig.BlkioDeviceOptions {
		opt := option.(map[string]interface{})
		for t, v := range opt {
			value := fmt.Sprintf("%v:%v", device, v)
			if t == blkioWeight {
				serviceConfig.BlkioWeightDevice = append(serviceConfig.BlkioWeightDevice, value)
			} else if t == blkioReadBps {
				serviceConfig.DeviceReadBps = append(serviceConfig.DeviceReadBps, value)
			} else if t == blkioReadIops {
				serviceConfig.DeviceReadIOps = append(serviceConfig.DeviceReadIOps, value)
			} else if t == blkioWriteBps {
				serviceConfig.DeviceWriteBps = append(serviceConfig.DeviceWriteBps, value)
			} else if t == blkioWriteIops {
				serviceConfig.DeviceWriteIOps = append(serviceConfig.DeviceWriteIOps, value)
			}
		}
	}
}

func convertBlkioOptionsStandalone(serviceConfig *config.ServiceConfig, container v3.Container) {
	for device, option := range container.BlkioDeviceOptions {
		opt := option.(map[string]interface{})
		for t, v := range opt {
			value := fmt.Sprintf("%v:%v", device, v)
			if t == blkioWeight {
				serviceConfig.BlkioWeightDevice = append(serviceConfig.BlkioWeightDevice, value)
			} else if t == blkioReadBps {
				serviceConfig.DeviceReadBps = append(serviceConfig.DeviceReadBps, value)
			} else if t == blkioReadIops {
				serviceConfig.DeviceReadIOps = append(serviceConfig.DeviceReadIOps, value)
			} else if t == blkioWriteBps {
				serviceConfig.DeviceWriteBps = append(serviceConfig.DeviceWriteBps, value)
			} else if t == blkioWriteIops {
				serviceConfig.DeviceWriteIOps = append(serviceConfig.DeviceWriteIOps, value)
			}
		}
	}
}
