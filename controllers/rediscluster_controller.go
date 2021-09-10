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

	"github.com/containersol/prescale-operator/internal/reconciler"
	"github.com/containersol/prescale-operator/internal/resources"
	"github.com/containersol/prescale-operator/internal/validations"
	g "github.com/containersol/prescale-operator/pkg/utils/global"
	redisalpha "github.com/containersolutions/redis-operator/api/v1alpha1"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

// RedisClusterWatcher reconciles a ScalingState object
type RedisClusterWatcher struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=list;watch;
// +kubebuilder:rbac:groups=redis.containersolutions.com,resources=redisclusters,verbs=get;list;watch;patch;update;
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile tries to reconcile the replicas of the opted-in deployments
func (r *RedisClusterWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := r.Log.
		WithValues("reconciler kind", "RedisClusterWatcher").
		WithValues("reconciler namespace", req.Namespace).
		WithValues("reconciler object", req.Name)
	// Fetch the deployment data
	redisCluster, err := resources.RedisClusterGetter(ctx, r.Client, req)
	if err != nil {
		log.Error(err, "Failed to get the deployment data")
		return ctrl.Result{}, err
	}
	deploymentItem := g.ConvertRedisClusterToItem(redisCluster)

	// After we have the deployment and state data, we are ready to reconcile the deployment
	// Only reconcile if the item is not in a failure state. Failure states are only handled by RectifyScaleItemsInFailureState() in reconciler_cron.go
	if !g.GetDenyList().IsDeploymentInFailureState(deploymentItem) {
		go reconciler.ReconcileScalingItem(ctx, r.Client, deploymentItem, false, r.Recorder, "REDISCLUSTERWATCHCONTROLLER")
	}

	log.Info("RedisCluster Reconciliation loop completed")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RedisClusterWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&redisalpha.RedisCluster{}).
		WithEventFilter(validations.PreFilter(r.Recorder)).
		WithEventFilter(validations.StartupFilter()).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
