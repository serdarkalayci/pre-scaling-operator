package quotas

import (
	"context"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_isAllowed(t *testing.T) {
	type args struct {
		rq           *corev1.ResourceQuotaList
		limitsneeded corev1.ResourceList
	}

	HardCPU := resource.NewQuantity(1000, resource.DecimalSI)

	UsedCPU := resource.NewQuantity(200, resource.DecimalSI)

	NeededCPU := resource.NewQuantity(100, resource.DecimalSI)

	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "TestingNoAvailableResources",
			args: args{
				rq: &corev1.ResourceQuotaList{
					TypeMeta: metav1.TypeMeta{},
					ListMeta: metav1.ListMeta{},
					Items: []corev1.ResourceQuota{
						{
							TypeMeta:   metav1.TypeMeta{},
							ObjectMeta: metav1.ObjectMeta{},
							Spec: corev1.ResourceQuotaSpec{
								Hard: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceLimitsCPU: *HardCPU,
								},
							},

							Status: corev1.ResourceQuotaStatus{
								Used: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceLimitsCPU: *UsedCPU,
								},
							},
						},
					},
				},
				limitsneeded: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU: *NeededCPU,
				},
			},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, got, err := isAllowed(tt.args.rq, tt.args.limitsneeded)
			if (err != nil) != tt.wantErr {
				t.Errorf("isAllowed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("isAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_resourceQuota(t *testing.T) {
	type args struct {
		ctx              context.Context
		namespace        string
		kubernetesclient kubernetes.Interface
	}

	tests := []struct {
		name    string
		args    args
		want    *corev1.ResourceQuotaList
		wantErr bool
	}{
		{
			name: "TestNoResourceQuotaPresent",
			args: args{
				ctx:              context.TODO(),
				namespace:        "default",
				kubernetesclient: fake.NewSimpleClientset(),
			},
			want:    &corev1.ResourceQuotaList{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resourceQuota(tt.args.ctx, tt.args.namespace, tt.args.kubernetesclient)
			if (err != nil) != tt.wantErr {
				t.Errorf("resourceQuota() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("resourceQuota() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResourceQuotaCheck(t *testing.T) {
	type args struct {
		ctx          context.Context
		namespace    string
		limitsneeded corev1.ResourceList
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "TestZeroDefinedObjectResources",
			args: args{
				ctx:          context.TODO(),
				namespace:    "default",
				limitsneeded: map[corev1.ResourceName]resource.Quantity{},
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, got, err := ResourceQuotaCheck(tt.args.ctx, tt.args.namespace, tt.args.limitsneeded)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResourceQuotaCheck() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResourceQuotaCheck() = %v, want %v", got, tt.want)
			}
		})
	}
}
