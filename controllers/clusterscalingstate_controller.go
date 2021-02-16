/*
Copyright 2021.

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
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	scalingv1alpha1 "github.com/containersol/prescale-operator/api/v1alpha1"

	sr "github.com/containersol/prescale-operator/internal"
	v1 "k8s.io/api/apps/v1"

	"k8s.io/apimachinery/pkg/labels"

	"k8s.io/apimachinery/pkg/selection"
)

// ClusterScalingStateReconciler reconciles a ClusterScalingState object
type ClusterScalingStateReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=scaling.prescale.com,resources=clusterscalingstates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=scaling.prescale.com,resources=clusterscalingstates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=scaling.prescale.com,resources=clusterscalingstates/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ClusterScalingState object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *ClusterScalingStateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := r.Log.
		WithValues("reconciler kind", "ClusterScalingState").
		WithValues("reconciler object", req.Name)

	clusterScalingState := &scalingv1alpha1.ClusterScalingState{}
	err := r.Get(ctx, req.NamespacedName, clusterScalingState)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("ClusterScalingState resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get ClusterScalingState")
		return ctrl.Result{}, err
	}

	cssd := &scalingv1alpha1.ClusterScalingStateDefinitionList{}
	r.List(ctx, cssd, &client.ListOptions{})

	if len(cssd.Items) == 0 {
		log.Info("No ClusterScalingStateDefinition Found. Doing Nothing.")
		return ctrl.Result{}, nil
	}

	if len(cssd.Items) >= 2 {
		log.Info("More than 1 ClusterScalingStateDefinition found. Merging is not yet supported. Doing Nothing.")
		return ctrl.Result{}, nil
	}

	// Next we need to fetch the current ScalingStates to determine which states are currently set in a namespace
	scalingStates := &scalingv1alpha1.ScalingStateList{}
	r.List(ctx, scalingStates, &client.ListOptions{})

	if len(scalingStates.Items) >= 2 {
		log.Info("More than 1 ScalingState found. Merging is not yet supported.")
		return ctrl.Result{}, nil
	}

	namespaceState := clusterScalingState.Spec.State
	if len(scalingStates.Items) == 0 {
		log.Info("No ScalingStates found to compare. Using only ClusterScalingState for calculations.")
	}

	// TODO: We need to take priority into consideration here before we decide which state we want to apply

	log.Info("State set for namespace.", "state", namespaceState)
	log.Info("Finding objects which are opted in for scale")

	// We now need to look for Deployments which are opted in,
	// then use their annotations to determine the correct scale
	deployments := &v1.DeploymentList{}
	requirements, err := labels.NewRequirement("scaler/opt-in", selection.Equals, []string{"true"})
	if err != nil {
		return ctrl.Result{}, err
	}
	selector := labels.Everything().Add(*requirements)
	r.List(ctx, deployments, &client.ListOptions{LabelSelector: selector, Namespace: ""})

	if len(deployments.Items) == 0 {
		log.Info("No deployments found to manage in namespace. Doing Nothing.")
	}

	for _, deployment := range deployments.Items {
		r.Log.Info("Checking replication state for deployment", "deployment", deployment.Name)
		stateReplicas, err := sr.NewStateReplicasFromAnnotations(deployment.GetAnnotations())
		if err != nil {
			log.WithValues("deployment", deployment.Name).
				WithValues("namespace", deployment.Namespace).
				Error(err, "Cannot calculate state replicas. Please check deployment annotations. Continuing.")
			continue
		}
		log.WithValues("state replicas", stateReplicas.GetStates()).Info("State replicas calculated")
		// Now we have all the state settings, we can set the replicas for the deployment accordingly
		stateReplica, err := stateReplicas.GetState(namespaceState)
		if err != nil {
			// TODO here we should do priority filtering, and go down one level of priority to find the lowest set one.
			// We will ignore any that are not set
			log.WithValues("set states", stateReplicas).WithValues("namespace state", namespaceState).Info("State could not be found")
		} else {
			log.Info("Updating deployment replicas for state", "replicas", stateReplica.Replicas)
			deployment.Spec.Replicas = &stateReplica.Replicas
			r.Update(ctx, &deployment, &client.UpdateOptions{})
		}
	}

	log.Info("Reconciliation loop completed successfully")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterScalingStateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&scalingv1alpha1.ClusterScalingState{}).
		Complete(r)
}
