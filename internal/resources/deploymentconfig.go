package resources

import (
	"context"
	"github.com/containersol/prescale-operator/internal/validations"
	v1 "github.com/openshift/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func DeploymentConfigLister(ctx context.Context, _client client.Client, namespace string, OptInLabel map[string]string) (v1.DeploymentConfigList, error) {

	deploymentconfigs := v1.DeploymentConfigList{}
	err := _client.List(ctx, &deploymentconfigs, client.MatchingLabels(OptInLabel), client.InNamespace(namespace))
	if err != nil {
		return v1.DeploymentConfigList{}, err
	}
	return deploymentconfigs, nil

}

func DeploymentConfigGetter(ctx context.Context, _client client.Client, req ctrl.Request) (v1.DeploymentConfig, error) {

	deploymentconfig := v1.DeploymentConfig{}
	err := _client.Get(ctx, req.NamespacedName, &deploymentconfig)
	if err != nil {
		return v1.DeploymentConfig{}, err
	}
	return deploymentconfig, nil

}

func DeploymentConfigScaler(ctx context.Context, _client client.Client, deploymentConfig v1.DeploymentConfig, replicas int32) error {

	deploymentConfig.Spec.Replicas = replicas
	err := _client.Update(ctx, &deploymentConfig, &client.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func DeploymentConfigOptinLabel(deploymentConfig v1.DeploymentConfig) (bool, error) {

	return validations.OptinLabelExists(deploymentConfig.GetLabels())
}
