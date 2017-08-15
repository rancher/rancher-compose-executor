package convert

import (
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/blkiodev"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/libcompose/utils"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/rancher-compose-executor/config"
	"github.com/rancher/rancher-compose-executor/project"
	"github.com/rancher/rancher-compose-executor/yaml"
)

const (
	readIops  = "readIops"
	writeIops = "writeIops"
	readBps   = "readBps"
	writeBps  = "writeBps"
	weight    = "weight"
)

func serviceConfigToLaunchConfig(serviceConfig config.ServiceConfig, p *project.Project) (client.LaunchConfig, error) {
	var launchConfig client.LaunchConfig

	launchConfig.BlkioWeight = int64(serviceConfig.BlkioWeight)
	launchConfig.CapAdd = serviceConfig.CapAdd
	launchConfig.CapDrop = serviceConfig.CapDrop
	launchConfig.CgroupParent = serviceConfig.CgroupParent
	launchConfig.Command = strslice.StrSlice(utils.CopySlice(serviceConfig.Command))
	launchConfig.CpuPeriod = int64(serviceConfig.CPUPeriod)
	launchConfig.CpuQuota = int64(serviceConfig.CPUQuota)
	launchConfig.CpuSet = serviceConfig.CPUSet
	launchConfig.CpuShares = int64(serviceConfig.CPUShares)
	launchConfig.DataVolumesFrom = serviceConfig.VolumesFrom
	launchConfig.DataVolumes = volumes(serviceConfig, p)
	launchConfig.Devices = setupDevice(serviceConfig.Devices)
	launchConfig.DnsOpt = serviceConfig.DNSOpt
	launchConfig.DnsSearch = serviceConfig.DNSSearch
	launchConfig.Dns = serviceConfig.DNS
	launchConfig.DomainName = serviceConfig.DomainName
	launchConfig.EntryPoint = strslice.StrSlice(utils.CopySlice(serviceConfig.Entrypoint))
	launchConfig.Environment = mapToMap(serviceConfig.Environment.ToMap())
	launchConfig.ExtraHosts = serviceConfig.ExtraHosts
	launchConfig.Expose = serviceConfig.Expose
	launchConfig.GroupAdd = serviceConfig.GroupAdd
	launchConfig.HealthCheck = serviceConfig.HealthCheck
	launchConfig.Hostname = serviceConfig.Hostname
	launchConfig.Image = serviceConfig.Image
	launchConfig.IpcMode = serviceConfig.Ipc
	launchConfig.Isolation = serviceConfig.Isolation
	launchConfig.Labels = labels(serviceConfig.Labels)
	launchConfig.LogConfig = toRancherLogOption(serviceConfig.Logging)
	launchConfig.Memory = int64(serviceConfig.MemLimit)
	launchConfig.MemoryMb = int64(serviceConfig.Memory)
	launchConfig.MemoryReservation = int64(serviceConfig.MemReservation)
	launchConfig.MemorySwap = int64(serviceConfig.MemSwapLimit)
	launchConfig.MemorySwappiness = int64(serviceConfig.MemSwappiness)
	launchConfig.NetworkMode = serviceConfig.NetworkMode
	launchConfig.OomKillDisable = serviceConfig.OomKillDisable
	launchConfig.OomScoreAdj = int64(serviceConfig.OomScoreAdj)
	launchConfig.PidMode = serviceConfig.Pid
	launchConfig.Ports = serviceConfig.Ports
	launchConfig.Privileged = serviceConfig.Privileged
	launchConfig.ReadOnly = serviceConfig.ReadOnly
	launchConfig.SecurityOpt = serviceConfig.SecurityOpt
	launchConfig.ShmSize = int64(serviceConfig.ShmSize)
	launchConfig.StdinOpen = serviceConfig.StdinOpen
	launchConfig.StopSignal = serviceConfig.StopSignal
	launchConfig.Sysctls = mapToMap(serviceConfig.Sysctls)
	launchConfig.Tmpfs = tmpfsToMap(serviceConfig.Tmpfs)
	launchConfig.Tty = serviceConfig.Tty
	launchConfig.Ulimits = toRancherUlimit(serviceConfig.Ulimits)
	launchConfig.User = serviceConfig.User
	launchConfig.Uts = serviceConfig.Uts
	launchConfig.VolumeDriver = serviceConfig.VolumeDriver
	launchConfig.WorkingDir = serviceConfig.WorkingDir

	options, err := toBlkioOptions(serviceConfig)
	if err != nil {
		return client.LaunchConfig{}, err
	}
	launchConfig.BlkioDeviceOptions = options

	if isContainerRef(launchConfig.NetworkMode) {
		launchConfig.NetworkContainerId = containerRefId(launchConfig.NetworkMode)
		launchConfig.NetworkMode = "container"
	}

	if isContainerRef(launchConfig.PidMode) {
		launchConfig.PidContainerId = containerRefId(launchConfig.PidMode)
		launchConfig.PidMode = "container"
	}

	if isContainerRef(launchConfig.IpcMode) {
		launchConfig.IpcContainerId = containerRefId(launchConfig.IpcMode)
		launchConfig.IpcMode = "container"
	}

	if strings.EqualFold(launchConfig.Kind, "virtual_machine") || strings.EqualFold(launchConfig.Kind, "virtualmachine") {
		launchConfig.Kind = "virtualMachine"
	}

	return launchConfig, nil
}

