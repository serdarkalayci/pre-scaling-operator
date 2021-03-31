package quotas

import (
	"context"
	"errors"
	"strings"

	c "github.com/containersol/prescale-operator/internal"
	"github.com/containersol/prescale-operator/internal/resources"
	sr "github.com/containersol/prescale-operator/internal/state_replicas"
	"github.com/containersol/prescale-operator/pkg/utils/math"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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

	allowed, err = isAllowed(rq, resources.LimitsNeededDeploymentList(deployments, scaleReplicalist))
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

	allowed, err = isAllowed(rq, resources.LimitsNeededDeployment(deployment, replicas))
	if err != nil {
		ctrl.Log.Error(err, "Cannot find namespace quotas")
		return false, err
	}

	return allowed, nil
}

func isAllowed(rq *corev1.ResourceQuotaList, limitsneeded corev1.ResourceList) (bool, error) {

	var leftovers corev1.ResourceList
	log := ctrl.Log.
		WithValues("Limits needed", limitsneeded)
	log.Info("Identified Resources")

	leftovers = math.Subtract(rq.Items[0].Status.Hard, rq.Items[0].Status.Used)
	log = ctrl.Log.
		WithValues("Rq name", rq.Items[0].Name).
		WithValues("Leftover resources", leftovers)
	log.Info("Resource Quota")

	checklimits := math.Subtract(leftovers, math.TranslateResourcesToQuotaResources(limitsneeded))

	log = ctrl.Log.
		WithValues("Limits", checklimits)
	log.Info("Final checks")

	if len(math.IsNegative(checklimits)) != 0 {
		return false, nil
	}

	return true, nil
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
