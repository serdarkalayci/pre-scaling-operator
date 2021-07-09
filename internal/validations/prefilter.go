package validations

import (
	"fmt"
	"reflect"
	"time"

	constants "github.com/containersol/prescale-operator/internal"
	"github.com/containersol/prescale-operator/pkg/utils/annotations"
	g "github.com/containersol/prescale-operator/pkg/utils/global"
	"github.com/containersol/prescale-operator/pkg/utils/labels"
	ocv1 "github.com/openshift/api/apps/v1"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// PreFilter for incoming changes of deployments or deploymentconfigs. We only care about changes on the opt-in label, replica count and annotations.
func PreFilter(r record.EventRecorder) predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			deploymentName := e.ObjectNew.GetName()
			nameSpace := e.ObjectNew.GetNamespace()
			var oldoptin, newoptin, replicaChange, annotationchange bool

			oldoptin = labels.GetLabelValueBool(e.ObjectOld.GetLabels(), "scaler/opt-in")
			newoptin = labels.GetLabelValueBool(e.ObjectNew.GetLabels(), "scaler/opt-in")

			// check if scalingclass has changed
			oldclass := labels.GetLabelValueString(e.ObjectOld.GetLabels(), "scaler/scaling-class")
			newclass := labels.GetLabelValueString(e.ObjectNew.GetLabels(), "scaler/scaling-class")
			if !reflect.DeepEqual(oldclass, newclass) && newoptin {
				return true
			}

			generateOptInLabelUpdateEvent(e, r, newoptin, oldoptin)

			item := g.ScalingInfo{
				Name:      deploymentName,
				Namespace: nameSpace,
			}

			// Deployment opted out. Don't do anything
			if !newoptin {
				if g.GetDenyList().IsBeingScaled(item) {
					// The deployment is being scaled at the moment! Notify scaler to abort.
					log := ctrl.Log.
						WithValues("Name", item.Name).
						WithValues("Namespace", item.Namespace).
						WithValues("NewOptIn", newoptin)
					log.Info("The deployment has been opted out and is being scaled at the moment. Trying to intercept the step scaler to stop scaling!")
					g.GetDenyList().SetScalingItemOnList(item, true, "Opt-In is false!", -1)
				}
				return false
			}

			replicaChange = AssesReplicaChange(e)
			annotationchange = AssessAnnotationChange(e)

			// don't reconcile if the stepscale annotation is present.
			//stepScaleActive := AssessStepScaleAnnotation(e)

			// Let reconciler_cron handle those deployments
			if g.GetDenyList().IsDeploymentInFailureState(item) {
				return false
			}

			// Don't reconcile on any change except when Annotation may have changed while the deployment was on the deny list.
			if g.GetDenyList().IsBeingScaled(item) && !annotationchange {
				return false
			}

			// eval if we need to reconcile.
			if !annotationchange && !replicaChange && newoptin && oldoptin {
				// Something else on the deployment has changed. Don't reconcile.
				return false
			}

			log := ctrl.Log.
				WithValues("Annotationchange", annotationchange).
				WithValues("Replicachange", replicaChange).
				WithValues("OldOptIn", oldoptin).
				WithValues("NewOptIn", newoptin)
			log.Info("Reconciling for object " + deploymentName)

			return true

		},
		CreateFunc: func(e event.CreateEvent) bool {

			uptime := time.Since(constants.StartTime)
			if uptime.Seconds() < 5 {
				return false
			}

			newlabels := e.Object.GetLabels()

			newoptin := labels.GetLabelValueBool(newlabels, "scaler/opt-in")
			deploymentName := e.Object.GetName()

			if newoptin {
				generateOptInLabelCreateEvent(e, r, newoptin)
				log := ctrl.Log
				log.Info("(CreateEvent) New opted-in deployment detected: " + deploymentName)
			}

			return newoptin
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// The deployment got deleted. Regardless of the failure state, we need to delete the item from the list.
			item := g.ScalingInfo{
				Name:      e.Object.GetName(),
				Namespace: e.Object.GetNamespace(),
			}

			if g.GetDenyList().IsInConcurrentList(item) {
				g.GetDenyList().RemoveFromList(item)
			}

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

func generateOptInLabelUpdateEvent(e event.UpdateEvent, r record.EventRecorder, newoptin, oldoptin bool) {

	if newoptin != oldoptin {

		if reflect.TypeOf(e.ObjectNew) == reflect.TypeOf(&ocv1.DeploymentConfig{}) {

			if newoptin {
				r.Event(e.ObjectNew.(*ocv1.DeploymentConfig), "Normal", "PreScalingOperator", fmt.Sprintf("The %s object has just opted-in", e.ObjectNew.(*ocv1.DeploymentConfig).Name))
			} else {
				r.Event(e.ObjectNew.(*ocv1.DeploymentConfig), "Normal", "PreScalingOperator", fmt.Sprintf("The %s object has just opted-out", e.ObjectNew.(*ocv1.DeploymentConfig).Name))
			}

		} else {

			if newoptin {
				r.Event(e.ObjectNew.(*v1.Deployment), "Normal", "PreScalingOperator", fmt.Sprintf("The %s object has just opted-in", e.ObjectNew.(*v1.Deployment).Name))
			} else {
				r.Event(e.ObjectNew.(*v1.Deployment), "Normal", "PreScalingOperator", fmt.Sprintf("The %s object has just opted-out", e.ObjectNew.(*v1.Deployment).Name))
			}
		}

	}

}

func generateOptInLabelCreateEvent(e event.CreateEvent, r record.EventRecorder, newoptin bool) {

	if reflect.TypeOf(e.Object) == reflect.TypeOf(&ocv1.DeploymentConfig{}) {
		r.Event(e.Object.(*ocv1.DeploymentConfig), "Normal", "PreScalingOperator", fmt.Sprintf("The %s object has just opted-in", e.Object.(*ocv1.DeploymentConfig).Name))

	} else {
		r.Event(e.Object.(*v1.Deployment), "Normal", "PreScalingOperator", fmt.Sprintf("The %s object has just opted-in", e.Object.(*v1.Deployment).Name))
	}
}
