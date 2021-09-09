package resources

import (
	"context"
	"fmt"
	"time"

	g "github.com/containersol/prescale-operator/pkg/utils/global"
	"github.com/go-logr/logr"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func StepScale(ctx context.Context, _client client.Client, deploymentItem g.ScalingInfo, recorder record.EventRecorder, log logr.Logger) error {
	var retryErr error = nil

	stepReplicaCount := deploymentItem.SpecReplica
	oldReplicaCount := deploymentItem.SpecReplica
	desiredReplicaCount := deploymentItem.DesiredReplicas
	initialDesiredReplicaCount := deploymentItem.DesiredReplicas

	// Loop step by step until deploymentItem has reached desiredreplica count.
	var stepCondition bool = true
	for stepCondition {
		// Refresh the deploymentItem
		deploymentItem, _ = g.GetDenyList().GetDeploymentInfoFromList(deploymentItem)
		// Get the desired replica count
		desiredReplicaCount = deploymentItem.DesiredReplicas
		// Check (and wait until) deployment is ready to scale
		err := WaitForReady(ctx, _client, deploymentItem, recorder, log)
		if err != nil {
			log.Error(err, "Error waiting for deployment to become ready")
			return err
		}
		// Not sure what this is for?
		if desiredReplicaCount == -1 {
			desiredReplicaCount = initialDesiredReplicaCount
		}
		oldReplicaCount = deploymentItem.SpecReplica
		// decide if we need to step up or down
		if oldReplicaCount < desiredReplicaCount {
			stepReplicaCount = oldReplicaCount + 1
		} else if oldReplicaCount > desiredReplicaCount {
			stepReplicaCount = oldReplicaCount - 1
		}
		// check if desired is reached from a fresh item
		if deploymentItem.ReadyReplicas == deploymentItem.DesiredReplicas {
			stepCondition = false
		} else {
			// Attempt to scale by 1 step
			retryErr = DoScaling(ctx, _client, deploymentItem, stepReplicaCount)
		}

		if retryErr != nil {
			deploymentItem.IsBeingScaled = false
			// Set failure state to true
			g.GetDenyList().SetScalingItemOnList(deploymentItem, true, retryErr.Error(), desiredReplicaCount)
			RegisterEvents(ctx, _client, recorder, retryErr, deploymentItem)
			return retryErr
		}
	}
	return nil
}

func RapidScale(ctx context.Context, _client client.Client, deploymentItem g.ScalingInfo, recorder record.EventRecorder, log logr.Logger) error {
	oldReplicaCount := deploymentItem.SpecReplica
	desiredReplicaCount := deploymentItem.DesiredReplicas
	// We need to skip this check in case of failure in order to get a new object from DoScaling() to check on the state on the cluster.
	if oldReplicaCount == desiredReplicaCount && !deploymentItem.Failure {
		return nil
	}

	var retryErr error = nil
	deploymentItem.IsBeingScaled = true
	g.GetDenyList().SetScalingItemOnList(deploymentItem, deploymentItem.Failure, deploymentItem.FailureMessage, desiredReplicaCount)

	err := WaitForReady(ctx, _client, deploymentItem, recorder, log)

	if err != nil {
		log.Error(err, "Error waiting for deployment to become ready")
		return err
	}

	retryErr = DoScaling(ctx, _client, deploymentItem, desiredReplicaCount)

	if retryErr != nil {
		deploymentItem.IsBeingScaled = false
		// Set failure state to true
		g.GetDenyList().SetScalingItemOnList(deploymentItem, true, retryErr.Error(), desiredReplicaCount)
		RegisterEvents(ctx, _client, recorder, retryErr, deploymentItem)
		return retryErr
	}

	return nil

}

func WaitForReady(ctx context.Context, _client client.Client, deploymentItem g.ScalingInfo, recorder record.EventRecorder, log logr.Logger) error {
	var err error

	stepReplicaCount := deploymentItem.SpecReplica
	desiredReplicaCount := deploymentItem.DesiredReplicas

	// Wait until deploymentItem is ready for the next step and check if it's failing for some reason
	waitTime := time.Duration(time.Duration(deploymentItem.ProgressDeadline))*time.Second + time.Second

	// Wait 2s at a time until timeout is reached
	for stay, timeout := true, time.After(waitTime); stay; {
		select {
		case <-timeout:
			// If timeout
			timeoutErr := ScaleError{
				msg: fmt.Sprintf("Message on the cluster: %s | The operator decided that it can't scale that deployment or deploymentconfig!", deploymentItem.ConditionReason),
			}
			deploymentItem.IsBeingScaled = false
			RegisterEvents(ctx, _client, recorder, timeoutErr, deploymentItem)
			// Set failure state to true
			g.GetDenyList().SetScalingItemOnList(deploymentItem, true, timeoutErr.msg, deploymentItem.DesiredReplicas)
			return timeoutErr
		default:
			// While not timeout
			time.Sleep(time.Second * 2)
			// Refresh the deploymentItem
			deploymentItem, err = GetRefreshedScalingItem(ctx, _client, deploymentItem)
			if err != nil {
				log.Error(err, "Error getting refreshed deploymentItem in wait for Readiness loop")
				// The deployment does not exist anymore. Not putting it in failure state.
				RegisterEvents(ctx, _client, recorder, nil, deploymentItem)
				g.GetDenyList().RemoveFromList(deploymentItem)
				return err
			}

			if deploymentItem.ReadyReplicas == stepReplicaCount || deploymentItem.SpecReplica == deploymentItem.ReadyReplicas {
				stay = false
			}
			// k8s can't handle the deployment for some reason. We can't scale
			if deploymentItem.ConditionReason == "ProgressDeadlineExceeded" {
				scaleErr := ScaleError{
					msg: "The deployment is in a failing state on the cluster! ProgressDeadlineExceeded!",
				}
				deploymentItem.IsBeingScaled = false
				g.GetDenyList().SetScalingItemOnList(deploymentItem, true, "ProgressDeadlineExceeded", desiredReplicaCount)
				RegisterEvents(ctx, _client, recorder, scaleErr, deploymentItem)
				return scaleErr
			}
		}
	}
	return nil
}
