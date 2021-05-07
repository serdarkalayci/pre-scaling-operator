package resources

import (
	"context"
	"reflect"
	"testing"

	v1 "github.com/openshift/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestDeploymentConfigGetter(t *testing.T) {
	type args struct {
		ctx     context.Context
		_client client.Client
		req     ctrl.Request
	}

	_ = v1.AddToScheme(scheme.Scheme)

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
