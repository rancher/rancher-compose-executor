package handlers

import (
	"errors"

	"os"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/rancher-compose-executor/kubectl"
	"github.com/rancher/rancher-compose-executor/parser/kubernetes"
)

func RemoveStack(event *events.Event, apiClient *client.RancherClient) error {
	return doRemove(event, apiClient, "Delete Stack")
}

func doRemove(event *events.Event, apiClient *client.RancherClient, msg string) error {
	logger := logrus.WithFields(logrus.Fields{
		"resourceId": event.ResourceID,
		"eventId":    event.ID,
	})

	logger.Infof("%s Event Received", msg)

	if err := stackRemove(event, apiClient); err != nil {
		logger.Errorf("%s Event Failed: %v", msg, err)
		publishTransitioningReply(err.Error(), event, apiClient, true)
		return err
	}

	logger.Infof("%s Event Done", msg)
	return emptyReply(event, apiClient)
}

func stackRemove(event *events.Event, apiClient *client.RancherClient) error {
	stack, err := apiClient.Stack.ById(event.ResourceID)
	if err != nil {
		return err
	}
	if stack == nil {
		return errors.New("Failed to find stack")
	}

	cluster, err := apiClient.Cluster.ById(stack.ClusterId)
	if err != nil {
		return err
	}
	if cluster == nil {
		return errors.New("Failed to find cluster")
	}
	if cluster.K8sClientConfig == nil {
		return nil
	}

	endpoint, err := kubectl.GetClusterEndpoint(apiClient, cluster.Id)
	if err != nil {
		return err
	}
	namespace, err := kubectl.GetNamespaceName(apiClient, stack)
	if err != nil {
		return err
	}
	kubeconfigLocation, err := kubectl.CreateKubeconfig(endpoint, cluster.K8sClientConfig.BearerToken)
	if err != nil {
		return err
	}
	defer os.Remove(kubeconfigLocation)

	for _, template := range stack.Templates {
		resources, err := kubernetes.GetResources([]byte(template))
		if err != nil {
			continue
		}
		for _, resource := range resources {
			if err := kubectl.Delete(kubeconfigLocation, resource.CombinedName,
				namespace, resource); err != nil {
				return err
			}
		}
	}
	return nil
}
