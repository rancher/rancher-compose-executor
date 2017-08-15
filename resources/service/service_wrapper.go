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
	service, err = s.project.Client.Service.Create(service)
	if err != nil {
		return err
	}
	return wait(s.project.Client, service)
}

func (s *ServiceWrapper) Image() string {
	return s.project.Config.Services[s.name].Image
}

func (s *ServiceWrapper) Labels() map[string]interface{} {
	return utils.ToMapInterface(s.project.Config.Services[s.name].Labels)
}

func (s *ServiceWrapper) upgrade(service *client.Service, options options.Options) error {
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

	service, err = s.project.Client.Service.Update(service, updates)
	if err != nil {
		return err
	}

	return wait(s.project.Client, service)
}

func (s *ServiceWrapper) rollback(service *client.Service) error {
	service, err := s.project.Client.Service.ActionRollback(service, nil)
	if err != nil {
		return err
	}

	return wait(s.project.Client, service)
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
		return s.rollback(service)
	}

	if service.State == "upgraded" {
		service, err = s.project.Client.Service.ActionFinishupgrade(service)
		if err != nil {
			return err
		}
		if err = wait(s.project.Client, service); err != nil {
			return err
		}
	}

	if service.State == "inactive" {
		service, err = s.project.Client.Service.ActionActivate(service)
		if err != nil {
			return err
		}
		if err = wait(s.project.Client, service); err != nil {
			return err
		}
	}

	return s.upgrade(service, options)
}
