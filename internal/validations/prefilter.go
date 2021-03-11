package validations

import (
	"github.com/containersol/prescale-operator/pkg/utils/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// PreFilter for incoming changes of deployments. We only care about changes on opt-in label.
func PreFilter() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			deploymentName := e.ObjectNew.GetName()

			var oldoptin, newoptin bool
			newoptin = labels.GetLabelValue(e.ObjectNew.GetLabels(), "scaler/opt-in")
			oldoptin = labels.GetLabelValue(e.ObjectOld.GetLabels(), "scaler/opt-in")

			if newoptin == true {
				log := ctrl.Log
				log.Info("(UpdateEvent) Deployment is opted in. Reconciling.. " + deploymentName)
				return true
			} else if oldoptin == true && newoptin == false {
				log := ctrl.Log.
					WithValues("old", oldoptin).
					WithValues("new", newoptin)
				log.Info("(UpdateEvent) Labels are for deployment. Deployment" + deploymentName + " opted out. Trying to reconcile back to default replica count")
				return true
			}

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
