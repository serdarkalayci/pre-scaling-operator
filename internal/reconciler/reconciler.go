package reconciler

import (
	"context"
	"errors"

	c "github.com/containersol/prescale-operator/internal"
	"github.com/containersol/prescale-operator/internal/quotas"
	"github.com/containersol/prescale-operator/internal/resources"
	"github.com/containersol/prescale-operator/internal/state_replicas"
	"github.com/containersol/prescale-operator/internal/states"
	"github.com/containersol/prescale-operator/pkg/utils/math"
	ocv1 "github.com/openshift/api/apps/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NamespaceEvents struct {
	QuotaExceeded string
}

func ReconcileNamespace(ctx context.Context, _client client.Client, namespace string, stateDefinitions states.States, clusterState states.State) (NamespaceEvents, string, error) {

	var objectsToReconcile int
	var deploymentConfigs ocv1.DeploymentConfigList
	var nsEvents NamespaceEvents

	var scaleReplicalistDC []state_replicas.StateReplica
	var limitsneeded corev1.ResourceList

	log := ctrl.Log.
		WithValues("namespace", namespace)

	log.Info("Reconciling namespace")

	finalState, err := GetAppliedState(ctx, _client, namespace, stateDefinitions, clusterState)
	if err != nil {
		log.Error(err, "Cannot determine applied state for namespace")
		return nsEvents, finalState.Name, err
	}

	// We now need to look for objects (currently supported deployments and deploymentConfigs) which are opted in,
	// then use their annotations to determine the correct scale
	deployments, err := resources.DeploymentLister(ctx, _client, namespace, c.OptInLabel)
	if err != nil {
		log.Error(err, "Cannot list deployments in namespace")
		return nsEvents, finalState.Name, err
	}
	objectsToReconcile = objectsToReconcile + len(deployments.Items)

	scaleReplicalist, err := resources.DeploymentStateReplicasList(finalState, deployments)
	if err != nil {
		log.Error(err, "Cannot fetch replicas of all opted-in deployments")
		return nsEvents, finalState.Name, err
	}

	//Here we calculate the resource limits we need from all deployments combined
	limitsneeded = resources.LimitsNeededDeploymentList(deployments, scaleReplicalist)

	log.WithValues("deploy to Openshift", c.OpenshiftCluster).
		Info("Cluster")
	if c.OpenshiftCluster {

		deploymentConfigs, err = resources.DeploymentConfigLister(ctx, _client, namespace, c.OptInLabel)
		if err != nil {
			log.Error(err, "Cannot list deploymentConfigs in namespace")
			return nsEvents, finalState.Name, err
		}
		objectsToReconcile = objectsToReconcile + len(deploymentConfigs.Items)

		scaleReplicalistDC, err = resources.DeploymentConfigStateReplicasList(finalState, deploymentConfigs)

		if err != nil {
			log.Error(err, "Cannot fetch replicas of all opted-in deploymentconfigs")
			return nsEvents, finalState.Name, err
		}

		//In case of Openshift, we calculate the resource limits we need from all deploymentconfigs combined and we add it to the total number
		limitsneeded = math.Add(limitsneeded, resources.LimitsNeededDeploymentConfigList(deploymentConfigs, scaleReplicalistDC))

	}

	// After we have calculated the resources needed from all workloads in a given namespace, we can determine if the scaling should be allowed to go through
	allowed, err := quotas.ResourceQuotaCheck(ctx, namespace, limitsneeded)
	if err != nil {
		log.Error(err, "Cannot calculate the resource quotas")
		return nsEvents, finalState.Name, err
	}

	log = ctrl.Log.
		WithValues("Allowed", allowed)
	log.Info("Namespace Quota Check")

	if allowed {
		for i, deployment := range deployments.Items {
			err := resources.ScaleDeployment(ctx, _client, deployment, scaleReplicalist[i])

			if err != nil {
				log.Error(err, "Error scaling the deployment")
				continue
			}
		}
		if c.OpenshiftCluster {
			for i, deploymentConfig := range deploymentConfigs.Items {
				log := ctrl.Log.
					WithValues("deploymentconfig", deploymentConfig.Name).
					WithValues("namespace", deploymentConfig.Namespace)

				err := resources.ScaleDeploymentConfig(ctx, _client, deploymentConfig, scaleReplicalistDC[i])
				if err != nil {
					log.Error(err, "Error scaling the deploymentconfig")
					continue
				}
			}
		}
	} else {
		nsEvents.QuotaExceeded = namespace
	}

	if objectsToReconcile == 0 {
		log.Info("No objects to reconcile. Doing Nothing.")
		return nsEvents, finalState.Name, err
	}

	return nsEvents, finalState.Name, err
}

func ReconcileDeployment(ctx context.Context, _client client.Client, deployment v1.Deployment, state states.State) error {
	log := ctrl.Log.
		WithValues("deployment", deployment.Name).
		WithValues("namespace", deployment.Namespace)

	stateReplica, err := resources.DeploymentStateReplicas(state, deployment)
	if err != nil {
		log.Error(err, "Error getting the state replicas")
		return err
	}
	allowed, err := quotas.ResourceQuotaCheck(ctx, deployment.Namespace, resources.LimitsNeededDeployment(deployment, stateReplica.Replicas))
	if err != nil {
		log.Error(err, "Cannot calculate the resource quotas")
		return err
	}

	log = ctrl.Log.
		WithValues("Allowed", allowed)
	log.Info("Quota Check")

	if allowed {
		err = resources.ScaleDeployment(ctx, _client, deployment, stateReplica)
		if err != nil {
			log.Error(err, "Error scaling the deployment")
			return err
		}
	}

	return nil
}

func ReconcileDeploymentConfig(ctx context.Context, _client client.Client, deploymentConfig ocv1.DeploymentConfig, state states.State) error {
	log := ctrl.Log.
		WithValues("deploymentconfig", deploymentConfig.Name).
		WithValues("namespace", deploymentConfig.Namespace)

	stateReplica, err := resources.DeploymentConfigStateReplicas(state, deploymentConfig)
	if err != nil {
		log.Error(err, "Error getting the state replicas")
		return err
	}

	allowed, err := quotas.ResourceQuotaCheck(ctx, deploymentConfig.Namespace, resources.LimitsNeededDeploymentConfig(deploymentConfig, stateReplica.Replicas))
	if err != nil {
		log.Error(err, "Cannot calculate the resource quotas")
		return err
	}

	log = ctrl.Log.
		WithValues("Allowed", allowed)
	log.Info("Quota Check")

	if allowed {
		err = resources.ScaleDeploymentConfig(ctx, _client, deploymentConfig, stateReplica)
		if err != nil {
			log.Error(err, "Error scaling the deploymentconfig")
			return err
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
		err = errors.New("cannot continue as no states are set for namespace")
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
