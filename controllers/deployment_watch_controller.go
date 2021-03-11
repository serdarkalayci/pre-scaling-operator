/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
   http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"strings"

	c "github.com/containersol/prescale-operator/internal"
	"github.com/containersol/prescale-operator/internal/reconciler"
	"github.com/containersol/prescale-operator/internal/resources"
	"github.com/containersol/prescale-operator/internal/states"
	"github.com/containersol/prescale-operator/pkg/utils/labels"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// DeploymentWatcher reconciles a ScalingState object
type DeploymentWatcher struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=list;watch;
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;patch;update;

// WatchForDeployments creates watcher for the deployment objects
func (r *DeploymentWatcher) WatchForDeployments(client client.Client, c controller.Controller) error {

	return c.Watch(&source.Kind{Type: &v1.Deployment{}}, &handler.EnqueueRequestForObject{})
}

// Filters the incoming changes to deployments. We only care about changes on the labels.
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
			log.Info("(CreateEvent) New opted-in deployment detected. Reconciling.. " + deploymentName)

			return newoptin
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
	}
}

// Reconcile tries to reconcile the replicas of the opted-in deployments
func (r *DeploymentWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := r.Log.
		WithValues("reconciler kind", "DeploymentWatcher").
		WithValues("reconciler namespace", req.Namespace).
		WithValues("reconciler object", req.Name)
	// Fetch the deployment data
	deployment := v1.Deployment{}
	err := r.Client.Get(ctx, req.NamespacedName, &deployment)
	if err != nil {
		log.Error(err, "Failed to get the deployment data")
		return ctrl.Result{}, err
	}

	// The first thing we need to do is determine if the deployment has the opt-in label and if it's set to true
	// If neither of these conditions is met, then we won't reconcile.
	optinLabel, err := resources.DeploymentOptinLabel(deployment)
	if err != nil {
		if strings.Contains(err.Error(), c.LabelNotFound) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to validate the opt-in label")
		return ctrl.Result{}, err
	}

	// Next step after we are certain that we have an object to reconcile, we need to get the state definitions
	stateDefinitions, err := states.GetClusterScalingStateDefinitions(ctx, r.Client)
	if err != nil {
		log.Error(err, "Failed to get ClusterStateDefinitions")
		return ctrl.Result{}, err
	}

	// We need to calculate the desired state before we try to reconcile the deployment
	finalState, err := reconciler.GetAppliedState(ctx, r.Client, req.Namespace, stateDefinitions, states.State{})
	if err != nil {
		log.Error(err, "Cannot determine applied state for namespace")
		return ctrl.Result{}, err
	}

	// After we have the deployment and state data, we are ready to reconcile the deployment
	err = reconciler.ReconcileDeployment(ctx, r.Client, deployment, finalState, optinLabel)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Reconciliation loop completed successfully")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeploymentWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Deployment{}).
		WithEventFilter(PreFilter()).
		Complete(r)
}
