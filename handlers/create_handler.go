package handlers

import (
	"errors"
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/docker/libcompose/project"
	"github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-rancher/client"
	"github.com/rancher/rancher-compose-executor/lookup"
	"github.com/rancher/rancher-compose/rancher"
)

func CreateEnvironment(event *events.Event, apiClient *client.RancherClient) error {
	logger := logrus.WithFields(logrus.Fields{
		"resourceId": event.ResourceId,
		"eventId":    event.Id,
	})

	logger.Info("Stack Create Event Received")

	if err := createEnvironment(logger, event, apiClient); err != nil {
		logger.Errorf("Stack Create Event Failed: %v", err)
		publishTransitioningReply(err.Error(), event, apiClient)
		return err
	}

	logger.Info("Stack Create Event Done")
	return nil
}

func createEnvironment(logger *logrus.Entry, event *events.Event, apiClient *client.RancherClient) error {
	env, err := apiClient.Environment.ById(event.ResourceId)
	if err != nil {
		return err
	}

	if env == nil {
		return errors.New("Failed to find stack")
	}

	if env.DockerCompose == "" {
		return emptyReply(event, apiClient)
	}

	project, err := constructProject(logger, env, apiClient.Opts.Url, apiClient.Opts.AccessKey, apiClient.Opts.SecretKey)
	if err != nil {
		return err
	}

	publishTransitioningReply("Creating stack", event, apiClient)

	if err := project.Create(); err != nil {
		return err
	}

	return emptyReply(event, apiClient)
}

func constructProject(logger *logrus.Entry, env *client.Environment, url, accessKey, secretKey string) (*project.Project, error) {
	context := rancher.Context{
		Context: project.Context{
			ProjectName:  env.Name,
			ComposeBytes: []byte(env.DockerCompose),
			EnvironmentLookup: &lookup.MapEnvLookup{
				Env: env.Environment,
			},
		},
		Url:                 fmt.Sprintf("%s/projects/%s/schemas", url, env.AccountId),
		AccessKey:           accessKey,
		SecretKey:           secretKey,
		RancherComposeBytes: []byte(env.RancherCompose),
	}

	p, err := rancher.NewProject(&context)
	if err != nil {
		return nil, err
	}

	p.AddListener(NewListenLogger(logger, p))
	return p, p.Parse()
}
