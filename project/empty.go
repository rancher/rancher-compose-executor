package project

import (
	"golang.org/x/net/context"

	"github.com/rancher/rancher-compose/config"
	"github.com/rancher/rancher-compose/project/events"
	"github.com/rancher/rancher-compose/project/options"
)

// this ensures EmptyService implements Service
// useful since it's easy to forget adding new functions to EmptyService
var _ Service = (*EmptyService)(nil)

// EmptyService is a struct that implements Service but does nothing.
type EmptyService struct {
}

// Create implements Service.Create but does nothing.
func (e *EmptyService) Create(ctx context.Context, options options.Create) error {
	return nil
}

// Build implements Service.Build but does nothing.
func (e *EmptyService) Build(ctx context.Context, buildOptions options.Build) error {
	return nil
}

// Up implements Service.Up but does nothing.
func (e *EmptyService) Up(ctx context.Context, options options.Up) error {
	return nil
}

// Log implements Service.Log but does nothing.
func (e *EmptyService) Log(ctx context.Context, follow bool) error {
	return nil
}

// RemoveImage implements Service.RemoveImage but does nothing.
func (e *EmptyService) RemoveImage(ctx context.Context, imageType options.ImageType) error {
	return nil
}

// Events implements Service.Events but does nothing.
func (e *EmptyService) Events(ctx context.Context, events chan events.ContainerEvent) error {
	return nil
}

// DependentServices implements Service.DependentServices with empty slice.
func (e *EmptyService) DependentServices() []ServiceRelationship {
	return []ServiceRelationship{}
}

// Config implements Service.Config with empty config.
func (e *EmptyService) Config() *config.ServiceConfig {
	return &config.ServiceConfig{}
}

// Name implements Service.Name with empty name.
func (e *EmptyService) Name() string {
	return ""
}

// this ensures EmptyNetworks implements Networks
var _ Networks = (*EmptyNetworks)(nil)

// EmptyNetworks is a struct that implements Networks but does nothing.
type EmptyNetworks struct {
}

// Initialize implements Networks.Initialize but does nothing.
func (e *EmptyNetworks) Initialize(ctx context.Context) error {
	return nil
}

// Remove implements Networks.Remove but does nothing.
func (e *EmptyNetworks) Remove(ctx context.Context) error {
	return nil
}
