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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/containersol/prescale-operator/api/v1alpha1"
	scalingv1alpha1 "github.com/containersol/prescale-operator/api/v1alpha1"
	"github.com/containersol/prescale-operator/internal/reconciler"
	"github.com/containersol/prescale-operator/internal/states"
	"github.com/containersol/prescale-operator/internal/validations"
)

// ScalingStateReconciler reconciles a ScalingState object
type ScalingStateReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=scaling.prescale.com,resources=scalingstates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=scaling.prescale.com,resources=scalingstates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=scaling.prescale.com,resources=scalingstates/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

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

	ss := &v1alpha1.ScalingState{}
	err := r.Get(ctx, req.NamespacedName, ss)
	if err != nil {
		log.Error(err, "Scalingstate could not be found! It might've been deleted. Reconciling.")
	}

	clusterStateDefinitions, err := states.GetClusterScalingStates(ctx, r.Client)
	if err != nil {
		// If we encounter an error trying to retrieve the state definitions,
		// we will not be able to compute anything else.
		log.Error(err, "Failed to get ClusterStateDefinitions")
		return ctrl.Result{}, err
	}

	log.WithValues("Namespace", req.Namespace).
		Info("Scalingstate Controller: Reconciling namespace")

	nsInfos, _, err := reconciler.PrepareForNamespaceReconcile(ctx, r.Client, req.Namespace, clusterStateDefinitions, states.State{}, r.Recorder, ss.Config.DryRun)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(nsInfos) == 0 && ss.Config.DryRun {
		r.Recorder.Event(ss, "Normal", "DryRun", "DryRun: No changes in any namespace would be made!")
	}

	for _, nsInfo := range nsInfos {
		if !ss.Config.DryRun {

			if nsInfo.NSEvents.QuotaExceeded != "" {
				r.Recorder.Event(ss, "Warning", "QuotaExceeded", fmt.Sprintf("Not enough available resources for namespace %s", nsInfo.NSEvents.QuotaExceeded))
			}

			log.Info("Scalingstate Reconciliation loop completed successfully")

		} else {

			r.Recorder.Event(ss, "Normal", "DryRun", fmt.Sprintf("DryRun: %s", nsInfo.NSEvents.DryRunInfo))

		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ScalingStateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&scalingv1alpha1.ScalingState{}).
		WithEventFilter(validations.StartupFilter()).
		WithOptions(controller.Options{MaxConcurrentReconciles: 5}).
		Owns(&scalingv1alpha1.ClusterScalingState{}).
		Complete(r)
}
