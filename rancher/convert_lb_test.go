package rancher

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/rancher/rancher-compose-executor/config"
)

func TestGenerateHAProxyConf(t *testing.T) {
	conf := generateHAProxyConf("daemon\nmaxconn 256", "mode http")
	expectedConf := `global
    daemon
    maxconn 256
defaults
    mode http`
	if conf != expectedConf {
		t.Fail()
	}

	conf = generateHAProxyConf("daemon\n", "")
	expectedConf = "global\n    daemon\n    \n"
	if conf != expectedConf {
		t.Fail()
	}

	conf = generateHAProxyConf("", "mode http")
	expectedConf = "defaults\n    mode http"
	if conf != expectedConf {
		t.Fail()
	}
}

func testRewritePorts(t *testing.T, in, out string) {
	updatedPorts, err := rewritePorts([]string{in})
	if err != nil {
		t.Fatal(err)
	}

	if len(updatedPorts) != 1 {
		t.Fail()
	}

	if updatedPorts[0] != out {
		t.Fail()
	}
}

func TestRewritePorts(t *testing.T) {
	testRewritePorts(t, "80", "80")
	testRewritePorts(t, "80/tcp", "80/tcp")
	testRewritePorts(t, "80:80", "80")
	testRewritePorts(t, "80:80/tcp", "80/tcp")
}

func testConvertLb(t *testing.T, ports, links, externalLinks []string, selector string, expectedPortRules []config.PortRule) {
	portRules, err := convertLb(ports, links, externalLinks, selector)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(portRules, expectedPortRules) {
		fmt.Println(portRules, expectedPortRules)
		t.Fail()
	}
}

func TestConvertLb(t *testing.T) {
	testConvertLb(t, []string{
		"8080:80",
	}, []string{"web1", "web2"}, []string{"external/web3"}, "", []config.PortRule{
		{
			SourcePort: 8080,
			TargetPort: 80,
			Service:    "web1",
			Protocol:   "http",
		},
		{
			SourcePort: 8080,
			TargetPort: 80,
			Service:    "web2",
			Protocol:   "http",
		},
		{
			SourcePort: 8080,
			TargetPort: 80,
			Service:    "external/web3",
			Protocol:   "http",
		},
	})

	testConvertLb(t, []string{
		"80",
	}, []string{"web1", "web2"}, []string{}, "", []config.PortRule{
		{
			SourcePort: 80,
			TargetPort: 80,
			Service:    "web1",
			Protocol:   "http",
		},
		{
			SourcePort: 80,
			TargetPort: 80,
			Service:    "web2",
			Protocol:   "http",
		},
	})

	testConvertLb(t, []string{
		"80/tcp",
	}, []string{"web1", "web2"}, []string{}, "", []config.PortRule{
		{
			SourcePort: 80,
			TargetPort: 80,
			Service:    "web1",
			Protocol:   "tcp",
		},
		{
			SourcePort: 80,
			TargetPort: 80,
			Service:    "web2",
			Protocol:   "tcp",
		},
	})

	testConvertLb(t, []string{
		"80/tcp",
	}, nil, nil, "foo=bar", []config.PortRule{
		{
			SourcePort: 80,
			TargetPort: 80,
			Selector:   "foo=bar",
			Protocol:   "tcp",
		},
	})
}

func testConvertLabel(t *testing.T, label string, expectedPortRules []config.PortRule) {
	portRules, err := convertLbLabel(label)
	if err != nil {
		t.Fail()
	}
	if !reflect.DeepEqual(portRules, expectedPortRules) {
		fmt.Println(portRules, expectedPortRules)
		t.Fail()
	}
}

func TestConvertLabel(t *testing.T) {
	testConvertLabel(t, "example2.com:80/path=81", []config.PortRule{
		{
			Hostname:   "example2.com",
			SourcePort: 80,
			Path:       "/path",
			TargetPort: 81,
		},
	})
	testConvertLabel(t, "example2.com:80/path/a", []config.PortRule{
		{
			Hostname:   "example2.com",
			SourcePort: 80,
			Path:       "/path/a",
		},
	})
	testConvertLabel(t, "example2.com:80=81", []config.PortRule{
		{
			Hostname:   "example2.com",
			SourcePort: 80,
			TargetPort: 81,
		},
	})
	testConvertLabel(t, "example2.com:80", []config.PortRule{
		{
			Hostname:   "example2.com",
			SourcePort: 80,
		},
	})
	testConvertLabel(t, "example2.com/path/b/c=81", []config.PortRule{
		{
			Hostname:   "example2.com",
			Path:       "/path/b/c",
			TargetPort: 81,
		},
	})
	testConvertLabel(t, "example2.com/path", []config.PortRule{
		{
			Hostname: "example2.com",
			Path:     "/path",
		},
	})
	testConvertLabel(t, "example2.com=81", []config.PortRule{
		{
			Hostname:   "example2.com",
			TargetPort: 81,
		},
	})
	testConvertLabel(t, "example2.com", []config.PortRule{
		{
			Hostname: "example2.com",
		},
	})

	testConvertLabel(t, "80/path=81", []config.PortRule{
		{
			SourcePort: 80,
			Path:       "/path",
			TargetPort: 81,
		},
	})
	testConvertLabel(t, "80/path", []config.PortRule{
		{
			SourcePort: 80,
			Path:       "/path",
		},
	})
	testConvertLabel(t, "80=81", []config.PortRule{
		{
			SourcePort: 80,
			TargetPort: 81,
		},
	})
	testConvertLabel(t, "/path=81", []config.PortRule{
		{
			Path:       "/path",
			TargetPort: 81,
		},
	})
	testConvertLabel(t, "www.abc.com", []config.PortRule{
		{
			Hostname: "www.abc.com",
		},
	})
	testConvertLabel(t, "www.abc2.com", []config.PortRule{
		{
			Hostname: "www.abc2.com",
		},
	})
	testConvertLabel(t, "/path", []config.PortRule{
		{
			Path: "/path",
		},
	})
	testConvertLabel(t, "www.abc2.com/service.html", []config.PortRule{
		{
			Hostname: "www.abc2.com",
			Path:     "/service.html",
		},
	})
	testConvertLabel(t, "81", []config.PortRule{
		{
			TargetPort: 81,
		},
	})

	testConvertLabel(t, "81,82", []config.PortRule{
		{
			TargetPort: 81,
		},
		{
			TargetPort: 82,
		},
	})
	testConvertLabel(t, "example2.com:80/path=81,example2.com:82/path2=83", []config.PortRule{
		{
			Hostname:   "example2.com",
			SourcePort: 80,
			Path:       "/path",
			TargetPort: 81,
		},
		{
			Hostname:   "example2.com",
			SourcePort: 82,
			Path:       "/path2",
			TargetPort: 83,
		},
	})
}

