package convert

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/docker/docker/api/types/blkiodev"
	"github.com/docker/docker/api/types/container"
	shlex "github.com/flynn/go-shlex"
	"github.com/rancher/rancher-compose-executor/config"
	"github.com/rancher/rancher-compose-executor/lookup"
	"github.com/rancher/rancher-compose-executor/project"
	"github.com/rancher/rancher-compose-executor/yaml"
	"github.com/stretchr/testify/assert"
)

func TestParseCommand(t *testing.T) {
	exp := []string{"sh", "-c", "exec /opt/bin/flanneld -logtostderr=true -iface=${NODE_IP}"}
	cmd, err := shlex.Split("sh -c 'exec /opt/bin/flanneld -logtostderr=true -iface=${NODE_IP}'")
	assert.Nil(t, err)
	assert.Equal(t, exp, cmd)
}

func TestParseBindsAndVolumes(t *testing.T) {
	ctx := project.Context{}
	ctx.ComposeFiles = []string{"foo/docker-compose.yml"}
	ctx.ResourceLookup = &lookup.FileResourceLookup{}

	abs, err := filepath.Abs(".")
	assert.Nil(t, err)
	cfg, hostCfg, err := Convert(&config.ServiceConfig{
		Volumes: &yaml.Volumes{
			Volumes: []*yaml.Volume{
				{
					Destination: "/foo",
				},
				{
					Source:      "/home",
					Destination: "/home",
				},
				{
					Destination: "/bar/baz",
				},
				{
					Source:      ".",
					Destination: "/home",
				},
				{
					Source:      "/usr/lib",
					Destination: "/usr/lib",
					AccessMode:  "ro",
				},
			},
		},
	}, ctx)
	assert.Nil(t, err)
	assert.Equal(t, map[string]struct{}{"/foo": {}, "/bar/baz": {}}, cfg.Volumes)
	assert.Equal(t, []string{"/home:/home", abs + "/foo:/home", "/usr/lib:/usr/lib:ro"}, hostCfg.Binds)
}

func TestParseLabels(t *testing.T) {
	ctx := project.Context{}
	ctx.ComposeFiles = []string{"foo/docker-compose.yml"}
	ctx.ResourceLookup = &lookup.FileResourceLookup{}
	bashCmd := "bash"
	fooLabel := "foo.label"
	fooLabelValue := "service.config.value"
	sc := &config.ServiceConfig{
		Entrypoint: yaml.Command([]string{bashCmd}),
		Labels:     yaml.SliceorMap{fooLabel: "service.config.value"},
	}
	cfg, _, err := Convert(sc, ctx)
	assert.Nil(t, err)

	cfg.Labels[fooLabel] = "FUN"
	cfg.Entrypoint[0] = "less"

	assert.Equal(t, fooLabelValue, sc.Labels[fooLabel])
	assert.Equal(t, "FUN", cfg.Labels[fooLabel])

	assert.Equal(t, yaml.Command{bashCmd}, sc.Entrypoint)
	assert.Equal(t, []string{"less"}, []string(cfg.Entrypoint))
}

func TestGroupAdd(t *testing.T) {
	ctx := project.Context{}
	sc := &config.ServiceConfig{
		GroupAdd: []string{
			"root",
			"1",
		},
	}
	_, hostCfg, err := Convert(sc, ctx)
	assert.Nil(t, err)

	assert.True(t, reflect.DeepEqual([]string{
		"root",
		"1",
	}, hostCfg.GroupAdd))
}

func TestBlkioWeight(t *testing.T) {
	ctx := project.Context{}
	sc := &config.ServiceConfig{
		BlkioWeight: 10,
	}
	_, hostCfg, err := Convert(sc, ctx)
	assert.Nil(t, err)

	assert.Equal(t, uint16(10), hostCfg.BlkioWeight)
}

func TestBlkioWeightDevices(t *testing.T) {
	ctx := project.Context{}
	sc := &config.ServiceConfig{
		BlkioWeightDevice: []string{
			"/dev/sda:10",
		},
	}
	_, hostCfg, err := Convert(sc, ctx)
	assert.Nil(t, err)

	assert.True(t, reflect.DeepEqual([]*blkiodev.WeightDevice{
		&blkiodev.WeightDevice{
			Path:   "/dev/sda",
			Weight: 10,
		},
	}, hostCfg.BlkioWeightDevice))
}

func TestCPUPeriod(t *testing.T) {
	ctx := project.Context{}
	sc := &config.ServiceConfig{
		CPUPeriod: 50000,
	}
	_, hostCfg, err := Convert(sc, ctx)
	assert.Nil(t, err)

	assert.Equal(t, int64(50000), hostCfg.CPUPeriod)
}

func TestDNSOpt(t *testing.T) {
	ctx := project.Context{}
	sc := &config.ServiceConfig{
		DNSOpt: []string{
			"use-vc",
			"no-tld-query",
		},
	}
	_, hostCfg, err := Convert(sc, ctx)
	assert.Nil(t, err)

	assert.True(t, reflect.DeepEqual([]string{
		"use-vc",
		"no-tld-query",
	}, hostCfg.DNSOptions))
}

func TestInit(t *testing.T) {
	ctx := project.Context{}
	sc := &config.ServiceConfig{
		Init: true,
	}
	_, hostCfg, err := Convert(sc, ctx)
	assert.Nil(t, err)

	assert.True(t, *hostCfg.Init)
}

