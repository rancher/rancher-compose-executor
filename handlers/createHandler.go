package handlers

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/libcompose/cli/logger"
	"github.com/docker/libcompose/lookup"
	"github.com/docker/libcompose/project"
	"github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-rancher/client"
	"github.com/rancher/rancher-compose/rancher"
	"gopkg.in/yaml.v2"
)

func CreateEnvironment(event *events.Event, apiClient *client.RancherClient) (err error) {
	log.WithFields(log.Fields{
		"resourceId": event.ResourceId,
		"eventId":    event.Id,
	}).Info("Environment Create Event Received")

	env, err := getEnvironment(event.ResourceId, apiClient)
	if err != nil {
		return handleByIdError(err, event, apiClient)
	}

	if env.DockerCompose == "" {
		reply := newReply(event)
		return publishReply(reply, apiClient)
	}

	composeUrl := os.Getenv("CATTLE_URL") + "/projects/" + env.AccountId + "/schema"
	projectName := env.Name
	composeBytes := []byte(env.DockerCompose)
	rancherComposeMap := map[string]rancher.RancherConfig{}
	if env.RancherCompose != "" {
		err := yaml.Unmarshal([]byte(env.RancherCompose), rancherComposeMap)
		if err != nil {
			return handleByIdError(err, event, apiClient)
		}
	}

	publishChan := make(chan string, 10)
	defer func() {
		close(publishChan)
	}()
	go republishTransitioningReply(publishChan, event, apiClient)

	publishChan <- "Starting rancher-compose"

	if err := createEnv(composeUrl, projectName, composeBytes, rancherComposeMap, env); err != nil {
		return handleByIdError(err, event, apiClient)
	}

	if err != nil {
		//This remains the most important use of publish Chan - to communicate the reason for error back to cattle
		publishChan <- err.Error()
	} else {
		publishChan <- "Finished creating service"
	}

	reply := newReply(event)
	return publishReply(reply, apiClient)
}

func createEnv(rancherUrl, projectName string, composeBytes []byte, rancherComposeMap map[string]rancher.RancherConfig, env *client.Environment) error {
	context := rancher.Context{
		Url:           rancherUrl,
		RancherConfig: rancherComposeMap,
		Uploader:      nil,
	}
	context.ProjectName = projectName
	context.ComposeBytes = composeBytes
	context.ConfigLookup = nil
	context.EnvironmentLookup = &lookup.OsEnvLookup{}
	context.LoggerFactory = logger.NewColorLoggerFactory()
	context.ServiceFactory = &rancher.RancherServiceFactory{
		Context: &context,
	}

	p := project.NewProject(&context.Context)

	err := p.Parse()
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Errorf("Error parsing docker-compose.yml")
		return err
	}

	apiClient, err := client.NewRancherClient(&client.ClientOpts{
		Url:       rancherUrl,
		AccessKey: os.Getenv("CATTLE_ACCESS_KEY"),
		SecretKey: os.Getenv("CATTLE_SECRET_KEY"),
	})

	context.Client = apiClient

	c := &context

	c.Environment = env

	context.SidekickInfo = rancher.NewSidekickInfo(p)

	err = p.Create([]string{}...)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Error while creating project.")
		return err
	}
	return nil
}
