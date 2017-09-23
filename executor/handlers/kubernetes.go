package handlers

import (
	"github.com/rancher/go-rancher/v3"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net/url"
	"strings"
)

type errClusterNotReady struct {
	err error
}

func (e errClusterNotReady) Error() string {
	return e.err.Error()
}

func NewErrClusterNotReady(err error) error {
	return errClusterNotReady{err}
}

func IsErrClusterNotReady(err error) bool {
	_, ok := err.(errClusterNotReady)
	return ok
}

func checkClusterReady(rancherClient *client.RancherClient, cluster *client.Cluster) error {
	if cluster.K8sClientConfig == nil {
		return nil
	}

	config := &rest.Config{
		Host:        getHost(rancherClient, cluster),
		BearerToken: cluster.K8sClientConfig.BearerToken,
	}

	if !strings.HasPrefix(cluster.K8sClientConfig.Address, "http://") {
		config.TLSClientConfig = rest.TLSClientConfig{
			// TODO
			Insecure: true,
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return NewErrClusterNotReady(err)
	}

	_, err = clientset.Discovery().ServerVersion()
	if err != nil {
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
