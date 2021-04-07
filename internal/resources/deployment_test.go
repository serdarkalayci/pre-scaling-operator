package resources

import (
	"context"
	"reflect"
	"testing"

	sr "github.com/containersol/prescale-operator/internal/state_replicas"
	"github.com/containersol/prescale-operator/internal/states"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestDeploymentLister(t *testing.T) {
	type args struct {
		ctx        context.Context
		_client    client.Client
		namespace  string
		OptInLabel map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    v1.DeploymentList
		wantErr bool
	}{
		{
			name: "TestEmptyLister",
			args: args{
				ctx:        context.TODO(),
				_client:    fake.NewClientBuilder().Build(),
				namespace:  "default",
				OptInLabel: map[string]string{},
			},
			want: v1.DeploymentList{
				TypeMeta: metav1.TypeMeta{
					Kind:       "DeploymentList",
					APIVersion: "apps/v1",
				},
				ListMeta: metav1.ListMeta{},
				Items:    []v1.Deployment{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DeploymentLister(tt.args.ctx, tt.args._client, tt.args.namespace, tt.args.OptInLabel)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeploymentLister() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeploymentLister() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeploymentGetter(t *testing.T) {
	type args struct {
		ctx     context.Context
		_client client.Client
		req     ctrl.Request
	}
	tests := []struct {
		name    string
		args    args
		want    v1.Deployment
		wantErr bool
	}{
		{
			name: "TestGetter",
			args: args{
				ctx: context.TODO(),
				_client: fake.NewClientBuilder().
					WithObjects(&v1.Deployment{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "test",
						},
						Spec:   v1.DeploymentSpec{},
						Status: v1.DeploymentStatus{},
					}).
					Build(),
				req: reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: "test",
					},
				},
			},
			want: v1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec:   v1.DeploymentSpec{},
				Status: v1.DeploymentStatus{},
			},
			wantErr: false,
		},
		{
			name: "TestEmptyGetter",
			args: args{
				ctx: context.TODO(),
				_client: fake.NewClientBuilder().
					WithObjects(&v1.Deployment{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "test",
						},
						Spec:   v1.DeploymentSpec{},
						Status: v1.DeploymentStatus{},
					}).
					Build(),
				req: reconcile.Request{},
			},
			want:    v1.Deployment{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DeploymentGetter(tt.args.ctx, tt.args._client, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeploymentGetter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeploymentGetter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeploymentScaler(t *testing.T) {
	type args struct {
		ctx        context.Context
		_client    client.Client
		deployment v1.Deployment
		replicas   int32
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "TestScaler",
			args: args{
				ctx: context.TODO(),
				_client: fake.NewClientBuilder().
					WithObjects(&v1.Deployment{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "foo",
						},
						Spec:   v1.DeploymentSpec{},
						Status: v1.DeploymentStatus{},
					}).
					Build(),
				deployment: v1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "foo",
					},
					Spec:   v1.DeploymentSpec{},
					Status: v1.DeploymentStatus{},
				},
				replicas: 4,
			},
			wantErr: false,
		},
		{
			name: "TestWrongScaler",
			args: args{
				ctx: context.TODO(),
				_client: fake.NewClientBuilder().
					WithObjects(&v1.Deployment{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "foo",
						},
						Spec:   v1.DeploymentSpec{},
						Status: v1.DeploymentStatus{},
					}).
					Build(),
				deployment: v1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "bar",
					},
					Spec:   v1.DeploymentSpec{},
					Status: v1.DeploymentStatus{},
				},
				replicas: 4,
			},
			wantErr: true,
		},
		{
			name: "TestAutoscaler",
			args: args{
				ctx: context.TODO(),
				_client: fake.NewClientBuilder().
					WithObjects(&v1.Deployment{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "foo",
						},
						Spec:   v1.DeploymentSpec{},
						Status: v1.DeploymentStatus{},
					}).
					Build(),
				deployment: v1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:        "foo",
						Annotations: map[string]string{"scaler/allow-autoscaling": "true"},
					},
					Spec: v1.DeploymentSpec{
						Replicas: new(int32),
					},
					Status: v1.DeploymentStatus{
						Replicas: 5,
					},
				},
				replicas: 2,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DeploymentScaler(tt.args.ctx, tt.args._client, tt.args.deployment, tt.args.replicas); (err != nil) != tt.wantErr {
				t.Errorf("DeploymentScaler() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeploymentOptinLabel(t *testing.T) {
	type args struct {
		deployment v1.Deployment
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "TestEmptyLabel",
			args: args{
				deployment: v1.Deployment{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{},
					},
					Spec:   v1.DeploymentSpec{},
					Status: v1.DeploymentStatus{},
				},
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DeploymentOptinLabel(tt.args.deployment)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeploymentOptinLabel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DeploymentOptinLabel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeploymentStateReplicas(t *testing.T) {
	type args struct {
		state      states.State
		deployment v1.Deployment
		optIn      bool
	}
	tests := []struct {
		name    string
		args    args
		want    sr.StateReplica
		wantErr bool
	}{
		{
			name: "TestEmptyAnnotation",
			args: args{
				state: states.State{
					Name: "foo",
				},
				deployment: v1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
				},
				optIn: false,
			},
			want:    sr.StateReplica{},
			wantErr: true,
		},
		{
			name: "TestFooAnnotation",
			args: args{
				state: states.State{
					Name: "foo",
				},
				deployment: v1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "bar",
						Annotations: map[string]string{
							"scaler/state-foo-replicas":     "5",
							"scaler/state-default-replicas": "3",
						},
					},
				},
				optIn: false,
			},
			want: sr.StateReplica{
				Name:     "default",
				Replicas: 3,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DeploymentStateReplicas(tt.args.state, tt.args.deployment, tt.args.optIn)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeploymentStateReplicas() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeploymentStateReplicas() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeploymentStateReplicasList(t *testing.T) {
	type args struct {
		state       states.State
		deployments v1.DeploymentList
	}
	tests := []struct {
		name    string
		args    args
		want    []sr.StateReplica
		wantErr bool
	}{
		{
			name: "TestOptedOutDeployment",
			args: args{
				state: states.State{
					Name: "foo",
				},
				deployments: v1.DeploymentList{
					TypeMeta: metav1.TypeMeta{},
					ListMeta: metav1.ListMeta{},
					Items: []v1.Deployment{
						{
							TypeMeta: metav1.TypeMeta{
								Kind:       "Deployment",
								APIVersion: "apps/v1",
							},
							ObjectMeta: metav1.ObjectMeta{
								Name: "foo",
								Labels: map[string]string{
									"scaler/opt-in": "false",
								},
								Annotations: map[string]string{
									"scaler/state-foo-replicas":     "2",
									"scaler/state-default-replicas": "1",
								},
							},
						},
						{
							TypeMeta: metav1.TypeMeta{
								Kind:       "Deployment",
								APIVersion: "apps/v1",
							},
							ObjectMeta: metav1.ObjectMeta{
								Name: "bar",
								Labels: map[string]string{
									"scaler/opt-in": "false",
								},
								Annotations: map[string]string{
									"scaler/state-foo-replicas":     "5",
									"scaler/state-default-replicas": "3",
								},
							},
						},
					},
				},
			},
			want: []sr.StateReplica{
				{
					Name:     "default",
					Replicas: 1,
				},
				{
					Name:     "default",
					Replicas: 3,
				},
			},
			wantErr: false,
		},
		{
			name: "TestAbsentReplicaState",
			args: args{
				state: states.State{
					Name: "foo",
				},
				deployments: v1.DeploymentList{
					TypeMeta: metav1.TypeMeta{},
					ListMeta: metav1.ListMeta{},
					Items: []v1.Deployment{
						{
							TypeMeta: metav1.TypeMeta{
								Kind:       "Deployment",
								APIVersion: "apps/v1",
							},
							ObjectMeta: metav1.ObjectMeta{
								Name: "foo",
								Labels: map[string]string{
									"scaler/opt-in": "false",
								},
							},
						},
						{
							TypeMeta: metav1.TypeMeta{
								Kind:       "Deployment",
								APIVersion: "apps/v1",
							},
							ObjectMeta: metav1.ObjectMeta{
								Name: "bar",
								Labels: map[string]string{
									"scaler/opt-in": "false",
								},
							},
						},
					},
				},
			},
			want:    []sr.StateReplica{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DeploymentStateReplicasList(tt.args.state, tt.args.deployments)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeploymentStateReplicasList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeploymentStateReplicasList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLimitsNeededDeployment(t *testing.T) {
	type args struct {
		deployment v1.Deployment
		replicas   int32
	}
	tests := []struct {
		name string
		args args
		want corev1.ResourceList
	}{
		{
			name: "TestLimitsNeeded",
			args: args{
				deployment: v1.Deployment{
					Spec: v1.DeploymentSpec{
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{},
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Resources: corev1.ResourceRequirements{
											Limits: map[corev1.ResourceName]resource.Quantity{},
										},
									},
								},
							},
						},
					},
					Status: v1.DeploymentStatus{},
				},
				replicas: 5,
			},
			want: map[corev1.ResourceName]resource.Quantity{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LimitsNeededDeployment(tt.args.deployment, tt.args.replicas); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LimitsNeededDeployment() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLimitsNeededDeploymentList(t *testing.T) {
	type args struct {
		deployments      v1.DeploymentList
		scaleReplicalist []sr.StateReplica
	}
	tests := []struct {
		name string
		args args
		want corev1.ResourceList
	}{
		{
			name: "TestLimitsNeededList",
			args: args{
				deployments: v1.DeploymentList{
					TypeMeta: metav1.TypeMeta{},
					ListMeta: metav1.ListMeta{},
					Items: []v1.Deployment{
						{
							Spec: v1.DeploymentSpec{
								Template: corev1.PodTemplateSpec{
									ObjectMeta: metav1.ObjectMeta{},
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Resources: corev1.ResourceRequirements{
													Limits: map[corev1.ResourceName]resource.Quantity{},
												},
											},
										},
									},
								},
							},
							Status: v1.DeploymentStatus{},
						},
						{
							Spec: v1.DeploymentSpec{
								Template: corev1.PodTemplateSpec{
									ObjectMeta: metav1.ObjectMeta{},
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Resources: corev1.ResourceRequirements{
													Limits: map[corev1.ResourceName]resource.Quantity{},
												},
											},
										},
									},
								},
							},
							Status: v1.DeploymentStatus{},
						},
					},
				},
				scaleReplicalist: []sr.StateReplica{
					{
						Name:     "foo",
						Replicas: 3,
					},
					{
						Name:     "bar",
						Replicas: 4,
					},
				},
			},
			want: map[corev1.ResourceName]resource.Quantity{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LimitsNeededDeploymentList(tt.args.deployments, tt.args.scaleReplicalist); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LimitsNeededDeploymentList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScaleDeployment(t *testing.T) {
	type args struct {
		ctx          context.Context
		_client      client.Client
		deployment   v1.Deployment
		stateReplica sr.StateReplica
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "TestScalingDeployment",
			args: args{
				ctx:     context.TODO(),
				_client: fake.NewClientBuilder().Build(),
				deployment: v1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
					},
					Spec: v1.DeploymentSpec{
						Replicas: new(int32),
					},
					Status: v1.DeploymentStatus{
						Replicas: 5,
					},
				},
				stateReplica: sr.StateReplica{
					Name:     "test",
					Replicas: 7,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ScaleDeployment(tt.args.ctx, tt.args._client, tt.args.deployment, tt.args.stateReplica); (err != nil) != tt.wantErr {
				t.Errorf("ScaleDeployment() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
