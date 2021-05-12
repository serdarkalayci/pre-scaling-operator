package reconciler

import (
	"context"

	"github.com/containersol/prescale-operator/internal/states"
	g "github.com/containersol/prescale-operator/pkg/utils/global"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RectifyScaleItemsInFailureState(client client.Client, recorder record.EventRecorder) error {

	log := ctrl.Log
	for inList := range g.GetDenyList().IterOverItemsInFailureState() {
		item := inList.Value
		log.WithValues("Name", item.Name).
			WithValues("Namespace", item.Namespace).
			WithValues("IsDeploymentconfig", item.IsDeploymentConfig).
			WithValues("Failure", item.Failure).
			WithValues("Failure Message", item.FailureMessage).
			Info("Trying to rectify ScaleItem in failure state")

		stateDefinitions, stateDefErr := states.GetClusterScalingStates(context.TODO(), client)
		if stateDefErr != nil {
			log.Error(stateDefErr, "Failed to get ClusterStateDefinitions")
			return stateDefErr
		}

		// We need to calculate the desired state before we try to reconcile the deployment
		finalState, stateErr := GetAppliedState(context.TODO(), client, item.Namespace, stateDefinitions, states.State{})
		if stateErr != nil {
			return stateErr
		}
		err := ReconcileScalingItem(context.TODO(), client, item, finalState, true, recorder, "CRONJOB")
		if err != nil {
			log.WithValues("Deployment", item.Name).
				WithValues("Namespace", item.Namespace).
				WithValues("IsDeploymentConfig", item.IsDeploymentConfig).
				WithValues("Failure", item.Failure).
				WithValues("Failuremessage", item.FailureMessage).
				Error(err, "Failed to rectify the Failure state for the ScalingItem!")
		} else {
			// Succesfully rectified. Remove from failure state
			g.GetDenyList().RemoveFromList(item)
			log.WithValues("Deployment", item.Name).
				WithValues("Namespace", item.Namespace).
				WithValues("IsDeploymentConfig", item.IsDeploymentConfig).
				WithValues("DesiredReplicas", item.DesiredReplicas).
				Info("Successfully rectified the failing ScalingItem!")
		}
	}
	return nil
}
