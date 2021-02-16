package states

import (
	"context"
	scalingv1alpha1 "github.com/containersol/prescale-operator/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestGetPrioritisedState(t *testing.T) {
	stateA := State{
		Name:     "peak",
		Priority: 1,
	}
	stateB := State{
		Name:     "bay",
		Priority: 5,
	}
	got := GetPrioritisedState(stateA, stateB)
	if got != stateA {
		t.Errorf("Priority state not being returned. Expected %s, Got %s", stateA, got)
	}
	got = GetPrioritisedState(stateB, stateA)
	if got != stateA {
		t.Errorf("Priority state not being returned. Expected %s, Got %s", stateA, got)
	}
}

func Test_getNamespaceScalingStateNameReturnsCorrectStateName(t *testing.T) {
	_ = scalingv1alpha1.AddToScheme(scheme.Scheme)
	client := fake.
		NewClientBuilder().
		WithObjects(&scalingv1alpha1.ScalingState{
			TypeMeta: v1.TypeMeta{
				Kind:       "ScalingState",
				APIVersion: "scaling.prescale.com/v1alpha1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name:      "product-scaling-state",
				Namespace: "product",
			},
			Spec: scalingv1alpha1.ScalingStateSpec{
				State: "peak",
			},
		}).
		WithScheme(scheme.Scheme).
		Build()
	got, _ := getNamespaceScalingStateName(context.TODO(), client, "product")
	if got != "peak" {
		t.Errorf("Did not return expected state name. Expected %s, Got %s", "peak", got)
	}
}

func Test_getNamespaceScalingStateNameReturnsCorrectErrorIfNoStatesExist(t *testing.T) {
	_ = scalingv1alpha1.AddToScheme(scheme.Scheme)
	client := fake.
		NewClientBuilder().
		WithScheme(scheme.Scheme).
		Build()
	_, err := getNamespaceScalingStateName(context.TODO(), client, "product")
	if _, ok := err.(stateNotFound); !ok {
		t.Errorf("Received incorrect error. Expected stateNotFound, got %s", err)
	}
}

func Test_getNamespaceScalingStateNameReturnsCorrectErrorIfTooManyStatesExist(t *testing.T) {
	_ = scalingv1alpha1.AddToScheme(scheme.Scheme)
	client := fake.
		NewClientBuilder().
		WithObjects(&scalingv1alpha1.ScalingState{
			TypeMeta: v1.TypeMeta{
				Kind:       "ScalingState",
				APIVersion: "scaling.prescale.com/v1alpha1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name:      "product-scaling-state",
				Namespace: "product",
			},
			Spec: scalingv1alpha1.ScalingStateSpec{
				State: "peak",
			},
		}, &scalingv1alpha1.ScalingState{
			TypeMeta: v1.TypeMeta{
				Kind:       "ScalingState",
				APIVersion: "scaling.prescale.com/v1alpha1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name:      "product-scaling-state-contender",
				Namespace: "product",
			},
			Spec: scalingv1alpha1.ScalingStateSpec{
				State: "bau",
			},
		}).
		WithScheme(scheme.Scheme).
		Build()
	_, err := getNamespaceScalingStateName(context.TODO(), client, "product")
	if _, ok := err.(tooManyStates); !ok {
		t.Errorf("Received incorrect error. Expected tooManyStates, got %s", err)
	}
}
