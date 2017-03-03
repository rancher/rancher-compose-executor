package rancher

import (
	"errors"
	"io"
	"strings"
	"sync"

	"golang.org/x/net/context"

	"github.com/Sirupsen/logrus"
	"github.com/docker/libcompose/labels"
	"github.com/gorilla/websocket"
	"github.com/rancher/go-rancher/hostaccess"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/rancher-compose-executor/docker/service"
	"github.com/rancher/rancher-compose-executor/project"
	rUtils "github.com/rancher/rancher-compose-executor/utils"
)

type Link struct {
	ServiceName, Alias string
}

func (r *RancherService) Metadata() map[string]interface{} {
	return rUtils.NestedMapsToMapInterface(r.serviceConfig.Metadata)
}

// TODO: is this still needed?
func (r *RancherService) HealthCheck(service string) *client.InstanceHealthCheck {
	return r.serviceConfig.HealthCheck
}

func (r *RancherService) setupLinks(service *client.Service, update bool) error {
	// Don't modify links for selector based linking, don't want to conflict
	// Don't modify links for load balancers, they're created by cattle
	if service.SelectorLink != "" || FindServiceType(r) == ExternalServiceType || FindServiceType(r) == LbServiceType {
		return nil
	}

	existingLinks, err := r.context.Client.ServiceConsumeMap.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"serviceId": service.Id,
		},
	})
	if err != nil {
		return err
	}

	if len(existingLinks.Data) > 0 && !update {
		return nil
	}

	links, err := r.getServiceLinks()
	_, err = r.context.Client.Service.ActionSetservicelinks(service, &client.SetServiceLinksInput{
		ServiceLinks: links,
	})
	return err
}

func (r *RancherService) SelectorContainer() string {
	return r.serviceConfig.Labels["io.rancher.service.selector.container"]
}

func (r *RancherService) SelectorLink() string {
	return r.serviceConfig.Labels["io.rancher.service.selector.link"]
}

func (r *RancherService) getServiceLinks() ([]client.ServiceLink, error) {
	links, err := r.getLinks()
	if err != nil {
		return nil, err
	}

	result := []client.ServiceLink{}
	for link, id := range links {
		result = append(result, client.ServiceLink{
			Name:      link.Alias,
			ServiceId: id,
		})
	}

	return result, nil
}

func (r *RancherService) getLinks() (map[Link]string, error) {
	result := map[Link]string{}

	for _, link := range append(r.serviceConfig.Links, r.serviceConfig.ExternalLinks...) {
		parts := strings.SplitN(link, ":", 2)
		name := parts[0]
		alias := ""
		if len(parts) == 2 {
			alias = parts[1]
		}

		name = strings.TrimSpace(name)
		alias = strings.TrimSpace(alias)

		linkedService, err := r.FindExisting(name)
		if err != nil {
			return nil, err
		}

		if linkedService == nil {
			if _, ok := r.context.Project.ServiceConfigs.Get(name); !ok {
				logrus.Warnf("Failed to find service %s to link to", name)
			}
		} else {
			result[Link{
				ServiceName: name,
				Alias:       alias,
			}] = linkedService.Id
		}
	}

	return result, nil
}

func (r *RancherService) containers() ([]client.Container, error) {
	service, err := r.FindExisting(r.name)
	if err != nil {
		return nil, err
	}

	var instances client.ContainerCollection

	err = r.context.Client.GetLink(service.Resource, "instances", &instances)
	if err != nil {
		return nil, err
	}

	return instances.Data, nil
}

func (r *RancherService) Log(ctx context.Context, follow bool) error {
	service, err := r.FindExisting(r.name)
	if err != nil || service == nil {
		return err
	}

	if service.Type != "service" {
		return nil
	}

	containers, err := r.containers()
	if err != nil {
		logrus.Errorf("Failed to list containers to log: %v", err)
		return err
	}

	for _, container := range containers {
		websocketClient := (*hostaccess.RancherWebsocketClient)(r.context.Client)
		conn, err := websocketClient.GetHostAccess(container.Resource, "logs", nil)
		if err != nil {
			logrus.Errorf("Failed to get logs for %s: %v", container.Name, err)
			continue
		}

		go r.pipeLogs(&container, conn)
	}

	return nil
}

