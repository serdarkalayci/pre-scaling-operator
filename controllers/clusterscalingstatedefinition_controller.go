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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	scalingv1alpha1 "github.com/containersol/prescale-operator/api/v1alpha1"
	"github.com/containersol/prescale-operator/internal/reconciler"
	"github.com/containersol/prescale-operator/internal/states"
)

// ClusterScalingStateDefinitionReconciler reconciles a ClusterScalingStateDefinition object
type ClusterScalingStateDefinitionReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=scaling.prescale.com,resources=clusterscalingstatedefinitions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=scaling.prescale.com,resources=clusterscalingstatedefinitions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=scaling.prescale.com,resources=clusterscalingstatedefinitions/finalizers,verbs=update

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
	log := r.Log.
		WithValues("reconciler kind", "ClusterScalingStatesDefinition").
		WithValues("reconciler object", req.Name)

	clusterStateDefinitions, err := states.GetClusterScalingStateDefinitions(ctx, r.Client)
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

	for _, namespace := range namespaces.Items {

		err = reconciler.ReconcileNamespace(ctx, r.Client, namespace.Name, clusterStateDefinitions, states.State{})

		if err != nil {
			return ctrl.Result{}, err
		}

	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterScalingStateDefinitionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&scalingv1alpha1.ClusterScalingStateDefinition{}).
		Complete(r)
}
