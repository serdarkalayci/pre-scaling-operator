package resources

import (
	"context"
	"reflect"
	"testing"

	v1 "k8s.io/api/apps/v1"
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
