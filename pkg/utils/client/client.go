package client

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func getConfig() clientcmd.ClientConfig {
	configLoadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		configLoadingRules,
		&clientcmd.ConfigOverrides{})
}

func GetClientSet() (*kubernetes.Clientset, error) {

	var kubernetesclient *kubernetes.Clientset

	restConfig, err := getConfig().ClientConfig()
	if err != nil {
		return kubernetesclient, err
	}

	kubernetesclient, err = kubernetes.NewForConfig(restConfig)
	if err != nil {
		return kubernetesclient, err
	}

	return kubernetesclient, nil
}