func TestMemSwappiness(t *testing.T) {
	ctx := project.Context{}
	sc := &config.ServiceConfig{
		MemSwappiness: yaml.StringorInt(10),
	}
	_, hostCfg, err := Convert(sc, ctx)
	assert.Nil(t, err)

	assert.Equal(t, int64(10), *hostCfg.MemorySwappiness)
}

func TestMemReservation(t *testing.T) {
	ctx := project.Context{}
	sc := &config.ServiceConfig{
		MemReservation: 100000,
	}
	_, hostCfg, err := Convert(sc, ctx)
	assert.Nil(t, err)

	assert.Equal(t, int64(100000), hostCfg.MemoryReservation)
}

func TestOomScoreAdj(t *testing.T) {
	ctx := project.Context{}
	sc := &config.ServiceConfig{
		OomScoreAdj: 500,
	}
	_, hostCfg, err := Convert(sc, ctx)
	assert.Nil(t, err)

	assert.Equal(t, 500, hostCfg.OomScoreAdj)
}

func TestIsolation(t *testing.T) {
	ctx := project.Context{}
	sc := &config.ServiceConfig{
		Isolation: "default",
	}
	_, hostCfg, err := Convert(sc, ctx)
	assert.Nil(t, err)

	assert.Equal(t, container.Isolation("default"), hostCfg.Isolation)
}

func TestStopSignal(t *testing.T) {
	ctx := project.Context{}
	sc := &config.ServiceConfig{
		StopSignal: "SIGTERM",
	}
	cfg, _, err := Convert(sc, ctx)
	assert.Nil(t, err)

	assert.Equal(t, "SIGTERM", cfg.StopSignal)
}

func TestSysctls(t *testing.T) {
	ctx := project.Context{}
	sc := &config.ServiceConfig{
		Sysctls: yaml.SliceorMap{
			"net.core.somaxconn": "1024",
		},
	}
	_, hostCfg, err := Convert(sc, ctx)
	assert.Nil(t, err)

	assert.True(t, reflect.DeepEqual(map[string]string{
		"net.core.somaxconn": "1024",
	}, hostCfg.Sysctls))
}

func TestTmpfs(t *testing.T) {
	ctx := project.Context{}
	sc := &config.ServiceConfig{
		Tmpfs: yaml.Stringorslice{"/run"},
	}
	_, hostCfg, err := Convert(sc, ctx)
	assert.Nil(t, err)

	assert.True(t, reflect.DeepEqual(map[string]string{
		"/run": "",
	}, hostCfg.Tmpfs))

	sc = &config.ServiceConfig{
		Tmpfs: yaml.Stringorslice{"/run:rw,noexec,nosuid,size=65536k"},
	}
	_, hostCfg, err = Convert(sc, ctx)
	assert.Nil(t, err)

	assert.True(t, reflect.DeepEqual(map[string]string{
		"/run": "rw,noexec,nosuid,size=65536k",
	}, hostCfg.Tmpfs))
}

func TestOomKillDisable(t *testing.T) {
	ctx := project.Context{}
	sc := &config.ServiceConfig{
		OomKillDisable: true,
	}
	_, hostCfg, err := Convert(sc, ctx)
	assert.Nil(t, err)

	assert.Equal(t, true, *hostCfg.OomKillDisable)
}

func TestBlkioDeviceReadBps(t *testing.T) {
	ctx := project.Context{}
	sc := &config.ServiceConfig{
		DeviceReadBps: yaml.MaporColonSlice([]string{
			"/dev/sda:100000",
		}),
	}
	_, hostCfg, err := Convert(sc, ctx)
	assert.Nil(t, err)

	assert.True(t, reflect.DeepEqual([]*blkiodev.ThrottleDevice{
		&blkiodev.ThrottleDevice{
			Path: "/dev/sda",
			Rate: 100000,
		},
	}, hostCfg.BlkioDeviceReadBps))
}

func TestBlkioDeviceReadIOps(t *testing.T) {
	ctx := project.Context{}
	sc := &config.ServiceConfig{
		DeviceReadIOps: yaml.MaporColonSlice([]string{
			"/dev/sda:100000",
		}),
	}
	_, hostCfg, err := Convert(sc, ctx)
	assert.Nil(t, err)

	assert.True(t, reflect.DeepEqual([]*blkiodev.ThrottleDevice{
		&blkiodev.ThrottleDevice{
			Path: "/dev/sda",
			Rate: 100000,
		},
	}, hostCfg.BlkioDeviceReadIOps))
}

func TestBlkioDeviceWriteBps(t *testing.T) {
	ctx := project.Context{}
	sc := &config.ServiceConfig{
		DeviceWriteBps: yaml.MaporColonSlice([]string{
			"/dev/sda:100000",
		}),
	}
	_, hostCfg, err := Convert(sc, ctx)
	assert.Nil(t, err)

	assert.True(t, reflect.DeepEqual([]*blkiodev.ThrottleDevice{
		&blkiodev.ThrottleDevice{
			Path: "/dev/sda",
			Rate: 100000,
		},
	}, hostCfg.BlkioDeviceWriteBps))
}

func TestBlkioDeviceWriteIOps(t *testing.T) {
	ctx := project.Context{}
	sc := &config.ServiceConfig{
		DeviceWriteIOps: yaml.MaporColonSlice([]string{
			"/dev/sda:100000",
		}),
	}
	_, hostCfg, err := Convert(sc, ctx)
	assert.Nil(t, err)

	assert.True(t, reflect.DeepEqual([]*blkiodev.ThrottleDevice{
		&blkiodev.ThrottleDevice{
			Path: "/dev/sda",
			Rate: 100000,
		},
	}, hostCfg.BlkioDeviceWriteIOps))
}
