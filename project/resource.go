package project

import (
	"golang.org/x/net/context"

	"github.com/rancher/rancher-compose-executor/project/options"
)

type ResourceSet interface {
	Initialize(ctx context.Context, options options.Options) error
}

type Starter interface {
	Start(ctx context.Context, options options.Options) error
}

// Optionally ResourceSet can implement Starter
type ResourceFactory func(p *Project) (ResourceSet, error)
