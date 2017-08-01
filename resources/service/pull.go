package service

import (
	"errors"

	log "github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/rancher-compose-executor/utils"
)

func pullImage(c *client.RancherClient, image string, labels map[string]string, pullCached bool) error {
	taskOpts := &client.PullTask{
		Mode:   "all",
		Labels: utils.ToMapInterface(labels),
		Image:  image,
	}

	if pullCached {
		taskOpts.Mode = "cached"
	}

	task, err := c.PullTask.Create(taskOpts)
	if err != nil {
		return err
	}

	printed := map[string]string{}
	lastMessage := ""
	WaitFor(c, &task.Resource, task, func() string {
		if task.TransitioningMessage != "" && task.TransitioningMessage != "In Progress" && task.TransitioningMessage != lastMessage {
			printStatus(task.Image, printed, task.Status)
			lastMessage = task.TransitioningMessage
		}

		return task.Transitioning
	})

	if task.Transitioning == "error" {
		return errors.New(task.TransitioningMessage)
	}

	if !printStatus(task.Image, printed, task.Status) {
		return errors.New("Pull failed on one of the hosts")
	}

	log.Infof("Finished pulling %s", task.Image)
	return nil
}
