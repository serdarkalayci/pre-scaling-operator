package reconciler

import (
	"context"
	"errors"
	"fmt"

	c "github.com/containersol/prescale-operator/internal"
	"github.com/containersol/prescale-operator/internal/quotas"
	"github.com/containersol/prescale-operator/internal/resources"
	"github.com/containersol/prescale-operator/internal/states"
	g "github.com/containersol/prescale-operator/pkg/utils/global"
	ocv1 "github.com/openshift/api/apps/v1"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NamespaceEvents struct {
	QuotaExceeded    string
	ReconcileSuccess []string
	ReconcileFailure []string
}

func ReconcileNamespace(ctx context.Context, _client client.Client, namespace string, stateDefinitions states.States, clusterState states.State, recorder record.EventRecorder) (NamespaceEvents, string, error) {

	var objectsToReconcile int
	var nsEvents NamespaceEvents

	var limitsneeded corev1.ResourceList

	log := ctrl.Log.
		WithValues("namespace", namespace)

	log.Info("Reconciling namespace")

	finalState, err := GetAppliedState(ctx, _client, namespace, stateDefinitions, clusterState)
	if err != nil {
		return nsEvents, finalState.Name, err
	}

	// We now need to look for objects (currently supported deployments and deploymentConfigs) which are opted in,
	// then use their annotations to determine the correct scale
	deployments, err := resources.ScalingItemLister(ctx, _client, namespace, c.OptInLabel)
	if err != nil {
		log.Error(err, "Cannot list deployments in namespace")
		return nsEvents, finalState.Name, err
	}
	objectsToReconcile = objectsToReconcile + len(deployments)

	scaleReplicalist, err := resources.StateReplicasList(finalState, deployments)
	if err != nil {
		log.Error(err, "Cannot fetch replicas of all opted-in deployments")
		return nsEvents, finalState.Name, err
	}

	//Here we calculate the resource limits we need from all deployments combined
	limitsneeded = resources.LimitsNeededList(deployments, scaleReplicalist)

	// After we have calculated the resources needed from all workloads in a given namespace, we can determine if the scaling should be allowed to go through
	allowed, err := quotas.ResourceQuotaCheck(ctx, namespace, limitsneeded)
	if err != nil {
		log.Error(err, "Cannot calculate the resource quotas")
		return nsEvents, finalState.Name, err
	}

	if allowed {
		for i, deployment := range deployments {
			// Don't scale if we don't need to
			if deployment.SpecReplica == scaleReplicalist[i].Replicas {
				return nsEvents, finalState.Name, nil
			}

			scalingItem, notFoundErr := g.GetDenyList().GetDeploymentInfoFromList(deployment)
			if notFoundErr == nil {
				if scalingItem.DesiredReplicas != scaleReplicalist[i].Replicas {
					g.GetDenyList().SetScalingItemOnList(scalingItem, scalingItem.Failure, scalingItem.FailureMessage, scaleReplicalist[i].Replicas)

					log.WithValues("Name: ", scalingItem.Name).
						WithValues("Namespace: ", scalingItem.Namespace).
						WithValues("DeploymentConfig: ", scalingItem.IsDeploymentConfig).
						WithValues("DesiredReplicaount: ", scalingItem.DesiredReplicas).
						WithValues("Failure: ", scalingItem.Failure).
						WithValues("Failure message: ", scalingItem.FailureMessage).
						Info("Deployment is already being scaled at the moment. Updated desired replica count")
				}
				continue
			}

			err := resources.ScaleOrStepScale(ctx, _client, deployment, scaleReplicalist[i], "NSSCALER")

			RegisterEvents(ctx, _client, recorder, err, scalingItem)
			if !g.GetDenyList().IsDeploymentInFailureState(scalingItem) {
				g.GetDenyList().RemoveFromList(scalingItem)
			}

			if err != nil {
				log.Error(err, "Error scaling the deployment")
				continue
			}
		}
	} else {
		nsEvents.QuotaExceeded = namespace
	}

	if objectsToReconcile == 0 {
		return nsEvents, finalState.Name, err
	}

	return nsEvents, finalState.Name, err
}

func ReconcileScalingItem(ctx context.Context, _client client.Client, scalingItem g.ScalingInfo, state states.State, forceReconcile bool, recorder record.EventRecorder, whereFromScalingItem string) error {
	log := ctrl.Log.
		WithValues("deploymentItem", scalingItem.Name).
		WithValues("namespace", scalingItem.Namespace)

	stateReplica, err := resources.StateReplicas(state, scalingItem)
	if err != nil {
		log.Error(err, "Error getting the state replicas")
		return err
	}

	// Don't scale if we don't need to
	if scalingItem.SpecReplica == stateReplica.Replicas {
		return nil
	}

	allowed, err := quotas.ResourceQuotaCheck(ctx, scalingItem.Namespace, resources.LimitsNeeded(scalingItem, stateReplica.Replicas))
	if err != nil {
		log.Error(err, "Cannot calculate the resource quotas")
		return err
	}

	log = ctrl.Log.
		WithValues("Allowed", allowed)
	log.Info("Quota Check")

	if allowed {
		if g.GetDenyList().IsBeingScaled(scalingItem) && !g.GetDenyList().IsDeploymentInFailureState(scalingItem) {
			if scalingItem.DesiredReplicas != stateReplica.Replicas {
				// Update the desired replica count with a "jump ahead". Because the scaler is active with this ScaleItem we need to tell them via the concurrent list that the desiredreplicacount has changed
				g.GetDenyList().SetScalingItemOnList(scalingItem, scalingItem.Failure, scalingItem.FailureMessage, stateReplica.Replicas)

				log.WithValues("Deployment: ", scalingItem.Name).
					WithValues("Namespace: ", scalingItem.Namespace).
					WithValues("DesiredReplicaount: ", scalingItem.DesiredReplicas).
					WithValues("Failure: ", scalingItem.Failure).
					WithValues("Failure message: ", scalingItem.FailureMessage).
					Info("Deployment is already being scaled at the moment. Updated desired replica count")
			}
		} else {
			_ = whereFromScalingItem
			err = resources.ScaleOrStepScale(ctx, _client, scalingItem, stateReplica, "deployScaler")
			RegisterEvents(ctx, _client, recorder, err, scalingItem)
			if !g.GetDenyList().IsDeploymentInFailureState(scalingItem) {
				g.GetDenyList().RemoveFromList(scalingItem)
			}
			if err != nil {
				log.Error(err, "Error scaling the deployment")
				return err
			}
		}

	}

	return nil
}

func GetAppliedState(ctx context.Context, _client client.Client, namespace string, stateDefinitions states.States, clusterState states.State) (states.State, error) {
	// Here we allow overriding the cluster state by passing it in.
	// This allows us to not recall the client when looping namespaces
	if clusterState == (states.State{}) {
		var err error
		clusterState, err = fetchClusterState(ctx, _client, stateDefinitions)
		if err != nil {
			return states.State{}, err
		}
	}

	// If we receive an error here, we cannot handle it and should return
	namespaceState, err := fetchNameSpaceState(ctx, _client, stateDefinitions, namespace)
	if err != nil {
		return states.State{}, err
	}

	if namespaceState == (states.State{}) && clusterState == (states.State{}) {
		return states.State{}, err
	}

	finalState := stateDefinitions.FindPriorityState(namespaceState, clusterState)
	return finalState, nil
}

func fetchClusterState(ctx context.Context, _client client.Client, stateDefinitions states.States) (states.State, error) {
	clusterStateName, err := states.GetClusterScalingState(ctx, _client)
	if err != nil {
		switch err.(type) {
		case states.NotFound:
		case states.TooMany:
			ctrl.Log.V(3).Info("Could not process cluster state, but continuing safely.")
		default:
			// For the moment, we cannot deal with any other error.
			return states.State{}, errors.New("could not retrieve cluster states")
		}
	}
	clusterState := states.State{}
	if clusterStateName != "" {
		err = stateDefinitions.FindState(clusterStateName, &clusterState)
		if err != nil {
			ctrl.Log.
				V(3).
				WithValues("state name", clusterStateName).
				Error(err, "Could not find ClusterScalingState within ClusterStateDefinitions. Continuing without considering ClusterScalingState.")
		}
	}
	return clusterState, nil
}

func fetchNameSpaceState(ctx context.Context, _client client.Client, stateDefinitions states.States, namespace string) (states.State, error) {
	namespaceStateName, err := states.GetNamespaceScalingStateName(ctx, _client, namespace)
	if err != nil {
		switch err.(type) {
		case states.NotFound:
		case states.TooMany:
			ctrl.Log.V(3).Info("Could not process namespaced state, but continuing safely.")
		default:
			return states.State{}, err
		}
	}
	namespaceState := states.State{}
	if namespaceStateName != "" {
		err = stateDefinitions.FindState(namespaceStateName, &namespaceState)
		if err != nil {
			ctrl.Log.
				V(3).
				WithValues("state name", namespaceStateName).
				Error(err, "Could not find ScalingState within ClusterStateDefinitions. Continuing without considering ScalingState.")
		}
	}
	return namespaceState, nil
}

func RegisterEvents(ctx context.Context, _client client.Client, recorder record.EventRecorder, scalerErr error, scalingItem g.ScalingInfo) {
	// refresh the item to get newest replica count
	scalingItem, _ = g.GetDenyList().GetDeploymentInfoFromList(scalingItem)
	if scalingItem.IsDeploymentConfig {
		deplConf := ocv1.DeploymentConfig{}
		deplConf, getErr := resources.DeploymentConfigGetterByScaleItem(ctx, _client, scalingItem)
		if getErr == nil {
			if scalerErr != nil {
				recorder.Event(deplConf.DeepCopyObject(), "Warning", "Deploymentconfig scale error", scalerErr.Error()+" | "+fmt.Sprintf("Failed to scale the Deploymentconfig to %d replicas. Stuck on: %d replicas", scalingItem.DesiredReplicas, deplConf.Spec.Replicas))
			} else {
				recorder.Event(deplConf.DeepCopyObject(), "Normal", "Deploymentconfig scaled", fmt.Sprintf("Successfully scaled the Deploymentconfig to %d replicas", deplConf.Spec.Replicas))
			}
		} else {
			recorder.Event(deplConf.DeepCopyObject(), "Warning", "Deploymentconfig scale error", scalerErr.Error()+" | "+fmt.Sprintf("Failed to scale the Deploymentconfig to %d replicas. Most likely cause is that the Deploymentconfig doesn't exist anymore.", scalingItem.DesiredReplicas))
		}
	} else {
		depl := v1.Deployment{}
		depl, getErr := resources.DeploymentGetterByScaleItem(ctx, _client, scalingItem)
		if getErr == nil {
			if scalerErr != nil {
				recorder.Event(depl.DeepCopyObject(), "Warning", "Deployment scale error", scalerErr.Error()+" | "+fmt.Sprintf("Failed to scale the Deployment to %d replicas. Stuck on: %d replicas", scalingItem.DesiredReplicas, *depl.Spec.Replicas))
			} else {
				recorder.Event(depl.DeepCopyObject(), "Normal", "Deployment scaled", fmt.Sprintf("Successfully scaled the Deployment to %d replicas", *depl.Spec.Replicas))
			}
		} else {
			recorder.Event(depl.DeepCopyObject(), "Warning", "Deployment scale error", scalerErr.Error()+" | "+fmt.Sprintf("Failed to scale the Deployment to %d replicas. Most likely cause is that the Deployment doesn't exist anymore.", scalingItem.DesiredReplicas))
		}

	}

}
