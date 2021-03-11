package state_replicas

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	annotations2 "github.com/containersol/prescale-operator/pkg/utils/annotations"
)

const StateReplicaAnnotationPrefix = "scaler/state-"

type StateReplica struct {
	Name     string
	Replicas int32
}

func (sr StateReplica) String() string {
	str := fmt.Sprintf("{name: %s, replicas: %v}", sr.Name, sr.Replicas)
	return str
}

type StateReplicas struct {
	states []StateReplica
}

func (sr StateReplicas) String() string {
	states := make([]string, 0)
	for _, state := range sr.states {
		states = append(states, state.String())
	}
	return fmt.Sprintf("[%s]", strings.Join(states, ", "))
}

func (sr *StateReplicas) Add(replica StateReplica) {
	sr.states = append(sr.states, replica)
}

func (sr *StateReplicas) GetStates() []StateReplica {
	return sr.states
}

func (sr *StateReplicas) GetState(name string) (StateReplica, error) {
	for _, state := range sr.states {
		if state.Name == name {
			return state, nil
		}
	}
	return StateReplica{}, errors.New("no state found")
}

func NewStateReplicasFromAnnotations(annotations map[string]string) (StateReplicas, error) {
	stateReplicas := StateReplicas{}
	states := annotations2.FilterByKeyPrefix(StateReplicaAnnotationPrefix, annotations)
	for key, value := range states {
		stateName := key[len(StateReplicaAnnotationPrefix) : len(key)-len("-replicas")]
		replicas, err := strconv.Atoi(value)
		if err != nil {
			return stateReplicas, errors.New("replica count in annotation is not a valid integer")
		}
		stateReplicas.Add(StateReplica{
			Name:     stateName,
			Replicas: int32(replicas),
		})
	}
	return stateReplicas, nil
}
