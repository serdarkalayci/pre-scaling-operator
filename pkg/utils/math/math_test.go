package math

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestMul(t *testing.T) {
	type args struct {
		times     int32
		resources corev1.ResourceList
	}
	tests := []struct {
		name string
		args args
		want corev1.ResourceList
	}{
		{
			name: "TestingMultiplication",
			args: args{
				times:     4,
				resources: map[corev1.ResourceName]resource.Quantity{},
			},
			want: map[corev1.ResourceName]resource.Quantity{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Mul(tt.args.times, tt.args.resources); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Mul() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTranslateResourcesToQuotaResources(t *testing.T) {
	type args struct {
		resources corev1.ResourceList
	}

	cpu := resource.NewQuantity(500, resource.DecimalSI)
	mem := resource.NewQuantity(500, resource.BinarySI)

	tests := []struct {
		name string
		args args
		want corev1.ResourceList
	}{
		{
			name: "TestTranslationOfResourceList",
			args: args{
				resources: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU:    *cpu,
					corev1.ResourceMemory: *mem,
				},
			},
			want: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceLimitsCPU:    *cpu,
				corev1.ResourceLimitsMemory: *mem,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TranslateResourcesToQuotaResources(tt.args.resources); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TranslateResourcesToQuotaResources() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNegative(t *testing.T) {
	type args struct {
		a corev1.ResourceList
	}

	cpu := resource.NewQuantity(-500, resource.DecimalSI)
	mem := resource.NewQuantity(500, resource.BinarySI)

	tests := []struct {
		name string
		args args
		want []corev1.ResourceName
	}{
		{
			name: "TestNegativeCPU",
			args: args{
				a: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU:    *cpu,
					corev1.ResourceMemory: *mem,
				},
			},
			want: []corev1.ResourceName{"cpu"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNegative(tt.args.a); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("IsNegative() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSubtract(t *testing.T) {
	type args struct {
		a corev1.ResourceList
		b corev1.ResourceList
	}

	CPU1 := resource.NewQuantity(500, resource.DecimalSI)

	CPU2 := resource.NewQuantity(200, resource.DecimalSI)
	MEM2 := resource.NewQuantity(200, resource.BinarySI)
	negativeMEM2 := resource.NewQuantity(-200, resource.BinarySI)

	CPU1plusCPU2 := resource.NewQuantity(300, resource.DecimalSI)

	tests := []struct {
		name string
		args args
		want corev1.ResourceList
	}{
		{
			name: "TestingSubstractAndNegative",
			args: args{
				a: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU: *CPU1,
				},
				b: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU:    *CPU2,
					corev1.ResourceMemory: *MEM2,
				},
			},
			want: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceCPU:    *CPU1plusCPU2,
				corev1.ResourceMemory: *negativeMEM2,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Subtract(tt.args.a, tt.args.b); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Subtract() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdd(t *testing.T) {
	type args struct {
		a corev1.ResourceList
		b corev1.ResourceList
	}

	CPU1 := resource.NewQuantity(500, resource.DecimalSI)

	CPU2 := resource.NewQuantity(200, resource.DecimalSI)
	MEM2 := resource.NewQuantity(200, resource.BinarySI)

	CPU3 := resource.NewQuantity(700, resource.DecimalSI)

	tests := []struct {
		name string
		args args
		want corev1.ResourceList
	}{
		{
			name: "",
			args: args{
				a: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU: *CPU1,
				},
				b: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU:    *CPU2,
					corev1.ResourceMemory: *MEM2,
				},
			},
			want: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceCPU:    *CPU3,
				corev1.ResourceMemory: *MEM2,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Add(tt.args.a, tt.args.b); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Add() = %v, want %v", got, tt.want)
			}
		})
	}
}
