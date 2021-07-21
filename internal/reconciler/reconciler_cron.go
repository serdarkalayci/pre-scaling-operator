package reconciler

import (
	"context"

	"github.com/containersol/prescale-operator/api/v1alpha1"
	"github.com/containersol/prescale-operator/internal/states"
	g "github.com/containersol/prescale-operator/pkg/utils/global"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RectifyScaleItemsInFailureState(client client.Client, recorder record.EventRecorder) error {

	log := ctrl.Log
	for inList := range g.GetDenyList().IterOverItemsInFailureState() {
		deploymentItem := inList.Value
		log.WithValues("Name", deploymentItem.Name).
			WithValues("Namespace", deploymentItem.Namespace).
			WithValues("IsDeploymentconfig", deploymentItem.ScalingItemType).
			WithValues("Failure", deploymentItem.Failure).
			WithValues("Failure Message", deploymentItem.FailureMessage).
			Info("Trying to rectify ScaleItem in failure state")

		//We are certain that we have an object to reconcile, we need to get the state definitions
		stateDefinitions, err := states.GetClusterScalingStates(context.TODO(), client)
		if err != nil {
			log.Error(err, "Failed to get ClusterStateDefinitions")
			return err
		}
		namespaceState, nsStateErr := states.FetchNameSpaceState(context.TODO(), client, stateDefinitions, inList.Value.Namespace)
		if err != nsStateErr {
			return nsStateErr
		}

		// get all css
		clusterScalingStates := v1alpha1.ClusterScalingStateList{}
		cssErr := client.List(context.TODO(), &clusterScalingStates)
		if cssErr != nil {
			return cssErr
		}

		deploymentItem = states.GetAppliedStateAndClassOnItem(deploymentItem, namespaceState, clusterScalingStates, stateDefinitions)

		reconcileErr := ReconcileScalingItem(context.TODO(), client, deploymentItem, true, recorder, "CRONJOB")
		if reconcileErr != nil {
			log.WithValues("Deployment", deploymentItem.Name).
				WithValues("Namespace", deploymentItem.Namespace).
				WithValues("IsDeploymentConfig", deploymentItem.ScalingItemType).
				WithValues("Failure", deploymentItem.Failure).
				WithValues("Failuremessage", deploymentItem.FailureMessage).
				Error(reconcileErr, "Failed to rectify the Failure state for the ScalingItem!")
		} else {
			// Succesfully rectified. Remove from failure state
			g.GetDenyList().RemoveFromList(deploymentItem)
			log.WithValues("Deployment", deploymentItem.Name).
				WithValues("Namespace", deploymentItem.Namespace).
				WithValues("IsDeploymentConfig", deploymentItem.ScalingItemType).
				WithValues("DesiredReplicas", deploymentItem.DesiredReplicas).
				Info("Successfully rectified the failing ScalingItem!")
		}
	}
	return nil
}
