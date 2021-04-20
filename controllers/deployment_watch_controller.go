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

	"github.com/containersol/prescale-operator/internal/reconciler"
	"github.com/containersol/prescale-operator/internal/states"
	"github.com/containersol/prescale-operator/internal/validations"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// DeploymentWatcher reconciles a ScalingState object
type DeploymentWatcher struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=list;watch;
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;patch;update;
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// WatchForDeployments creates watcher for the deployment objects
func (r *DeploymentWatcher) WatchForDeployments(client client.Client, c controller.Controller) error {

	return c.Watch(&source.Kind{Type: &v1.Deployment{}}, &handler.EnqueueRequestForObject{})
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

	// We are certain that we have an object to reconcile, we need to get the state definitions
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
	err = reconciler.ReconcileDeployment(ctx, r.Client, deployment, finalState)
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
		WithEventFilter(validations.PreFilter()).
		Complete(r)
}
