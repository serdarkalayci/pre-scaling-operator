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
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	scalingv1alpha1 "github.com/containersol/prescale-operator/api/v1alpha1"
)

// ScalingStateReconciler reconciles a ScalingState object
type ScalingStateReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=scaling.prescale.com,resources=scalingstates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=scaling.prescale.com,resources=scalingstates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=scaling.prescale.com,resources=scalingstates/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ScalingState object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *ScalingStateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.
		WithValues("reconciler kind", "ScalingState").
		WithValues("reconciler namespace", req.Namespace).
		WithValues("reconciler object", req.Name)

	// cssd here stand for ClusterScalingStateDefinitino
	scalingState := &scalingv1alpha1.ScalingState{}
	err := r.Get(ctx, req.NamespacedName, scalingState)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("ScalingState resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get ScalingState")
		return ctrl.Result{}, err
	}

	// When a ScalingState is created or updated,
	// we need to check both it and the ClusterState in order to determine the actual state the namespace should be in.
	cssd := &scalingv1alpha1.ClusterScalingStateDefinitionList{}
	r.List(ctx, cssd, &client.ListOptions{})

	if len(cssd.Items) == 0 {
		log.Info("No ClusterScalingStateDefinition Found. Doing Nothing.")
		// TODO Should we add errors here to crash the controller and make it explicit that one should be set ?
		return ctrl.Result{}, nil
	}

	if len(cssd.Items) >= 2 {
		log.Info("More than 1 ClusterScalingStateDefinition found. Merging is not yet supported. Doing Nothing.")
		return ctrl.Result{}, nil
	}

	// We now have the definitions of which states are available to developers.
	// @TODO implement priority overrides, once the priority is set for a clusterstatedefinition

	// Next we need to fetch the ClusterScalingState to determine which states are currently set in a namespace
	clusterScalingStates := &scalingv1alpha1.ClusterScalingStateList{}
	r.List(ctx, clusterScalingStates, &client.ListOptions{})

	if len(clusterScalingStates.Items) >= 2 {
		log.Info("More than 1 ClusterScalingState found. Merging is not yet supported.")
		return ctrl.Result{}, nil
	}

	namespaceState := scalingState.Spec.State
	if len(clusterScalingStates.Items) == 1 {
		log.Info("No ClusterScalingStates found to compare. Using only ScalingState for calculations.")
		// @TODO Here the logic for priority needs to be added.
		// We need to use the definition to decide which has higher priority, and mark that as the namespace state
	}

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
	r.List(ctx, deployments, &client.ListOptions{LabelSelector: selector})

	if len(deployments.Items) == 0 {
		log.Info("No deployments found to manage in namespace. Doing Nothing.")
	}

	for _, deployment := range deployments.Items {
		r.Log.Info("Checking replication state for deployment", "deployment", deployment.Name)
		r.Log.Info("Annotations", "annotations", deployment.GetAnnotations())
	}

	log.Info("Reconciling")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ScalingStateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&scalingv1alpha1.ScalingState{}).
		Complete(r)
}
