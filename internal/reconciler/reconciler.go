package reconciler

import (
	"context"
	"errors"

	c "github.com/containersol/prescale-operator/internal"
	"github.com/containersol/prescale-operator/internal/quotas"
	"github.com/containersol/prescale-operator/internal/resources"
	"github.com/containersol/prescale-operator/internal/states"

	ocv1 "github.com/openshift/api/apps/v1"
	v1 "k8s.io/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ReconcileNamespace(ctx context.Context, _client client.Client, namespace string, stateDefinitions states.States, clusterState states.State) error {

	var objectsToReconcile int

	log := ctrl.Log.
		WithValues("namespace", namespace)

	log.Info("Reconciling namespace")

	finalState, err := GetAppliedState(ctx, _client, namespace, stateDefinitions, clusterState)
	if err != nil {
		log.Error(err, "Cannot determine applied state for namespace")
		return err
	}

	// We now need to look for objects (currently supported deployments and deploymentConfigs) which are opted in,
	// then use their annotations to determine the correct scale
	deployments, err := resources.DeploymentLister(ctx, _client, namespace, c.OptInLabel)
	if err != nil {
		log.Error(err, "Cannot list deployments in namespace")
		return err
	}
	objectsToReconcile = objectsToReconcile + len(deployments.Items)

	scaleReplicalist, err := resources.DeploymentStateReplicasList(finalState, deployments)
	if err != nil {
		log.Error(err, "Cannot fetch replicas of all opted-in deployments")
		return err
	}

	allowed, err := quotas.ResourceQuotaCheckforNamespace(ctx, deployments, scaleReplicalist, namespace)
	if err != nil {
		log.Error(err, "Cannot calculate the resource quotas")
		return err
	}

	log = ctrl.Log.
		WithValues("Allowed", allowed)
	log.Info("Namespace Quota Check")

	if allowed {

		for i, deployment := range deployments.Items {
			log := ctrl.Log.
				WithValues("deployment", deployment.Name).
				WithValues("namespace", deployment.Namespace)

			err := resources.ScaleDeployment(ctx, _client, deployment, scaleReplicalist[i])
			if err != nil {
				log.Error(err, "Error scaling the deployment")
				continue
			}
		}
	}
	log.WithValues("env is", c.OpenshiftCluster).
		Info("Cluster")
	if c.OpenshiftCluster {
		deploymentConfigs, err := resources.DeploymentConfigLister(ctx, _client, namespace, c.OptInLabel)
		if err != nil {
			log.Error(err, "Cannot list deploymentConfigs in namespace")
			return err
		}
		objectsToReconcile = objectsToReconcile + len(deploymentConfigs.Items)

		scaleReplicalist, err := resources.DeploymentConfigStateReplicasList(finalState, deploymentConfigs)
		if err != nil {
			log.Error(err, "Cannot fetch replicas of all opted-in deploymentconfigs")
			return err
		}

		allowed, err := quotas.ResourceQuotaCheckforNamespaceDC(ctx, deploymentConfigs, scaleReplicalist, namespace)
		if err != nil {
			log.Error(err, "Cannot calculate the resource quotas")
			return err
		}

		log = ctrl.Log.
			WithValues("Allowed", allowed)
		log.Info("Namespace Quota Check")

		if allowed {

			for i, deploymentConfig := range deploymentConfigs.Items {
				log := ctrl.Log.
					WithValues("deploymentconfig", deploymentConfig.Name).
					WithValues("namespace", deploymentConfig.Namespace)

				err := resources.ScaleDeploymentConfig(ctx, _client, deploymentConfig, scaleReplicalist[i])
				if err != nil {
					log.Error(err, "Error scaling the deploymentconfig")
					continue
				}
			}
		}
	}

	if objectsToReconcile == 0 {
		log.Info("No objects to reconcile. Doing Nothing.")
		return nil
	}

	return nil
}

func ReconcileDeployment(ctx context.Context, _client client.Client, deployment v1.Deployment, state states.State, optIn bool) error {

	log := ctrl.Log.
		WithValues("deployment", deployment.Name).
		WithValues("namespace", deployment.Namespace)

	stateReplica, err := resources.DeploymentStateReplicas(state, deployment, optIn)
	if err != nil {
		log.Error(err, "Error getting the state replicas")
		return err
	}

	allowed, err := quotas.ResourceQuotaCheck(ctx, deployment, stateReplica.Replicas, deployment.Namespace)
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

func ReconcileDeploymentConfig(ctx context.Context, _client client.Client, deploymentConfig ocv1.DeploymentConfig, state states.State, optIn bool) error {
	log := ctrl.Log.
		WithValues("deploymentconfig", deploymentConfig.Name).
		WithValues("namespace", deploymentConfig.Namespace)

	stateReplica, err := resources.DeploymentConfigStateReplicas(state, deploymentConfig, optIn)
	if err != nil {
		log.Error(err, "Error getting the state replicas")
		return err
	}

	allowed, err := quotas.ResourceQuotaCheckDC(ctx, deploymentConfig, stateReplica.Replicas, deploymentConfig.Namespace)
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
