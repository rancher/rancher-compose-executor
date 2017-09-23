package project

import (
	"github.com/rancher/go-rancher/v3"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net/url"
	"strings"
)

type ErrClusterNotReady struct {
	err error
}

func (e ErrClusterNotReady) Error() string {
	return e.err.Error()
}

func NewErrClusterNotReady(err error) error {
	return ErrClusterNotReady{err}
}

func IsErrClusterNotReady(err error) bool {
	_, ok := err.(ErrClusterNotReady)
	return ok
}

func (p *Project) checkClusterReady() error {
	if p.Cluster.K8sClientConfig == nil || len(p.Config.KubernetesResources) == 0 {
		return nil
	}

	config := &rest.Config{
		Host:        getHost(p.Client, p.Cluster),
		BearerToken: p.Cluster.K8sClientConfig.BearerToken,
	}

	if !strings.HasPrefix(p.Cluster.K8sClientConfig.Address, "http://") {
		config.TLSClientConfig = rest.TLSClientConfig{
			// TODO
			Insecure: true,
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return NewErrClusterNotReady(err)
	}

	if _, err = clientset.Discovery().ServerVersion(); err != nil {
		return NewErrClusterNotReady(err)
	}

	return nil
}

// TODO: move this code into go-rancher
func getHost(rancherClient *client.RancherClient, cluster *client.Cluster) string {
	u, _ := url.Parse(rancherClient.GetOpts().Url)
	u.Path = "/k8s/clusters/"
	clusterOverrideURL := u.String()
	if clusterOverrideURL != "" {
		return clusterOverrideURL + cluster.Id
	}
	if strings.HasSuffix(cluster.K8sClientConfig.Address, "443") {
		return "https://" + cluster.K8sClientConfig.Address
	}
	return "http://" + cluster.K8sClientConfig.Address
}
