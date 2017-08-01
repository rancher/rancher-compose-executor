package server

import (
	"fmt"
	"strings"

	"github.com/rancher/go-rancher/v3"
)

type RancherServerLookup struct {
	stackID string
	c       *client.RancherClient
}

func NewLookup(stackID string, client *client.RancherClient) *RancherServerLookup {
	return &RancherServerLookup{
		stackID: stackID,
		c:       client,
	}
}

func resolveNameAndStackId(c *client.RancherClient, currentStackId, name string) (string, string, error) {
	parts := strings.SplitN(name, "/", 2)
	if len(parts) == 1 {
		return name, currentStackId, nil
	}

	stacks, err := c.Stack.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"name":         parts[0],
			"removed_null": nil,
		},
	})
	if err != nil {
		return "", "", err
	}

	if len(stacks.Data) == 0 {
		return "", "", fmt.Errorf("Failed to find stack: %s", parts[0])
	}

	return parts[1], stacks.Data[0].Id, nil
}
