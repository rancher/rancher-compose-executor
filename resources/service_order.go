package resources

import (
	"errors"
	"strings"

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
		if config.Image == convert.LegacyLBImage {
			continue
		}
		if config.LbConfig != nil && containsServicePortRules(config.LbConfig) {
			continue
		}
		add(name, &order, added)
	}

	for i := 0; i < 100; i++ {
		for name, config := range services {
			if config.LbConfig != nil {
				targetsAdded := true
				for _, portRule := range config.LbConfig.PortRules {
					if _, ok := added[portRule.Service]; !ok {
						targetsAdded = false
						break
					}
				}
				if targetsAdded {
					add(name, &order, added)
				}
			} else if config.Image == convert.LegacyLBImage {
				targetsAdded := true
				for _, link := range config.Links {
					parts := strings.SplitN(link, ":", 2)
					if len(parts) > 1 {
						link = parts[1]
					}
					if _, ok := added[link]; !ok {
						targetsAdded = false
						break
					}
				}
				if targetsAdded {
					add(name, &order, added)
				}
			}
		}
	}

	if len(order) != len(containers)+len(services) {
		return nil, errors.New("Failed to determine correct order to create services")
	}

	return order, nil
}

func containsServicePortRules(lbConfig *config.LBConfig) bool {
	for _, portRule := range lbConfig.PortRules {
		if portRule.Service != "" {
			return true
		}
	}
	return false
}

func add(name string, order *[]string, added map[string]bool) {
	if _, ok := added[name]; ok {
		return
	}
	*order = append(*order, name)
	added[name] = true
}
