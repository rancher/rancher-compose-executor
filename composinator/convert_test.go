package composinator

import (
	"testing"

	v3 "github.com/rancher/go-rancher/v3"
	"github.com/rancher/rancher-compose-executor/config"
	cyaml "github.com/rancher/rancher-compose-executor/yaml"
	"gopkg.in/check.v1"
	"gopkg.in/yaml.v2"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	check.TestingT(t)
}

type ConvertTestSuite struct {
}

var _ = check.Suite(&ConvertTestSuite{})

func (s *ConvertTestSuite) SetUpSuite(c *check.C) {}

func (s *ConvertTestSuite) TestExportConfig(c *check.C) {
	service := v3.Service{}
	metadata := map[string]interface{}{
		"io.rancher.service.hash": "088b54be-2b79-99e30b3a1a24",
		"$bar": map[string]interface{}{
			"metadata": []map[string]interface{}{
				{
					"$id$$foo$bar$$": "${HOSTNAME}",
				},
			},
		},
	}
	restartPolicy := v3.RestartPolicy{
		MaximumRetryCount: 2,
		Name:              "on-failure",
	}
	service.LaunchConfig = &v3.LaunchConfig{
		Image: "strongmonkey/test",
		Labels: map[string]string{
			"io.rancher.scheduler.global": "true",
			"io.rancher.service.hash":     "088b54be-2b79-99e30b3a1a24",
		},
		CpuSetCpu:           "0,1",
		RestartPolicy:       &restartPolicy,
		PidMode:             "host",
		Memory:              1048576,
		MemorySwap:          2097152,
		MemoryReservation:   4194304,
		MilliCpuReservation: 1000,
		Devices:             []string{"/dev/sdc:/dev/xsdc:rwm"},
		LogConfig: &v3.LogConfig{
			Driver: "json-file",
			Config: map[string]string{
				"labels": "foo",
			},
		},
		BlkioWeight:      100,
		CpuPeriod:        10000,
		CpuQuota:         20000,
		MemorySwappiness: 50,
		OomScoreAdj:      500,
		ShmSize:          67108864,
		Uts:              "host",
		Tty:              true,
		IpcMode:          "host",
		StopSignal:       "SIGTERM",
		GroupAdd:         []string{"root"},
		CgroupParent:     "parent",
		ExtraHosts:       []string{"host1", "host2"},
		SecurityOpt:      []string{"sopt1", "sopt2"},
		ReadOnly:         true,
		OomKillDisable:   true,
		Isolation:        "hyper-v",
		DnsOpt:           []string{"opt"},
		DnsSearch:        []string{"192.168.1.1"},
		CpuShares:        100,
		BlkioDeviceOptions: map[string]interface{}{
			"/dev/sda": map[string]interface{}{
				"readIops":  1000,
				"writeIops": 2000,
			},
			"/dev/null": map[string]interface{}{
				"readBps":  3000,
				"writeBps": 3000,
				"weight":   3000,
			},
		},
		Tmpfs: map[string]string{
			"/run": "rw",
		},
		Sysctls: map[string]string{
			"net.ipv4.ip_forward": "1",
		},
		Ulimits: []v3.Ulimit{
			{
				Name: "cpu",
				Soft: 1234,
				Hard: 1234,
			},
			{
				Name: "nporc",
				Soft: 1234,
			},
		},
	}
	service.ServiceLinks = []v3.Link{
		{
			Name:  "default1/link-1",
			Alias: "l1",
		},
		{
			Name:  "default/link-2",
			Alias: "l2",
		},
	}
	service.Metadata = metadata
	service.LaunchConfig.RetainIp = true
	service.Name = "strongmonkey"
	service.Id = "test"
	stackData := StackData{
		Services:             map[string]v3.Service{},
		StandaloneContainers: map[string]v3.Container{},
		VolumeTemplates:      map[string]v3.VolumeTemplate{},
		Certificates:         map[string]v3.Certificate{},
		PortRuleServices:     map[string]v3.Service{},
		PortRuleContainers:   map[string]v3.Container{},
		Secrets:              map[string]v3.Secret{},
	}
	stackData.Services[service.Id] = service
	stackData.StackName = "default"
	dockerCompose, rancherCompose, _, err := createComposeData(stackData, "split")
	if err != nil {
		c.Fatal(err)
	}
	dockerConfig := config.Config{}
	if err := yaml.Unmarshal([]byte(dockerCompose), &dockerConfig); err != nil {
		c.Fatal(err)
	}
	rancherConfig := config.Config{}
	if err := yaml.Unmarshal([]byte(rancherCompose), &rancherConfig); err != nil {
		c.Fatal(err)
	}
	c.Assert(dockerConfig.Version, check.Equals, "2")
	c.Assert(dockerConfig.Services[service.Name], check.NotNil)
	serviceConfig := dockerConfig.Services[service.Name]
	c.Assert(serviceConfig.Image, check.Equals, "strongmonkey/test")
	c.Assert(serviceConfig.CPUSet, check.Equals, "0,1")
	c.Assert(serviceConfig.Labels, check.DeepEquals, cyaml.SliceorMap{
		"io.rancher.scheduler.global": "true",
	})
	c.Assert(serviceConfig.DeviceReadIOps, check.DeepEquals, cyaml.MaporColonSlice{"/dev/sda:1000"})
	c.Assert(serviceConfig.DeviceWriteIOps, check.DeepEquals, cyaml.MaporColonSlice{"/dev/sda:2000"})
	c.Assert(serviceConfig.BlkioWeightDevice, check.DeepEquals, []string{"/dev/null:3000"})
	c.Assert(serviceConfig.DeviceWriteBps, check.DeepEquals, cyaml.MaporColonSlice{"/dev/null:3000"})
	c.Assert(serviceConfig.DeviceReadBps, check.DeepEquals, cyaml.MaporColonSlice{"/dev/null:3000"})
	c.Assert(serviceConfig.Restart, check.Equals, "on-failure")
	c.Assert(serviceConfig.Logging.Driver, check.Equals, "json-file")
	c.Assert(serviceConfig.Logging.Options, check.DeepEquals, map[string]string{
		"labels": "foo",
	})
	c.Assert(serviceConfig.Pid, check.Equals, "host")
	c.Assert(serviceConfig.MemLimit, check.Equals, cyaml.MemStringorInt(1048576))
	c.Assert(serviceConfig.MemSwapLimit, check.DeepEquals, cyaml.MemStringorInt(2097152))
	c.Assert(serviceConfig.MemReservation, check.DeepEquals, cyaml.MemStringorInt(4194304))
	c.Assert(serviceConfig.Devices, check.DeepEquals, []string{"/dev/sdc:/dev/xsdc:rwm"})
	c.Assert(serviceConfig.BlkioWeight, check.Equals, cyaml.StringorInt(100))
	c.Assert(serviceConfig.CPUPeriod, check.Equals, cyaml.StringorInt(10000))
	c.Assert(serviceConfig.CPUQuota, check.Equals, cyaml.StringorInt(20000))
	c.Assert(serviceConfig.MemSwappiness, check.Equals, cyaml.StringorInt(50))
	c.Assert(serviceConfig.OomScoreAdj, check.Equals, cyaml.StringorInt(500))
	c.Assert(serviceConfig.ShmSize, check.Equals, cyaml.MemStringorInt(67108864))
	c.Assert(serviceConfig.Uts, check.Equals, "host")
	c.Assert(serviceConfig.Tty, check.Equals, true)
	c.Assert(serviceConfig.Ipc, check.Equals, "host")
	c.Assert(serviceConfig.StopSignal, check.Equals, "SIGTERM")
	c.Assert(serviceConfig.GroupAdd, check.DeepEquals, []string{"root"})
	c.Assert(serviceConfig.CgroupParent, check.Equals, "parent")
	c.Assert(serviceConfig.ExtraHosts, check.DeepEquals, []string{"host1", "host2"})
	c.Assert(serviceConfig.SecurityOpt, check.DeepEquals, []string{"sopt1", "sopt2"})
	c.Assert(serviceConfig.ReadOnly, check.Equals, true)
	c.Assert(serviceConfig.OomKillDisable, check.Equals, true)
	c.Assert(serviceConfig.Isolation, check.Equals, "hyper-v")
	c.Assert(serviceConfig.DNSOpt, check.DeepEquals, []string{"opt"})
	c.Assert(serviceConfig.DNSSearch, check.DeepEquals, cyaml.Stringorslice([]string{"192.168.1.1"}))
	c.Assert(serviceConfig.CPUShares, check.Equals, cyaml.StringorInt(100))
	c.Assert(serviceConfig.Tmpfs, check.DeepEquals, cyaml.Stringorslice([]string{"/run:rw"}))
	c.Assert(serviceConfig.Ulimits, check.DeepEquals, cyaml.Ulimits{
		Elements: []cyaml.Ulimit{cyaml.NewUlimit("cpu", 1234, 1234), cyaml.NewUlimit("nporc", 1234, 0)},
	})
	c.Assert(serviceConfig.Sysctls, check.DeepEquals, cyaml.SliceorMap{
		"net.ipv4.ip_forward": "1",
	})
	c.Assert(serviceConfig.Links, check.DeepEquals, cyaml.MaporColonSlice{"l1:default1/link-1", "l2:link-2"})
	c.Assert(rancherConfig.Version, check.Equals, "2")
	rancherServiceConfig := rancherConfig.Services[service.Name]
	c.Assert(rancherServiceConfig.Scale, check.Equals, cyaml.StringorInt(0))
	c.Assert(rancherServiceConfig.Metadata, check.HasLen, 1)
	c.Assert(rancherServiceConfig.RetainIp, check.Equals, true)
	c.Assert(rancherServiceConfig.MilliCpuReservation, check.Equals, cyaml.StringorInt(1000))
}

