package main

import (
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-machine-service/events"
	"github.com/rancher/go-rancher/client"
	"github.com/rancher/rancher-compose-executor/handlers"
)

var (
	GITCOMMIT = "HEAD"
)

func main() {
	logger := logrus.WithFields(logrus.Fields{
		"gitcommit": GITCOMMIT,
	})

	logger.Info("Starting rancher-compose-executor")

	eventHandlers := map[string]events.EventHandler{
		"environment.create": handlers.CreateEnvironment,
		"ping": func(event *events.Event, apiClient *client.RancherClient) error {
			return nil
		},
	}

	router, err := events.NewEventRouter("rancher-compose-executor", 2000,
		os.Getenv("CATTLE_URL"),
		os.Getenv("CATTLE_ACCESS_KEY"),
		os.Getenv("CATTLE_SECRET_KEY"),
		nil, eventHandlers, "environment", 10)
	if err != nil {
		logrus.WithField("error", err).Fatal("Unable to create event router")
	}

	if err := router.Start(nil); err != nil {
		logrus.WithField("error", err).Fatal("Unable to start event router")
	}

	logger.Info("Exiting rancher-compose-executor")
}
