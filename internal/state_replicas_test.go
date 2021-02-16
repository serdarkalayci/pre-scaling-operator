package state_replicas_test

import (
	state_replicas "github.com/containersol/prescale-operator/internal"
	"reflect"
	"testing"
)

func TestNewStateReplicasFromAnnotationsCalculatesCorrectly(t *testing.T) {
	annotations := map[string]string{
		"scaler/state-peak-replicas":    "5",
		"scaler/state-bau-replicas":     "2",
	}
	got, err := state_replicas.NewStateReplicasFromAnnotations(annotations)
	if err != nil {
		t.Errorf("Failed to process state replicas")
	}
	expectedOneWay := []state_replicas.StateReplica{
		{Name: "peak", Replicas: 5},
		{Name: "bau", Replicas: 2},
	}
	expectedTheOtherWay := []state_replicas.StateReplica{
		{Name: "bau", Replicas: 2},
		{Name: "peak", Replicas: 5},
	}
	if !reflect.DeepEqual(got.GetStates(), expectedOneWay) && !reflect.DeepEqual(got.GetStates(), expectedTheOtherWay) {
		t.Errorf("Could not calculate stsate replicas. Expected %s, Got %s", expectedOneWay, got.GetStates())
	}
}

func TestNewStateReplicasFromAnnotationsCanReturnNoStates(t *testing.T) {
	annotations := map[string]string{
		"no-match":                      "5",
		"state-bau-replicas":            "2",
	}
	got, err := state_replicas.NewStateReplicasFromAnnotations(annotations)
	if err != nil {
		t.Errorf("Failed to process state replicas")
	}
	expected := state_replicas.StateReplicas{}
	if !reflect.DeepEqual(expected.GetStates(), got.GetStates()) {
		t.Errorf("Could not calculate state replicas. Expected %s, Got %s", expected, got)
	}
}

func TestNewStateReplicasFromAnnotationsFailsIfReplicasNotInt(t *testing.T) {
	annotations := map[string]string{
		"scaler/state-peak-replicas": "foo",
	}
	_, err := state_replicas.NewStateReplicasFromAnnotations(annotations)
	if err == nil {
		t.Errorf("NewStateReplicasFromAnnotations expected to fail but passed")
	}
}

func TestStateReplicas_GetStateReturnsCorrectState(t *testing.T) {
	stateReplicas := state_replicas.StateReplicas{}
	stateReplicas.Add(state_replicas.StateReplica{Name: "peak", Replicas: 5})
	stateReplicas.Add(state_replicas.StateReplica{Name: "bau", Replicas: 2})
	stateReplicas.Add(state_replicas.StateReplica{Name: "default", Replicas: 1})

	got, err := stateReplicas.GetState("peak")
	if err != nil {
		t.Errorf("Failed to fetch state")
	}
	if !reflect.DeepEqual(state_replicas.StateReplica{Name: "peak", Replicas: 5}, got) {
		t.Errorf("Incorrect state returned. Expected %s, Got %s", state_replicas.StateReplica{Name: "peak", Replicas: 5}, got)
	}
}

func TestStateReplicas_GetStateReturnErrorIfNotStatesMatch(t *testing.T) {
	stateReplicas := state_replicas.StateReplicas{}
	stateReplicas.Add(state_replicas.StateReplica{Name: "peak", Replicas: 5})
	stateReplicas.Add(state_replicas.StateReplica{Name: "bau", Replicas: 2})
	stateReplicas.Add(state_replicas.StateReplica{Name: "default", Replicas: 1})

	_, err := stateReplicas.GetState("foobar")
	if err == nil {
		t.Errorf("Expected GetState to fail, but it passed")
	}
}
