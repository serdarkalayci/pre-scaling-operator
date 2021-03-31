package quotas

import (
	"context"
	"errors"
	"strings"

	c "github.com/containersol/prescale-operator/internal"
	sr "github.com/containersol/prescale-operator/internal/state_replicas"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
)

func getConfig() clientcmd.ClientConfig {
	configLoadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		configLoadingRules,
		&clientcmd.ConfigOverrides{})
}

func isAllowedforNamespace(ctx context.Context, deployments v1.DeploymentList, scaleReplicalist []sr.StateReplica, rq *corev1.ResourceQuotaList) (bool, error) {

	var limitsneeded, leftovers corev1.ResourceList

	for i, deployment := range deployments.Items {

		limitsneeded = add(limitsneeded, mul(scaleReplicalist[i].Replicas, deployment.Spec.Template.Spec.Containers[0].Resources.Limits))
	}

	log := ctrl.Log.
		WithValues("Limits needed", limitsneeded)
	log.Info("Namespace Identified Resources")

	leftovers = subtract(rq.Items[0].Status.Hard, rq.Items[0].Status.Used)
	log = ctrl.Log.
		WithValues("Rq name", rq.Items[0].Name).
		WithValues("Leftover resources", leftovers)
	log.Info("Namespace Resource Quota")

	checklimits := subtract(leftovers, translateResourcesToQuotaResources(limitsneeded))

	log = ctrl.Log.
		WithValues("Limits", checklimits)
	log.Info("Namespace Final checks")

	if len(isNegative(checklimits)) != 0 {
		return false, nil
	}

	return true, nil
}

func isAllowed(ctx context.Context, deployment v1.Deployment, replicas int32, rq *corev1.ResourceQuotaList) (bool, error) {

	containers := deployment.Spec.Template.Spec.Containers

	var limitsneeded, leftovers corev1.ResourceList

	for _, c := range containers {

		limitsneeded = mul(replicas, c.Resources.Limits)

		log := ctrl.Log.
			WithValues("Name", c.Name).
			WithValues("Limits needed", limitsneeded)
		log.Info("Identified Resources")
	}

	for _, q := range rq.Items {
		leftovers = subtract(q.Status.Hard, q.Status.Used)
		log := ctrl.Log.
			WithValues("Rq name", q.Name).
			WithValues("Leftover resources", leftovers)
		log.Info("Resource Quota")
	}

	checklimits := subtract(leftovers, translateResourcesToQuotaResources(limitsneeded))

	log := ctrl.Log.
		WithValues("Limits", checklimits)
	log.Info("Final checks")

	if len(isNegative(checklimits)) != 0 {
		return false, nil
	}

	return true, nil
}

func ResourceQuotaCheckforNamespace(ctx context.Context, deployments v1.DeploymentList, scaleReplicalist []sr.StateReplica, namespace string) (bool, error) {
	var allowed bool
	rq, err := resourceQuota(ctx, namespace)
	if err != nil {
		if strings.Contains(err.Error(), c.RQNotFound) {
			ctrl.Log.Info("WARNING: No Resource Quotas found for this namespace")
			return true, nil
		}
		return false, err
	}

	allowed, err = isAllowedforNamespace(ctx, deployments, scaleReplicalist, rq)
	if err != nil {
		ctrl.Log.Error(err, "Cannot find namespace quotas")
		return false, err
	}

	return allowed, nil
}

func ResourceQuotaCheck(ctx context.Context, deployment v1.Deployment, replicas int32, namespace string) (bool, error) {

	var allowed bool
	rq, err := resourceQuota(ctx, namespace)
	if err != nil {
		if strings.Contains(err.Error(), c.RQNotFound) {
			ctrl.Log.Info("WARNING: No Resource Quotas found for this namespace")
			return true, nil
		}
		return false, err
	}

	allowed, err = isAllowed(ctx, deployment, replicas, rq)
	if err != nil {
		ctrl.Log.Error(err, "Cannot find namespace quotas")
		return false, err
	}

	return allowed, nil
}

func resourceQuota(ctx context.Context, namespace string) (*corev1.ResourceQuotaList, error) {

	rq := &corev1.ResourceQuotaList{}

	restConfig, err := getConfig().ClientConfig()
	if err != nil {
		return &corev1.ResourceQuotaList{}, err
	}

	kubernetesclient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return &corev1.ResourceQuotaList{}, err
	}

	rq, err = kubernetesclient.CoreV1().ResourceQuotas(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return &corev1.ResourceQuotaList{}, err
	}

	if len(rq.Items) == 0 {
		return &corev1.ResourceQuotaList{}, errors.New(c.RQNotFound)
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

func mul(times int32, resources corev1.ResourceList) corev1.ResourceList {
	result := corev1.ResourceList{}
	for i := 0; int32(i) < times; i++ {
		result = add(result, resources)
	}
	return result
}

func add(a corev1.ResourceList, b corev1.ResourceList) corev1.ResourceList {
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

func translateResourcesToQuotaResources(resources corev1.ResourceList) corev1.ResourceList {
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

func isNegative(a corev1.ResourceList) []corev1.ResourceName {
	results := []corev1.ResourceName{}
	zero := resource.MustParse("0")
	for k, v := range a {
		if v.Cmp(zero) < 0 {
			results = append(results, k)
		}
	}
	return results
}
