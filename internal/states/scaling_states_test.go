package states

import (
	"context"
	"reflect"
	"testing"

	scalingv1alpha1 "github.com/containersol/prescale-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
	got, _ := GetNamespaceScalingStateName(context.TODO(), client, "product")
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
	_, err := GetNamespaceScalingStateName(context.TODO(), client, "product")
	if _, ok := err.(NotFound); !ok {
		t.Errorf("Received incorrect error. Expected NotFound, got %s", err)
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
	_, err := GetNamespaceScalingStateName(context.TODO(), client, "product")
	if _, ok := err.(TooMany); !ok {
		t.Errorf("Received incorrect error. Expected TooMany, got %s", err)
	}
}

func TestGetAppliedState(t *testing.T) {
	type args struct {
		ctx              context.Context
		_client          client.Client
		namespace        string
		stateDefinitions States
		clusterState     State
	}

	_ = scalingv1alpha1.AddToScheme(scheme.Scheme)

	tests := []struct {
		name    string
		args    args
		want    State
		wantErr bool
	}{
		{
			name: "TestEmptyClusterState",
			args: args{
				ctx: context.TODO(),
				_client: fake.
					NewClientBuilder().
					WithObjects(&scalingv1alpha1.ClusterScalingState{
						TypeMeta: metav1.TypeMeta{
							Kind:       "ScalingState",
							APIVersion: "scaling.prescale.com/v1alpha1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "product-scaling-state",
							Namespace: "product",
						},
						Spec: scalingv1alpha1.ClusterScalingStateSpec{
							State: "peak",
						},
					}).
					WithScheme(scheme.Scheme).
					Build(),
				namespace: "default",
				stateDefinitions: []State{
					{
						Name:     "bau",
						Priority: 0,
					},
					{
						Name:     "peak",
						Priority: 3,
					},
				},
				clusterState: State{
					Name:     "bau",
					Priority: 0,
				},
			},
			want: State{
				Name:     "bau",
				Priority: 0,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetAppliedState(tt.args.ctx, tt.args._client, tt.args.namespace, tt.args.stateDefinitions, tt.args.clusterState)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAppliedState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAppliedState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_fetchNameSpaceState(t *testing.T) {
	type args struct {
		ctx              context.Context
		_client          client.Client
		stateDefinitions States
		namespace        string
	}

	_ = scalingv1alpha1.AddToScheme(scheme.Scheme)

	tests := []struct {
		name    string
		args    args
		want    State
		wantErr bool
	}{
		{
			name: "",
			args: args{
				ctx: context.TODO(),
				_client: fake.
					NewClientBuilder().
					WithScheme(scheme.Scheme).
					Build(),
				stateDefinitions: []State{},
				namespace:        "default",
			},
			want:    State{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fetchNameSpaceState(tt.args.ctx, tt.args._client, tt.args.stateDefinitions, tt.args.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("fetchNameSpaceState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fetchNameSpaceState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_fetchClusterState(t *testing.T) {
	type args struct {
		ctx              context.Context
		_client          client.Client
		stateDefinitions States
	}

	_ = scalingv1alpha1.AddToScheme(scheme.Scheme)

	tests := []struct {
		name    string
		args    args
		want    State
		wantErr bool
	}{
		{
			name: "TestClusterState",
			args: args{
				ctx: context.TODO(),
				_client: fake.
					NewClientBuilder().
					WithScheme(scheme.Scheme).
					Build(),
				stateDefinitions: []State{},
			},
			want:    State{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fetchClusterState(tt.args.ctx, tt.args._client, tt.args.stateDefinitions)
			if (err != nil) != tt.wantErr {
				t.Errorf("fetchClusterState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fetchClusterState() = %v, want %v", got, tt.want)
			}
		})
	}
}
