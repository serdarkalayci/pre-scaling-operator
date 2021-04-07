package resources

import (
	"context"
	"reflect"
	"testing"

	sr "github.com/containersol/prescale-operator/internal/state_replicas"
	"github.com/containersol/prescale-operator/internal/states"
	dc "github.com/openshift/api/apps/v1"
	v1 "github.com/openshift/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestDeploymentConfigLister(t *testing.T) {
	type args struct {
		ctx        context.Context
		_client    client.Client
		namespace  string
		OptInLabel map[string]string
	}

	_ = dc.AddToScheme(scheme.Scheme)

	tests := []struct {
		name    string
		args    args
		want    v1.DeploymentConfigList
		wantErr bool
	}{
		{
			name: "TestEmptyLister",
			args: args{
				ctx: context.TODO(),
				_client: fake.
					NewClientBuilder().
					WithScheme(scheme.Scheme).
					Build(),
				namespace:  "default",
				OptInLabel: map[string]string{},
			},
			want: v1.DeploymentConfigList{
				TypeMeta: metav1.TypeMeta{
					Kind:       "DeploymentConfigList",
					APIVersion: "apps.openshift.io/v1",
				},
				ListMeta: metav1.ListMeta{},
				Items:    []v1.DeploymentConfig{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DeploymentConfigLister(tt.args.ctx, tt.args._client, tt.args.namespace, tt.args.OptInLabel)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeploymentConfigLister() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeploymentConfigLister() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeploymentConfigGetter(t *testing.T) {
	type args struct {
		ctx     context.Context
		_client client.Client
		req     ctrl.Request
	}

	_ = dc.AddToScheme(scheme.Scheme)

	tests := []struct {
		name    string
		args    args
		want    v1.DeploymentConfig
		wantErr bool
	}{
		{
			name: "TestGetter",
			args: args{
				ctx: context.TODO(),
				_client: fake.
					NewClientBuilder().
					WithObjects(&v1.DeploymentConfig{
						TypeMeta: metav1.TypeMeta{
							Kind:       "DeploymentConfig",
							APIVersion: "apps.openshift.oi/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "foo",
						},
						Spec:   v1.DeploymentConfigSpec{},
						Status: v1.DeploymentConfigStatus{},
					}).
					WithScheme(scheme.Scheme).
					Build(),
				req: reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: "foo",
					},
				},
			},
			want: v1.DeploymentConfig{
				TypeMeta: metav1.TypeMeta{
					Kind:       "DeploymentConfig",
					APIVersion: "apps.openshift.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec:   v1.DeploymentConfigSpec{},
				Status: v1.DeploymentConfigStatus{},
			},
			wantErr: false,
		},
		{
			name: "TestEmptyGetter",
			args: args{
				ctx: context.TODO(),
				_client: fake.NewClientBuilder().
					WithObjects(&v1.DeploymentConfig{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "foo",
						},
						Spec:   v1.DeploymentConfigSpec{},
						Status: v1.DeploymentConfigStatus{},
					}).
					WithScheme(scheme.Scheme).
					Build(),
				req: reconcile.Request{},
			},
			want:    v1.DeploymentConfig{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DeploymentConfigGetter(tt.args.ctx, tt.args._client, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeploymentConfigGetter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeploymentConfigGetter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeploymentConfigScaler(t *testing.T) {
	type args struct {
		ctx              context.Context
		_client          client.Client
		deploymentConfig v1.DeploymentConfig
		replicas         int32
	}

	_ = dc.AddToScheme(scheme.Scheme)

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
					WithObjects(&v1.DeploymentConfig{
						TypeMeta: metav1.TypeMeta{
							Kind:       "DeploymentConfig",
							APIVersion: "apps.openshift.io/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "foo",
						},
						Spec:   v1.DeploymentConfigSpec{},
						Status: v1.DeploymentConfigStatus{},
					}).
					WithScheme(scheme.Scheme).
					Build(),
				deploymentConfig: v1.DeploymentConfig{
					TypeMeta: metav1.TypeMeta{
						Kind:       "DeploymentConfig",
						APIVersion: "apps.openshift.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "foo",
					},
					Spec:   v1.DeploymentConfigSpec{},
					Status: v1.DeploymentConfigStatus{},
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
					WithObjects(&v1.DeploymentConfig{
						TypeMeta: metav1.TypeMeta{
							Kind:       "DeploymentConfig",
							APIVersion: "apps.openshift.io/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "foo",
						},
						Spec:   v1.DeploymentConfigSpec{},
						Status: v1.DeploymentConfigStatus{},
					}).
					WithScheme(scheme.Scheme).
					Build(),
				deploymentConfig: v1.DeploymentConfig{
					TypeMeta: metav1.TypeMeta{
						Kind:       "DeploymentConfig",
						APIVersion: "apps.openshift.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "bar",
					},
					Spec:   v1.DeploymentConfigSpec{},
					Status: v1.DeploymentConfigStatus{},
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
					WithObjects(&v1.DeploymentConfig{
						TypeMeta: metav1.TypeMeta{
							Kind:       "DeploymentConfig",
							APIVersion: "apps.openshift.io/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "foo",
						},
						Spec:   v1.DeploymentConfigSpec{},
						Status: v1.DeploymentConfigStatus{},
					}).
					WithScheme(scheme.Scheme).
					Build(),
				deploymentConfig: v1.DeploymentConfig{
					TypeMeta: metav1.TypeMeta{
						Kind:       "DeploymentConfig",
						APIVersion: "apps.openshift.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:        "foo",
						Annotations: map[string]string{"scaler/allow-autoscaling": "true"},
					},
					Spec: v1.DeploymentConfigSpec{
						Replicas: 4,
					},
					Status: v1.DeploymentConfigStatus{},
				},
				replicas: 2,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DeploymentConfigScaler(tt.args.ctx, tt.args._client, tt.args.deploymentConfig, tt.args.replicas); (err != nil) != tt.wantErr {
				t.Errorf("DeploymentConfigScaler() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeploymentConfigOptinLabel(t *testing.T) {
	type args struct {
		deploymentConfig v1.DeploymentConfig
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
				deploymentConfig: v1.DeploymentConfig{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{},
					},
					Spec:   v1.DeploymentConfigSpec{},
					Status: v1.DeploymentConfigStatus{},
				},
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DeploymentConfigOptinLabel(tt.args.deploymentConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeploymentConfigOptinLabel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DeploymentConfigOptinLabel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeploymentConfigStateReplicas(t *testing.T) {
	type args struct {
		state            states.State
		deploymentConfig v1.DeploymentConfig
		optIn            bool
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
				deploymentConfig: v1.DeploymentConfig{
					TypeMeta: metav1.TypeMeta{
						Kind:       "DeploymentConfig",
						APIVersion: "apps.openshift.io/v1",
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
				deploymentConfig: v1.DeploymentConfig{
					TypeMeta: metav1.TypeMeta{
						Kind:       "DeploymentConfig",
						APIVersion: "apps.openshift.io/v1",
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
			got, err := DeploymentConfigStateReplicas(tt.args.state, tt.args.deploymentConfig, tt.args.optIn)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeploymentConfigStateReplicas() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeploymentConfigStateReplicas() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeploymentConfigStateReplicasList(t *testing.T) {
	type args struct {
		state             states.State
		deploymentConfigs v1.DeploymentConfigList
	}
	tests := []struct {
		name    string
		args    args
		want    []sr.StateReplica
		wantErr bool
	}{
		{
			name: "TestOptedOutDeploymentConfig",
			args: args{
				state: states.State{Name: "foo"},
				deploymentConfigs: v1.DeploymentConfigList{
					TypeMeta: metav1.TypeMeta{},
					ListMeta: metav1.ListMeta{},
					Items: []v1.DeploymentConfig{
						{
							TypeMeta: metav1.TypeMeta{
								Kind:       "DeploymentConfig",
								APIVersion: "apps.openshift.io/v1",
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
								Kind:       "DeploymentConfig",
								APIVersion: "apps.openshift.io/v1",
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
				deploymentConfigs: v1.DeploymentConfigList{
					TypeMeta: metav1.TypeMeta{},
					ListMeta: metav1.ListMeta{},
					Items: []v1.DeploymentConfig{
						{
							TypeMeta: metav1.TypeMeta{
								Kind:       "DeploymentConfig",
								APIVersion: "apps.openshift.io/v1",
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
								Kind:       "DeploymentConfig",
								APIVersion: "apps.openshift.io/v1",
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
			got, err := DeploymentConfigStateReplicasList(tt.args.state, tt.args.deploymentConfigs)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeploymentConfigStateReplicasList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeploymentConfigStateReplicasList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLimitsNeededDeploymentConfig(t *testing.T) {
	type args struct {
		deploymentConfig v1.DeploymentConfig
		replicas         int32
	}
	tests := []struct {
		name string
		args args
		want corev1.ResourceList
	}{
		{
			name: "TestLimitsNeeded",
			args: args{
				deploymentConfig: v1.DeploymentConfig{
					Spec: v1.DeploymentConfigSpec{
						Template: &corev1.PodTemplateSpec{
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
					Status: v1.DeploymentConfigStatus{},
				},
				replicas: 5,
			},
			want: map[corev1.ResourceName]resource.Quantity{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LimitsNeededDeploymentConfig(tt.args.deploymentConfig, tt.args.replicas); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LimitsNeededDeploymentConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLimitsNeededDeploymentConfigList(t *testing.T) {
	type args struct {
		deploymentConfigs v1.DeploymentConfigList
		scaleReplicalist  []sr.StateReplica
	}
	tests := []struct {
		name string
		args args
		want corev1.ResourceList
	}{
		{
			name: "TestLimitsNeededList",
			args: args{
				deploymentConfigs: v1.DeploymentConfigList{
					TypeMeta: metav1.TypeMeta{},
					ListMeta: metav1.ListMeta{},
					Items: []v1.DeploymentConfig{
						{
							Spec: v1.DeploymentConfigSpec{
								Template: &corev1.PodTemplateSpec{
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
							Status: v1.DeploymentConfigStatus{},
						},
						{
							Spec: v1.DeploymentConfigSpec{
								Template: &corev1.PodTemplateSpec{
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
							Status: v1.DeploymentConfigStatus{},
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
			if got := LimitsNeededDeploymentConfigList(tt.args.deploymentConfigs, tt.args.scaleReplicalist); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LimitsNeededDeploymentConfigList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScaleDeploymentConfig(t *testing.T) {
	type args struct {
		ctx              context.Context
		_client          client.Client
		deploymentConfig v1.DeploymentConfig
		stateReplica     sr.StateReplica
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "TestScalingDeploymentConfig",
			args: args{
				ctx:     context.TODO(),
				_client: fake.NewClientBuilder().Build(),
				deploymentConfig: v1.DeploymentConfig{
					TypeMeta: metav1.TypeMeta{
						Kind:       "DeploymentConfig",
						APIVersion: "apps.openshift.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
					},
					Spec: v1.DeploymentConfigSpec{
						Replicas: *new(int32),
					},
					Status: v1.DeploymentConfigStatus{
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
			if err := ScaleDeploymentConfig(tt.args.ctx, tt.args._client, tt.args.deploymentConfig, tt.args.stateReplica); (err != nil) != tt.wantErr {
				t.Errorf("ScaleDeploymentConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