func testMergePortRules(t *testing.T, baseRules, overrideRules, expectedPortRules []config.PortRule) {
	portRules := mergePortRules(baseRules, overrideRules)
	if !reflect.DeepEqual(portRules, expectedPortRules) {
		fmt.Println(portRules, expectedPortRules)
		t.Fail()
	}
}

func TestMergePortRules(t *testing.T) {
	testMergePortRules(t, []config.PortRule{
		{
			Service:    "web",
			SourcePort: 80,
			TargetPort: 80,
		},
	}, []config.PortRule{
		{
			Service:    "web",
			SourcePort: 80,
			TargetPort: 81,
		},
	}, []config.PortRule{
		{
			Service:    "web",
			SourcePort: 80,
			TargetPort: 81,
		},
	})

	testMergePortRules(t, []config.PortRule{
		{
			Service:    "web",
			SourcePort: 80,
			TargetPort: 80,
		},
	}, []config.PortRule{
		{
			Service:    "web",
			Path:       "/path",
			SourcePort: 80,
		},
	}, []config.PortRule{
		{
			Service:    "web",
			Path:       "/path",
			SourcePort: 80,
			TargetPort: 80,
		},
	})

	testMergePortRules(t, []config.PortRule{
		{
			Service:    "web",
			SourcePort: 80,
			TargetPort: 80,
		},
		{
			Service:    "web",
			SourcePort: 81,
			TargetPort: 81,
		},
	}, []config.PortRule{
		{
			Service: "web",
			Path:    "/path",
		},
	}, []config.PortRule{
		{
			Service:    "web",
			Path:       "/path",
			SourcePort: 80,
			TargetPort: 80,
		},
		{
			Service:    "web",
			Path:       "/path",
			SourcePort: 81,
			TargetPort: 81,
		},
	})

	testMergePortRules(t, []config.PortRule{
		{
			Service:    "web",
			SourcePort: 80,
			TargetPort: 80,
		},
		{
			Service:    "web",
			SourcePort: 81,
			TargetPort: 81,
		},
	}, []config.PortRule{
		{
			Service:    "web",
			TargetPort: 90,
			Hostname:   "www.example2.com",
			Path:       "/path",
		},
	}, []config.PortRule{
		{
			Service:    "web",
			Hostname:   "www.example2.com",
			Path:       "/path",
			SourcePort: 80,
			TargetPort: 90,
		},
		{
			Service:    "web",
			Hostname:   "www.example2.com",
			Path:       "/path",
			SourcePort: 81,
			TargetPort: 90,
		},
	})

	testMergePortRules(t, []config.PortRule{
		{
			Service:    "web",
			SourcePort: 80,
			TargetPort: 80,
		},
	}, []config.PortRule{
		{
			Service:  "web",
			Hostname: "www.example1.com",
			Path:     "/path1",
		},
		{
			Service:  "web",
			Hostname: "www.example2.com",
			Path:     "/path2",
		},
	}, []config.PortRule{
		{
			Service:    "web",
			Hostname:   "www.example1.com",
			Path:       "/path1",
			SourcePort: 80,
			TargetPort: 80,
		},
		{
			Service:    "web",
			Hostname:   "www.example2.com",
			Path:       "/path2",
			SourcePort: 80,
			TargetPort: 80,
		},
	})

	testMergePortRules(t, []config.PortRule{
		{
			Service:    "web",
			SourcePort: 80,
			TargetPort: 80,
		},
		{
			Service:    "web2",
			SourcePort: 80,
			TargetPort: 80,
		},
		{
			Service:    "web3",
			SourcePort: 80,
			TargetPort: 80,
		},
	}, []config.PortRule{
		{
			Service:  "web",
			Hostname: "www.example1.com",
			Path:     "/path1",
		},
		{
			Service:  "web",
			Hostname: "www.example2.com",
			Path:     "/path2",
		},
		{
			Service:    "web3",
			TargetPort: 90,
		},
	}, []config.PortRule{
		{
			Service:    "web",
			Hostname:   "www.example1.com",
			Path:       "/path1",
			SourcePort: 80,
			TargetPort: 80,
		},
		{
			Service:    "web",
			Hostname:   "www.example2.com",
			Path:       "/path2",
			SourcePort: 80,
			TargetPort: 80,
		},
		{
			Service:    "web2",
			SourcePort: 80,
			TargetPort: 80,
		},
		{
			Service:    "web3",
			SourcePort: 80,
			TargetPort: 90,
		},
	})
}
