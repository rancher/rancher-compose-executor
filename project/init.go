package project

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/docker/libcompose/logger"
	"github.com/rancher/go-rancher/catalog"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/rancher-compose-executor/lookup"
	"github.com/rancher/rancher-compose-executor/lookup/server"
	"github.com/rancher/rancher-compose-executor/utils"
)

func (p *Project) LoadFromTemplateVersion(templateVersion catalog.TemplateVersion, answers map[string]string) error {
	p.TemplateVersion = &templateVersion
	return p.Load(templateVersion.Files, answers)
}

func (p *Project) Load(templates map[string]interface{}, answers map[string]string) error {
	var err error

	p.Templates = utils.ToMapByte(templates)
	p.Answers = answers

	if p.LoggerFactory == nil {
		p.LoggerFactory = &logger.NullLogger{}
	}

	if p.ResourceLookup == nil {
		p.ResourceLookup = &lookup.MemoryResourceLookup{
			Content: p.Templates,
		}
	}

	if p.Name == "" {
		return errors.New("Name is required")
	}

	if stackSchema, ok := p.Client.GetTypes()["stack"]; !ok || !utils.Contains(stackSchema.CollectionMethods, "POST") {
		return fmt.Errorf("Can not create a stack, check API key [%s] for [%s]",
			p.Client.GetOpts().AccessKey,
			p.Client.GetOpts().Url)
	}

	if p.Stack == nil {
		stack, err := loadStack(p.Name, p.Client)
		if err != nil {
			return err
		}

		p.Stack = stack
	}

	if p.ServerResourceLookup == nil {
		p.ServerResourceLookup = server.NewLookup(p.Stack.Id, p.Client)
	}

	defer p.Config.Complete()

	composeBytes, file, err := p.ResourceLookup.Lookup("compose.yml", ".")
	if err == nil {
		return p.load(file, composeBytes)
	}

	composeBytes, file, err = p.ResourceLookup.Lookup("docker-compose.yml", ".")
	if err == nil {
		if err := p.load(file, composeBytes); err != nil {
			return err
		}
	} else {
		composeBytes, file, err = p.ResourceLookup.Lookup("docker-compose.yml.tpl", ".")
		if err != nil {
			return err
		}
		if err := p.load(file, composeBytes); err != nil {
			return err
		}
	}

	composeBytes, file, err = p.ResourceLookup.Lookup("rancher-compose.yml", ".")
	if err == nil {
		err = p.load(file, composeBytes)
		if err != nil {
			return err
		}
	}

	return nil
}

func loadStack(projectName string, c *client.RancherClient) (*client.Stack, error) {
	logrus.Debugf("Looking for stack %s", projectName)
	// First try by name
	stacks, err := c.Stack.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"name":         projectName,
			"removed_null": nil,
		},
	})
	if err != nil {
		return nil, err
	}

	for _, stack := range stacks.Data {
		if strings.EqualFold(projectName, stack.Name) {
			logrus.Debugf("Found stack: %s(%s)", stack.Name, stack.Id)
			return &stack, nil
		}
	}

	// Now try not by name for case sensitive databases
	stacks, err = c.Stack.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"removed_null": nil,
		},
	})
	if err != nil {
		return nil, err
	}

	for _, stack := range stacks.Data {
		if strings.EqualFold(projectName, stack.Name) {
			logrus.Debugf("Found stack: %s(%s)", stack.Name, stack.Id)
			return &stack, nil
		}
	}

	logrus.Infof("Creating stack %s", projectName)
	stack, err := c.Stack.Create(&client.Stack{
		Name: projectName,
	})
	if err != nil {
		return nil, err
	}

	return stack, nil
}
