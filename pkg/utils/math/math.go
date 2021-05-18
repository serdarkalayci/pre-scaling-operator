package math

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func Subtract(a corev1.ResourceList, b corev1.ResourceList) corev1.ResourceList {
	result := corev1.ResourceList{}
	for key, value := range a {
		quantity := value.DeepCopy()
		if other, found := b[key]; found {
			quantity.Sub(other)
		}
		result[key] = quantity
	}
	for key, value := range b {
		if _, found := result[key]; !found {
			quantity := value.DeepCopy()
			quantity.Neg()
			result[key] = quantity
		}
	}
	return result
}

func Mul(times int32, resources corev1.ResourceList) corev1.ResourceList {
	result := corev1.ResourceList{}
	for i := 0; int32(i) < times; i++ {
		result = Add(result, resources)
	}
	return result
}

func Add(a corev1.ResourceList, b corev1.ResourceList) corev1.ResourceList {
	result := corev1.ResourceList{}
	for key, value := range a {
		quantity := value.DeepCopy()
		if other, found := b[key]; found {
			quantity.Add(other)

		}
		result[key] = quantity
	}
	for key, value := range b {
		if _, found := result[key]; !found {
			quantity := value.DeepCopy()
			result[key] = quantity
		}
	}
	return result
}

func TranslateResourcesToQuotaResources(resources corev1.ResourceList) corev1.ResourceList {
	result := make(corev1.ResourceList)
	cpu, ok := resources[corev1.ResourceCPU]
	if ok {
		result[corev1.ResourceLimitsCPU] = cpu
	}
	mem, ok := resources[corev1.ResourceMemory]
	if ok {
		result[corev1.ResourceLimitsMemory] = mem
	}
	return result
}

func IsNegative(a corev1.ResourceList) []corev1.ResourceName {
	results := []corev1.ResourceName{}
	zero := resource.MustParse("0")
	for k, v := range a {
		if v.Cmp(zero) < 0 {
			results = append(results, k)
		}
	}
	return results
}

func IsZero(a corev1.ResourceList) bool {
	zero := resource.MustParse("0")
	for _, v := range a {
		if v.Cmp(zero) != 0 {
			return false
		}
	}
	return true
}

func ReplicaCalc(a, b int32) int32 {

	c := a - b

	if c == 0 {
		return a
	}

	return c
}

// Abs returns the absolute value of x.
func Abs(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}
