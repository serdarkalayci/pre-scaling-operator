package reconciler

import (
	"context"
	"reflect"
	"testing"

	scalingv1alpha1 "github.com/containersol/prescale-operator/api/v1alpha1"
	"github.com/containersol/prescale-operator/internal/states"
	g "github.com/containersol/prescale-operator/pkg/utils/global"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReconcileNamespace(t *testing.T) {
	type args struct {
		ctx              context.Context
		_client          client.Client
		namespace        string
		stateDefinitions states.States
		clusterState     states.State
	}

	_ = scalingv1alpha1.AddToScheme(scheme.Scheme)

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "StateNotEmpty",
			args: args{
				ctx: context.TODO(),
				_client: fake.
					NewClientBuilder().
					WithScheme(scheme.Scheme).
					Build(),
				namespace: "default",
				stateDefinitions: []states.State{
					{
						Name:     "bau",
						Priority: 0,
					},
					{
						Name:     "peak",
						Priority: 3,
					},
				},
				clusterState: states.State{
					Name:     "bau",
					Priority: 0,
				},
			},
			wantErr: false,
		},
		{
			name: "StateEmpty",
			args: args{
				ctx: context.TODO(),
				_client: fake.
					NewClientBuilder().
					WithScheme(scheme.Scheme).
					Build(),
				namespace: "default",
				stateDefinitions: []states.State{
					{
						Name:     "bau",
						Priority: 0,
					},
					{
						Name:     "peak",
						Priority: 3,
					},
				},
				clusterState: states.State{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := ReconcileNamespace(tt.args.ctx, tt.args._client, tt.args.namespace, tt.args.stateDefinitions, tt.args.clusterState, record.NewFakeRecorder(10)); (err != nil) != tt.wantErr {
				t.Errorf("ReconcileNamespace() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReconcileDeployment(t *testing.T) {
	type args struct {
		ctx            context.Context
		_client        client.Client
		deploymentItem g.ScalingInfo
		state          states.State
		optIn          bool
	}

	_ = scalingv1alpha1.AddToScheme(scheme.Scheme)

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "TestReconcileStateNotFound",
			args: args{
				ctx: context.TODO(),
				_client: fake.
					NewClientBuilder().
					WithScheme(scheme.Scheme).
					Build(),
				deploymentItem: g.ScalingInfo{
					Name:               "foo",
					Namespace:          "bar",
					Annotations:        map[string]string{},
					Labels:             map[string]string{"scaler/opt-in": "false"},
					SpecReplica:        2,
					IsDeploymentConfig: false,
					Failure:            false,
					FailureMessage:     "",
					ReadyReplicas:      2,
					DesiredReplicas:    4,
				},
				state: states.State{
					Name: "peak",
				},
				optIn: false,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ReconcileScalingItem(tt.args.ctx, tt.args._client, tt.args.deploymentItem, tt.args.state, false, record.NewFakeRecorder(10)); (err != nil) != tt.wantErr {
				t.Errorf("ReconcileDeployment() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetAppliedState(t *testing.T) {
	type args struct {
		ctx              context.Context
		_client          client.Client
		namespace        string
		stateDefinitions states.States
		clusterState     states.State
	}

	_ = scalingv1alpha1.AddToScheme(scheme.Scheme)

	tests := []struct {
		name    string
		args    args
		want    states.State
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
				stateDefinitions: []states.State{
					{
						Name:     "bau",
						Priority: 0,
					},
					{
						Name:     "peak",
						Priority: 3,
					},
				},
				clusterState: states.State{
					Name:     "bau",
					Priority: 0,
				},
			},
			want: states.State{
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
		stateDefinitions states.States
		namespace        string
	}

	_ = scalingv1alpha1.AddToScheme(scheme.Scheme)

	tests := []struct {
		name    string
		args    args
		want    states.State
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
				stateDefinitions: []states.State{},
				namespace:        "default",
			},
			want:    states.State{},
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
		stateDefinitions states.States
	}

	_ = scalingv1alpha1.AddToScheme(scheme.Scheme)

	tests := []struct {
		name    string
		args    args
		want    states.State
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
				stateDefinitions: []states.State{},
			},
			want:    states.State{},
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

func TestReconcileDeploymentConfig(t *testing.T) {
	type args struct {
		ctx            context.Context
		_client        client.Client
		deploymentItem g.ScalingInfo
		state          states.State
		optIn          bool
	}

	_ = scalingv1alpha1.AddToScheme(scheme.Scheme)

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "TestReconcileDeploymentConfigNoStateFound",
			args: args{
				ctx: context.TODO(),
				_client: fake.
					NewClientBuilder().
					WithScheme(scheme.Scheme).
					Build(),
				deploymentItem: g.ScalingInfo{
					Name:               "foo",
					Namespace:          "bar",
					Annotations:        map[string]string{},
					Labels:             map[string]string{"scaler/opt-in": "false"},
					SpecReplica:        2,
					IsDeploymentConfig: true,
					Failure:            false,
					FailureMessage:     "",
					ReadyReplicas:      2,
					DesiredReplicas:    4,
				},
				state: states.State{
					Name: "peak",
				},
				optIn: false,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ReconcileScalingItem(tt.args.ctx, tt.args._client, tt.args.deploymentItem, tt.args.state, false, record.NewFakeRecorder(10)); (err != nil) != tt.wantErr {
				t.Errorf("ReconcileDeploymentConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
