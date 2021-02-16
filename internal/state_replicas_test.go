package state_replicas_test

import (
	state_replicas "github.com/containersol/prescale-operator/internal"
	"reflect"
	"testing"
)

func TestNewStateReplicasFromAnnotations(t *testing.T) {
	annotations := map[string]string{
		"scaler/state-peak-replicas":    "5",
		"scaler/state-bau-replicas":     "2",
		"scaler/state-default-replicas": "1",
	}
	got, err := state_replicas.NewStateReplicasFromAnnotations(annotations)
	if err != nil {
		t.Errorf("Failed to process state replicas")
	}
	expected := state_replicas.StateReplicas{}
	expected.Add(state_replicas.StateReplica{Name: "peak", Replicas: 5})
	expected.Add(state_replicas.StateReplica{Name: "bau", Replicas: 2})
	expected.Add(state_replicas.StateReplica{Name: "default", Replicas: 1})
	if !reflect.DeepEqual(expected.GetStates(), got.GetStates()) {
		t.Errorf("Could not calculate state replicas. Expected %s, Got %s", expected, got)
	}
}