func isContainerRef(ref string) bool {
	return strings.HasPrefix(ref, "container:")
}

func containerRefId(ref string) string {
	return strings.TrimPrefix(ref, "container:")
}

func labels(labels map[string]string) map[string]interface{} {
	result := map[string]interface{}{}
	for k, v := range labels {
		// Remove legacy lb labels
		if !strings.HasPrefix(k, "io.rancher.loadbalancer") && !strings.HasPrefix(k, "io.rancher.service.selector") {
			result[k] = v
		}
	}
	return result
}

func mapToMap(m map[string]string) map[string]interface{} {
	r := map[string]interface{}{}
	for k, v := range m {
		r[k] = v
	}
	return r
}

func volumes(c config.ServiceConfig, p *project.Project) []string {
	if c.Volumes == nil {
		return []string{}
	}
	volumes := []string{}
	for _, v := range c.Volumes.Volumes {
		volumes = append(volumes, v.String())
	}
	return volumes
}

func toBlkioOptions(c config.ServiceConfig) (map[string]interface{}, error) {
	opts := make(map[string]map[string]uint64)

	blkioDeviceReadBps, err := getThrottleDevice(c.DeviceReadBps)
	if err != nil {
		return nil, err
	}

	blkioDeviceReadIOps, err := getThrottleDevice(c.DeviceReadIOps)
	if err != nil {
		return nil, err
	}

	blkioDeviceWriteBps, err := getThrottleDevice(c.DeviceWriteBps)
	if err != nil {
		return nil, err
	}

	blkioDeviceWriteIOps, err := getThrottleDevice(c.DeviceWriteIOps)
	if err != nil {
		return nil, err
	}

	blkioWeight, err := getThrottleDevice(c.BlkioWeightDevice)
	if err != nil {
		return nil, err
	}

	for _, rbps := range blkioDeviceReadBps {
		_, ok := opts[rbps.Path]
		if !ok {
			opts[rbps.Path] = map[string]uint64{}
		}
		opts[rbps.Path][readBps] = rbps.Rate
	}

	for _, riops := range blkioDeviceReadIOps {
		_, ok := opts[riops.Path]
		if !ok {
			opts[riops.Path] = map[string]uint64{}
		}
		opts[riops.Path][readIops] = riops.Rate
	}

	for _, wbps := range blkioDeviceWriteBps {
		_, ok := opts[wbps.Path]
		if !ok {
			opts[wbps.Path] = map[string]uint64{}
		}
		opts[wbps.Path][writeBps] = wbps.Rate
	}

	for _, wiops := range blkioDeviceWriteIOps {
		_, ok := opts[wiops.Path]
		if !ok {
			opts[wiops.Path] = map[string]uint64{}
		}
		opts[wiops.Path][writeIops] = wiops.Rate
	}

	for _, w := range blkioWeight {
		_, ok := opts[w.Path]
		if !ok {
			opts[w.Path] = map[string]uint64{}
		}
		opts[w.Path][weight] = w.Rate
	}

	result := make(map[string]interface{})
	for k, v := range opts {
		result[k] = v
	}
	return result, nil
}

func getThrottleDevice(throttleConfig yaml.MaporColonSlice) ([]*blkiodev.ThrottleDevice, error) {
	throttleDevice := []*blkiodev.ThrottleDevice{}
	for _, deviceWriteIOps := range throttleConfig {
		split := strings.Split(deviceWriteIOps, ":")
		rate, err := strconv.ParseUint(split[1], 10, 64)
		if err != nil {
			return nil, err
		}

		throttleDevice = append(throttleDevice, &blkiodev.ThrottleDevice{
			Path: split[0],
			Rate: rate,
		})
	}

	return throttleDevice, nil
}

func tmpfsToMap(tmpfs []string) map[string]interface{} {
	r := make(map[string]interface{})
	for _, v := range tmpfs {
		parts := strings.SplitN(v, ":", 2)
		if len(parts) == 1 {
			r[parts[0]] = ""
		} else if len(parts) == 2 {
			r[parts[0]] = parts[1]
		}
	}
	return r
}

func toRancherUlimit(ulimits yaml.Ulimits) []client.Ulimit {
	r := []client.Ulimit{}
	for _, u := range ulimits.Elements {
		r = append(r, client.Ulimit{Name: u.Name, Soft: u.Soft, Hard: u.Hard})
	}
	return r
}

func toRancherLogOption(log config.Log) *client.LogConfig {
	var r client.LogConfig
	r.Driver = log.Driver
	r.Config = mapToMap(log.Options)
	return &r
}

func setupDevice(devices []string) []string {
	r := []string{}
	for _, d := range devices {
		tmp := d
		parts := strings.SplitN(d, ":", 3)
		if len(parts) == 2 {
			tmp = tmp + ":rwm"
		}
		r = append(r, tmp)
	}
	return r
}

func isNamedVolume(volume string) bool {
	return !strings.HasPrefix(volume, ".") && !strings.HasPrefix(volume, "/") && !strings.HasPrefix(volume, "~")
}
