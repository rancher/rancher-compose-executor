package rancher

import (
	"fmt"

	"golang.org/x/net/context"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/libcompose/utils"
	"github.com/rancher/go-rancher/hostaccess"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/rancher-compose-executor/config"
	"github.com/rancher/rancher-compose-executor/digest"
	"github.com/rancher/rancher-compose-executor/docker/service"
	"github.com/rancher/rancher-compose-executor/project"
	"github.com/rancher/rancher-compose-executor/project/options"
)

type RancherContainer struct {
	name          string
	serviceConfig *config.ServiceConfig
	context       *Context
}

func (r *RancherContainer) Name() string {
	return r.name
}

func (r *RancherContainer) Config() *config.ServiceConfig {
	return r.serviceConfig
}

func (r *RancherContainer) Context() *Context {
	return r.context
}

func NewContainer(name string, config *config.ServiceConfig, context *Context) *RancherContainer {
	return &RancherContainer{
		name:          name,
		serviceConfig: config,
		context:       context,
	}
}

func (r *RancherContainer) config() (*client.Container, error) {
	service := NewService(r.name, r.serviceConfig, r.context)
	launchConfig, err := createLaunchConfig(service, r.Name(), r.Config())
	if err != nil {
		return nil, err
	}
	launchConfig.HealthCheck = r.serviceConfig.HealthCheck

	var container client.Container
	if err := utils.Convert(launchConfig, &container); err != nil {
		return nil, err
	}

	container.Name = r.Name()
	container.StackId = r.context.Stack.Id

	hash, err := digest.CreateServiceHash(nil, &launchConfig, nil)
	if err != nil {
		return nil, err
	}

	if container.Labels == nil {
		container.Labels = make(map[string]interface{})
	}
	container.Labels[digest.ServiceHashKey] = hash.LaunchConfig

	links, err := r.getLinks()
	if err != nil {
		return nil, err
	}
	if len(links) > 0 {
		container.InstanceLinks = make(map[string]interface{})
		for alias, containerId := range links {
			container.InstanceLinks[alias] = containerId
		}
	}

	return &container, nil
}

func (r *RancherContainer) Create(ctx context.Context, options options.Create) error {
	return r.up(false)
}

func (r *RancherContainer) Up(ctx context.Context, options options.Up) error {
	return r.up(true)
}

func (r *RancherContainer) up(start bool) error {
	existing, err := r.FindExisting(r.name)
	if err != nil {
		return nil
	}

	container, err := r.config()
	if err != nil {
		return err
	}

	if existing != nil {
		existingHash, ok := existing.Labels[digest.ServiceHashKey]
		if ok && existingHash != container.Labels[digest.ServiceHashKey] {
			log.Warnf("Container %s is out of sync with local configuration file", r.Name())
		}
		return nil
	}

	container, err = r.Client().Container.Create(container)
	if err != nil {
		return err
	}

	return err
}

func (r *RancherContainer) Build(ctx context.Context, buildOptions options.Build) error {
	return nil
}

func (r *RancherContainer) Log(ctx context.Context, follow bool) error {
	existing, err := r.FindExisting(r.name)
	if err != nil {
		return nil
	}

	websocketClient := (*hostaccess.RancherWebsocketClient)(r.context.Client)
	conn, err := websocketClient.GetHostAccess(existing.Resource, "logs", nil)
	if err != nil {
		return fmt.Errorf("Failed to get logs for %s: %v", existing.Name, err)
	}

	go r.pipeLogs(existing, conn)

	return nil
}

func (r *RancherContainer) DependentServices() []project.ServiceRelationship {
	return service.DefaultDependentServices(r.context.Project.ContainerConfigs, r)
}

func (r *RancherContainer) Client() *client.RancherClient {
	return r.context.Client
}

func (r *RancherContainer) Pull(ctx context.Context) error {
	return r.pullImage(r.serviceConfig.Image, r.serviceConfig.Labels)
}
