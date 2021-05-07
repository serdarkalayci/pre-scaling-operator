package reconciler

import (
	"context"

	"github.com/containersol/prescale-operator/internal/states"
	g "github.com/containersol/prescale-operator/pkg/utils/global"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RectifyDeploymentsInFailureState(client client.Client) {

	log := ctrl.Log
	var failureList []g.DeploymentInfo
	for _, deployment := range failureList {
		err := ReconcileDeploymentOrDeploymentConfig(context.TODO(), client, deployment, states.State{}, true)
		if err != nil {
			log.WithValues("Deployment", deployment.Name).
				WithValues("Namespace", deployment.Namespace).
				WithValues("IsDeploymentConfig", deployment.IsDeploymentConfig).
				WithValues("Failure", deployment.Failure).
				WithValues("Failuremessage", deployment.FailureMessage).
				WithValues("DesiredReplicas", deployment.DesiredReplicas).
				Error(err, "Failed to rectify the Failure state for the deployment!")
		} else {
			log.WithValues("Deployment", deployment.Name).
				WithValues("Namespace", deployment.Namespace).
				WithValues("IsDeploymentConfig", deployment.IsDeploymentConfig).
				WithValues("Failure", deployment.Failure).
				WithValues("Failuremessage", deployment.FailureMessage).
				WithValues("DesiredReplicas", deployment.DesiredReplicas).
				Info("Successfully rectified the failing deployment!")
		}
	}
}
