package service

import (
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/rancher-compose-executor/convert"
	"github.com/rancher/rancher-compose-executor/project"
	"github.com/rancher/rancher-compose-executor/project/options"
	"github.com/rancher/rancher-compose-executor/utils"
	"golang.org/x/net/context"
)

type ContainerWrapper struct {
	name    string
	project *project.Project
}

func (s *ContainerWrapper) Exists() (bool, error) {
	val, err := s.project.ServerResourceLookup.Container(s.name)
	return val != nil, err
}

func (s *ContainerWrapper) Create(ctx context.Context, options options.Options) error {
	container, err := convert.Container(s.project, s.name)
	if err != nil {
		return err
	}

	logrus.Debugf("Creating service %s", s.name)
	container, err = s.project.Client.Container.Create(container)
	if err != nil {
		return err
	}
	return waitContainer(s.project.Client, container)
}

func (s *ContainerWrapper) Image() string {
	return s.project.Config.Services[s.name].Image
}

func (s *ContainerWrapper) Labels() map[string]interface{} {
	return utils.ToMapInterface(s.project.Config.Services[s.name].Labels)
}

func (s *ContainerWrapper) upgrade(container *client.Container, options options.Options) error {
	if options.NoRecreate {
		return nil
	}

	updates, err := convert.ContainerConfig(s.project, s.name)
	if err != nil {
		return err
	}

	rev, err := s.project.Client.Container.ActionUpgrade(container, &client.ContainerUpgrade{
		Config: *updates,
	})
	if err != nil || rev == nil {
		return err
	}

	for i := 0; i < 3; i++ {
		err := waitContainer(s.project.Client, container)
		if err != nil {
			return err
		}

		if container.Desired {
			time.Sleep(time.Second)
		} else {
			break
		}
	}

	containers, err := s.project.Client.Container.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"revisionId": rev.Id,
		},
	})
	if err != nil {
		return err
	}

	if len(containers.Data) > 0 {
		return waitContainer(s.project.Client, &containers.Data[0])
	}

	return nil
}

func (s *ContainerWrapper) Up(ctx context.Context, options options.Options) error {
	container, err := s.project.ServerResourceLookup.Container(s.name)
	if err != nil {
		return err
	}
	if container == nil {
		return fmt.Errorf("Failed to find container %s", s.name)
	}

	if options.Rollback {
		return nil
	}

	return s.upgrade(container, options)
}
