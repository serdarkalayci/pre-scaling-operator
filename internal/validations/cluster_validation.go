package validations

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Load load the configuration
func Load() *kubernetes.Clientset {
	config, err := GetConfig().ClientConfig()

	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	return clientset
}

// GetConfig return the client configuration
func GetConfig() clientcmd.ClientConfig {
	configLoadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		configLoadingRules,
		&clientcmd.ConfigOverrides{})
}

// DefaultNamespace return the client configuration
func DefaultNamespace() (namespace string) {
	configLoadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	clientconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(configLoadingRules, &clientcmd.ConfigOverrides{})
	namespace, _, err := clientconfig.Namespace()
	if err != nil {
		panic(err)
	}
	return namespace
}

// ClusterCheck checks if we are operating in an Openshift cluster
func ClusterCheck() (bool, error) {

	restConfig, err := GetConfig().ClientConfig()
	if err != nil {
		return false, err
	}

	kubernetesclient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return false, err
	}

	_, err = kubernetesclient.DiscoveryClient.ServerResourcesForGroupVersion("apps.openshift.io/v1")
	if err != nil {
		return false, err
	}

	return false, nil
}
