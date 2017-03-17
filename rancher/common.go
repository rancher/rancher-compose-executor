package rancher

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/libcompose/logger"
	"github.com/gorilla/websocket"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/rancher-compose-executor/config"
	"github.com/rancher/rancher-compose-executor/utils"
)

func (r *RancherService) Wait(service *client.Service) error {
	return r.WaitFor(&service.Resource, service, func() string {
		return service.Transitioning
	})
}

func (r *RancherService) waitInstance(instance *client.Instance) error {
	return r.WaitFor(&instance.Resource, instance, func() string {
		return instance.Transitioning
	})
}

func (r *RancherService) WaitFor(resource *client.Resource, output interface{}, transitioning func() string) error {
	return WaitFor(r.Client(), resource, output, transitioning)
}

func (r *RancherContainer) Wait(container *client.Container) error {
	return WaitFor(r.Client(), &container.Resource, container, func() string {
		return container.Transitioning
	})
}

func WaitFor(client *client.RancherClient, resource *client.Resource, output interface{}, transitioning func() string) error {
	for {
		if transitioning() != "yes" {
			return nil
		}

		time.Sleep(150 * time.Millisecond)

		err := client.Reload(resource, output)
		if err != nil {
			return err
		}
	}
}

func (r *RancherService) FindExisting(name string) (*client.Service, error) {
	return FindExistingService(r.Client(), r.context.Stack.Id, name)
}

func (r *RancherService) FindExistingContainer(name string) (*client.Container, error) {
	return FindExistingContainer(r.Client(), r.context.Stack.Id, name)
}

func (r *RancherContainer) FindExisting(name string) (*client.Container, error) {
	return FindExistingContainer(r.Client(), r.context.Stack.Id, name)
}

func FindExistingService(c *client.RancherClient, currentStackId, name string) (*client.Service, error) {
	log.Debugf("Finding service %s", name)

	name, stackId, err := resolveNameAndStackId(c, currentStackId, name)
	if err != nil {
		return nil, err
	}

	services, err := c.Service.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"stackId":      stackId,
			"name":         name,
			"removed_null": nil,
		},
	})

	if err != nil {
		return nil, err
	}

	if len(services.Data) == 0 {
		return nil, nil
	}

	log.Debugf("Found service %s", name)
	return &services.Data[0], nil
}

func FindExistingContainer(c *client.RancherClient, currentStackId, name string) (*client.Container, error) {
	name, stackId, err := resolveNameAndStackId(c, currentStackId, name)
	if err != nil {
		return nil, err
	}

	containers, err := c.Container.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"stackId":      stackId,
			"name":         name,
			"removed_null": nil,
		},
	})
	if err != nil {
		return nil, err
	}

	if len(containers.Data) == 0 {
		return nil, nil
	}

	return &containers.Data[0], nil
}

func resolveNameAndStackId(c *client.RancherClient, currentStackId, name string) (string, string, error) {
	parts := strings.SplitN(name, "/", 2)
	if len(parts) == 1 {
		return name, currentStackId, nil
	}

	stacks, err := c.Stack.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"name":         parts[0],
			"removed_null": nil,
		},
	})
	if err != nil {
		return "", "", err
	}

	if len(stacks.Data) == 0 {
		return "", "", fmt.Errorf("Failed to find stack: %s", parts[0])
	}

	return parts[1], stacks.Data[0].Id, nil
}

func (r *RancherService) pipeLogs(container *client.Container, conn *websocket.Conn) {
	pipeLogs(container, conn, r.context.LoggerFactory, r.context.ProjectName)
}

func (r *RancherContainer) pipeLogs(container *client.Container, conn *websocket.Conn) {
	pipeLogs(container, conn, r.context.LoggerFactory, r.context.ProjectName)
}

