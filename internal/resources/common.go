package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	c "github.com/containersol/prescale-operator/internal"
	sr "github.com/containersol/prescale-operator/internal/state_replicas"
	"github.com/containersol/prescale-operator/internal/states"
	"github.com/containersol/prescale-operator/internal/validations"
	g "github.com/containersol/prescale-operator/pkg/utils/global"
	"github.com/containersol/prescale-operator/pkg/utils/math"
	ocv1 "github.com/openshift/api/apps/v1"
	"github.com/prometheus/common/log"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ScaleError struct {
	msg string
}

func (err ScaleError) Error() string {
	return err.msg
}

func DoScaling(ctx context.Context, _client client.Client, scalingItem g.ScalingInfo, replicas int32) error {

	if v, found := scalingItem.Annotations["scaler/allow-autoscaling"]; found {
		if v == "true" {
			if replicas <= int32(scalingItem.SpecReplica) {
				return nil
			}
		}
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		time.Sleep(time.Second * 1)

		// We need to get a newer version of the object from the client
		deploymentItem, err := GetRefreshedScalingItem(ctx, _client, scalingItem)
		if err != nil || (deploymentItem.Name == "" || deploymentItem.Namespace == "") {
			log.Error(err, fmt.Sprintf("Error getting refreshed deploymentItem in conflict resolution. Name: %s , Namespace: %s", scalingItem.Name, scalingItem.Namespace))
		}

		// Skip if we couldn't get the deploymentItem
		if err == nil {
			deploymentItem.SpecReplica = replicas

			var updateErr error = nil
			if !deploymentItem.Failure {
				exists, labelErr := OptinLabel(deploymentItem)
				if !exists || labelErr != nil {
					return DeploymentScaleError{
						msg: "Error scaling the Deployment! The deployment is opted out!",
					}
				}
				updateErr = UpdateScalingItem(ctx, _client, deploymentItem)
			} else {
				return DeploymentScaleError{
					msg: "Error scaling the Deployment!. The Deployment is in failure state! Message: " + deploymentItem.FailureMessage,
				}
			}
			if updateErr != nil {
				return updateErr
			}
		}
		return err
	})
}

//DeploymentOptinLabel returns true if the optin-label is found and is true for the deploymentItem
func OptinLabel(deploymentItem g.ScalingInfo) (bool, error) {

	return validations.OptinLabelExists(deploymentItem.Labels)
}

func StateReplicas(state states.State, deploymentItem g.ScalingInfo) (sr.StateReplica, error) {
	log := ctrl.Log.
		WithValues("deploymentItem", deploymentItem.Name).
		WithValues("namespace", deploymentItem.Namespace)
	stateReplicas, err := sr.NewStateReplicasFromAnnotations(deploymentItem.Annotations)
	if err != nil {
		log.WithValues("deploymentItem", deploymentItem.Name).
			WithValues("namespace", deploymentItem.Namespace).
			Error(err, "Cannot calculate state replicas. Please check deploymentItem annotations. Continuing.")
		return sr.StateReplica{}, err
	}
	// Now we have all the state settings, we can set the replicas for the deploymentItem accordingly
	stateReplica, err := stateReplicas.GetState(state.Name)
	if err != nil {
		// TODO here we should do priority filtering, and go down one level of priority to find the lowest set one.
		// We will ignore any that are not set
		log.WithValues("set states", stateReplicas).
			WithValues("namespace state", state.Name).
			Info("State could not be found")
		return sr.StateReplica{}, err
	}
	return stateReplica, nil
}

func StateReplicasList(state states.State, deployments []g.ScalingInfo) ([]sr.StateReplica, error) {

	var stateReplicaList []sr.StateReplica
	var err error

	for _, deploymentItem := range deployments {
		log := ctrl.Log.
			WithValues("deploymentItem", deploymentItem.Name).
			WithValues("namespace", deploymentItem.Namespace)
		stateReplicas, err := sr.NewStateReplicasFromAnnotations(deploymentItem.Annotations)
		if err != nil {
			log.WithValues("deploymentItem", deploymentItem.Name).
				WithValues("namespace", deploymentItem.Namespace).
				Error(err, "Cannot calculate state replicas. Please check deploymentItem annotations. Continuing.")
			return []sr.StateReplica{}, err
		}

		optIn, err := OptinLabel(deploymentItem)
		if err != nil {
			if strings.Contains(err.Error(), c.LabelNotFound) {
				return []sr.StateReplica{}, nil
			}
			log.Error(err, "Failed to validate the opt-in label")
			return []sr.StateReplica{}, err
		}

		// Now we have all the state settings, we can set the replicas for the deploymentItem accordingly
		if !optIn {
			// the deploymentItem opted out. We need to set back to default.
			log.Info("The deploymentItem opted out. Will scale back to default")
			state.Name = c.DefaultReplicaAnnotation
		}

		stateReplica, err := stateReplicas.GetState(state.Name)
		if err != nil {
			// TODO here we should do priority filtering, and go down one level of priority to find the lowest set one.
			// We will ignore any that are not set
			log.WithValues("set states", stateReplicas).
				WithValues("namespace state", state.Name).
				Info("State could not be found")
			return []sr.StateReplica{}, err
		}

		stateReplicaList = append(stateReplicaList, stateReplica)

	}

	return stateReplicaList, err
}

