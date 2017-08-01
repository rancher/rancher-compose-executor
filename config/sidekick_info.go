package config

import (
	"strings"
)

type SidekickInfo struct {
	PrimariesToSidekicks map[string][]string
	Primaries            map[string]bool
	SidekickToPrimaries  map[string][]string
}

func (c *Config) Complete() {
	result := &SidekickInfo{
		PrimariesToSidekicks: map[string][]string{},
		Primaries:            map[string]bool{},
		SidekickToPrimaries:  map[string][]string{},
	}

	for name, config := range c.Services {
		sidekicks := []string{}

		for key, value := range config.Labels {
			if key != "io.rancher.sidekicks" {
				continue
			}

			for _, part := range strings.Split(strings.TrimSpace(value), ",") {
				part = strings.TrimSpace(part)
				result.Primaries[name] = true

				sidekicks = append(sidekicks, part)

				list, ok := result.SidekickToPrimaries[part]
				if !ok {
					list = []string{}
				}
				result.SidekickToPrimaries[part] = append(list, name)
			}
		}

		result.PrimariesToSidekicks[name] = sidekicks
	}

	c.SidekickInfo = result
}
