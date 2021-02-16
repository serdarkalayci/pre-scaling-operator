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
	"errors"
	sr "github.com/containersol/prescale-operator/internal"
	"github.com/containersol/prescale-operator/internal/states"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/apps/v1"
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

	clusterStateDefinitions, err := states.GetClusterScalingStateDefinitions(ctx, r.Client)
	if err != nil {
		// If we encounter an error trying to retrieve the state definitions,
		// we will not be able to compute anything else.
		log.Error(err, "Failed to get ClusterStateDefinitions")
		return ctrl.Result{}, err
	}

	clusterStateName, err := states.GetClusterScalingState(ctx, r.Client)
	if err != nil {
		switch err.(type) {
		case states.NotFound:
			log.Info("No ClusterScalingState was found to compare. Using namespaced state instead.")
		case states.TooMany:
			log.Info("Too many ClusterScalingStates were found. Continuing on to using only namespaced state")
		default:
			// For the moment, we cannot deal with any other error.
			log.Error(err, "Could not get ClusterScalingStates.")
			return ctrl.Result{}, err
		}
	}
	clusterState := states.State{}
	if clusterStateName != "" {
		err = clusterStateDefinitions.FindState(clusterStateName, &clusterState)
		if err != nil {
			log.WithValues("state name", clusterStateName).
				Error(err, "Could not find ClusterScalingState within ClusterStateDefinitions. Continuing without considering ClusterScalingState.")
		}
	}

	namespaceStateName, err := states.GetNamespaceScalingStateName(ctx, r.Client, req.Namespace)
	if err != nil {
		switch err.(type) {
		case states.NotFound:
			log.Info("No ScalingState was found. Using cluster state instead.")
		case states.TooMany:
			log.Info("Too many ScalingStates were found. Using cluster state instead.")
		default:
			// For the moment, we cannot deal with any other error.
			log.Error(err, "Could not get ScalingStates.")
			return ctrl.Result{}, err
		}
	}
	namespaceState := states.State{}
	if namespaceStateName != "" {
		err = clusterStateDefinitions.FindState(namespaceStateName, &namespaceState)
		if err != nil {
			log.WithValues("state name", namespaceStateName).
				WithValues("error", err).
				Info("Could not find ScalingState within ClusterStateDefinitions. Continuing without considering ScalingState.")
		}
	}

	if namespaceState == (states.State{}) && clusterState == (states.State{}) {
		err = errors.New("no states defined for namespace. doing nothing")
		log.Error(err, "Cannot continue as no states are set for namespace.")
		return ctrl.Result{}, err
	}

	finalState := clusterStateDefinitions.FindPriorityState(namespaceState, clusterState)
	log.Info("State set for namespace.", "state", finalState.Name)

	log.Info("Searching Objects which are opted in to the scaler")
	// We now need to look for Deployments which are opted in,
	// then use their annotations to determine the correct scale
	deployments := v1.DeploymentList{}
	r.List(
		ctx,
		&deployments,
		client.MatchingLabels(map[string]string{"scaler/opt-in": "true"}),
		client.InNamespace(req.Namespace),
	)

	if len(deployments.Items) == 0 {
		log.Info("No deployments found to manage in namespace. Doing Nothing.")
		return ctrl.Result{}, nil
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
		stateReplica, err := stateReplicas.GetState(finalState.Name)
		if err != nil {
			// TODO here we should do priority filtering, and go down one level of priority to find the lowest set one.
			// We will ignore any that are not set
			log.WithValues("set states", stateReplicas).
				WithValues("namespace state", finalState.Name).
				Info("State could not be found")
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
func (r *ScalingStateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&scalingv1alpha1.ScalingState{}).
		Complete(r)
}
