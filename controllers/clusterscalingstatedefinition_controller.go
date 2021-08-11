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
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/containersol/prescale-operator/api/v1alpha1"
	scalingv1alpha1 "github.com/containersol/prescale-operator/api/v1alpha1"
	c "github.com/containersol/prescale-operator/internal"
	constants "github.com/containersol/prescale-operator/internal"
	"github.com/containersol/prescale-operator/internal/reconciler"
	"github.com/containersol/prescale-operator/internal/states"
	"github.com/containersol/prescale-operator/internal/validations"
)

// ClusterScalingStateDefinitionReconciler reconciles a ClusterScalingStateDefinition object
type ClusterScalingStateDefinitionReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=scaling.prescale.com,resources=clusterscalingstatedefinitions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=scaling.prescale.com,resources=clusterscalingstatedefinitions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=scaling.prescale.com,resources=clusterscalingstatedefinitions/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ClusterScalingStateDefinition object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *ClusterScalingStateDefinitionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var eventsList []string
	var appliedStates []string
	var appliedStateNamespaceList []string
	var dryRunCluster string
	log := r.Log.
		WithValues("reconciler kind", "ClusterScalingStatesDefinition").
		WithValues("reconciler object", req.Name)

	cssd := &v1alpha1.ClusterScalingStateDefinition{}
	err := r.Get(ctx, req.NamespacedName, cssd)
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

	log.Info("Reconciling")

	namespaces := corev1.NamespaceList{}
	err = r.Client.List(ctx, &namespaces)
	if err != nil {
		log.Error(err, "Cannot list namespaces")
		return ctrl.Result{}, err
	}
	log.Info("Clusterscalingstatedefinition Controller: Reconciling namespaces")

	nsInfos, retrigger, err := reconciler.PrepareForNamespaceReconcile(ctx, r.Client, "", clusterStateDefinitions, states.State{}, r.Recorder, cssd.Config.DryRun)
	if err != nil {
		return ctrl.Result{}, err
	}

	if nsInfos == nil && !retrigger && err == nil {
		return ctrl.Result{}, nil
	}

	// Loop over all the namespace events of the namespaces which have been reconciled
	for namespaceKey, nsInfo := range nsInfos {
		if !cssd.Config.DryRun {

			if nsInfo.NSEvents.QuotaExceeded != "" {
				eventsList = append(eventsList, nsInfo.NSEvents.QuotaExceeded)
			}

			appliedStateNamespaceList = append(appliedStateNamespaceList, namespaceKey)
			appliedStates = append(appliedStates, nsInfo.AppliedState)

		} else {
			dryRunCluster = dryRunCluster + nsInfo.NSEvents.DryRunInfo
		}
	}

	if !cssd.Config.DryRun {

		if len(eventsList) != 0 {
			r.Recorder.Event(cssd, "Warning", "QuotaExceeded", fmt.Sprintf("Not enough available resources for the following %d namespaces: %s", len(eventsList), eventsList))
		}

		log.Info("Clusterscalingstatedefinition Reconciliation loop completed")

	} else {
		if dryRunCluster == "" {
			r.Recorder.Event(cssd, "Normal", "DryRun", "DryRun: No changes in any namespace would be made!")

		} else {
			r.Recorder.Event(cssd, "Normal", "DryRun", fmt.Sprintf("DryRun: %s", dryRunCluster))
		}

	}
	if retrigger {
		log.Info(fmt.Sprintf("Not all namespaces reconciled. Retriggering ClusterScalingStateDefinitionController in %d seconds", c.RetriggerControllerSeconds))
		return ctrl.Result{RequeueAfter: time.Second * c.RetriggerControllerSeconds}, nil
	}

	// Trigger again in order to catch resources which have been created or updated within the first 10 seconds of startup
	if time.Since(constants.StartTime).Seconds() < 12 {
		log.Info("Startup complete. Retriggering ClusterScalingStateDefinitionController one more time")
		return ctrl.Result{RequeueAfter: time.Second * c.RetriggerControllerSeconds}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterScalingStateDefinitionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&scalingv1alpha1.ClusterScalingStateDefinition{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		WithEventFilter(validations.DeleteFilter()).
		Complete(r)
}
