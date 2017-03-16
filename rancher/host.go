package rancher

import (
	"fmt"

	"golang.org/x/net/context"

	log "github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/rancher-compose-executor/config"
	"github.com/rancher/rancher-compose-executor/project"
)

type RancherHostsFactory struct {
	Context *Context
}

func (f *RancherHostsFactory) Create(projectName string, hostConfigs map[string]*config.HostConfig) (project.Hosts, error) {
	hosts := make([]*Host, 0, len(hostConfigs))
	for name, config := range hostConfigs {
		count := config.Count
		if count == 0 {
			count = 1
		}
		hosts = append(hosts, &Host{
			context:     f.Context,
			name:        name,
			projectName: projectName,
			hostConfig:  &config.Host,
			count:       count,
		})
	}
	return &Hosts{
		hosts: hosts,
	}, nil
}

type Hosts struct {
	hosts   []*Host
	Context *Context
}

func (h *Hosts) Initialize(ctx context.Context) error {
	for _, host := range h.hosts {
		if err := host.EnsureItExists(ctx); err != nil {
			return err
		}
	}
	return nil
}

type Host struct {
	context     *Context
	name        string
	projectName string
	hostConfig  *client.Host
	count       int
}

func (h *Host) EnsureItExists(ctx context.Context) error {
	existingHosts, err := h.context.Client.Host.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"stackId": h.context.Stack.Id,
		},
	})
	if err != nil {
		return err
	}

	existingNames := map[string]bool{}
	for _, existingHost := range existingHosts.Data {
		existingNames[existingHost.Name] = true
	}

	var hostsToCreate []client.Host

	if h.count == 0 {
		return nil
	} else if h.count == 1 {
		name := fmt.Sprintf("%s-%s", h.context.Stack.Name, h.name)
		if _, ok := existingNames[name]; !ok {
			host := *h.hostConfig
			host.Name = name
			host.Hostname = name
			host.StackId = h.context.Stack.Id
			hostsToCreate = append(hostsToCreate, host)
		}
	} else {
		for i := 1; i < h.count+1; i++ {
			name := fmt.Sprintf("%s-%s-%d", h.context.Stack.Name, h.name, i)
			if _, ok := existingNames[name]; !ok {
				host := *h.hostConfig
				host.Name = name
				host.Hostname = name
				host.StackId = h.context.Stack.Id
				hostsToCreate = append(hostsToCreate, host)
			}
		}
	}

	for _, host := range hostsToCreate {
		log.Infof("Creating host %s", host.Name)
		if _, err := h.context.Client.Host.Create(&host); err != nil {
			return err
		}
	}

	return nil
}
