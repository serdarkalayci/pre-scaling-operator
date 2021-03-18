package reconciler

import (
	"context"
	"errors"

	c "github.com/containersol/prescale-operator/internal"
	"github.com/containersol/prescale-operator/internal/resources"
	sr "github.com/containersol/prescale-operator/internal/state_replicas"
	"github.com/containersol/prescale-operator/internal/states"
	"github.com/containersol/prescale-operator/pkg/utils/labels"

	// "github.com/containersol/prescale-operator/internal/validations"
	ocv1 "github.com/openshift/api/apps/v1"
	v1 "k8s.io/api/apps/v1"
	retry "k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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

	for _, deployment := range deployments.Items {
		optin := labels.GetLabelValue(deployment.GetLabels(), "scaler/opt-in")
		err = ReconcileDeployment(ctx, _client, deployment, finalState, optin)
		if err != nil {
			log.Error(err, "Could not reconcile deployment.")
			continue
		}
	}

	if err != nil {
		log.Error(err, "unable to identify cluster")
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

		for _, deploymentConfig := range deploymentConfigs.Items {

			err = ReconcileDeploymentConfig(ctx, _client, deploymentConfig, finalState, true)
			if err != nil {
				log.Error(err, "Could not reconcile deploymentConfig.")
				continue
			}
		}
	}

	if objectsToReconcile == 0 {
		log.Info("No objects to reconcile. Doing Nothing.")
		return nil
	}

	return nil
}

func ReconcileDeployment(ctx context.Context, _client client.Client, deployment v1.Deployment, state states.State, optinLabel bool) error {
	log := ctrl.Log.
		WithValues("deployment", deployment.Name).
		WithValues("namespace", deployment.Namespace)
	stateReplicas, err := sr.NewStateReplicasFromAnnotations(deployment.GetAnnotations())
	if err != nil {
		ctrl.Log.
			WithValues("deployment", deployment.Name).
			WithValues("namespace", deployment.Namespace).
			Error(err, "Cannot calculate state replicas. Please check deployment annotations. Continuing.")
		return err
	}
	// Now we have all the state settings, we can set the replicas for the deployment accordingly
	if !optinLabel {
		// the deployment opted out. We need to set back to default.
		log.Info("The deployment opted out. Will scale back to default")
		state.Name = c.DefaultReplicaAnnotation
	}
	stateReplica, err := stateReplicas.GetState(state.Name)
	if err != nil {
		// TODO here we should do priority filtering, and go down one level of priority to find the lowest set one.
		// We will ignore any that are not set
		log.WithValues("set states", stateReplicas).
			WithValues("namespace state", state.Name).
			Info("State could not be found")
		return err
	}
	var oldReplicaCount int32
	oldReplicaCount = *deployment.Spec.Replicas
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if oldReplicaCount == stateReplica.Replicas {
			log.Info("No Update on deployment. Desired replica count already matches current.")
			return nil
		}
		log.Info("Updating deploymentconfig replicas for state", "replicas", stateReplica.Replicas)
		updateErr := resources.DeploymentScaler(ctx, _client, deployment, stateReplica.Replicas)
		if updateErr == nil {
			log.WithValues("Deployment", deployment.Name).
				WithValues("StateReplica mode", stateReplica.Name).
				WithValues("Old Replica count", oldReplicaCount).
				WithValues("New Replica count", stateReplica.Replicas).
				Info("Deployment succesfully updated")
			return nil
		}
		log.Info("Updating deployment failed due to a conflict! Retrying..")
		// We need to get a newer version of the object from the client
		var req reconcile.Request
		req.NamespacedName.Namespace = deployment.Namespace
		req.NamespacedName.Name = deployment.Name
		deployment, err = resources.DeploymentGetter(ctx, _client, req)
		if err != nil {
			log.Error(err, "Error getting refreshed deployment in conflict resolution")
		}
		return updateErr

	})
	if retryErr != nil {
		log.Error(retryErr, "Unable to scale the deployment, err: %v")
	}
	return nil
}

func ReconcileDeploymentConfig(ctx context.Context, _client client.Client, deploymentConfig ocv1.DeploymentConfig, state states.State, optinLabel bool) error {
	log := ctrl.Log.
		WithValues("deploymentConfig", deploymentConfig.Name).
		WithValues("namespace", deploymentConfig.Namespace)
	stateReplicas, err := sr.NewStateReplicasFromAnnotations(deploymentConfig.GetAnnotations())
	if err != nil {
		ctrl.Log.
			WithValues("deploymentConfig", deploymentConfig.Name).
			WithValues("namespace", deploymentConfig.Namespace).
			Error(err, "Cannot calculate state replicas. Please check deploymentConfig annotations. Continuing.")
		return err
	}
	// Now we have all the state settings, we can set the replicas for the deploymentConfig accordingly
	if !optinLabel {
		// the deployment opted out. We need to set back to default.
		log.Info("The deploymentconfig opted out. Will scale back to default")
		state.Name = c.DefaultReplicaAnnotation
	}
	stateReplica, err := stateReplicas.GetState(state.Name)
	if err != nil {
		// TODO here we should do priority filtering, and go down one level of priority to find the lowest set one.
		// We will ignore any that are not set
		log.WithValues("set states", stateReplicas).
			WithValues("namespace state", state.Name).
			Info("State could not be found")
		return err
	}
	var oldReplicaCount int32
	oldReplicaCount = *&deploymentConfig.Spec.Replicas
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if oldReplicaCount == stateReplica.Replicas {
			log.Info("No Update on deployment. Desired replica count already matches current.")
			return nil
		}
		log.Info("Updating deploymentconfig replicas for state", "replicas", stateReplica.Replicas)
		updateErr := resources.DeploymentConfigScaler(ctx, _client, deploymentConfig, stateReplica.Replicas)
		if updateErr == nil {
			log.WithValues("Deploymentconfig", deploymentConfig.Name).
				WithValues("StateReplica mode", stateReplica.Name).
				WithValues("Old Replica count", oldReplicaCount).
				WithValues("New Replica count", stateReplica.Replicas).
				Info("Deploymentconfig succesfully updated")
			return nil
		}
		log.Info("Updating deployment failed due to a conflict! Retrying..")
		// We need to get a newer version of the object from the client
		var req reconcile.Request
		req.NamespacedName.Namespace = deploymentConfig.Namespace
		req.NamespacedName.Name = deploymentConfig.Name
		deploymentConfig, err = resources.DeploymentConfigGetter(ctx, _client, req)
		if err != nil {
			log.Error(err, "Error getting refreshed deployment in conflict resolution")
		}
		return updateErr

	})
	if retryErr != nil {
		log.Error(retryErr, "Unable to scale the deploymentconfig, err: %v")
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
