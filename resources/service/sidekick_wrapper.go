package service

import (
	"github.com/rancher/rancher-compose-executor/project"
	"github.com/rancher/rancher-compose-executor/project/options"
	"github.com/rancher/rancher-compose-executor/utils"
	"golang.org/x/net/context"
)

type SidekickWrapper struct {
	name    string
	project *project.Project
}

func (s *SidekickWrapper) Exists() (bool, error) {
	for _, primary := range s.getPrimaries() {
		val, err := s.project.ServerResourceLookup.Service(primary)
		if err != nil {
			return false, err
		}
		if val == nil {
			return false, nil
		}
	}
	return true, nil
}

func (s *SidekickWrapper) Image() string {
	return s.project.Config.Services[s.name].Image
}

func (s *SidekickWrapper) getPrimaries() []string {
	return s.project.Config.SidekickInfo.SidekickToPrimaries[s.name]
}

func (s *SidekickWrapper) Labels() map[string]interface{} {
	return utils.ToMapInterface(s.project.Config.Services[s.name].Labels)
}

func (s *SidekickWrapper) Create(ctx context.Context, options options.Options) error {
	for _, primary := range s.getUnSelectedPrimaries(options) {
		primaryService := ServiceWrapper{
			name:    primary,
			project: s.project,
		}
		if err := primaryService.Create(ctx, options); err != nil {
			return err
		}
	}
	return nil
}

func (s *SidekickWrapper) Up(ctx context.Context, options options.Options) error {
	for _, primary := range s.getUnSelectedPrimaries(options) {
		primaryService := ServiceWrapper{
			name:    primary,
			project: s.project,
		}
		if err := primaryService.Up(ctx, options); err != nil {
			return err
		}
	}
	return nil
}

func (s *SidekickWrapper) getUnSelectedPrimaries(options options.Options) []string {
	result := []string{}
	for _, primary := range s.getPrimaries() {
		if !utils.IsSelected(options.Services, primary) {
			result = append(result, primary)
		}
	}
	return result
}
