package resources

import (
	"errors"

	"github.com/rancher/rancher-compose-executor/config"
	"github.com/rancher/rancher-compose-executor/convert"
)

func getServiceOrder(containers, services map[string]*config.ServiceConfig) ([]string, error) {
	var order []string
	added := map[string]bool{}

	for name := range containers {
		add(name, &order, added)
	}

	for name, config := range services {
		if config.LbConfig == nil || len(config.LbConfig.PortRules) == 0 || config.Image != convert.LegacyLBImage {
			add(name, &order, added)
		}
	}

	for i := 0; i < 100; i++ {
		for name, config := range services {
			if config.LbConfig == nil {
				continue
			}
			targetsAdded := true
			for _, portRule := range config.LbConfig.PortRules {
				if _, ok := added[portRule.Service]; !ok {
					targetsAdded = false
					break
				}
			}
			if !targetsAdded {
				continue
			}
			add(name, &order, added)
		}
	}

	// make sure legacy lb image is added
	for name := range services {
		add(name, &order, added)
	}

	if len(order) != len(containers)+len(services) {
		return nil, errors.New("Failed to determine correct order to create services")
	}

	return order, nil
}

func add(name string, order *[]string, added map[string]bool) {
	if _, ok := added[name]; ok {
		return
	}
	*order = append(*order, name)
	added[name] = true
}
