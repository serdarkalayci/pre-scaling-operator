package validations

import (
	"strings"

	constants "github.com/containersol/prescale-operator/internal"
	"github.com/containersol/prescale-operator/pkg/utils/client"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Checks if the RedisCluster CRD is in place
func RedisClusterInstalled() (bool, error) {
	kubernetesclient, err := client.GetClientSet()
	if err != nil {
		return false, err
	}

	_, err = kubernetesclient.DiscoveryClient.ServerResourcesForGroupVersion(constants.RedisClusterObjectGroup)
	if err != nil {
		if strings.Contains(err.Error(), constants.ResourceNotFound) {
			return false, nil
		}
		return false, err
	}
	ctrl.Log.Info("RedisCluster CRD found. Activating the RedisCluster objects watcher")
	return true, nil
}

// OpenshiftClusterCheck checks if we are operating in an Openshift cluster
func OpenshiftClusterCheck() (bool, error) {

	kubernetesclient, err := client.GetClientSet()
	if err != nil {
		return false, err
	}

	//We use the discovery client to identify the API resources of a given group and version
	openshiftObjects, err := kubernetesclient.DiscoveryClient.ServerResourcesForGroupVersion(constants.OpenshiftObjectGroup)
	if err != nil {
		if strings.Contains(err.Error(), constants.ResourceNotFound) {
			return false, nil
		}
		return false, err
	}

	// We enable the deploymentconfig watcher only if we verify that the deploymentconfig API resource exists in the API server
	for resource := range openshiftObjects.APIResources {
		if openshiftObjects.APIResources[resource].Kind == constants.OpenshiftResources {
			ctrl.Log.
				WithValues("kind", openshiftObjects.APIResources[resource].Kind).
				Info("Openshift resources found. Activating the Openshift objects watcher")
			return true, nil
		}
	}

	return false, nil
}
