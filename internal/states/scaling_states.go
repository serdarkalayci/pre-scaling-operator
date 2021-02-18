package states

import (
	"context"
	"fmt"
	scalingv1alpha1 "github.com/containersol/prescale-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type State struct {
	Name     string
	Priority int32
}

func (s State) String() string {
	return fmt.Sprintf("{Name: %s, Priority: %v}", s.Name, s.Priority)
}

// GetPrioritisedState returns the state which has a higher priority
// If both state priorities are the same, return the second for consistency
func GetPrioritisedState(a State, b State) State {
	// It might be confusing that we return the "lower priority",
	// but keep in mind that priority 1 ranks higher thank priority 10
	if a == (State{}) {
		return b
	}
	if b == (State{}) {
		return a
	}
	if a.Priority < b.Priority {
		return a
	}
	return b
}

type States []State

func (s States) FindState(name string, _state *State) error {
	for _, state := range s {
		if state.Name == name {
			*_state = state
			return nil
		}
	}
	return NotFound{msg: "Could not find state"}
}

func (s States) FindPriorityState(a State, b State) State {
	return GetPrioritisedState(a, b)
}

func GetNamespaceScalingStateName(ctx context.Context, _client client.Client, namespace string) (string, error) {
	scalingStates := &scalingv1alpha1.ScalingStateList{}
	err := _client.List(ctx, scalingStates, &client.ListOptions{Namespace: namespace})
	if err != nil {
		return "", err
	}
	if len(scalingStates.Items) == 0 {
		return "", NotFound{}
	}
	if len(scalingStates.Items) > 1 {
		return "", TooMany{
			msg:   "Too many ScalingState objects found in namespace",
			count: len(scalingStates.Items),
		}
	}
	return scalingStates.Items[0].Spec.State, nil
}

func GetClusterScalingStateDefinitions(ctx context.Context, _client client.Client) (States, error) {
	// When a ScalingState is created or updated,
	// we need to check both it and the ClusterState in order to determine the actual state the namespace should be in.
	cssd := &scalingv1alpha1.ClusterScalingStateDefinitionList{}
	_client.List(ctx, cssd, &client.ListOptions{})

	if len(cssd.Items) == 0 {
		return States{}, NotFound{
			msg: "No cluster state definitions found",
		}
	}

	if len(cssd.Items) >= 2 {
		return States{}, TooMany{
			msg:   "Too many cluster states found",
			count: len(cssd.Items),
		}
	}

	clusterStateDefinitions := States{}
	for _, state := range cssd.Items[0].Spec {
		clusterStateDefinitions = append(clusterStateDefinitions, State{
			Name:     state.Name,
			Priority: state.Priority,
		})
	}
	return clusterStateDefinitions, nil
}

func GetClusterScalingState(ctx context.Context, _client client.Client) (string, error) {
	clusterScalingStates := &scalingv1alpha1.ClusterScalingStateList{}
	_client.List(ctx, clusterScalingStates, &client.ListOptions{})

	if len(clusterScalingStates.Items) == 0 {
		return "", NotFound{
			msg: "No ClusterScalingState objects found.",
		}
	}

	if len(clusterScalingStates.Items) > 1 {
		return "", TooMany{
			msg: "Too many ClusterScalingState objects found.",
		}
	}

	return clusterScalingStates.Items[0].Spec.State, nil
}

//func GetNamespaceState(ctx context.Context, client client.Client, namespace string) State {
//	// cssd here stand for ClusterScalingStateDefinitino
//	scalingState := &scalingv1alpha1.ScalingState{}
//	err := client.Get(ctx, req.NamespacedName, scalingState)
//	if err != nil {
//		if errors.IsNotFound(err) {
//			// Request object not found, could have been deleted after reconcile request.
//			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
//			// Return and don't requeue
//			log.Info("ScalingState resource not found. Ignoring since object must be deleted.")
//			return ctrl.Result{}, nil
//		}
//		// Error reading the object - requeue the request.
//		log.Error(err, "Failed to get ScalingState")
//		return ctrl.Result{}, err
//	}
//
//	// When a ScalingState is created or updated,
//	// we need to check both it and the ClusterState in order to determine the actual state the namespace should be in.
//	cssd := &scalingv1alpha1.ClusterScalingStateDefinitionList{}
//	r.List(ctx, cssd, &client.ListOptions{})
//
//	if len(cssd.Items) == 0 {
//		log.Info("No ClusterScalingStateDefinition Found. Doing Nothing.")
//		// TODO Should we add errors here to crash the controller and make it explicit that one should be set ?
//		return ctrl.Result{}, nil
//	}
//
//	if len(cssd.Items) >= 2 {
//		log.Info("More than 1 ClusterScalingStateDefinition found. Merging is not yet supported. Doing Nothing.")
//		return ctrl.Result{}, nil
//	}
//
//	clusterStateDefinitions := states.States{}
//	for _, state := range cssd.Items[0].Spec {
//		clusterStateDefinitions = append(clusterStateDefinitions, states.State{
//			Name:     state.Name,
//			Priority: state.Priority,
//		})
//	}
//
//	// We now have the definitions of which states are available to developers.
//	// @TODO implement priority overrides, once the priority is set for a clusterstatedefinition
//
//	// Next we need to fetch the ClusterScalingState to determine which states are currently set in a namespace
//	clusterScalingStates := &scalingv1alpha1.ClusterScalingStateList{}
//	r.List(ctx, clusterScalingStates, &client.ListOptions{})
//
//	if len(clusterScalingStates.Items) >= 2 {
//		log.Info("More than 1 ClusterScalingState found. Merging is not yet supported.")
//		return ctrl.Result{}, nil
//	}
//
//	if len(clusterScalingStates.Items) == 0 {
//		log.Info("No ClusterScalingStates found to compare. Using only ScalingState for calculations.")
//	}
//
//	selectedState := states.State{}
//	namespaceState, err := clusterStateDefinitions.FindState(scalingState.Spec.State)
//	if err != nil {
//		log.Error(err, "Could not determine state from namespace state name", "state", scalingState.Spec.State)
//	}
//	if len(clusterScalingStates.Items) == 1 {
//		clusterState, err := clusterStateDefinitions.FindState(clusterScalingStates.Items[0].Spec.State)
//		if err != nil {
//			log.Error(err, "Could not determine state from cluster state name", "state", clusterScalingStates.Items[0].Spec.State)
//		}
//		selectedState = states.GetPrioritisedState(namespaceState, clusterState)
//	}
//}