// Main function to make scaling decisions. The step scaler scales 1 by 1 towards the desired replica count.
func ScaleOrStepScale(ctx context.Context, _client client.Client, deploymentItem g.ScalingInfo, stateReplica sr.StateReplica, whereFrom string, recorder record.EventRecorder) error {

	log := ctrl.Log.
		WithValues("deploymentItem", deploymentItem.Name).
		WithValues("namespace", deploymentItem.Namespace)
	var err error

	var oldReplicaCount int32
	oldReplicaCount = deploymentItem.SpecReplica
	desiredReplicaCount := stateReplica.Replicas

	// We need to skip this check in case of failure in order to get a new object from DoScaling() to check on the state on the cluster.
	if oldReplicaCount == stateReplica.Replicas && !deploymentItem.Failure {
		log.Info("No Update on deploymentItem. Desired replica count already matches current.")
		return nil
	}

	var stepReplicaCount int32
	var stepCondition bool = true
	var retryErr error = nil

	stepReplicaCount = deploymentItem.SpecReplica
	rateLimitingEnabled := states.GetStepScaleSetting(ctx, _client)
	log.Info("Putting deploymentItem on denylist")
	deploymentItem.IsBeingScaled = true
	g.GetDenyList().SetScalingItemOnList(deploymentItem, deploymentItem.Failure, deploymentItem.FailureMessage, desiredReplicaCount)
	if rateLimitingEnabled {
		log.WithValues("Deployment: ", deploymentItem.Name).
			WithValues("Namespace: ", deploymentItem.Namespace).
			WithValues("Wherefrom: ", whereFrom).
			WithValues("DesiredReplicaCount", desiredReplicaCount).
			Info("Going into step scaler..")
		// Loop step by step until deploymentItem has reached desiredreplica count.
		for stepCondition {

			deploymentItem, _ = g.GetDenyList().GetDeploymentInfoFromList(deploymentItem)
			desiredReplicaCount = deploymentItem.DesiredReplicas
			oldReplicaCount = deploymentItem.SpecReplica
			if desiredReplicaCount == -1 {
				desiredReplicaCount = stateReplica.Replicas
			}

			// Wait until deploymentItem is ready for the next step and check if it's failing for some reason
			waitTime := time.Duration(time.Duration(deploymentItem.ProgressDeadline))*time.Second + time.Second
			for stay, timeout := true, time.After(waitTime); stay; {
				select {
				case <-timeout:
					timeoutErr := ScaleError{
						msg: fmt.Sprintf("Message on the cluster: %s | The operator decided that it can't scale that deployment or deploymentconfig!", deploymentItem.ConditionReason),
					}
					deploymentItem.IsBeingScaled = false
					RegisterEvents(ctx, _client, recorder, timeoutErr, deploymentItem)
					g.GetDenyList().SetScalingItemOnList(deploymentItem, true, timeoutErr.msg, desiredReplicaCount)
					return timeoutErr
				default:
					time.Sleep(time.Second * 2)
					deploymentItem, err = GetRefreshedScalingItem(ctx, _client, deploymentItem)
					if err != nil {
						log.Error(err, "Error getting refreshed deploymentItem in wait for Readiness loop")
						// The deployment does not exist anymore. Not putting it in failure state.
						RegisterEvents(ctx, _client, recorder, nil, deploymentItem)
						g.GetDenyList().RemoveFromList(deploymentItem)
						return err
					}

					if deploymentItem.ReadyReplicas == stepReplicaCount {
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

			// decide if we need to step up or down
			if oldReplicaCount < desiredReplicaCount {
				stepReplicaCount = oldReplicaCount + 1
			} else if oldReplicaCount > desiredReplicaCount {
				stepReplicaCount = oldReplicaCount - 1
			}
			// else if oldReplicaCount == desiredReplicaCount {
			// 	log.Info("Finished scaling. Leaving early due to an update from another goroutine.")
			// 	RegisterEvents(ctx, _client, recorder, nil, deploymentItem)
			// 	g.GetDenyList().RemoveFromList(deploymentItem)
			// 	return nil
			// }

			log.WithValues("ScalingItem: ", deploymentItem.Name).
				WithValues("Namespace: ", deploymentItem.Namespace).
				WithValues("Stepreplicacount", stepReplicaCount).
				WithValues("Oldreplicacount", oldReplicaCount).
				WithValues("Desiredreplicacount", desiredReplicaCount).
				WithValues("Wherefrom: ", whereFrom).
				Info("Step Scaling!")

			retryErr = DoScaling(ctx, _client, deploymentItem, stepReplicaCount)

			if retryErr != nil {
				//log.Error(retryErr, "Unable to scale the deploymentItem, err: %v")
				deploymentItem.IsBeingScaled = false
				g.GetDenyList().SetScalingItemOnList(deploymentItem, true, retryErr.Error(), stateReplica.Replicas)
				RegisterEvents(ctx, _client, recorder, retryErr, deploymentItem)
				return retryErr
			}

			// check if desired is reached from a fresh item
			deploymentItem, _ = GetRefreshedScalingItem(ctx, _client, deploymentItem)
			if deploymentItem.ReadyReplicas == deploymentItem.DesiredReplicas {
				stepCondition = false
			}
		}
	} else {
		// Rapid scale. No Step Scale
		retryErr = DoScaling(ctx, _client, deploymentItem, desiredReplicaCount)

		if retryErr != nil {
			//log.Error(retryErr, "Unable to scale the deploymentItem, err: %v")
			deploymentItem.IsBeingScaled = false
			g.GetDenyList().SetScalingItemOnList(deploymentItem, true, retryErr.Error(), stateReplica.Replicas)
			RegisterEvents(ctx, _client, recorder, retryErr, deploymentItem)
			return retryErr
		}
	}

	log.WithValues("Desired Replica Count", deploymentItem.DesiredReplicas).
		WithValues("Deployment Name", deploymentItem.Name).
		WithValues("Namespace", deploymentItem.Namespace).
		Info("Finished scaling deploymentItem to desired replica count")

	// Success
	RegisterEvents(ctx, _client, recorder, nil, deploymentItem)
	g.GetDenyList().RemoveFromList(deploymentItem)
	return nil
}

func LimitsNeeded(deploymentItem g.ScalingInfo, replicas int32) corev1.ResourceList {

	return math.Mul(math.ReplicaCalc(replicas, deploymentItem.SpecReplica), deploymentItem.ResourceList)
}

func LimitsNeededList(deployments []g.ScalingInfo, scaleReplicalist []sr.StateReplica) corev1.ResourceList {

	var limitsneeded corev1.ResourceList
	for i, deploymentItem := range deployments {
		limitsneeded = math.Add(limitsneeded, math.Mul(math.ReplicaCalc(scaleReplicalist[i].Replicas, deploymentItem.SpecReplica), deploymentItem.ResourceList))
	}
	return limitsneeded
}

func GetRefreshedScalingItemSetError(ctx context.Context, _client client.Client, deploymentInfo g.ScalingInfo, failure bool) (g.ScalingInfo, error) {
	item, err := GetRefreshedScalingItem(ctx, _client, deploymentInfo)
	return g.GetDenyList().SetScalingItemOnList(item, failure, "", -1), err
}

// Returns a new scaling item from the cluster
func GetRefreshedScalingItem(ctx context.Context, _client client.Client, deploymentInfo g.ScalingInfo) (g.ScalingInfo, error) {
	// First we need to get an updated version from the list
	deploymentInfo, _ = g.GetDenyList().GetDeploymentInfoFromList(deploymentInfo)

	var req reconcile.Request
	req.NamespacedName.Namespace = deploymentInfo.Namespace
	req.NamespacedName.Name = deploymentInfo.Name
	itemToReturn := g.ScalingInfo{}
	if deploymentInfo.ScalingItemType.ItemTypeName == "DeploymentConfig" {
		deploymentconfig := ocv1.DeploymentConfig{}
		err := _client.Get(ctx, req.NamespacedName, &deploymentconfig)
		if err != nil {
			return g.ScalingInfo{}, err
		}
		itemToReturn = g.ConvertDeploymentConfigToItem(deploymentconfig)
	} else {
		// deployment
		deployment := v1.Deployment{}
		err := _client.Get(ctx, req.NamespacedName, &deployment)
		if err != nil {
			return g.ScalingInfo{}, err
		}
		itemToReturn = g.ConvertDeploymentToItem(deployment)
	}
	// Refresh the item on the list as well
	itemToReturn.IsBeingScaled = deploymentInfo.IsBeingScaled
	g.GetDenyList().SetScalingItemOnList(itemToReturn, itemToReturn.Failure, itemToReturn.FailureMessage, deploymentInfo.DesiredReplicas)
	item, _ := g.GetDenyList().GetDeploymentInfoFromList(itemToReturn)
	return item, nil
}

//DeploymentLister lists all deployments in a namespace
func ScalingItemLister(ctx context.Context, _client client.Client, namespace string, OptInLabel map[string]string) ([]g.ScalingInfo, error) {

	returnList := []g.ScalingInfo{}
	deployments := v1.DeploymentList{}
	deploymentconfigs := ocv1.DeploymentConfigList{}
	err := _client.List(ctx, &deployments, client.MatchingLabels(OptInLabel), client.InNamespace(namespace))
	if err != nil {
		return []g.ScalingInfo{}, err
	}

	if c.OpenshiftCluster {
		err := _client.List(ctx, &deploymentconfigs, client.MatchingLabels(OptInLabel), client.InNamespace(namespace))
		if err != nil {
			return []g.ScalingInfo{}, err
		}
	}

	for _, deployment := range deployments.Items {
		returnList = append(returnList, g.ConvertDeploymentToItem(deployment))
	}

	for _, deploymentConfig := range deploymentconfigs.Items {
		returnList = append(returnList, g.ConvertDeploymentConfigToItem(deploymentConfig))
	}

	return returnList, nil

}

func UpdateScalingItem(ctx context.Context, _client client.Client, deploymentItem g.ScalingInfo) error {
	var req reconcile.Request
	req.NamespacedName.Namespace = deploymentItem.Namespace
	req.NamespacedName.Name = deploymentItem.Name

	var updateErr error = nil
	var getErr error = nil
	deployment := v1.Deployment{}
	deploymentConfig := ocv1.DeploymentConfig{}

	if deploymentItem.ScalingItemType.ItemTypeName == "DeploymentConfig" {
		deploymentConfig, getErr = DeploymentConfigGetter(ctx, _client, req)
		if getErr != nil {
			return getErr
		}
		deploymentConfig.Spec.Replicas = deploymentItem.SpecReplica
		updateErr = _client.Update(ctx, &deploymentConfig, &client.UpdateOptions{})
	} else {
		deployment, getErr = DeploymentGetter(ctx, _client, req)
		if getErr != nil {
			return getErr
		}
		deployment.Spec.Replicas = &deploymentItem.SpecReplica
		updateErr = _client.Update(ctx, &deployment, &client.UpdateOptions{})
	}

	return updateErr
}

func RegisterEvents(ctx context.Context, _client client.Client, recorder record.EventRecorder, scalerErr error, scalingItem g.ScalingInfo) {
	// refresh the item to get newest replica count
	scalingItem, _ = g.GetDenyList().GetDeploymentInfoFromList(scalingItem)
	if scalingItem.ScalingItemType.ItemTypeName == "DeploymentConfig" {
		deplConf := ocv1.DeploymentConfig{}
		deplConf, getErr := DeploymentConfigGetterByScaleItem(ctx, _client, scalingItem)
		if getErr == nil {
			if scalerErr != nil {
				recorder.Event(deplConf.DeepCopyObject(), "Warning", "Deploymentconfig scale error", scalerErr.Error()+" | "+fmt.Sprintf("Failed to scale the Deploymentconfig to %d replicas. Stuck on: %d replicas", scalingItem.DesiredReplicas, deplConf.Spec.Replicas))
			} else {
				recorder.Event(deplConf.DeepCopyObject(), "Normal", "Deploymentconfig scaled", fmt.Sprintf("Successfully scaled the Deploymentconfig to %d replicas", deplConf.Spec.Replicas))
			}
		}
	} else {
		depl := v1.Deployment{}
		depl, getErr := DeploymentGetterByScaleItem(ctx, _client, scalingItem)
		if getErr == nil {
			if scalerErr != nil {
				recorder.Event(depl.DeepCopyObject(), "Warning", "Deployment scale error", scalerErr.Error()+" | "+fmt.Sprintf("Failed to scale the Deployment to %d replicas. Stuck on: %d replicas", scalingItem.DesiredReplicas, *depl.Spec.Replicas))
			} else {
				recorder.Event(depl.DeepCopyObject(), "Normal", "Deployment scaled", fmt.Sprintf("Successfully scaled the Deployment to %d replicas", *depl.Spec.Replicas))
			}
		}

	}

}
