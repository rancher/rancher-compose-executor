package service

import (
	"golang.org/x/net/context"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types/container"
	"github.com/rancher/rancher-compose-executor/project"
	"github.com/rancher/rancher-compose-executor/project/options"
	"github.com/rancher/rancher-compose-executor/utils"
)

type Link struct {
	ServiceName, Alias string
}

type ContainerInspect struct {
	Name       string
	Config     *container.Config
	HostConfig *container.HostConfig
}

type Service struct {
	name    string
	project *project.Project
	wrapper Wrapper
}

func (s *Service) Name() string {
	return s.name
}

func NewContainer(name string, p *project.Project) *Service {
	return &Service{
		name:    name,
		project: p,
		wrapper: &ContainerWrapper{
			name:    name,
			project: p,
		},
	}
}

func NewService(name string, p *project.Project) *Service {
	return &Service{
		name:    name,
		project: p,
		wrapper: &ServiceWrapper{
			name:    name,
			project: p,
		},
	}
}

func NewSidekick(name string, p *project.Project) *Service {
	return &Service{
		name:    name,
		project: p,
		wrapper: &SidekickWrapper{
			name:    name,
			project: p,
		},
	}
}

func (s *Service) Create(ctx context.Context, options options.Options) error {
	exists, err := s.wrapper.Exists()
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	return s.wrapper.Create(ctx, options)
}

func (s *Service) Up(ctx context.Context, options options.Options) error {
	return s.wrapper.Up(ctx, options)
}

func (s *Service) Pull(ctx context.Context, options options.Pull) (err error) {
	image := s.wrapper.Image()
	if image == "" {
		return
	}

	labels := s.wrapper.Labels()

	return pullImage(s.project.Client, image, utils.ToMapString(labels), options.Cached)
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
