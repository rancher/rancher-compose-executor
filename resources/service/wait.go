package service

import (
	"time"

	"github.com/rancher/go-rancher/v3"
)

func wait(client *client.RancherClient, service *client.Service) error {
	return WaitFor(client, &service.Resource, service, func() string {
		return service.Transitioning
	})
}

func waitContainer(client *client.RancherClient, instance *client.Container) error {
	return WaitFor(client, &instance.Resource, instance, func() string {
		return instance.Transitioning
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
