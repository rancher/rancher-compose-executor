package project

import (
	"os"

	"github.com/rancher/rancher-compose-executor/kubectl"
	"github.com/rancher/rancher-compose-executor/project/options"
	"golang.org/x/net/context"
)

func (p *Project) Create(ctx context.Context, options options.Options) error {
	return p.create(ctx, options, false)
}

func (p *Project) Up(ctx context.Context, options options.Options) error {
	return p.create(ctx, options, true)
}

func (p *Project) Delete(ctx context.Context) error {
	endpoint, err := kubectl.GetClusterEndpoint(p.Client, p.Cluster.Id)
	if err != nil {
		return err
	}
	namespace, err := kubectl.GetNamespaceName(p.Client, p.Stack)
	if err != nil {
		return err
	}
	kubeconfigLocation, err := kubectl.CreateKubeconfig(endpoint, p.Cluster.K8sClientConfig.BearerToken)
	if err != nil {
		return err
	}
	defer os.Remove(kubeconfigLocation)

	for name, resource := range p.Config.KubernetesResources {
		if err := kubectl.Delete(kubeconfigLocation, name, namespace, resource); err != nil {
			return err
		}
	}

	return nil
}
