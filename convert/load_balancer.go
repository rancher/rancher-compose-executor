package convert

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/rancher-compose-executor/config"
	"github.com/rancher/rancher-compose-executor/lookup"
)

type Link struct {
	ServiceName, Alias string
}

func populateLb(resourceLookup lookup.ServerResourceLookup, serviceConfig config.ServiceConfig, launchConfig *client.LaunchConfig, service *client.Service) error {
	legacy := serviceConfig.Image == LegacyLBImage

	if !legacy && serviceConfig.LbConfig == nil {
		return nil
	}

	if serviceConfig.LbConfig == nil {
		// just avoid some nil checks
		serviceConfig.LbConfig = &config.LBConfig{}
	}

	return populateLbFields(legacy, resourceLookup, serviceConfig, launchConfig, service)
}

func populateLbFields(legacy bool, resourceLookup lookup.ServerResourceLookup, config config.ServiceConfig, launchConfig *client.LaunchConfig, service *client.Service) error {
	var err error

	service.LbConfig = &client.LbConfig{
		CertificateIds:       config.Certs,
		Config:               generateHAProxyConf(config),
		StickinessPolicy:     getStickynessPolicy(config),
		DefaultCertificateId: config.DefaultCert,
	}


	portRules := config.LbConfig.PortRules

	if legacy {
		legacyPortRules, haProxyConfig, err := getLegacyPortRules(config)
		if err != nil {
			return err
		}

		service.LbConfig.Config += haProxyConfig
		portRules = append(portRules, legacyPortRules...)
	}

	for _, portRule := range portRules {
		finalPortRule := client.PortRule{
			SourcePort:  int64(portRule.SourcePort),
			Protocol:    portRule.Protocol,
			Path:        portRule.Path,
			Hostname:    portRule.Hostname,
			TargetPort:  int64(portRule.TargetPort),
			Priority:    int64(portRule.Priority),
			BackendName: portRule.BackendName,
			Selector:    portRule.Selector,
		}

		if portRule.Service != "" {
			targetService, err := resourceLookup.Service(portRule.Service)
			if err != nil {
				return err
			}
			if targetService == nil {
				return NewErrDependencyNotFound(fmt.Errorf("Failed to find existing service: %s", portRule.Service))
			}
			finalPortRule.ServiceId = targetService.Id
		}

		if portRule.Container != "" {
			targetContainer, err := resourceLookup.Container(portRule.Container)
			if err != nil {
				return err
			}
			if targetContainer == nil {
				return NewErrDependencyNotFound(fmt.Errorf("Failed to find existing container: %s", portRule.Service))
			}
			finalPortRule.InstanceId = targetContainer.Id
		}

		service.LbConfig.PortRules = append(service.LbConfig.PortRules, finalPortRule)
	}

	// Strip target ports from lb service config
	if legacy {
		launchConfig.Ports, err = rewritePorts(config.Ports)
		if err != nil {
			return err
		}
		// Remove expose from config
		launchConfig.Expose = nil
	}

	return populateCerts(resourceLookup, service, config.DefaultCert, config.Certs)
}

func getLegacyPortRules(config config.ServiceConfig) ([]config.PortRule, string, error) {
	haProxyConfig := ""
	portRules, err := convertLb(config.Ports, config.Links, config.ExternalLinks, "")
	if err != nil {
		return nil, "", err
	}

	exposeRules, err := convertLb(config.Expose, config.Links, config.ExternalLinks, "")
	if err != nil {
		return nil, "", err
	}

	portRules = append(portRules, exposeRules...)

	labelName := "io.rancher.service.selector.link"
	if label, ok := config.Labels[labelName]; ok {
		selectorPortRules, err := convertLb(config.Ports, nil, nil, label)
		if err != nil {
			return nil, "", err
		}
		portRules = append(portRules, selectorPortRules...)

		selectorExposeRules, err := convertLb(config.Expose, nil, nil, label)
		if err != nil {
			return nil, "", err
		}
		portRules = append(portRules, selectorExposeRules...)
	}

	for _, link := range getLinks(config) {
		labelName = "io.rancher.loadbalancer.target." + link.ServiceName
		if label, ok := config.Labels[labelName]; ok {
			newPortRules, err := convertLbLabel(label)
			if err != nil {
				return nil, "", err
			}
			for i := range newPortRules {
				newPortRules[i].Service = link.ServiceName
			}
			portRules = mergePortRules(portRules, newPortRules)
		}
	}

	labelName = "io.rancher.loadbalancer.ssl.ports"
	if label, ok := config.Labels[labelName]; ok {
		split := strings.Split(label, ",")
		for _, portString := range split {
			port, err := strconv.ParseInt(portString, 10, 32)
			if err != nil {
				return nil, "", err
			}
			for i, portRule := range portRules {
				if portRule.SourcePort == int(port) {
					portRules[i].Protocol = "https"
				}
			}
		}
	}

	labelName = "io.rancher.loadbalancer.proxy-protocol.ports"
	if label, ok := config.Labels[labelName]; ok {
		split := strings.Split(label, ",")
		for _, portString := range split {
			haProxyConfig += fmt.Sprintf(`
frontend %s
    accept-proxy`, portString)
		}
	}

	return portRules, haProxyConfig, nil
}

