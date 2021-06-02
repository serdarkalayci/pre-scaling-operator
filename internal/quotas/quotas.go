package quotas

import (
	"context"
	"errors"
	"fmt"
	"strings"

	c "github.com/containersol/prescale-operator/internal"
	"github.com/containersol/prescale-operator/pkg/utils/client"
	"github.com/containersol/prescale-operator/pkg/utils/math"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

// +kubebuilder:rbac:groups="",resources=resourcequotas,verbs=list;

//ResourceQuotaCheck function first checks if there are resourceQuota objects present and then validates if the namespace has enough resources to scale
func ResourceQuotaCheck(ctx context.Context, namespace string, limitsneeded corev1.ResourceList) (bool, error) {

	var allowed bool

	kubernetesclient, err := client.GetClientSet()
	if err != nil {
		return false, err
	}

	rq, err := resourceQuota(ctx, namespace, kubernetesclient)
	if err != nil {
		if strings.Contains(err.Error(), c.RQNotFound) {
			ctrl.Log.Info(fmt.Sprintf("WARNING: No Resource Quotas found for this namespace: %s", namespace))
			return true, nil
		}
		return false, err
	}

	if len(math.IsNegative(limitsneeded)) > 0 {
		return true, nil
	}

	if math.IsZero(limitsneeded) {
		ctrl.Log.Info("WARNING: No Resource limits are specified in the target object")
		return true, nil
	}

	allowed, err = isAllowed(rq, limitsneeded)
	if err != nil {
		ctrl.Log.Error(err, "Cannot find namespace quotas")
		return false, err
	}

	return allowed, nil
}

//This function will determine if we exceed the available resources in at least one resourcequota object
func isAllowed(rql *corev1.ResourceQuotaList, limitsneeded corev1.ResourceList) (bool, error) {

	var leftovers corev1.ResourceList
	log := ctrl.Log.
		WithValues("Limits needed", limitsneeded)
	log.Info("Identified Resources")

	for _, rq := range rql.Items {
		// we don't care about ResourceQuotas that contain persistentvolumeclaims or storageRequests
		rq = sanitizeRQFromStorageAndPVC(rq)
		if len(rq.Status.Hard) == 0 && len(rq.Status.Used) == 0 {
			continue
		}
		leftovers = math.Subtract(rq.Status.Hard, rq.Status.Used)
		log = ctrl.Log.
			WithValues("Rq name", rq.Name).
			WithValues("Leftover resources", leftovers)
		log.Info("Resource Quota")

		checklimits := math.Subtract(leftovers, math.TranslateResourcesToQuotaResources(limitsneeded))

		log = ctrl.Log.
			WithValues("Limits", checklimits)
		log.Info("Final checks")

		if len(math.IsNegative(checklimits)) != 0 {
			return false, nil
		}
	}

	return true, nil
}

func sanitizeRQFromStorageAndPVC(rq corev1.ResourceQuota) corev1.ResourceQuota {
	listHard := rq.Status.Hard
	for key, _ := range listHard {
		if key == "persistentvolumeclaims" || key == "requests.storage" {
			delete(listHard, key)
		}
	}
	rq.Status.Hard = listHard

	listUsed := rq.Status.Used
	for key, _ := range listUsed {
		if key == "persistentvolumeclaims" || key == "requests.storage" {
			delete(listUsed, key)
		}
	}
	rq.Status.Used = listUsed
	return rq
}

func resourceQuota(ctx context.Context, namespace string, kubernetesclient kubernetes.Interface) (*corev1.ResourceQuotaList, error) {

	rq := &corev1.ResourceQuotaList{}

	rq, err := kubernetesclient.CoreV1().ResourceQuotas(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return &corev1.ResourceQuotaList{}, err
	}

	if len(rq.Items) == 0 {
		return &corev1.ResourceQuotaList{}, errors.New(c.RQNotFound)
	}

	return rq, nil
}