func (s *ConvertTestSuite) TestLoadBalanceExport(c *check.C) {
	service := v3.Service{}
	service.LaunchConfig = &v3.LaunchConfig{}
	service.LbConfig = &v3.LbConfig{
		DefaultCertificateId: "1c1",
		CertificateIds:       []string{"1c2"},
		Config:               "config",
		PortRules: []v3.PortRule{
			{
				Hostname:    "foo",
				Path:        "bar",
				SourcePort:  32,
				Priority:    10,
				ServiceId:   "1s2",
				TargetPort:  42,
				BackendName: "myBackend",
			},
		},
		StickinessPolicy: &v3.LoadBalancerCookieStickinessPolicy{
			Name:     "policy2",
			Cookie:   "cookie1",
			Domain:   ".test.com",
			Indirect: true,
			Nocache:  true,
			Postonly: true,
			Mode:     "insert",
		},
	}
	service.Name = "strongmonkey"
	serviceMap := map[string]v3.Service{
		"1s2": {
			Name: "test",
		},
	}
	certMap := map[string]v3.Certificate{
		"1c1": {
			Name: "cert1",
		},
		"1c2": {
			Name: "cert2",
		},
	}
	service.Id = "test"
	stackData := StackData{
		Services:             map[string]v3.Service{},
		StandaloneContainers: map[string]v3.Container{},
		VolumeTemplates:      map[string]v3.VolumeTemplate{},
		Certificates:         map[string]v3.Certificate{},
		PortRuleServices:     map[string]v3.Service{},
		PortRuleContainers:   map[string]v3.Container{},
		Secrets:              map[string]v3.Secret{},
	}
	stackData.Services[service.Id] = service
	stackData.PortRuleServices = serviceMap
	stackData.Certificates = certMap
	_, rancherCompose, _, err := createComposeData(stackData, "split")
	if err != nil {
		c.Fatal(err)
	}
	rancherConfig := config.Config{}
	if err := yaml.Unmarshal([]byte(rancherCompose), &rancherConfig); err != nil {
		c.Fatal(err)
	}
	rancherServiceConfig := rancherConfig.Services[service.Name]
	c.Assert(rancherConfig.Version, check.Equals, "2")
	c.Assert(rancherServiceConfig.LbConfig.PortRules, check.DeepEquals, []config.PortRule{
		{
			Hostname:    "foo",
			Path:        "bar",
			SourcePort:  32,
			Priority:    10,
			Service:     "test",
			TargetPort:  42,
			BackendName: "myBackend",
		},
	})
	c.Assert(rancherServiceConfig.LbConfig.Config, check.DeepEquals, "config")
	c.Assert(rancherServiceConfig.LbConfig.DefaultCert, check.DeepEquals, "cert1")
	c.Assert(rancherServiceConfig.LbConfig.Certs, check.DeepEquals, []string{"cert2"})
	c.Assert(*rancherServiceConfig.LbConfig.StickinessPolicy, check.DeepEquals, config.LBStickinessPolicy{
		Name:     "policy2",
		Cookie:   "cookie1",
		Domain:   ".test.com",
		Indirect: true,
		Nocache:  true,
		Postonly: true,
		Mode:     "insert",
	})
}

