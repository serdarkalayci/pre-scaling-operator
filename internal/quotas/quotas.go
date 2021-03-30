package quotas

import (
	"context"

	sr "github.com/containersol/prescale-operator/internal/state_replicas"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Quotas struct {
	RequestCPU    string
	LimitCPU      string
	RequestMemory string
	LimitMemory   string
}

func getConfig() clientcmd.ClientConfig {
	configLoadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		configLoadingRules,
		&clientcmd.ConfigOverrides{})
}

func IsAllowedforNamespace(ctx context.Context, deployments v1.DeploymentList, scaleReplicalist []sr.StateReplica) (bool, error) {

	var limitsneeded, leftovers corev1.ResourceList

	for i, deployment := range deployments.Items {

		limitsneeded = Add(limitsneeded, Mul(scaleReplicalist[i].Replicas, deployment.Spec.Template.Spec.Containers[0].Resources.Limits))
	}

	log := ctrl.Log.
		WithValues("Limits needed", limitsneeded)
	log.Info("Namespace Identified Resources")

	rq, err := resourceQuota(ctx)
	if err != nil {
		return false, err
	}

	leftovers = subtract(rq.Items[0].Status.Hard, rq.Items[0].Status.Used)
	log = ctrl.Log.
		WithValues("Rq name", rq.Items[0].Name).
		WithValues("Leftover resources", leftovers)
	log.Info("Namespace Resource Quota")

	checklimits := subtract(leftovers, TranslateResourcesToQuotaResources(limitsneeded))

	log = ctrl.Log.
		WithValues("Limits", checklimits)
	log.Info("Namespace Final checks")

	if len(IsNegative(checklimits)) != 0 {
		return false, nil
	}

	return true, nil
}

func IsAllowed(ctx context.Context, deployment v1.Deployment, replicas int32) (bool, error) {

	containers := deployment.Spec.Template.Spec.Containers

	var limitsneeded, leftovers corev1.ResourceList

	for _, c := range containers {

		limitsneeded = Mul(replicas, c.Resources.Limits)

		log := ctrl.Log.
			WithValues("Name", c.Name).
			WithValues("Limits needed", limitsneeded)
		log.Info("Identified Resources")
	}

	rq, err := resourceQuota(ctx)
	if err != nil {
		return false, err
	}

	for _, q := range rq.Items {
		leftovers = subtract(q.Status.Hard, q.Status.Used)
		log := ctrl.Log.
			WithValues("Rq name", q.Name).
			WithValues("Leftover resources", leftovers)
		log.Info("Resource Quota")
	}

	checklimits := subtract(leftovers, TranslateResourcesToQuotaResources(limitsneeded))

	log := ctrl.Log.
		WithValues("Limits", checklimits)
	log.Info("Final checks")

	if len(IsNegative(checklimits)) != 0 {
		return false, nil
	}

	return true, nil
}

func resourceQuota(ctx context.Context) (*corev1.ResourceQuotaList, error) {

	rq := &corev1.ResourceQuotaList{}

	restConfig, err := getConfig().ClientConfig()
	if err != nil {
		return &corev1.ResourceQuotaList{}, err
	}

	kubernetesclient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return &corev1.ResourceQuotaList{}, err
	}

	rq, err = kubernetesclient.CoreV1().ResourceQuotas("default").List(ctx, metav1.ListOptions{})
	if err != nil {
		return &corev1.ResourceQuotaList{}, err
	}

	return rq, nil
}

func subtract(a corev1.ResourceList, b corev1.ResourceList) corev1.ResourceList {
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

// Add returns the result of a + b for each named resource
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

// IsNegative returns the set of resource names that have a negative value.
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
