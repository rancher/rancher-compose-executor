package resources

import (
	"golang.org/x/net/context"

	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/rancher-compose-executor/kubectl"
	"github.com/rancher/rancher-compose-executor/project"
	"github.com/rancher/rancher-compose-executor/project/options"
	"os"
)

func KubernetesResourcesCreate(p *project.Project) (project.ResourceSet, error) {
	endpoint, err := kubectl.GetClusterEndpoint(p.Client, p.Cluster.Id)
	if err != nil {
		return nil, err
	}
	namespace, err := kubectl.GetNamespaceName(p.Client, p.Stack)
	if err != nil {
		return nil, err
	}
	return &KubernetesResources{
		resources: p.Config.KubernetesResources,
		cluster:   p.Cluster,
		endpoint:  endpoint,
		namespace: namespace,
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

	kubeconfigLocation, err := kubectl.CreateKubeconfig(h.endpoint, h.cluster.K8sClientConfig.BearerToken)
	if err != nil {
		return err
	}
	defer os.Remove(kubeconfigLocation)

	for name, resource := range h.resources {
		if err := kubectl.Apply(kubeconfigLocation, name, h.namespace, resource); err != nil {
			return err
		}
	}
	return nil
}
