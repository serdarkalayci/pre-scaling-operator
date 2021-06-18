package reconciler

import (
	"context"
	"testing"

	scalingv1alpha1 "github.com/containersol/prescale-operator/api/v1alpha1"
	"github.com/containersol/prescale-operator/internal/states"
	g "github.com/containersol/prescale-operator/pkg/utils/global"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// func TestReconcileNamespace(t *testing.T) {
// 	type args struct {
// 		ctx              context.Context
// 		_client          client.Client
// 		namespace        string
// 		deployment       v1.Deployment
// 		stateDefinitions states.States
// 		clusterState     states.State
// 		dryRun           bool
// 	}

// 	_ = scalingv1alpha1.AddToScheme(scheme.Scheme)

// 	tests := []struct {
// 		name    string
// 		args    args
// 		wantErr bool
// 	}{
// 		{
// 			name: "StateNotEmpty",
// 			args: args{
// 				ctx: context.TODO(),
// 				_client: fake.
// 					NewClientBuilder().
// 					WithScheme(scheme.Scheme).
// 					Build(),
// 				namespace: "default",
// 				deployment: v1.Deployment{
// 					Spec: v1.DeploymentSpec{
// 						Replicas:                new(int32),
// 						ProgressDeadlineSeconds: new(int32),
// 						Template: corev1.PodTemplateSpec{
// 							ObjectMeta: metav1.ObjectMeta{
// 								Namespace: "default",
// 								Name:      "foo",
// 								Annotations: map[string]string{
// 									"scaler/state-bau-replicas":  "2",
// 									"scaler/state-peak-replicas": "5"},
// 								Labels: map[string]string{"scaler/opt-in": "true"},
// 							},
// 							Spec: corev1.PodSpec{Containers: []corev1.Container{
// 								{
// 									Resources: corev1.ResourceRequirements{
// 										Limits: map[corev1.ResourceName]resource.Quantity{},
// 									},
// 								},
// 							},
// 							},
// 						},
// 					},
// 					Status: v1.DeploymentStatus{
// 						Replicas: 3,
// 					},
// 				},
// 				stateDefinitions: []states.State{
// 					{
// 						Name:     "bau",
// 						Priority: 0,
// 					},
// 					{
// 						Name:     "peak",
// 						Priority: 3,
// 					},
// 				},
// 				clusterState: states.State{
// 					Name:     "bau",
// 					Priority: 0,
// 				},
// 				dryRun: false,
// 			},
// 			wantErr: false,
// 		},
// 		{
// 			name: "StateEmpty",
// 			args: args{
// 				ctx: context.TODO(),
// 				_client: fake.
// 					NewClientBuilder().
// 					WithScheme(scheme.Scheme).
// 					Build(),
// 				namespace: "default",
// 				deployment: v1.Deployment{
// 					Spec: v1.DeploymentSpec{
// 						Replicas:                new(int32),
// 						ProgressDeadlineSeconds: new(int32),
// 						Template: corev1.PodTemplateSpec{
// 							ObjectMeta: metav1.ObjectMeta{
// 								Namespace: "default",
// 								Name:      "foo",
// 								Annotations: map[string]string{
// 									"scaler/state-bau-replicas":  "2",
// 									"scaler/state-peak-replicas": "5"},
// 								Labels: map[string]string{"scaler/opt-in": "true"},
// 							},
// 							Spec: corev1.PodSpec{Containers: []corev1.Container{
// 								{
// 									Resources: corev1.ResourceRequirements{
// 										Limits: map[corev1.ResourceName]resource.Quantity{},
// 									},
// 								},
// 							},
// 							},
// 						},
// 					},
// 					Status: v1.DeploymentStatus{
// 						Replicas: 3,
// 					},
// 				},
// 				stateDefinitions: []states.State{
// 					{
// 						Name:     "bau",
// 						Priority: 0,
// 					},
// 					{
// 						Name:     "peak",
// 						Priority: 3,
// 					},
// 				},
// 				clusterState: states.State{},
// 			},
// 			wantErr: false,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			// Objects to track in the fake client.

// 			tt.args._client.Create(tt.args.ctx, &tt.args.deployment, &client.CreateOptions{})

// 			if _, _, err := PrepareForNamespaceReconcile(tt.args.ctx, tt.args._client, tt.args.namespace, tt.args.stateDefinitions, tt.args.clusterState, record.NewFakeRecorder(10), tt.args.dryRun); (err != nil) != tt.wantErr {
// 				t.Errorf("ReconcileNamespace() error = %v, wantErr %v", err, tt.wantErr)
// 			}
// 		})
// 	}
// }

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
					Name:            "foo",
					Namespace:       "bar",
					Annotations:     map[string]string{},
					Labels:          map[string]string{"scaler/opt-in": "false"},
					SpecReplica:     2,
					ScalingItemType: g.ScalingItemType{ItemTypeName: "Deployment"},
					Failure:         false,
					FailureMessage:  "",
					ReadyReplicas:   2,
					DesiredReplicas: 4,
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
			if err := ReconcileScalingItem(tt.args.ctx, tt.args._client, tt.args.deploymentItem, tt.args.state, false, record.NewFakeRecorder(10), "UNIT TEST"); (err != nil) != tt.wantErr {
				t.Errorf("ReconcileDeployment() error = %v, wantErr %v", err, tt.wantErr)
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
					Name:            "foo",
					Namespace:       "bar",
					Annotations:     map[string]string{},
					Labels:          map[string]string{"scaler/opt-in": "false"},
					SpecReplica:     2,
					ScalingItemType: g.ScalingItemType{ItemTypeName: "Deployment"},
					Failure:         false,
					FailureMessage:  "",
					ReadyReplicas:   2,
					DesiredReplicas: 4,
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
			if err := ReconcileScalingItem(tt.args.ctx, tt.args._client, tt.args.deploymentItem, tt.args.state, false, record.NewFakeRecorder(10), "UNIT TEST"); (err != nil) != tt.wantErr {
				t.Errorf("ReconcileDeploymentConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
