package states

import (
	"context"
	"fmt"
	"strconv"

	scalingv1alpha1 "github.com/containersol/prescale-operator/api/v1alpha1"
	"github.com/containersol/prescale-operator/pkg/utils/annotations"
	g "github.com/containersol/prescale-operator/pkg/utils/global"
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

func GetRapidScalingSetting(deploymentItem g.ScalingInfo) bool {

	scalingAnnotation := annotations.FilterByKeyPrefix("scaler/rapid-", deploymentItem.Annotations)

	if len(scalingAnnotation) == 0 {
		return false
	}

	rapidScaling, _ := strconv.ParseBool(scalingAnnotation["scaler/rapid-scaling"])

	return rapidScaling
}

func GetClusterScalingStateDefinitionsList(ctx context.Context, _client client.Client) (scalingv1alpha1.ClusterScalingStateDefinitionList, error) {
	cssd := &scalingv1alpha1.ClusterScalingStateDefinitionList{}
	_client.List(ctx, cssd, &client.ListOptions{})
	if len(cssd.Items) == 0 {
		return scalingv1alpha1.ClusterScalingStateDefinitionList{}, NotFound{
			msg: "No cluster state definitions found",
		}
	}

	if len(cssd.Items) >= 2 {
		return scalingv1alpha1.ClusterScalingStateDefinitionList{}, TooMany{
			msg:   "Too many cluster states found",
			count: len(cssd.Items),
		}
	}
	return *cssd, nil
}

func GetClusterScalingStates(ctx context.Context, _client client.Client) (States, error) {
	// When a ScalingState is created or updated,
	// we need to check both it and the ClusterState in order to determine the actual state the namespace should be in.
	cssd, err := GetClusterScalingStateDefinitionsList(ctx, _client)
	if err != nil {
		return States{}, err
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
