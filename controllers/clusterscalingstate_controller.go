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
	"fmt"

	"github.com/containersol/prescale-operator/api/v1alpha1"
	scalingv1alpha1 "github.com/containersol/prescale-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/containersol/prescale-operator/internal/reconciler"
	"github.com/containersol/prescale-operator/internal/states"
)

// ClusterScalingStateReconciler reconciles a ClusterScalingState object
type ClusterScalingStateReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=scaling.prescale.com,resources=clusterscalingstates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=scaling.prescale.com,resources=clusterscalingstates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=scaling.prescale.com,resources=clusterscalingstates/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

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
	var eventsList []string
	var appliedStates []string
	var appliedStateNamespaceList []string
	var dryRunCluster string
	log := r.Log.
		WithValues("reconciler kind", "ClusterScalingState").
		WithValues("reconciler object", req.Name)

	css := &v1alpha1.ClusterScalingState{}
	err := r.Get(ctx, req.NamespacedName, css)
	if err != nil {
		return ctrl.Result{}, err
	}

	clusterStateDefinitions, err := states.GetClusterScalingStates(ctx, r.Client)
	if err != nil {
		// If we encounter an error trying to retrieve the state definitions,
		// we will not be able to compute anything else.
		log.Error(err, "Failed to get ClusterStateDefinitions")
		return ctrl.Result{}, err
	}

	namespaces := corev1.NamespaceList{}
	err = r.Client.List(ctx, &namespaces)
	if err != nil {
		log.Error(err, "Cannot list namespaces")
		return ctrl.Result{}, err
	}
	log.Info("Clusterscalingstate Controller: Reconciling namespaces")
	for _, namespace := range namespaces.Items {

		events, state, err := reconciler.PrepareForNamespaceReconcile(ctx, r.Client, clusterStateDefinitions, states.State{}, r.Recorder, css.Config.DryRun)
		if err != nil {
			return ctrl.Result{}, err
		}

		if !css.Config.DryRun {

			if events.QuotaExceeded != "" {
				eventsList = append(eventsList, events.QuotaExceeded)
			}

			appliedStateNamespaceList = append(appliedStateNamespaceList, namespace.Name)
			appliedStates = append(appliedStates, state)

		} else {
			dryRunCluster = dryRunCluster + events.DryRunInfo
		}
	}

	if !css.Config.DryRun {

		if len(eventsList) != 0 {
			r.Recorder.Event(css, "Warning", "QuotaExceeded", fmt.Sprintf("Not enough available resources for the following %d namespaces: %s", len(eventsList), eventsList))
		}

		r.Recorder.Event(css, "Normal", "AppliedStates", fmt.Sprintf("The applied state for each of the %s namespaces is %s", appliedStateNamespaceList, appliedStates))

		log.Info("Clusterscalingstate Reconciliation loop completed successfully")

	} else {

		r.Recorder.Event(css, "Normal", "DryRun", fmt.Sprintf("DryRun: %s", dryRunCluster))

	}

	return ctrl.Result{}, nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterScalingStateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&scalingv1alpha1.ClusterScalingState{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Owns(&scalingv1alpha1.ScalingState{}).
		Complete(r)
}
