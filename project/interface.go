package project

import (
	"golang.org/x/net/context"

	"github.com/rancher/rancher-compose/config"
	"github.com/rancher/rancher-compose/project/events"
	"github.com/rancher/rancher-compose/project/options"
)

// APIProject defines the methods a libcompose project should implement.
type APIProject interface {
	events.Notifier
	events.Emitter

	Build(ctx context.Context, options options.Build, sevice ...string) error
	Create(ctx context.Context, options options.Create, services ...string) error
	Log(ctx context.Context, follow bool, services ...string) error
	Up(ctx context.Context, options options.Up, services ...string) error

	Parse() error
	CreateService(name string) (Service, error)
	AddConfig(name string, config *config.ServiceConfig) error
	Load(bytes []byte) error

	GetServiceConfig(service string) (*config.ServiceConfig, bool)
}

// Filter holds filter element to filter containers
type Filter struct {
	State State
}

// State defines the supported state you can filter on
type State string

// Definitions of filter states
const (
	AnyState = State("")
	Running  = State("running")
	Stopped  = State("stopped")
)

// RuntimeProject defines runtime-specific methods for a libcompose implementation.
type RuntimeProject interface {
	RemoveOrphans(ctx context.Context, projectName string, serviceConfigs *config.ServiceConfigs) error
}
