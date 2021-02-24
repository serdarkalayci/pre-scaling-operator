/*
Copyright 2019 LitmusChaos Authors
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

	"github.com/go-logr/logr"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Watcher reconciles a ScalingState object
type Watcher struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// WatchForDeployments creates watcher for Chaos Runner Pod
func (r *Watcher) WatchForDeployments(client client.Client, c controller.Controller) error {

	return c.Watch(&source.Kind{Type: &v1.Deployment{}}, &handler.EnqueueRequestForObject{})
}

// Reconcile tries to reconcile the replicas of the opted-in deployments
func (r *Watcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := r.Log.
		WithValues("reconciler kind", "Wachter").
		WithValues("reconciler namespace", req.Namespace).
		WithValues("reconciler object", req.Name)

	// Fetch the deployment
	deployment := v1.Deployment{}
	err := r.Client.Get(ctx, req.NamespacedName, &deployment)
	if err != nil {
		log.Error(err, "Failed to get the deployment data")
		return ctrl.Result{}, err
	}

	// The first thing we need to do is determine if the deployment has the opt-in label and if it's set to true
	// If neither of these conditions is met, then we won't reconcile.
	labels := deployment.GetLabels()
	if v, found := labels["scaler/opt-in"]; found {
		if v != "true" {
			log.Info("Opted-out deployment. Doing nothing")
			return ctrl.Result{}, nil
		}
	} else {
		return ctrl.Result{}, nil
	}

	stateDefinitions, err := states.GetClusterScalingStateDefinitions(ctx, r.Client)
	if err != nil {
		// If we encounter an error trying to retrieve the state definitions,
		// we will not be able to compute anything else.
		log.Error(err, "Failed to get ClusterStateDefinitions")
		return ctrl.Result{}, err
	}

	// We need to calculate the desired state before we try to reconcile the deployment
	finalState, err := reconciler.GetAppliedState(ctx, r.Client, req.Namespace, stateDefinitions, states.State{})
	if err != nil {
		log.Error(err, "Cannot determine applied state for namespace")
	}

	err = reconciler.ReconcileDeployment(ctx, r.Client, deployment, finalState)

	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Reconciliation loop completed successfully")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Watcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Deployment{}).
		Complete(r)
}
