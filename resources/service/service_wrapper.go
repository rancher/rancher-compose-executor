package service

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/rancher-compose-executor/convert"
	"github.com/rancher/rancher-compose-executor/project"
	"github.com/rancher/rancher-compose-executor/project/options"
	"github.com/rancher/rancher-compose-executor/utils"
	"golang.org/x/net/context"
)

const (
	rollback      = "rollback"
	finishupgrade = "finishupgrade"
	activate      = "activate"
)

type ServiceWrapper struct {
	name    string
	project *project.Project
}

func (s *ServiceWrapper) Exists() (bool, error) {
	val, err := s.project.ServerResourceLookup.Service(s.name)
	return val != nil, err
}

func (s *ServiceWrapper) Create(ctx context.Context, options options.Options) error {
	service, err := convert.Service(s.project, s.name)
	if err != nil {
		return err
	}

	logrus.Debugf("Creating service %s", s.name)
	service.CreateOnly = true
	service.CompleteUpdate = true
	if service.LaunchConfig != nil {
		service.LaunchConfig.CompleteUpdate = true
	}
	for i := range service.SecondaryLaunchConfigs {
		service.SecondaryLaunchConfigs[i].CompleteUpdate = true
	}
	service, err = s.project.Client.Service.Create(service)
	if err != nil {
		return err
	}
	return wait(ctx, s.project.Client, service)
}

func (s *ServiceWrapper) Image() string {
	return s.project.Config.Services[s.name].Image
}

func (s *ServiceWrapper) Labels() map[string]interface{} {
	return utils.ToMapInterface(s.project.Config.Services[s.name].Labels)
}

func (s *ServiceWrapper) upgrade(ctx context.Context, service *client.Service, options options.Options) error {
	if options.NoRecreate {
		return nil
	}

	updates, err := convert.Service(s.project, s.name)
	if err != nil {
		return err
	}

	if options.ForceRecreate {
		if utils.IsSelected(options.Services, s.name) && updates.LaunchConfig != nil {
			updates.LaunchConfig.ForceUpgrade = true
		}
		for i, lc := range updates.SecondaryLaunchConfigs {
			if utils.IsSelected(options.Services, lc.Name) {
				updates.SecondaryLaunchConfigs[i].ForceUpgrade = true
			}
		}
	}

	if err = utils.RetryOnError(10, updateServiceWrapper(s.project.Client, service, updates)); err != nil {
		return err
	}

	return wait(ctx, s.project.Client, service)
}

func updateServiceWrapper(client *client.RancherClient, service *client.Service, updates *client.Service) func() error {
	return func() error {
		_, err := client.Service.Update(service, updates)
		if err != nil {
			return err
		}
		return nil
	}
}

func (s *ServiceWrapper) rollback(ctx context.Context, service *client.Service) error {
	if err := utils.RetryOnError(10, ActionWrapper(s.project.Client, service, rollback)); err != nil {
		return err
	}

	return wait(ctx, s.project.Client, service)
}

func (s *ServiceWrapper) Up(ctx context.Context, options options.Options) error {
	service, err := s.project.ServerResourceLookup.Service(s.name)
	if err != nil {
		return err
	}
	if service == nil {
		return fmt.Errorf("Failed to find service %s", s.name)
	}

	if options.Rollback {
		return s.rollback(ctx, service)
	}

	if service.State == "upgraded" {
		if err := utils.RetryOnError(10, ActionWrapper(s.project.Client, service, finishupgrade)); err != nil {
			return err
		}
		if err = wait(ctx, s.project.Client, service); err != nil {
			return err
		}
	}

	if service.State == "inactive" {
		if err := utils.RetryOnError(10, ActionWrapper(s.project.Client, service, activate)); err != nil {
			return err
		}
		if err = wait(ctx, s.project.Client, service); err != nil {
			return err
		}
	}

	return s.upgrade(ctx, service, options)
}

func ActionWrapper(c *client.RancherClient, service *client.Service, action string) func() error {
	return func() error {
		switch action {
		case rollback:
			_, err := c.Service.ActionRollback(service, nil)
			return err
		case finishupgrade:
			_, err := c.Service.ActionFinishupgrade(service)
			return err
		case activate:
			_, err := c.Service.ActionActivate(service)
			return err
		}
		return nil
	}
}