func (s *ConvertTestSuite) TestSecretExport(c *check.C) {
	service := v3.Service{}
	service.Name = "strongmonkey"
	service.LaunchConfig = &v3.LaunchConfig{
		Secrets: []v3.SecretReference{
			{
				Name:     "my_secret1",
				SecretId: "1s1",
			},
			{
				Name:     "my_secret2",
				SecretId: "1s2",
				Mode:     "444",
				Uid:      "0",
				Gid:      "0",
			},
		},
	}
	secretMap := map[string]v3.Secret{
		"1s1": {
			Name: "secret1",
		},
		"1s2": {
			Name: "secret2",
		},
	}
	service.Id = "test"
	stackData := StackData{
		Services:             map[string]v3.Service{},
		StandaloneContainers: map[string]v3.Container{},
		VolumeTemplates:      map[string]v3.VolumeTemplate{},
		Certificates:         map[string]v3.Certificate{},
		PortRuleServices:     map[string]v3.Service{},
		PortRuleContainers:   map[string]v3.Container{},
		Secrets:              map[string]v3.Secret{},
	}
	stackData.Services[service.Id] = service
	stackData.Secrets = secretMap
	dockerCompose, _, _, err := createComposeData(stackData, "split")
	if err != nil {
		c.Fatal(err)
	}
	dockerConfig := config.Config{}
	if err := yaml.Unmarshal([]byte(dockerCompose), &dockerConfig); err != nil {
		c.Fatal(err)
	}
	c.Assert(dockerConfig.Version, check.Equals, "2")
	c.Assert(dockerConfig.Secrets["secret1"].External, check.DeepEquals, "true")
	c.Assert(dockerConfig.Secrets["secret2"].External, check.DeepEquals, "true")
	serviceConfig := dockerConfig.Services[service.Name]
	c.Assert(serviceConfig.Secrets, check.DeepEquals, config.SecretReferences{
		{
			Source: "secret1",
			Target: "my_secret1",
		},
		{
			Source: "secret2",
			Target: "my_secret2",
			Uid:    "0",
			Gid:    "0",
			Mode:   "444",
		},
	})
}

