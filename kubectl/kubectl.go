package kubectl

import (
	"fmt"

	"bytes"
	log "github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v3"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/url"
	"os/exec"
	"path"
)

const (
	kubeconfigTemplate = string(`apiVersion: v1
kind: Config
clusters:
- name: rancher-compose-executor
  cluster:
    insecure-skip-tls-verify: true
    server: %s
contexts:
- context:
    cluster: rancher-compose-executor
    user: rancher-compose-executor
  name: rancher-compose-executor
current-context: rancher-compose-executor
users:
- name: rancher-compose-executor
  user:
    token: %s`)
)

func GetClusterEndpoint(rancherClient *client.RancherClient, clusterId string) (string, error) {
	u, err := url.Parse(rancherClient.GetOpts().Url)
	if err != nil {
		return "", err
	}
	u.Path = path.Join("/k8s/clusters", clusterId)
	return u.String(), nil
}

func GetNamespaceName(rancherClient *client.RancherClient, stack *client.Stack) (string, error) {
	account, err := rancherClient.Account.ById(stack.AccountId)
	if err != nil {
		return "", err
	}
	return account.ExternalId, nil
}

func CreateKubeconfig(endpoint, token string) (string, error) {
	kubeconfig, err := ioutil.TempFile("", "")
	if err != nil {
		return "", err
	}
	kubeconfigContents := []byte(fmt.Sprintf(kubeconfigTemplate, endpoint, token))
	if _, err := kubeconfig.Write(kubeconfigContents); err != nil {
		return "", err
	}
	if err := kubeconfig.Close(); err != nil {
		return "", err
	}
	return kubeconfig.Name(), nil
}

func Apply(kubeconfigLocation, name, namespace string, resource interface{}) error {
	resourceBytes, err := yaml.Marshal(resource)
	if err != nil {
		return err
	}

	cmd := exec.Command("kubectl", "--kubeconfig", kubeconfigLocation, "-n", namespace, "apply", "-f", "-")
	cmd.Stdin = bytes.NewReader(resourceBytes)

	log.Infof("Applying Kubernetes resource %s", name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("Failed to apply Kubernetes resource %s: %v (%s)", name, err, output)
	}
	return nil
}

func Delete(kubeconfigLocation, name, namespace string, resource interface{}) error {
	resourceBytes, err := yaml.Marshal(resource)
	if err != nil {
		return err
	}

	cmd := exec.Command("kubectl", "--kubeconfig", kubeconfigLocation, "-n", namespace, "delete", "-f", "-")
	cmd.Stdin = bytes.NewReader(resourceBytes)

	log.Infof("Deleting Kubernetes resource %s", name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("Failed to delete Kubernetes resource %s: %v (%s)", name, err, output)
	}
	return nil
}
