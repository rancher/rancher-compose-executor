package service

import (
	"time"

	"context"

	"github.com/pkg/errors"
	"github.com/rancher/go-rancher/v3"
)

func wait(ctx context.Context, client *client.RancherClient, service *client.Service) error {
	return WaitFor(ctx, client, &service.Resource, service, func() string {
		return service.Transitioning
	})
}

func waitContainer(ctx context.Context, client *client.RancherClient, instance *client.Container) error {
	return WaitFor(ctx, client, &instance.Resource, instance, func() string {
		return instance.Transitioning
	})
}

func WaitFor(ctx context.Context, client *client.RancherClient, resource *client.Resource, output interface{}, transitioning func() string) error {
	ticker := time.NewTicker(time.Millisecond * 150)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return errors.Errorf("Timeout. Context canceled for resource %v", resource.Id)
		case <-ticker.C:
			if transitioning() != "yes" {
				return nil
			}
			err := client.Reload(resource, output)
			if err != nil {
				return err
			}
		}
	}
}
