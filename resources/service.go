package resources

import (
	"fmt"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/rancher-compose-executor/config"
	"github.com/rancher/rancher-compose-executor/project"
	"github.com/rancher/rancher-compose-executor/project/options"
	"github.com/rancher/rancher-compose-executor/resources/service"
	rutils "github.com/rancher/rancher-compose-executor/utils"
	"golang.org/x/net/context"
	"golang.org/x/sync/errgroup"
)

type Service interface {
	Create(ctx context.Context, options options.Options) error
	Up(ctx context.Context, options options.Options) error

	//Config() *config.ServiceConfig
	Name() string
}

type Services struct {
	Project      *project.Project
	Services     map[string]Service
	ServiceOrder []string
}

func ServicesCreate(p *project.Project) (project.ResourceSet, error) {
	var err error

	s := &Services{
		Project:  p,
		Services: map[string]Service{},
	}

	for name, config := range s.Project.Config.Containers {
		config, err = injectEnv(p, *config)
		if err != nil {
			return nil, err
		}

		s.Services[name] = service.NewContainer(name, s.Project)
	}

	for name, config := range s.Project.Config.Services {
		config, err = injectEnv(p, *config)
		if err != nil {
			return nil, err
		}

		if len(s.Project.Config.SidekickInfo.SidekickToPrimaries[name]) > 0 {
			s.Services[name] = service.NewSidekick(name, s.Project)
		} else {
			s.Services[name] = service.NewService(name, s.Project)
		}
	}

	s.ServiceOrder, err = getServiceOrder(s.Project.Config.Containers, s.Project.Config.Services)
	if err != nil {
		return nil, err
	}

	logrus.Infof("Service order: %v", s.ServiceOrder)

	return project.ResourceSet(s), nil
}

func injectEnv(p *project.Project, config config.ServiceConfig) (*config.ServiceConfig, error) {
	parsedEnv := make([]string, 0, len(config.Environment))

	for _, env := range config.Environment {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) > 1 {
			parsedEnv = append(parsedEnv, env)
			continue
		} else {
			env = parts[0]
		}

		if val, ok := p.Answers[env]; ok {
			parsedEnv = append(parsedEnv, fmt.Sprintf("%s=%s"), env, val)
		}
	}

	config.Environment = parsedEnv
	return &config, nil
}

func (s *Services) Initialize(ctx context.Context, options options.Options) error {
	/*for name, service := range s.Services {
		if rutils.IsSelected(options.Services, name) {
			if err := service.Create(ctx, options); err != nil {
				return err
			}
		}
	}*/
	for _, name := range s.ServiceOrder {
		service := s.Services[name]
		if rutils.IsSelected(options.Services, name) {
			if err := service.Create(ctx, options); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Services) Start(ctx context.Context, options options.Options) error {
	g, ctx := errgroup.WithContext(ctx)
	for name, service := range s.Services {
		if rutils.IsSelected(options.Services, name) {
			g.Go(func() error {
				return service.Up(ctx, options)
			})
		}
	}

	return g.Wait()
}
