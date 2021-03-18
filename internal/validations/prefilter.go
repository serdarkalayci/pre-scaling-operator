package validations

import (
	"reflect"

	"github.com/containersol/prescale-operator/pkg/utils/annotations"
	"github.com/containersol/prescale-operator/pkg/utils/labels"
	v1 "k8s.io/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// PreFilter for incoming changes of deployments or deploymentconfigs. We only care about changes on the opt-in label, replica count and annotations.
func PreFilter() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			deploymentName := e.ObjectNew.GetName()
			var oldoptin, newoptin, replicaChange, annotationchange bool

			var annotationsnew, annotationsold map[string]string

			oldoptin = labels.GetLabelValue(e.ObjectOld.GetLabels(), "scaler/opt-in")
			newoptin = labels.GetLabelValue(e.ObjectNew.GetLabels(), "scaler/opt-in")

			// Both optin are false, there is no case where we need to reconcile
			if !oldoptin && !newoptin {
				return false
			}

			// optinLabel has changed, we need to reconcile in any case
			if (oldoptin != newoptin) {
				return true
			}

			oldDeployment, ok1 := e.ObjectOld.(*v1.Deployment)
			newDeployment, ok2 := e.ObjectNew.(*v1.Deployment)

			if !ok1 || !ok2 {
				return false
			}

			// Check if replicas count has changed
			if *oldDeployment.Spec.Replicas != *newDeployment.Spec.Replicas {
				replicaChange = true
			}

			// Check for changes in relevant annotations.
			annotationsnew = annotations.FilterByKeyPrefix("scaler", e.ObjectNew.GetAnnotations())
			annotationsold = annotations.FilterByKeyPrefix("scaler", e.ObjectOld.GetAnnotations())

			eq := reflect.DeepEqual(annotationsnew, annotationsold)

			if !eq {
				annotationchange = true
			}

			// eval if we need to reconcile.
			// (a || b) && c && d
			if ((annotationchange || replicaChange) && newoptin && oldoptin) {
				log := ctrl.Log.
					WithValues("Annotationchange", annotationchange).
					WithValues("Replicachange", replicaChange).
					WithValues("OldOptIn", oldoptin).
					WithValues("NewOptIn", newoptin)
				log.Info("Reconciling for deployment " + deploymentName)
				return true
			}
			// don't reconcile at all
			return false

		},
		CreateFunc: func(e event.CreateEvent) bool {
			newlabels := e.Object.GetLabels()

			newoptin := labels.GetLabelValue(newlabels, "scaler/opt-in")
			deploymentName := e.Object.GetName()
			log := ctrl.Log
			log.WithValues("OptIn is: ", newoptin)
			log.Info("(CreateEvent) New deployment detected: " + deploymentName)

			return newoptin
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
	}
}
