package resources

import (
	"fmt"

	"golang.org/x/net/context"

	"bytes"
	log "github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/rancher-compose-executor/project"
	"github.com/rancher/rancher-compose-executor/project/options"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/url"
	"os"
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

func KubernetesResourcesCreate(p *project.Project) (project.ResourceSet, error) {
	u, err := url.Parse(p.Client.GetOpts().Url)
	if err != nil {
		return nil, err
	}
	account, err := p.Client.Account.ById(p.Stack.AccountId)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join("/k8s/clusters", p.Cluster.Id)
	return &KubernetesResources{
		resources: p.Config.KubernetesResources,
		cluster:   p.Cluster,
		endpoint:  u.String(),
		namespace: account.ExternalId,
	}, nil
}

type KubernetesResources struct {
	resources map[string]interface{}
	cluster   *client.Cluster
	endpoint  string
	namespace string
}

func (h *KubernetesResources) Initialize(ctx context.Context, _ options.Options) error {
	if h.cluster.K8sClientConfig == nil {
		return nil
	}

	f, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())

	if _, err := f.Write(generateKubeconfig(h.endpoint, h.cluster.K8sClientConfig.BearerToken)); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	for name, resource := range h.resources {
		resourceBytes, err := yaml.Marshal(resource)
		if err != nil {
			return err
		}

		cmd := exec.Command("kubectl", "--kubeconfig", f.Name(), "-n", h.namespace, "apply", "-f", "-")
		cmd.Stdin = bytes.NewReader(resourceBytes)

		log.Infof("Creating Kubernetes resource %s", name)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("Failed to apply Kubernetes resource %s: %v (%s)", name, err, output)
		}
	}
	return nil
}

func generateKubeconfig(endpoint, token string) []byte {
	return []byte(fmt.Sprintf(kubeconfigTemplate, endpoint, token))
}
