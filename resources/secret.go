package resources

import (
	"encoding/base64"
	"fmt"

	"golang.org/x/net/context"

	log "github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/rancher-compose-executor/project"
	"github.com/rancher/rancher-compose-executor/project/options"
)

func SecretsCreate(p *project.Project) (project.ResourceSet, error) {
	secrets := make([]*Secret, 0, len(p.Config.Secrets))
	for name, config := range p.Config.Secrets {
		secrets = append(secrets, &Secret{
			project: p,
			name:        name,
			file:        config.File,
			external:    config.External,
		})
	}
	return &Secrets{
		secrets: secrets,
	}, nil
}

type Secrets struct {
	secrets []*Secret
}

func (s *Secrets) Initialize(ctx context.Context, _ options.Options) error {
	for _, secret := range s.secrets {
		if err := secret.EnsureItExists(ctx); err != nil {
			return err
		}
	}
	return nil
}

type Secret struct {
	project  *project.Project
	name     string
	file     string
	external string
}

func (s *Secret) EnsureItExists(ctx context.Context) error {
	existingSecrets, err := s.project.Client.Secret.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"name": s.name,
		},
	})
	if err != nil {
		return err
	}
	if len(existingSecrets.Data) > 0 {
		log.Infof("Secret %s already exists", s.name)
		return nil
	}
	if s.external != "" {
		return fmt.Errorf("Existing secret %s not found", s.name)
	}
	// TODO: use real relative path
	contents, filename, err := s.project.ResourceLookup.Lookup(s.file, "./")
	if err != nil {
		return err
	}
	log.Infof("Creating secret %s with contents from file %s", s.name, filename)
	_, err = s.project.Client.Secret.Create(&client.Secret{
		Name:  s.name,
		Value: base64.StdEncoding.EncodeToString(contents),
	})
	return err
}
