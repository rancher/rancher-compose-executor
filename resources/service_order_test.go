package resources

import (
	"testing"

	"github.com/rancher/rancher-compose-executor/config"
)

func TestGetServiceOrder(t *testing.T) {
	testGetServiceOrder(t, map[string]*config.ServiceConfig{
		"s1": &config.ServiceConfig{},
		"s2": &config.ServiceConfig{},
	}, []string{"s1", "s2"})

	testGetServiceOrder(t, map[string]*config.ServiceConfig{
		"s1": &config.ServiceConfig{},
		"s2": &config.ServiceConfig{},
		"lb": lbConfigFactory("s1", "s2"),
	}, []string{"s1", "s2"}, []string{"lb"})

	testGetServiceOrder(t, map[string]*config.ServiceConfig{
		"s1":  &config.ServiceConfig{},
		"s2":  &config.ServiceConfig{},
		"lb":  lbConfigFactory("s1", "s2"),
		"lb2": lbConfigFactory("lb"),
	}, []string{"s1", "s2"}, []string{"lb"}, []string{"lb2"})

	testGetServiceOrder(t, map[string]*config.ServiceConfig{
		"s1":  &config.ServiceConfig{},
		"s2":  &config.ServiceConfig{},
		"lb":  lbConfigFactory("s1", "s2"),
		"lb2": lbConfigFactory("lb"),
		"lb3": lbConfigFactory("lb"),
		"lb4": lbConfigFactory("lb2", "lb3"),
	}, []string{"s1", "s2"}, []string{"lb"}, []string{"lb2", "lb3"}, []string{"lb4"})
}

func TestGetServiceOrderCycleFails(t *testing.T) {
	_, err := getServiceOrder(nil, map[string]*config.ServiceConfig{
		"s1":  &config.ServiceConfig{},
		"s2":  &config.ServiceConfig{},
		"lb":  lbConfigFactory("s1", "s2"),
		"lb2": lbConfigFactory("lb3"),
		"lb3": lbConfigFactory("lb2"),
	})
	if err == nil {
		t.Fail()
	}
}

func lbConfigFactory(targetServices ...string) *config.ServiceConfig {
	var portRules []config.PortRule
	for _, service := range targetServices {
		portRules = append(portRules, config.PortRule{
			Service: service,
		})
	}
	return &config.ServiceConfig{
		RancherConfig: config.RancherConfig{
			LbConfig: &config.LBConfig{
				PortRules: portRules,
			},
		},
	}
}

func testGetServiceOrder(t *testing.T, services map[string]*config.ServiceConfig, expectedOrderSets ...[]string) {
	order, err := getServiceOrder(nil, services)
	if err != nil {
		t.Fatal(err)
	}
	for i, set := range expectedOrderSets {
		for _, name := range set {
			for j := 0; j < i; j++ {
				previousSet := expectedOrderSets[j]
				for _, previousName := range previousSet {
					if positionOf(t, name, order) < positionOf(t, previousName, order) {
						t.Fail()
					}
				}
			}
		}
	}
}

func positionOf(t *testing.T, search string, slice []string) int {
	for i, elem := range slice {
		if elem == search {
			return i
		}
	}
	t.Fail()
	return -1
}
