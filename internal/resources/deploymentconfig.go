package resources

import (
	"context"

	v1 "github.com/openshift/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeploymentConfigScaleError struct {
	msg string
}

func (err DeploymentConfigScaleError) Error() string {
	return err.msg
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