func pipeLogs(container *client.Container, conn *websocket.Conn, loggerFactory logger.Factory, projectName string) {
	defer conn.Close()

	logName := strings.TrimPrefix(container.Name, projectName+"_")
	logger := loggerFactory.CreateContainerLogger(logName)

	for {
		messageType, bytes, err := conn.ReadMessage()

		if err == io.EOF {
			return
		} else if err != nil {
			log.Errorf("Failed to read log: %v", err)
			return
		}

		if messageType != websocket.TextMessage || len(bytes) <= 3 {
			continue
		}

		if bytes[len(bytes)-1] != '\n' {
			bytes = append(bytes, '\n')
		}
		message := bytes[3:]

		if "01" == string(bytes[:2]) {
			logger.Out(message)
		} else {
			logger.Err(message)
		}
	}
}

func (r *RancherService) pullImage(image string, labels map[string]string) error {
	return pullImage(r.Client(), image, labels, r.context.PullCached)
}

func (r *RancherContainer) pullImage(image string, labels map[string]string) error {
	return pullImage(r.Client(), image, labels, r.context.PullCached)
}

func pullImage(c *client.RancherClient, image string, labels map[string]string, pullCached bool) error {
	taskOpts := &client.PullTask{
		Mode:   "all",
		Labels: utils.ToMapInterface(labels),
		Image:  image,
	}

	if pullCached {
		taskOpts.Mode = "cached"
	}

	task, err := c.PullTask.Create(taskOpts)
	if err != nil {
		return err
	}

	printed := map[string]string{}
	lastMessage := ""
	WaitFor(c, &task.Resource, task, func() string {
		if task.TransitioningMessage != "" && task.TransitioningMessage != "In Progress" && task.TransitioningMessage != lastMessage {
			printStatus(task.Image, printed, task.Status)
			lastMessage = task.TransitioningMessage
		}

		return task.Transitioning
	})

	if task.Transitioning == "error" {
		return errors.New(task.TransitioningMessage)
	}

	if !printStatus(task.Image, printed, task.Status) {
		return errors.New("Pull failed on one of the hosts")
	}

	log.Infof("Finished pulling %s", task.Image)
	return nil
}

func (r *RancherService) getLinks() (map[Link]string, error) {
	return getLinks(r.Client(), r.serviceConfig, r.context.Project.ServiceConfigs, func(name string) (string, error) {
		service, err := r.FindExisting(name)
		if err != nil {
			return "", err
		}
		if service == nil {
			return "", nil
		}
		return service.Id, nil
	})
}

func (r *RancherContainer) getLinks() (map[string]string, error) {
	links, err := getLinks(r.Client(), r.serviceConfig, r.context.Project.ContainerConfigs, func(name string) (string, error) {
		container, err := r.FindExisting(name)
		if err != nil {
			return "", err
		}
		if container == nil {
			return "", nil
		}
		return container.Id, nil
	})
	if err != nil {
		return nil, err
	}

	linksById := map[string]string{}
	for link, id := range links {
		linksById[link.Alias] = id
	}

	return linksById, nil
}

func getLinks(c *client.RancherClient, serviceConfig *config.ServiceConfig, serviceConfigs *config.ServiceConfigs, findExistingId func(string) (string, error)) (map[Link]string, error) {
	result := map[Link]string{}

	for _, link := range append(serviceConfig.Links, serviceConfig.ExternalLinks...) {
		parts := strings.SplitN(link, ":", 2)
		name := parts[0]
		alias := ""
		if len(parts) == 1 {
			alias = parts[0]
		} else {
			alias = parts[1]
		}

		name = strings.TrimSpace(name)
		alias = strings.TrimSpace(alias)

		linkedServiceOrContainerId, err := findExistingId(name)
		if err != nil {
			return nil, err
		}

		if linkedServiceOrContainerId == "" {
			if _, ok := serviceConfigs.Get(name); !ok {
				log.Warnf("Failed to find service %s to link to", name)
			}
		} else {
			result[Link{
				ServiceName: name,
				Alias:       alias,
			}] = linkedServiceOrContainerId
		}
	}

	return result, nil
}
