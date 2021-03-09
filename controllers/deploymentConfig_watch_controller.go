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
	c "github.com/containersol/prescale-operator/internal"
	"github.com/containersol/prescale-operator/internal/reconciler"
	"github.com/containersol/prescale-operator/internal/resources"
	"github.com/containersol/prescale-operator/internal/states"
	"github.com/go-logr/logr"
	ocv1 "github.com/openshift/api/apps/v1"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// DeploymentConfigWatcher reconciles a ScalingState object
type DeploymentConfigWatcher struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=list;watch;
// +kubebuilder:rbac:groups=apps.openshift.io,resources=deploymentconfigs,verbs=get;list;watch;patch;update;

// WatchForDeploymentConfigs creates watcher for the deploymentconfig objects
func (r *DeploymentConfigWatcher) WatchForDeploymentConfigs(client client.Client, c controller.Controller) error {

	return c.Watch(&source.Kind{Type: &ocv1.DeploymentConfig{}}, &handler.EnqueueRequestForObject{})
}

// Reconcile tries to reconcile the replicas of the opted-in deployments
func (r *DeploymentConfigWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := r.Log.
		WithValues("reconciler kind", "DeploymentConfigWatcher").
		WithValues("reconciler namespace", req.Namespace).
		WithValues("reconciler object", req.Name)

	deploymentconfig := ocv1.DeploymentConfig{}

	err := r.Client.Get(ctx, req.NamespacedName, &deploymentconfig)
	if err != nil {
		log.Error(err, "Failed to get the deployment data")
		return ctrl.Result{}, err
	}

	// The first thing we need to do is determine if the deployment has the opt-in label and if it's set to true
	// If neither of these conditions is met, then we won't reconcile.
	optinLabel, err := resources.DeploymentConfigOptinLabel(deploymentconfig)
	if err != nil {
		if strings.Contains(err.Error(), c.LabelNotFound) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to validate the opt-in label")
		return ctrl.Result{}, err
	}

	if !optinLabel {
		log.Info("Deployment opted out. No reconciliation")
		return ctrl.Result{}, nil
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
	err = reconciler.ReconcileDeploymentConfig(ctx, r.Client, deploymentconfig, finalState)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Reconciliation loop completed successfully for deploymentconfig")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeploymentConfigWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ocv1.DeploymentConfig{}).
		Complete(r)
}
