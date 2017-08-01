package project

import (
	"github.com/rancher/rancher-compose-executor/project/options"
	"golang.org/x/net/context"
)

func (p *Project) Create(ctx context.Context, options options.Options) error {
	return p.create(ctx, options, false)
}

func (p *Project) Up(ctx context.Context, options options.Options) error {
	return p.create(ctx, options, true)
}
