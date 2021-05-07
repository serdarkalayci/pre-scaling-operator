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
						Spec: v1.DeploymentSpec{
							Replicas: new(int32),
						},
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
				Spec: v1.DeploymentSpec{
					Replicas: new(int32),
				},
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
						Spec: v1.DeploymentSpec{
							Replicas: new(int32),
						},
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
