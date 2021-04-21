package validations

import (
	"reflect"

	"github.com/containersol/prescale-operator/pkg/utils/annotations"
	"github.com/containersol/prescale-operator/pkg/utils/labels"
	ocv1 "github.com/openshift/api/apps/v1"
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

			oldoptin = labels.GetLabelValue(e.ObjectOld.GetLabels(), "scaler/opt-in")
			newoptin = labels.GetLabelValue(e.ObjectNew.GetLabels(), "scaler/opt-in")

			// Deployment opted out. Don't do anything
			if !newoptin {
				return false
			}

			replicaChange = AssesReplicaChange(e)
			annotationchange = AssessAnnotationChange(e)

			// don't reconcile if the stepscale annotation is present.
			stepScaleActive := AssessStepScaleAnnotation(e)
			if stepScaleActive {
				return false
			}

			// eval if we need to reconcile.
			if !annotationchange && !replicaChange && newoptin && oldoptin {
				// Something else on the deployment has changed. Don't reconcile.
				return false
			} else {
				log := ctrl.Log.
					WithValues("Annotationchange", annotationchange).
					WithValues("Replicachange", replicaChange).
					WithValues("OldOptIn", oldoptin).
					WithValues("NewOptIn", newoptin)
				log.Info("Reconciling for deployment " + deploymentName)
				return true
			}

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

func AssessAnnotationChange(e event.UpdateEvent) bool {
	// Check for changes in relevant annotations.
	annotationsnew := annotations.FilterByKeyPrefix("scaler", e.ObjectNew.GetAnnotations())
	annotationsold := annotations.FilterByKeyPrefix("scaler", e.ObjectOld.GetAnnotations())

	eq := reflect.DeepEqual(annotationsnew, annotationsold)

	if !eq {
		return true
	}
	return false
}

func AssesReplicaChange(e event.UpdateEvent) bool {
	var replicasOld, replicasNew *int32
	if reflect.TypeOf(e.ObjectNew) == reflect.TypeOf(&ocv1.DeploymentConfig{}) {
		replicasOld = &e.ObjectOld.(*ocv1.DeploymentConfig).Spec.Replicas
		replicasNew = &e.ObjectNew.(*ocv1.DeploymentConfig).Spec.Replicas
	} else {
		replicasOld = e.ObjectOld.(*v1.Deployment).Spec.Replicas
		replicasNew = e.ObjectNew.(*v1.Deployment).Spec.Replicas
	}

	// Check if replicas count has changed
	if *replicasOld != *replicasNew {
		return true
	}
	return false
}

func AssessStepScaleAnnotation(e event.UpdateEvent) bool {
	annotationsold := annotations.FilterByKeyPrefix("scaler/step-scale-active", e.ObjectOld.GetAnnotations())
	if len(annotationsold) == 0 {
		return false
	}
	return true
}
