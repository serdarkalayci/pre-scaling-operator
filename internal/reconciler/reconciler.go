package reconciler

import (
	"context"
	"errors"
	sr "github.com/containersol/prescale-operator/internal"
	"github.com/containersol/prescale-operator/internal/states"
	v1 "k8s.io/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	OptInLabel = map[string]string{"scaler/opt-in": "true"}
)

func ReconcileNamespace(ctx context.Context, _client client.Client, namespace string, stateDefinitions states.States, clusterState states.State) error {
	log := ctrl.Log.
		WithValues("namespace", namespace)

	log.Info("Reconciling namespace")

	finalState, err := GetAppliedState(ctx, _client, namespace, stateDefinitions, clusterState)
	if err != nil {
		log.Error(err, "Cannot determine applied state for namespace")
		return err
	}

	// We now need to look for Deployments which are opted in,
	// then use their annotations to determine the correct scale
	deployments := v1.DeploymentList{}
	err = _client.List(ctx, &deployments, client.MatchingLabels(OptInLabel), client.InNamespace(namespace))
	if err != nil {
		log.Error(err, "Cannot list deployments in namespace")
		return err
	}

	if len(deployments.Items) == 0 {
		log.Info("No deployments to reconcile. Doing Nothing.")
		return nil
	}

	for _, deployment := range deployments.Items {
		err := ReconcileDeployment(ctx, _client, deployment, finalState)
		if err != nil {
			log.Error(err, "Could not reconcile deployment.")
			continue
		}
	}
	return nil
}

func ReconcileDeployment(ctx context.Context, _client client.Client, deployment v1.Deployment, state states.State) error {
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
	stateReplica, err := stateReplicas.GetState(state.Name)
	if err != nil {
		// TODO here we should do priority filtering, and go down one level of priority to find the lowest set one.
		// We will ignore any that are not set
		log.WithValues("set states", stateReplicas).
			WithValues("namespace state", state.Name).
			Info("State could not be found")
		return err
	}
	log.Info("Updating deployment replicas for state", "replicas", stateReplica.Replicas)
	deployment.Spec.Replicas = &stateReplica.Replicas
	err = _client.Update(ctx, &deployment, &client.UpdateOptions{})
	if err != nil {
		log.Error(err, "Could not scale deployment in namespace")
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
