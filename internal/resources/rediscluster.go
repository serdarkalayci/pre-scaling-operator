package resources

import (
	"context"

	g "github.com/containersol/prescale-operator/pkg/utils/global"
	redisalpha "github.com/containersolutions/redis-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type RedisClusterScaleError struct {
	msg string
}

func (err RedisClusterScaleError) Error() string {
	return err.msg
}

//RedisClusterGetter returns the specific deployment data given a reconciliation request
func RedisClusterGetter(ctx context.Context, _client client.Client, req ctrl.Request) (redisalpha.RedisCluster, error) {

	rediscluster := redisalpha.RedisCluster{}
	err := _client.Get(ctx, req.NamespacedName, &rediscluster)
	if err != nil {
		return redisalpha.RedisCluster{}, err
	}
	return rediscluster, nil

}

//DeploymentGetter returns the specific deployment data given a scaleitem
func RedisClusterGetterByScaleItem(ctx context.Context, _client client.Client, deploymentItem g.ScalingInfo) (redisalpha.RedisCluster, error) {
	var req reconcile.Request
	req.NamespacedName.Namespace = deploymentItem.Namespace
	req.NamespacedName.Name = deploymentItem.Name

	return RedisClusterGetter(ctx, _client, req)

}
