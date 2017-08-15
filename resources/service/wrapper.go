package service

import (
	"github.com/rancher/rancher-compose-executor/project/options"
	"golang.org/x/net/context"
)

type Wrapper interface {
	Exists() (bool, error)
	Create(ctx context.Context, options options.Options) error
	Up(ctx context.Context, options options.Options) error
	Image() string
	Labels() map[string]interface{}
}
