package reconciler

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/containersol/prescale-operator/api/v1alpha1"
	scalingv1alpha1 "github.com/containersol/prescale-operator/api/v1alpha1"
	constants "github.com/containersol/prescale-operator/internal"
	"github.com/containersol/prescale-operator/internal/states"
	g "github.com/containersol/prescale-operator/pkg/utils/global"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

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
			if err := ReconcileScalingItem(tt.args.ctx, tt.args._client, tt.args.deploymentItem, false, record.NewFakeRecorder(10), "UNIT TEST"); (err != nil) != tt.wantErr {
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
			if err := ReconcileScalingItem(tt.args.ctx, tt.args._client, tt.args.deploymentItem, false, record.NewFakeRecorder(10), "UNIT TEST"); (err != nil) != tt.wantErr {
				t.Errorf("ReconcileDeploymentConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPrepareNamespaces(t *testing.T) {
	type args struct {
		ctx              context.Context
		_client          client.Client
		namespace        string
		stateDefinitions states.States
		clusterState     states.State
		dryRun           bool
	}

	_ = scalingv1alpha1.AddToScheme(scheme.Scheme)
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "TestTwoDeploymentTwoNamespaces",
			args: args{
				ctx: context.TODO(),
				_client: fake.NewClientBuilder().
					WithScheme(scheme.Scheme).
					WithObjects(&v1.Deployment{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "ns1",
							Annotations: map[string]string{
								"scaler/state-bau-replicas":  "2",
								"scaler/state-peak-replicas": "5"},
							Labels: map[string]string{"scaler/opt-in": "true"},
						},
						Spec: v1.DeploymentSpec{
							Replicas:                new(int32),
							ProgressDeadlineSeconds: new(int32),
						},
						Status: v1.DeploymentStatus{},
					}, &v1.Deployment{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "ns2",
							Annotations: map[string]string{
								"scaler/state-bau-replicas":  "2",
								"scaler/state-peak-replicas": "5"},
							Labels: map[string]string{"scaler/opt-in": "true"},
						},
						Spec: v1.DeploymentSpec{
							Replicas:                new(int32),
							ProgressDeadlineSeconds: new(int32),
						},
						Status: v1.DeploymentStatus{},
					}, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: "ns1",
						}},
						&corev1.Namespace{
							ObjectMeta: metav1.ObjectMeta{
								Name: "ns2",
							},
						},
						&v1alpha1.ClusterScalingState{
							TypeMeta: metav1.TypeMeta{
								Kind:       "ClusterScalingState",
								APIVersion: "scaling.prescale.com/v1alpha1",
							},
							ObjectMeta: metav1.ObjectMeta{
								Name: "clusterscalingstate-sample-bau",
							},
							Spec: v1alpha1.ClusterScalingStateSpec{
								State: "bau",
							},
							Config: v1alpha1.ClusterScalingStateConfiguration{
								DryRun: false,
							},
						}, &v1alpha1.ScalingState{
							TypeMeta: metav1.TypeMeta{
								Kind:       "ScalingState",
								APIVersion: "scaling.prescale.com/v1alpha1",
							},
							ObjectMeta: metav1.ObjectMeta{
								Name:      "product-scaling-state",
								Namespace: "ns1",
							},
							Spec: scalingv1alpha1.ScalingStateSpec{
								State: "peak",
							},
						}).
					Build(),
				namespace: "",
				stateDefinitions: states.States{
					states.State{
						Name:     "peak",
						Priority: 1,
					}, {
						Name:     "bau",
						Priority: 5,
					},
				},
				clusterState: states.State{
					Name:     "bau",
					Priority: 5,
				},
				dryRun: false,
			},
			want: map[string]string{
				"ns1": "peak",
				"ns2": "",
			},
		},
		{
			name: "TestOneNameSpaceTwoDeployment",
			args: args{
				ctx: context.TODO(),
				_client: fake.NewClientBuilder().
					WithScheme(scheme.Scheme).
					WithObjects(&v1.Deployment{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test1",
							Namespace: "ns1",
							Annotations: map[string]string{
								"scaler/state-bau-replicas":  "2",
								"scaler/state-peak-replicas": "5"},
							Labels: map[string]string{"scaler/opt-in": "true"},
						},
						Spec: v1.DeploymentSpec{
							Replicas:                new(int32),
							ProgressDeadlineSeconds: new(int32),
						},
						Status: v1.DeploymentStatus{},
					}, &v1.Deployment{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test2",
							Namespace: "ns1",
							Annotations: map[string]string{
								"scaler/state-bau-replicas":  "2",
								"scaler/state-peak-replicas": "5"},
							Labels: map[string]string{"scaler/opt-in": "true"},
						},
						Spec: v1.DeploymentSpec{
							Replicas:                new(int32),
							ProgressDeadlineSeconds: new(int32),
						},
						Status: v1.DeploymentStatus{},
					}, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: "ns1",
						}},
						&v1alpha1.ClusterScalingState{
							TypeMeta: metav1.TypeMeta{
								Kind:       "ClusterScalingState",
								APIVersion: "scaling.prescale.com/v1alpha1",
							},
							ObjectMeta: metav1.ObjectMeta{
								Name: "clusterscalingstate-sample-bau",
							},
							Spec: v1alpha1.ClusterScalingStateSpec{
								State: "bau",
							},
							Config: v1alpha1.ClusterScalingStateConfiguration{
								DryRun: false,
							},
						}).
					Build(),
				namespace: "",
				stateDefinitions: states.States{
					states.State{
						Name:     "peak",
						Priority: 1,
					}, {
						Name:     "bau",
						Priority: 5,
					},
				},
				clusterState: states.State{
					Name:     "bau",
					Priority: 5,
				},
				dryRun: false,
			},
			want: map[string]string{
				"ns1": "",
			},
		},
		{
			name: "TestOneNameSpaceWithScalingStateOneDeployment",
			args: args{
				ctx: context.TODO(),
				_client: fake.NewClientBuilder().
					WithScheme(scheme.Scheme).
					WithObjects(&v1.Deployment{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test1",
							Namespace: "ns1",
							Annotations: map[string]string{
								"scaler/state-bau-replicas":  "2",
								"scaler/state-peak-replicas": "5"},
							Labels: map[string]string{"scaler/opt-in": "true"},
						},
						Spec: v1.DeploymentSpec{
							Replicas:                new(int32),
							ProgressDeadlineSeconds: new(int32),
						},
						Status: v1.DeploymentStatus{},
					}, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: "ns1",
						}},
						&v1alpha1.ClusterScalingState{
							TypeMeta: metav1.TypeMeta{
								Kind:       "ClusterScalingState",
								APIVersion: "scaling.prescale.com/v1alpha1",
							},
							ObjectMeta: metav1.ObjectMeta{
								Name: "clusterscalingstate-sample-bau",
							},
							Spec: v1alpha1.ClusterScalingStateSpec{
								State: "bau",
							},
							Config: v1alpha1.ClusterScalingStateConfiguration{
								DryRun: false,
							},
						}, &v1alpha1.ScalingState{
							TypeMeta: metav1.TypeMeta{
								Kind:       "ScalingState",
								APIVersion: "scaling.prescale.com/v1alpha1",
							},
							ObjectMeta: metav1.ObjectMeta{
								Name:      "product-scaling-state",
								Namespace: "ns1",
							},
							Spec: scalingv1alpha1.ScalingStateSpec{
								State: "peak",
							},
						}).
					Build(),
				namespace: "",
				stateDefinitions: states.States{
					states.State{
						Name:     "peak",
						Priority: 1,
					}, {
						Name:     "bau",
						Priority: 5,
					},
				},
				clusterState: states.State{
					Name:     "bau",
					Priority: 5,
				},
				dryRun: false,
			},
			want: map[string]string{
				"ns1": "peak",
			},
		},
		{
			name: "TestDryRun",
			args: args{
				ctx: context.TODO(),
				_client: fake.NewClientBuilder().
					WithScheme(scheme.Scheme).
					WithObjects(&v1.Deployment{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test1",
							Namespace: "ns1",
							Annotations: map[string]string{
								"scaler/state-bau-replicas":  "2",
								"scaler/state-peak-replicas": "5"},
							Labels: map[string]string{"scaler/opt-in": "true"},
						},
						Spec: v1.DeploymentSpec{
							Replicas:                new(int32),
							ProgressDeadlineSeconds: new(int32),
						},
						Status: v1.DeploymentStatus{},
					}, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: "ns1",
						}},
						&v1alpha1.ClusterScalingState{
							TypeMeta: metav1.TypeMeta{
								Kind:       "ClusterScalingState",
								APIVersion: "scaling.prescale.com/v1alpha1",
							},
							ObjectMeta: metav1.ObjectMeta{
								Name: "clusterscalingstate-sample-bau",
							},
							Spec: v1alpha1.ClusterScalingStateSpec{
								State: "bau",
							},
							Config: v1alpha1.ClusterScalingStateConfiguration{
								DryRun: false,
							},
						}).
					Build(),
				namespace: "",
				stateDefinitions: states.States{
					states.State{
						Name:     "peak",
						Priority: 1,
					}, {
						Name:     "bau",
						Priority: 5,
					},
				},
				clusterState: states.State{
					Name:     "bau",
					Priority: 5,
				},
				dryRun: true,
			},
			want: map[string]string{
				"ns1": "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(constants.EnvMaxConcurrentNamespaceReconciles, "2")
			nsInfoMap, _, err := PrepareForNamespaceReconcile(tt.args.ctx, tt.args._client, tt.args.namespace, tt.args.stateDefinitions, tt.args.clusterState, record.NewFakeRecorder(10), tt.args.dryRun)
			if err != nil {
				t.Errorf(fmt.Sprintf("Error during preparation/reconcile. %s", err))
			}

			if len(nsInfoMap) != len(tt.want) {
				t.Errorf(fmt.Sprintf("Returned infomap length not correct! Expected %d, got %d", len(tt.want), len(nsInfoMap)))
				t.Fail()
			}

			for key, value := range nsInfoMap {
				if value.AppliedState != tt.want[key] {
					t.Errorf(fmt.Sprintf("Applied state for ns %s is not correct. Expected %s. got %s", key, tt.want[key], value.AppliedState))
				}
				if value.Error != nil {
					t.Error(fmt.Sprintf("Got a NsScaleDecision error! Expected %s. Got %s ", "nil", value.Error))
				}

				if value.RetriggerMe == true {
					t.Error(fmt.Sprintf("Retrigger active! Expected %t. Got %t ", false, value.RetriggerMe))
				}

				if tt.args.dryRun && value.NSEvents.DryRunInfo == "" {
					t.Errorf("Dryrun info is empty although dryrun is active!")
				}

				if !tt.args.dryRun && value.NSEvents.DryRunInfo != "" {
					t.Errorf("Dryrun info is filled although dryrun shouldn't run!")
				}

			}

		})
	}
}
