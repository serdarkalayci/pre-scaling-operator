package resources

import (
	"context"
	"reflect"
	"testing"

	dc "github.com/openshift/api/apps/v1"
	v1 "github.com/openshift/api/apps/v1"
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
						Annotations: map[string]string{"scaler/type": "autoscale"},
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
