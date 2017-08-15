package resources

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/rancher-compose-executor/config"
	"github.com/rancher/rancher-compose-executor/project"
	"github.com/rancher/rancher-compose-executor/project/options"
)

func VolumesCreate(p *project.Project) (project.ResourceSet, error) {
	volumes := make([]*Volume, 0, len(p.Config.Volumes))
	for name, config := range p.Config.Volumes {
		volume := NewVolume(p, name, config)
		volumes = append(volumes, volume)
	}
	return &Volumes{
		volumes: volumes,
	}, nil
}

type Volumes struct {
	volumes []*Volume
}

func (v *Volumes) Initialize(ctx context.Context, _ options.Options) error {
	for _, volume := range v.volumes {
		if err := volume.EnsureItExists(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (v *Volumes) Remove(ctx context.Context) error {
	for _, volume := range v.volumes {
		if err := volume.Remove(ctx); err != nil {
			return err
		}
	}
	return nil
}

type Volume struct {
	project       *project.Project
	name          string
	driver        string
	driverOptions map[string]string
	external      bool
	perContainer  bool
}

// Inspect looks up a volume template
func (v *Volume) Inspect(ctx context.Context) (*client.VolumeTemplate, error) {
	volumes, err := v.project.Client.VolumeTemplate.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"name":    v.name,
			"stackId": v.project.Stack.Id,
		},
	})
	if err != nil {
		return nil, err
	}

	if len(volumes.Data) > 0 {
		return &volumes.Data[0], nil
	}

	return nil, nil
}

func (v *Volume) Remove(ctx context.Context) error {
	if v.external {
		return nil
	}

	volumeResource, err := v.Inspect(ctx)
	if err != nil {
		return err
	}
	return v.project.Client.VolumeTemplate.Delete(volumeResource)
}

func (v *Volume) EnsureItExists(ctx context.Context) error {
	volumeResource, err := v.Inspect(ctx)
	if err != nil {
		return err
	}

	if volumeResource == nil {
		logrus.Infof("Creating volume template %s", v.name)
		return v.create(ctx)
	} else {
		logrus.Infof("Existing volume template found for %s", v.name)
	}

	if v.driver != "" && volumeResource.Driver != v.driver {
		return fmt.Errorf("Volume %q needs to be recreated - driver has changed", v.name)
	}
	return nil
}

func (v *Volume) create(ctx context.Context) error {
	driverOptions := map[string]interface{}{}
	for k, v := range v.driverOptions {
		driverOptions[k] = v
	}
	_, err := v.project.Client.VolumeTemplate.Create(&client.VolumeTemplate{
		Name:         v.name,
		Driver:       v.driver,
		DriverOpts:   driverOptions,
		External:     v.external,
		PerContainer: v.perContainer,
		StackId:      v.project.Stack.Id,
	})
	return err
}

func NewVolume(p *project.Project, name string, config *config.VolumeConfig) *Volume {
	return &Volume{
		project:       p,
		name:          name,
		driver:        config.Driver,
		driverOptions: config.DriverOpts,
		external:      config.External.External,
		perContainer:  config.PerContainer,
	}
}
