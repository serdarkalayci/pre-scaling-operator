package resources

import (
	"context"
	"reflect"
	"testing"

	sr "github.com/containersol/prescale-operator/internal/state_replicas"
	"github.com/containersol/prescale-operator/internal/states"
	g "github.com/containersol/prescale-operator/pkg/utils/global"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestLister(t *testing.T) {
	type args struct {
		ctx        context.Context
		_client    client.Client
		namespace  string
		OptInLabel map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    []g.ScalingInfo
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
			want:    []g.ScalingInfo{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ScalingItemNamespaceLister(tt.args.ctx, tt.args._client, tt.args.namespace, tt.args.OptInLabel)
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

func TestGetter(t *testing.T) {
	type args struct {
		ctx     context.Context
		_client client.Client
		want    g.ScalingInfo
	}
	tests := []struct {
		name    string
		args    args
		want    g.ScalingInfo
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
							Name:      "test",
							Namespace: "bar",
						},
						Spec: v1.DeploymentSpec{
							Replicas:                new(int32),
							ProgressDeadlineSeconds: new(int32),
						},
						Status: v1.DeploymentStatus{},
					}).
					Build(),
				want: g.ScalingInfo{
					Name:            "test",
					Namespace:       "bar",
					ScalingItemType: g.ScalingItemType{ItemTypeName: "Deployment"},
					SpecReplica:     0,
					ReadyReplicas:   0,
					DesiredReplicas: -1,
					ResourceList:    corev1.ResourceList{},
				},
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
						Spec: v1.DeploymentSpec{
							Replicas:                new(int32),
							ProgressDeadlineSeconds: new(int32),
						},
						Status: v1.DeploymentStatus{},
					}).
					Build(),
				want: g.ScalingInfo{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		// Only step scaler can put the item on the list which we don't test here. So we do it manually
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetRefreshedScalingItem(tt.args.ctx, tt.args._client, tt.args.want)
			if (err != nil) != tt.wantErr {
				t.Errorf("ScalingItemGetter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.args.want) {
				t.Errorf("ScalingItemGetter() = %v, want %v", got, tt.args.want)
			}
		})
	}
}