func getStickynessPolicy(config config.ServiceConfig) *client.LoadBalancerCookieStickinessPolicy {
	if config.LegacyLoadBalancerConfig == nil || config.LegacyLoadBalancerConfig.LbCookieStickinessPolicy == nil {
		stickinessPolicy := config.LbConfig.StickinessPolicy
		if stickinessPolicy == nil {
			return nil
		}

		return &client.LoadBalancerCookieStickinessPolicy{
			Name:     stickinessPolicy.Name,
			Cookie:   stickinessPolicy.Cookie,
			Domain:   stickinessPolicy.Domain,
			Indirect: stickinessPolicy.Indirect,
			Nocache:  stickinessPolicy.Nocache,
			Postonly: stickinessPolicy.Postonly,
			Mode:     stickinessPolicy.Mode,
		}
	}

	legacyStickinessPolicy := config.LegacyLoadBalancerConfig.LbCookieStickinessPolicy
	return &client.LoadBalancerCookieStickinessPolicy{
		Cookie:   legacyStickinessPolicy.Cookie,
		Domain:   legacyStickinessPolicy.Domain,
		Indirect: legacyStickinessPolicy.Indirect,
		Mode:     legacyStickinessPolicy.Mode,
		Name:     legacyStickinessPolicy.Name,
		Nocache:  legacyStickinessPolicy.Nocache,
		Postonly: legacyStickinessPolicy.Postonly,
	}
}

func generateHAProxyConf(config config.ServiceConfig) string {
	if config.LegacyLoadBalancerConfig == nil || config.LegacyLoadBalancerConfig.HaproxyConfig == nil {
		return config.LbConfig.Config
	}

	global := config.LegacyLoadBalancerConfig.HaproxyConfig.Global
	defaults := config.LegacyLoadBalancerConfig.HaproxyConfig.Defaults

	conf := ""
	if global != "" {
		conf += "global"
		global = "\n" + global
		finalGlobal := ""
		for _, c := range global {
			if c == '\n' {
				finalGlobal += "\n    "
			} else {
				finalGlobal += string(c)
			}
		}
		conf += finalGlobal
		conf += "\n"
	}
	if defaults != "" {
		conf += "defaults"
		defaults = "\n" + defaults
		finalDefaults := ""
		for _, c := range defaults {
			if c == '\n' {
				finalDefaults += "\n    "
			} else {
				finalDefaults += string(c)
			}
		}
		conf += finalDefaults
	}
	return conf
}

func rewritePorts(ports []string) ([]string, error) {
	updatedPorts := []string{}

	for _, port := range ports {
		protocol := ""
		split := strings.Split(port, "/")
		if len(split) == 2 {
			protocol = split[1]
		}

		var source string
		var err error
		split = strings.Split(port, ":")
		if len(split) == 1 {
			source, _, err = readPort(split[0], 0)
			if err != nil {
				return nil, err
			}
		} else if len(split) == 2 {
			source = split[0]
		}

		if protocol == "" {
			updatedPorts = append(updatedPorts, source)
		} else {
			updatedPorts = append(updatedPorts, fmt.Sprintf("%s/%s", source, protocol))
		}
	}

	return updatedPorts, nil
}

func convertLb(ports, links, externalLinks []string, selector string) ([]config.PortRule, error) {
	portRules := []config.PortRule{}

	for _, port := range ports {
		protocol := "http"
		split := strings.Split(port, "/")
		if len(split) == 2 {
			protocol = split[1]
		}

		var sourcePort int64
		var targetPort int64
		var err error
		split = strings.Split(port, ":")
		if len(split) == 1 {
			singlePort, _, err := readPort(split[0], 0)
			if err != nil {
				return nil, err
			}
			sourcePort, err = strconv.ParseInt(singlePort, 10, 32)
			if err != nil {
				return nil, err
			}
			targetPort, err = strconv.ParseInt(singlePort, 10, 32)
			if err != nil {
				return nil, err
			}
		} else if len(split) == 2 {
			sourcePort, err = strconv.ParseInt(split[0], 10, 32)
			if err != nil {
				return nil, err
			}
			target, _, err := readPort(split[1], 0)
			if err != nil {
				return nil, err
			}
			targetPort, err = strconv.ParseInt(target, 10, 32)
			if err != nil {
				return nil, err
			}
		}
		for _, link := range links {
			split := strings.Split(link, ":")
			portRules = append(portRules, config.PortRule{
				SourcePort: int(sourcePort),
				TargetPort: int(targetPort),
				Service:    split[0],
				Protocol:   protocol,
			})
		}
		for _, externalLink := range externalLinks {
			split := strings.Split(externalLink, ":")
			portRules = append(portRules, config.PortRule{
				SourcePort: int(sourcePort),
				TargetPort: int(targetPort),
				Service:    split[0],
				Protocol:   protocol,
			})
		}
		if selector != "" {
			portRules = append(portRules, config.PortRule{
				SourcePort: int(sourcePort),
				TargetPort: int(targetPort),
				Selector:   selector,
				Protocol:   protocol,
			})
		}
	}

	return portRules, nil
}

