package reconciler

import (
	"context"
	"errors"

	c "github.com/containersol/prescale-operator/internal"
	"github.com/containersol/prescale-operator/internal/quotas"
	"github.com/containersol/prescale-operator/internal/resources"
	"github.com/containersol/prescale-operator/internal/states"
	g "github.com/containersol/prescale-operator/pkg/utils/global"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NamespaceEvents struct {
	QuotaExceeded string
}

func ReconcileNamespace(ctx context.Context, _client client.Client, namespace string, stateDefinitions states.States, clusterState states.State) (NamespaceEvents, string, error) {

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
	deployments, err := resources.DeploymentItemLister(ctx, _client, namespace, c.OptInLabel)
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
			deploymentItem, notFoundErr := g.GetDenyList().GetDeploymentInfoFromList(deployment)
			if notFoundErr == nil {
				if deploymentItem.Failure {
					log.WithValues("Deployment: ", deploymentItem.Name).
						WithValues("Namespace: ", deploymentItem.Namespace).
						WithValues("DesiredReplicaount: ", deploymentItem.DesiredReplicas).
						WithValues("Failure: ", deploymentItem.Failure).
						WithValues("Failure message: ", deploymentItem.FailureMessage).
						Info("Deployment is in failure state! Not going to scale")
					continue
				}
				if deploymentItem.DesiredReplicas != scaleReplicalist[i].Replicas {
					g.GetDenyList().SetDeploymentInfoOnList(deploymentItem, deploymentItem.Failure, deploymentItem.FailureMessage, scaleReplicalist[i].Replicas)

					log.WithValues("Name: ", deploymentItem.Name).
						WithValues("Namespace: ", deploymentItem.Namespace).
						WithValues("DeploymentConfig: ", deploymentItem.IsDeploymentConfig).
						WithValues("DesiredReplicaount: ", deploymentItem.DesiredReplicas).
						WithValues("Failure: ", deploymentItem.Failure).
						WithValues("Failure message: ", deploymentItem.FailureMessage).
						Info("Deployment is already being scaled at the moment. Updated desired replica count")
				}
				continue
			}

			err := resources.ScaleOrStepScale(ctx, _client, deployment, scaleReplicalist[i], "NSSCALER")

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

func ReconcileDeploymentOrDeploymentConfig(ctx context.Context, _client client.Client, deploymentItem g.DeploymentInfo, state states.State) error {
	log := ctrl.Log.
		WithValues("deploymentItem", deploymentItem.Name).
		WithValues("namespace", deploymentItem.Namespace)

	stateReplica, err := resources.StateReplicas(state, deploymentItem)
	if err != nil {
		log.Error(err, "Error getting the state replicas")
		return err
	}

	allowed, err := quotas.ResourceQuotaCheck(ctx, deploymentItem.Namespace, resources.LimitsNeeded(deploymentItem, stateReplica.Replicas))
	if err != nil {
		log.Error(err, "Cannot calculate the resource quotas")
		return err
	}

	log = ctrl.Log.
		WithValues("Allowed", allowed)
	log.Info("Quota Check")

	if allowed {
		deploymentItem, notFoundErr := g.GetDenyList().GetDeploymentInfoFromList(deploymentItem)
		if notFoundErr == nil {
			if deploymentItem.Failure {
				log.WithValues("Deployment: ", deploymentItem.Name).
					WithValues("Namespace: ", deploymentItem.Namespace).
					WithValues("DesiredReplicaount: ", deploymentItem.DesiredReplicas).
					WithValues("Failure: ", deploymentItem.Failure).
					WithValues("Failure message: ", deploymentItem.FailureMessage).
					Info("Deployment is in failure state! Not going to scale")
				return nil
			}
			if deploymentItem.DesiredReplicas != stateReplica.Replicas {
				g.GetDenyList().SetDeploymentInfoOnList(deploymentItem, deploymentItem.Failure, deploymentItem.FailureMessage, stateReplica.Replicas)

				log.WithValues("Deployment: ", deploymentItem.Name).
					WithValues("Namespace: ", deploymentItem.Namespace).
					WithValues("DesiredReplicaount: ", deploymentItem.DesiredReplicas).
					WithValues("Failure: ", deploymentItem.Failure).
					WithValues("Failure message: ", deploymentItem.FailureMessage).
					Info("Deployment is already being scaled at the moment. Updated desired replica count")
			}
		} else {
			err = resources.ScaleOrStepScale(ctx, _client, deploymentItem, stateReplica, "deployScaler")
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
