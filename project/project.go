package project

import (
	"fmt"

	"golang.org/x/net/context"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/libcompose/logger"
	"github.com/rancher/go-rancher/catalog"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/rancher-compose-executor/config"
	"github.com/rancher/rancher-compose-executor/lookup"
	"github.com/rancher/rancher-compose-executor/parser"
	"github.com/rancher/rancher-compose-executor/project/options"
)

var resourceFactories = []ResourceFactory{}

func SetResourceFactories(factories ...ResourceFactory) {
	resourceFactories = factories
}

type Project struct {
	Name   string
	Config *config.Config

	Templates            map[string][]byte
	Answers              map[string]string
	Version              string
	ResourceLookup       lookup.ResourceLookup
	ServerResourceLookup lookup.ServerResourceLookup
	LoggerFactory        logger.Factory
	Project              *Project
	TemplateVersion      *catalog.TemplateVersion

	Client *client.RancherClient
	Stack  *client.Stack
}

func NewProject(name string, client *client.RancherClient) *Project {
	return &Project{
		Config: config.NewConfig(),
		Name:   name,
		Client: client,
	}
}

func (p *Project) load(file string, bytes []byte) error {
	config, err := parser.Merge(p.Config.Services, p.Answers, p.ResourceLookup, p.TemplateVersion, file, bytes)
	if err != nil {
		log.Errorf("Could not parse config for project %s : %v", p.Name, err)
		return err
	}
	for name, config := range config.Services {
		p.Config.Services[name] = config
	}
	for name, config := range config.Containers {
		p.Config.Containers[name] = config
	}
	for name, config := range config.Dependencies {
		p.Config.Dependencies[name] = config
	}
	for name, config := range config.Volumes {
		p.Config.Volumes[name] = config
	}
	for name, config := range config.Networks {
		p.Config.Networks[name] = config
	}
	for name, config := range config.Secrets {
		p.Config.Secrets[name] = config
	}
	for name, config := range config.Hosts {
		p.Config.Hosts[name] = config
	}

	return nil
}

func (p *Project) create(ctx context.Context, options options.Options, start bool) error {
	if options.NoRecreate && options.ForceRecreate {
		return fmt.Errorf("no-recreate and force-recreate cannot be combined")
	}

	var resources []ResourceSet
	for _, factory := range resourceFactories {
		resourceSet, err := factory(p)
		if err != nil {
			return err
		}
		resources = append(resources, resourceSet)
	}

	for _, resource := range resources {
		if err := resource.Initialize(ctx, options); err != nil {
			return err
		}
	}

	if start {
		for _, resource := range resources {
			if starter, ok := resource.(Starter); ok {
				if err := starter.Start(ctx, options); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
