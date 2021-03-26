package resources

import (
	"context"

	"github.com/containersol/prescale-operator/internal/validations"
	v1 "github.com/openshift/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//DeploymentConfigLister lists all deploymentconfigs in a namespace
func DeploymentConfigLister(ctx context.Context, _client client.Client, namespace string, OptInLabel map[string]string) (v1.DeploymentConfigList, error) {

	deploymentconfigs := v1.DeploymentConfigList{}
	err := _client.List(ctx, &deploymentconfigs, client.MatchingLabels(OptInLabel), client.InNamespace(namespace))
	if err != nil {
		return v1.DeploymentConfigList{}, err
	}
	return deploymentconfigs, nil

}

//DeploymentConfigGetter returns the specific deploymentconfig data given a reconciliation request
func DeploymentConfigGetter(ctx context.Context, _client client.Client, req ctrl.Request) (v1.DeploymentConfig, error) {

	deploymentconfig := v1.DeploymentConfig{}
	err := _client.Get(ctx, req.NamespacedName, &deploymentconfig)
	if err != nil {
		return v1.DeploymentConfig{}, err
	}
	return deploymentconfig, nil

}

//DeploymentConfigScaler scales the deploymentconfig to the desired replica number
func DeploymentConfigScaler(ctx context.Context, _client client.Client, deploymentConfig v1.DeploymentConfig, replicas int32) error {

	if v, found := deploymentConfig.GetAnnotations()["scaler/type"]; found {
		if v == "autoscale" {
			if replicas <= deploymentConfig.Spec.Replicas {
				return nil
			}
		}
	}

	deploymentConfig.Spec.Replicas = replicas
	err := _client.Update(ctx, &deploymentConfig, &client.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

//DeploymentConfigOptinLabel returns true if the optin-label is found and is true for the deploymentconfig
func DeploymentConfigOptinLabel(deploymentConfig v1.DeploymentConfig) (bool, error) {

	return validations.OptinLabelExists(deploymentConfig.GetLabels())
}
