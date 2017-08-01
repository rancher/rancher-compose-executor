package resources

import (
	"golang.org/x/net/context"

	"github.com/rancher/rancher-compose-executor/project"
	"github.com/rancher/rancher-compose-executor/project/options"
)

func DependenciesCreate(p *project.Project) (project.ResourceSet, error) {
	dependencies := make([]*Dependency, 0, len(p.Config.Dependencies))
	for name, config := range p.Config.Dependencies {
		dependencies = append(dependencies, &Dependency{
			project:  p,
			name:     name,
			template: config.Template,
			version:  config.Version,
		})
	}
	return &Dependencies{
		dependencies: dependencies,
	}, nil
}

type Dependencies struct {
	dependencies []*Dependency
}

func (h *Dependencies) Initialize(ctx context.Context, _ options.Options) error {
	for _, dependency := range h.dependencies {
		if err := dependency.EnsureItExists(ctx); err != nil {
			return err
		}
	}
	return nil
}

type Dependency struct {
	project  *project.Project
	name     string
	template string
	version  string
}

func (d *Dependency) EnsureItExists(ctx context.Context) error {
	return nil
}