func (s *ConvertTestSuite) TestConvertVolume(c *check.C) {
	service := v3.Service{}
	service.Name = "strongmonkey"
	service.LaunchConfig = &v3.LaunchConfig{
		DataVolumes: []string{"foo:/data"},
	}
	volumeMap := map[string]v3.VolumeTemplate{
		"foo": {
			Name:   "foo",
			Driver: "nfs",
			DriverOpts: map[string]string{
				"size": "1",
			},
			PerContainer: true,
		},
	}
	service.Id = "test"
	stackData := StackData{
		Services:             map[string]v3.Service{},
		StandaloneContainers: map[string]v3.Container{},
		VolumeTemplates:      map[string]v3.VolumeTemplate{},
		Certificates:         map[string]v3.Certificate{},
		PortRuleServices:     map[string]v3.Service{},
		PortRuleContainers:   map[string]v3.Container{},
		Secrets:              map[string]v3.Secret{},
	}
	stackData.Services[service.Id] = service
	stackData.VolumeTemplates = volumeMap
	dockerCompose, _, _, err := createComposeData(stackData, "split")
	if err != nil {
		c.Fatal(err)
	}
	dockerConfig := config.Config{}
	if err := yaml.Unmarshal([]byte(dockerCompose), &dockerConfig); err != nil {
		c.Fatal(err)
	}
	volumeConfig := dockerConfig.Volumes["foo"]
	c.Assert(volumeConfig.PerContainer, check.Equals, true)
	c.Assert(volumeConfig.Driver, check.Equals, "nfs")
	c.Assert(volumeConfig.DriverOpts, check.DeepEquals, map[string]string{
		"size": "1",
	})
}

