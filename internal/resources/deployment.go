package resources

import (
	"context"

	g "github.com/containersol/prescale-operator/pkg/utils/global"
	v1 "k8s.io/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type DeploymentScaleError struct {
	msg string
}

func (err DeploymentScaleError) Error() string {
	return err.msg
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

//DeploymentGetter returns the specific deployment data given a scaleitem
func DeploymentGetterByScaleItem(ctx context.Context, _client client.Client, deploymentItem g.ScalingInfo) (v1.Deployment, error) {
	var req reconcile.Request
	req.NamespacedName.Namespace = deploymentItem.Namespace
	req.NamespacedName.Name = deploymentItem.Name

	return DeploymentGetter(ctx, _client, req)

}
