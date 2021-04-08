package resources

import (
	"context"

	"github.com/containersol/prescale-operator/internal/validations"
	v1 "k8s.io/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//DeploymentLister lists all deployments in a namespace
func DeploymentLister(ctx context.Context, _client client.Client, namespace string, OptInLabel map[string]string) (v1.DeploymentList, error) {

	deployments := v1.DeploymentList{}
	err := _client.List(ctx, &deployments, client.MatchingLabels(OptInLabel), client.InNamespace(namespace))
	if err != nil {
		return v1.DeploymentList{}, err
	}
	return deployments, nil

}

//DeploymentGetter returns the specific deployment data given a reconciliation request
func DeploymentGetter(ctx context.Context, _client client.Client, req ctrl.Request) (v1.Deployment, error) {

	deployment := v1.Deployment{}
	err := _client.Get(ctx, req.NamespacedName, &deployment)
	if err != nil {
		return v1.Deployment{}, err
	}
	return deployment, nil

}

//DeploymentScaler scales the deployment to the desired replica number
func DeploymentScaler(ctx context.Context, _client client.Client, deployment v1.Deployment, replicas int32) error {

	if v, found := deployment.GetAnnotations()["scaler/allow-autoscaling"]; found {
		if v == "true" {
			if replicas <= *deployment.Spec.Replicas {
				return nil
			}
		}
	}

	deployment.Spec.Replicas = &replicas
	err := _client.Update(ctx, &deployment, &client.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

//DeploymentOptinLabel returns true if the optin-label is found and is true for the deployment
func DeploymentOptinLabel(deployment v1.Deployment) (bool, error) {

	return validations.OptinLabelExists(deployment.GetLabels())
}