func TestScaler(t *testing.T) {
	type args struct {
		ctx         context.Context
		_client     client.Client
		scalingItem g.ScalingInfo
		deployment  v1.Deployment
		replicas    int32
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
							Name:      "foo",
							Namespace: "bar",
							Labels:    map[string]string{"scaler/opt-in": "true"},
						},
						Spec: v1.DeploymentSpec{
							Replicas:                new(int32),
							ProgressDeadlineSeconds: new(int32),
						},
						Status: v1.DeploymentStatus{},
					}).
					Build(),
				scalingItem: g.ScalingInfo{
					Name:        "foo",
					Namespace:   "bar",
					SpecReplica: 4,
					Labels:      map[string]string{"scaler/opt-in": "true"},
				},
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
						Replicas:                new(int32),
						ProgressDeadlineSeconds: new(int32),
					},
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
							Name:      "foo",
							Namespace: "bar",
							Labels:    map[string]string{"scaler/opt-in": "true"},
						},
						Spec: v1.DeploymentSpec{
							Replicas:                new(int32),
							ProgressDeadlineSeconds: new(int32),
						},
						Status: v1.DeploymentStatus{},
					}).
					Build(),
				scalingItem: g.ScalingInfo{
					Name:        "bar",
					Namespace:   "foo",
					SpecReplica: 4,
					Labels:      map[string]string{"scaler/opt-in": "true"},
				},
				deployment: v1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bar",
						Namespace: "foo",
						Labels:    map[string]string{"scaler/opt-in": "true"},
					},
					Spec: v1.DeploymentSpec{
						Replicas:                new(int32),
						ProgressDeadlineSeconds: new(int32),
					},
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
							Name:      "foo",
							Namespace: "bar",
							Labels:    map[string]string{"scaler/opt-in": "true"},
						},
						Spec: v1.DeploymentSpec{
							Replicas:                new(int32),
							ProgressDeadlineSeconds: new(int32),
						},
						Status: v1.DeploymentStatus{},
					}).
					Build(),
				scalingItem: g.ScalingInfo{
					Name:          "foo",
					Namespace:     "bar",
					Annotations:   map[string]string{"scaler/allow-autoscaling": "true"},
					SpecReplica:   0,
					ReadyReplicas: 5,
					Labels:        map[string]string{"scaler/opt-in": "true"},
				},
				deployment: v1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:        "foo",
						Namespace:   "bar",
						Annotations: map[string]string{"scaler/allow-autoscaling": "true"},
					},
					Spec: v1.DeploymentSpec{
						Replicas:                new(int32),
						ProgressDeadlineSeconds: new(int32),
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
			// Only step scaler can put the item on the list which we don't test here. So we do it manually
			g.GetDenyList().SetScalingItemOnList(tt.args.scalingItem, false, "", 0)

			if err := DoScaling(tt.args.ctx, tt.args._client, tt.args.scalingItem, tt.args.replicas); (err != nil) != tt.wantErr {
				t.Errorf("Scaler() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOptinLabel(t *testing.T) {
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
					Spec: v1.DeploymentSpec{
						Replicas:                new(int32),
						ProgressDeadlineSeconds: new(int32),
					},
					Status: v1.DeploymentStatus{},
				},
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := OptinLabel(g.ConvertDeploymentToItem(tt.args.deployment))
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

func TestStateReplicas(t *testing.T) {
	type args struct {
		state          states.State
		deploymentItem g.ScalingInfo
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
				deploymentItem: g.ScalingInfo{
					Name:        "foo",
					Namespace:   "bar",
					Annotations: map[string]string{},
				},
			},
			want:    sr.StateReplica{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StateReplicas(tt.args.state, tt.args.deploymentItem)
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

func TestStateReplicasList(t *testing.T) {
	type args struct {
		state           states.State
		deploymentItems []g.ScalingInfo
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
					Name: "default",
				},
				deploymentItems: []g.ScalingInfo{
					{
						Name:      "foo",
						Namespace: "bar",
						Annotations: map[string]string{
							"scaler/state-foo-replicas":     "2",
							"scaler/state-default-replicas": "1"},
						Labels:          map[string]string{"scaler/opt-in": "false"},
						SpecReplica:     1,
						ScalingItemType: g.ScalingItemType{ItemTypeName: "Deployment"},
						Failure:         false,
						FailureMessage:  "",
						ReadyReplicas:   1,
						DesiredReplicas: 2,
					},
					{
						Name:      "foo2",
						Namespace: "bar2",
						Annotations: map[string]string{
							"scaler/state-foo-replicas":     "5",
							"scaler/state-default-replicas": "3"},
						Labels:          map[string]string{"scaler/opt-in": "false"},
						SpecReplica:     1,
						ScalingItemType: g.ScalingItemType{ItemTypeName: "Deployment"},
						Failure:         false,
						FailureMessage:  "",
						ReadyReplicas:   1,
						DesiredReplicas: 2,
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
				deploymentItems: []g.ScalingInfo{
					{
						Name:            "foo",
						Namespace:       "bar",
						Labels:          map[string]string{"scaler/opt-in": "false"},
						SpecReplica:     1,
						ScalingItemType: g.ScalingItemType{ItemTypeName: "Deployment"},
						Failure:         false,
						FailureMessage:  "",
						ReadyReplicas:   1,
						DesiredReplicas: 2,
					},
					{
						Name:            "foo2",
						Namespace:       "bar2",
						Labels:          map[string]string{"scaler/opt-in": "false"},
						SpecReplica:     1,
						ScalingItemType: g.ScalingItemType{ItemTypeName: "Deployment"},
						Failure:         false,
						FailureMessage:  "",
						ReadyReplicas:   1,
						DesiredReplicas: 2,
					},
				},
			},
			want:    []sr.StateReplica{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DetermineDesiredReplicas(tt.args.state, tt.args.deploymentItems)
			if (err != nil) != tt.wantErr {
				t.Errorf("StateReplicasList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeploymentStateReplicasList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLimitsNeeded(t *testing.T) {
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
						Replicas:                new(int32),
						ProgressDeadlineSeconds: new(int32),
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{},
							Spec: corev1.PodSpec{Containers: []corev1.Container{
								{
									Resources: corev1.ResourceRequirements{
										Limits: map[corev1.ResourceName]resource.Quantity{},
									},
								},
							},
							},
						},
					},
					Status: v1.DeploymentStatus{
						Replicas: 3,
					},
				},
				replicas: 5,
			},
			want: map[corev1.ResourceName]resource.Quantity{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LimitsNeeded(g.ConvertDeploymentToItem(tt.args.deployment), tt.args.replicas); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LimitsNeededDeployment() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLimitsNeededList(t *testing.T) {
	type args struct {
		deploymentItems  []g.ScalingInfo
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
				deploymentItems: []g.ScalingInfo{
					{
						Name:         "foo",
						ResourceList: map[corev1.ResourceName]resource.Quantity{},
						SpecReplica:  3,
					},
					{
						Name:         "bar",
						ResourceList: map[corev1.ResourceName]resource.Quantity{},
						SpecReplica:  3,
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
			if got := LimitsNeededList(tt.args.deploymentItems, tt.args.scaleReplicalist); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LimitsNeededDeploymentList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNamespaceGrouping(t *testing.T) {
	type args struct {
		deploymentItems []g.ScalingInfo
	}
	tests := []struct {
		name string
		args args
		want map[string]int
	}{
		{
			name: "TestThreeNamespaces",
			args: args{
				deploymentItems: []g.ScalingInfo{
					{
						Name:         "foons2",
						Namespace:    "ns2",
						ResourceList: map[corev1.ResourceName]resource.Quantity{},
						SpecReplica:  3,
					},
					{
						Name:         "foons1",
						Namespace:    "ns1",
						ResourceList: map[corev1.ResourceName]resource.Quantity{},
						SpecReplica:  3,
					}, {
						Name:         "barns3",
						Namespace:    "ns3",
						ResourceList: map[corev1.ResourceName]resource.Quantity{},
						SpecReplica:  3,
					},
					{
						Name:         "barns1",
						Namespace:    "ns1",
						ResourceList: map[corev1.ResourceName]resource.Quantity{},
						SpecReplica:  3,
					}, {
						Name:         "bazns2",
						Namespace:    "ns2",
						ResourceList: map[corev1.ResourceName]resource.Quantity{},
						SpecReplica:  3,
					},
					{
						Name:         "barns2",
						Namespace:    "ns2",
						ResourceList: map[corev1.ResourceName]resource.Quantity{},
						SpecReplica:  3,
					},
					{
						Name:         "foons3",
						Namespace:    "ns3",
						ResourceList: map[corev1.ResourceName]resource.Quantity{},
						SpecReplica:  3,
					},
				},
			},
			want: map[string]int{
				"ns1": 2,
				"ns2": 3,
				"ns3": 2,
			},
		},
		{
			name: "TestOneNamespace",
			args: args{
				deploymentItems: []g.ScalingInfo{
					{
						Name:         "foons2",
						Namespace:    "ns2",
						ResourceList: map[corev1.ResourceName]resource.Quantity{},
						SpecReplica:  3,
					},
				},
			},
			want: map[string]int{
				"ns2": 1,
			},
		},
		{
			name: "TestNoItems",
			args: args{
				deploymentItems: []g.ScalingInfo{},
			},
			want: map[string]int{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GroupScalingItemByNamespace(tt.args.deploymentItems)
			for key := range got {
				expectedCount := tt.want[key]
				length := len(got[key])
				if length == 0 {
					t.Errorf("Couldn't find key %s", key)
				}

				if length != expectedCount {
					t.Errorf("Not correct count of objects in the map for key %s", key)
				}
			}

		})
	}
}

