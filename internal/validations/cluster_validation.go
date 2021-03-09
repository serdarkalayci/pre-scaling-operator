package validations

import (
	"strings"

	c "github.com/containersol/prescale-operator/internal"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
)

// GetConfig return the client configuration
func getConfig() clientcmd.ClientConfig {
	configLoadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		configLoadingRules,
		&clientcmd.ConfigOverrides{})
}

// ClusterCheck checks if we are operating in an Openshift cluster
func ClusterCheck() (bool, error) {

	restConfig, err := getConfig().ClientConfig()
	if err != nil {
		return false, err
	}

	kubernetesclient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return false, err
	}

	_, err = kubernetesclient.DiscoveryClient.ServerResourcesForGroupVersion(c.OpenshiftObjectGroup)
	if err != nil {
		if strings.Contains(err.Error(), c.ResourceNotFound) {
			return false, nil
		}
		return false, err
	}

	ctrl.Log.Info("Openshift resources found. Activating the Openshift objects watcher")

	return true, nil
}
