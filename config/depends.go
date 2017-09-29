package config

import (
	"github.com/pkg/errors"
)

type Dependencies map[string]Dependency

type Dependency struct {
	Condition string `yaml:"condition,omitempty"`
	Container bool   `yaml:"container,omitempty"`
}

func (d *Dependencies) UnmarshalYAML(unmarshal func(interface{}) error) error {
	strings := []string{}
	deps := map[string]Dependency{}

	if err := unmarshal(&strings); err == nil {
		for _, dep := range strings {
			deps[dep] = Dependency{
				Condition: "healthy",
			}
		}
	} else if err := unmarshal(&deps); err != nil {
		return errors.Wrap(err, "Failed to unmarshall dependencies")
	}

	*d = deps
	return nil
}
