package state_replicas

import "errors"

const StateReplicaAnnotationPrefix = "scaler/state-"

type StateReplica struct {
	Name string
	Replicas int32
}

type StateReplicas struct {
	states []StateReplica
}

func (sr *StateReplicas) Add (replica StateReplica) {
	sr.states = append(sr.states, replica)
}

func (sr *StateReplicas) GetStates () []StateReplica {
	return sr.states
}

func (sr *StateReplicas) GetState (name string) (StateReplica, error) {
	for _, state := range sr.states {
		if state.Name == name {
			return state, nil
		}
	}
	return StateReplica{}, errors.New("No state found")
}
