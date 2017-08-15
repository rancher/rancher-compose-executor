package executor

import (
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/rancher-compose-executor/executor/handlers"
	"github.com/rancher/rancher-compose-executor/version"
)

func Main() {
	logger := logrus.WithFields(logrus.Fields{
		"version": version.VERSION,
	})

	logger.Info("Starting rancher-compose-executor")

	eventHandlers := map[string]events.EventHandler{
		"stack.create": handlers.WithTimeout(handlers.CreateStack),
		"stack.update": handlers.WithTimeout(handlers.UpdateStack),
		"ping": func(event *events.Event, apiClient *client.RancherClient) error {
			return nil
		},
	}

	url := os.Getenv("CATTLE_URL")
	if url == "" {
		url = "http://localhost:8080/v3"
	}

	router, err := events.NewEventRouter("rancher-compose-executor", 2000,
		url,
		os.Getenv("CATTLE_ACCESS_KEY"),
		os.Getenv("CATTLE_SECRET_KEY"),
		nil, eventHandlers, "stack", 250, events.DefaultPingConfig)
	if err != nil {
		logrus.WithField("error", err).Fatal("Unable to create event router")
	}

	if err := router.Start(nil); err != nil {
		logrus.WithField("error", err).Fatal("Unable to start event router")
	}

	logger.Info("Exiting rancher-compose-executor")
}
