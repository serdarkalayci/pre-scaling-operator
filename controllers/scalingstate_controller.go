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
	}

	if len(cssd.Items) >= 2 {
		log.Info("More than 1 ClusterScalingStateDefinition found. Merging is not yet supported. Doing Nothing.")
	}

	// @TODO Here the logic for priority needs to be added.
	// We need to check which has higher priority, and use that as the scaling method for the namespace

	// Next we need to loop the deployments in the Namespace which have the opt-in label set

	log.WithValues("cluster state definitions", len(cssd.Items)).Info("Reconciling")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ScalingStateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&scalingv1alpha1.ScalingState{}).
		Complete(r)
}