func (r *RancherService) pipeLogs(container *client.Container, conn *websocket.Conn) {
	defer conn.Close()

	log_name := strings.TrimPrefix(container.Name, r.context.ProjectName+"_")
	logger := r.context.LoggerFactory.CreateContainerLogger(log_name)

	for {
		messageType, bytes, err := conn.ReadMessage()

		if err == io.EOF {
			return
		} else if err != nil {
			logrus.Errorf("Failed to read log: %v", err)
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

func (r *RancherService) DependentServices() []project.ServiceRelationship {
	result := []project.ServiceRelationship{}

	for _, rel := range service.DefaultDependentServices(r.context.Project, r) {
		if rel.Type == project.RelTypeLink {
			rel.Optional = true
			result = append(result, rel)
		}
	}

	// Load balancers should depend on non-external target services
	lbConfig := r.serviceConfig.LbConfig
	if lbConfig != nil {
		for _, portRule := range lbConfig.PortRules {
			if portRule.Service != "" && !strings.Contains(portRule.Service, "/") {
				result = append(result, project.NewServiceRelationship(portRule.Service, project.RelTypeLink))
			}
		}
	}

	return result
}

func (r *RancherService) Client() *client.RancherClient {
	return r.context.Client
}

func (r *RancherService) pullImage(image string, labels map[string]string) error {
	taskOpts := &client.PullTask{
		Mode:   "all",
		Labels: rUtils.ToMapInterface(labels),
		Image:  image,
	}

	if r.context.PullCached {
		taskOpts.Mode = "cached"
	}

	task, err := r.context.Client.PullTask.Create(taskOpts)
	if err != nil {
		return err
	}

	printed := map[string]string{}
	lastMessage := ""
	r.WaitFor(&task.Resource, task, func() string {
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

	logrus.Infof("Finished pulling %s", task.Image)
	return nil
}

func (r *RancherService) Pull(ctx context.Context) (err error) {
	config := r.Config()
	if config.Image == "" || FindServiceType(r) != RancherType {
		return
	}

	toPull := map[string]bool{config.Image: true}
	labels := config.Labels

	if secondaries, ok := r.context.SidekickInfo.primariesToSidekicks[r.name]; ok {
		for _, secondaryName := range secondaries {
			serviceConfig, ok := r.context.Project.ServiceConfigs.Get(secondaryName)
			if !ok {
				continue
			}

			labels = rUtils.MapUnion(labels, serviceConfig.Labels)
			if serviceConfig.Image != "" {
				toPull[serviceConfig.Image] = true
			}
		}
	}

	wg := sync.WaitGroup{}

	for image := range toPull {
		wg.Add(1)
		go func(image string) {
			if pErr := r.pullImage(image, labels); pErr != nil {
				err = pErr
			}
			wg.Done()
		}(image)
	}

	wg.Wait()
	return
}

func appendHash(service *RancherService, existingLabels map[string]interface{}) (map[string]interface{}, error) {
	ret := map[string]interface{}{}
	for k, v := range existingLabels {
		ret[k] = v
	}

	hashValue := "" //, err := hash(service)
	//if err != nil {
	//return nil, err
	//}

	ret[labels.HASH.Str()] = hashValue
	return ret, nil
}

func printStatus(image string, printed map[string]string, current map[string]interface{}) bool {
	good := true
	for host, objStatus := range current {
		status, ok := objStatus.(string)
		if !ok {
			continue
		}

		v := printed[host]
		if status != "Done" {
			good = false
		}

		if v == "" {
			logrus.Infof("Checking for %s on %s...", image, host)
			v = "start"
		} else if printed[host] == "start" && status == "Done" {
			logrus.Infof("Finished %s on %s", image, host)
			v = "done"
		} else if printed[host] == "start" && status != "Pulling" && status != v {
			logrus.Infof("Checking for %s on %s: %s", image, host, status)
			v = status
		}
		printed[host] = v
	}

	return good
}
