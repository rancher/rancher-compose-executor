package handlers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v3"
)

var (
	ErrTimeout = errors.New("Timeout waiting service")
)

func keepalive(request *events.Event, apiClient *client.RancherClient) (stopFunc func()) {
	ctx, cancel := context.WithCancel(context.Background())
	innerCtx, innerCancel := context.WithCancel(context.Background())
	go func() {
		defer innerCancel()
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
			publishTransitioningReply("", request, apiClient, false)
		}
	}()
	return func() {
		cancel()
		<-innerCtx.Done()
	}
}

func emptyReply(request *events.Event, apiClient *client.RancherClient) error {
	reply := newReply(request)
	return publishReply(reply, apiClient)
}

func publishReply(reply *client.Publish, apiClient *client.RancherClient) error {
	_, err := apiClient.Publish.Create(reply)
	return err
}

func publishTransitioningReply(msg string, request *events.Event, apiClient *client.RancherClient, isError bool) {
	// Since this is only updating the msg for the state transition, we will ignore errors here
	replyT := newReply(request)
	if isError {
		replyT.Transitioning = "error"
	} else {
		replyT.Transitioning = "yes"
	}

	replyT.TransitioningMessage = msg
	publishReply(replyT, apiClient)
}

func newReply(event *events.Event) *client.Publish {
	return &client.Publish{
		Name:        event.ReplyTo,
		PreviousIds: []string{event.ID},
	}
}

func WithTimeout(f func(event *events.Event, apiClient *client.RancherClient) error) func(event *events.Event, apiClient *client.RancherClient) error {
	return func(event *events.Event, apiClient *client.RancherClient) error {
		err := f(event, apiClient)
		if err == ErrTimeout {
			logrus.Infof("Timeout processing %s", fmt.Sprintf("%s:%s", event.ResourceType, event.ResourceID))
			return nil
		}
		return nil
	}
}