func (s *ConvertTestSuite) TestConvertCombined(c *check.C) {
	service := v3.Service{}
	service.LaunchConfig = &v3.LaunchConfig{
		Image: "strongmonkey/test",
	}
	service.LaunchConfig.RetainIp = true
	service.Name = "strongmonkey"
	service.Id = "test"
	stackData := StackData{
		Services:             map[string]v3.Service{},
		StandaloneContainers: map[string]v3.Container{},
		VolumeTemplates:      map[string]v3.VolumeTemplate{},
		Certificates:         map[string]v3.Certificate{},
		PortRuleServices:     map[string]v3.Service{},
		PortRuleContainers:   map[string]v3.Container{},
		Secrets:              map[string]v3.Secret{},
	}
	stackData.Services[service.Id] = service
	_, _, compose, err := createComposeData(stackData, "combined")
	if err != nil {
		c.Fatal(err)
	}
	config := config.Config{}
	if err := yaml.Unmarshal([]byte(compose), &config); err != nil {
		c.Fatal(err)
	}
	sconfig := config.Services[service.Name]
	c.Assert(sconfig.Image, check.Equals, "strongmonkey/test")
	c.Assert(sconfig.RetainIp, check.Equals, true)
}

func (s *ConvertTestSuite) TestConvertStandaloneContainer(c *check.C) {
	container := v3.Container{}
	container.Image = "strongmonkey/test"
	container.RetainIp = true
	container.Name = "strongmonkey"
	container.Id = "test"
	stackData := StackData{
		Services:             map[string]v3.Service{},
		StandaloneContainers: map[string]v3.Container{},
		VolumeTemplates:      map[string]v3.VolumeTemplate{},
		Certificates:         map[string]v3.Certificate{},
		PortRuleServices:     map[string]v3.Service{},
		PortRuleContainers:   map[string]v3.Container{},
		Secrets:              map[string]v3.Secret{},
	}
	stackData.StandaloneContainers[container.Id] = container
	_, _, compose, err := createComposeData(stackData, "combined")
	if err != nil {
		c.Fatal(err)
	}
	config := config.Config{}
	if err := yaml.Unmarshal([]byte(compose), &config); err != nil {
		c.Fatal(err)
	}
	sconfig := config.Containers[container.Name]
	c.Assert(sconfig.Image, check.Equals, "strongmonkey/test")
	c.Assert(sconfig.RetainIp, check.Equals, true)
}