func isNum(c uint8) bool {
	return c >= '0' && c <= '9'
}

func readHostname(label string, pos int) (string, int, error) {
	var hostname bytes.Buffer
	if isNum(label[pos]) {
		return hostname.String(), pos, nil
	}
	for ; pos < len(label); pos++ {
		c := label[pos]
		if c == '=' {
			return hostname.String(), pos + 1, nil
		}
		if c == ':' {
			return hostname.String(), pos + 1, nil
		}
		if c == '/' {
			return hostname.String(), pos, nil
		}
		hostname.WriteByte(c)
	}
	return hostname.String(), pos, nil
}

func readPort(label string, pos int) (string, int, error) {
	var port bytes.Buffer
	for ; pos < len(label); pos++ {
		c := label[pos]
		if !isNum(c) {
			return port.String(), pos, nil
		}
		port.WriteByte(c)
	}
	return port.String(), pos, nil
}

func readPath(label string, pos int) (string, int, error) {
	var path bytes.Buffer
	for ; pos < len(label); pos++ {
		c := label[pos]
		if c == '=' {
			return path.String(), pos + 1, nil
		}
		path.WriteByte(c)
	}
	return path.String(), pos, nil
}

func convertLbLabel(label string) ([]config.PortRule, error) {
	var portRules []config.PortRule

	labels := strings.Split(label, ",")
	for _, label := range labels {
		label = strings.Trim(label, " \t\n")

		hostname, pos, err := readHostname(label, 0)
		if err != nil {
			return nil, err
		}

		sourcePort, pos, err := readPort(label, pos)
		if err != nil {
			return nil, err
		}

		path, pos, err := readPath(label, pos)
		if err != nil {
			return nil, err
		}

		targetPort, pos, err := readPort(label, pos)
		if err != nil {
			return nil, err
		}

		var source int64
		if sourcePort == "" {
			source = 0
		} else {
			source, err = strconv.ParseInt(sourcePort, 10, 32)
			if err != nil {
				return nil, err
			}
		}

		var target int64
		if targetPort == "" {
			target = 0
		} else {
			target, err = strconv.ParseInt(targetPort, 10, 32)
			if err != nil {
				return nil, err
			}
		}

		if hostname == "" && path == "" && target == 0 {
			portRules = append(portRules, config.PortRule{
				TargetPort: int(source),
			})
			continue
		}

		if target == 0 && strings.Contains(label, "=") {
			portRules = append(portRules, config.PortRule{
				Hostname:   hostname,
				TargetPort: int(source),
			})
			continue
		}

		portRules = append(portRules, config.PortRule{
			Hostname:   hostname,
			SourcePort: int(source),
			Path:       path,
			TargetPort: int(target),
		})
	}

	return portRules, nil
}

func mergePortRules(baseRules, overrideRules []config.PortRule) []config.PortRule {
	newRules := []config.PortRule{}
	for _, baseRule := range baseRules {
		prevLength := len(newRules)
		for _, overrideRule := range overrideRules {
			if baseRule.Service == overrideRule.Service && (overrideRule.SourcePort == 0 || baseRule.SourcePort == overrideRule.SourcePort) {
				newRule := baseRule
				newRule.Path = overrideRule.Path
				newRule.Hostname = overrideRule.Hostname
				if overrideRule.TargetPort != 0 {
					newRule.TargetPort = overrideRule.TargetPort
				}
				newRules = append(newRules, newRule)
			}
		}
		// If no rules were overwritten, just copy over base rule
		if len(newRules) == prevLength {
			newRules = append(newRules, baseRule)
		}
	}
	return newRules
}

func getLinks(serviceConfig config.ServiceConfig) []Link {
	result := []Link{}

	for _, link := range append(serviceConfig.Links, serviceConfig.ExternalLinks...) {
		parts := strings.SplitN(link, ":", 2)
		name := parts[0]
		alias := ""
		if len(parts) == 2 {
			alias = parts[1]
		}

		result = append(result, Link{
			ServiceName: strings.TrimSpace(name),
			Alias:       strings.TrimSpace(alias),
		})
	}

	return result
}
