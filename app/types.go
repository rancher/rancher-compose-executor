package app

import (
	"github.com/rancher/rancher-compose-executor/project"
	"github.com/urfave/cli"
)

type ProjectFactory interface {
	Create(c *cli.Context, dryRun bool) (*project.Project, error)
}
