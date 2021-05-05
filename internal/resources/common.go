package resources

import (
	"context"
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

//DeploymentScaler scales the deploymentItem to the desired replica number
func Scaler(ctx context.Context, _client client.Client, deploymentItem g.DeploymentInfo, replicas int32) error {

	if v, found := deploymentItem.Annotations["scaler/allow-autoscaling"]; found {
		if v == "true" {
			if replicas <= int32(deploymentItem.SpecReplica) {
				return nil
			}
		}
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		time.Sleep(time.Second * 1)

		// We need to get a newer version of the object from the client
		deploymentItem, err := GetDeploymentItem(ctx, _client, deploymentItem)
		if err != nil {
			log.Error(err, "Error getting refreshed deploymentItem in conflict resolution")
		}

		// Skip if we couldn't get the deploymentItem
		if err == nil {
			deploymentItem.SpecReplica = replicas

			deploymentItem := g.DeploymentInfo{
				Name:      deploymentItem.Name,
				Namespace: deploymentItem.Namespace,
			}
			var updateErr error = nil
			if !g.GetDenyList().IsDeploymentInFailureState(deploymentItem) {
				//updateErr = _client.Update(ctx, &deploymentItem, &client.UpdateOptions{})
				updateErr = UpdateDeploymentOrDeploymentConfig(ctx, _client, deploymentItem)
			} else {
				return DeploymentScaleError{
					msg: "Error scaling the Deployment!. The Deployment is in failure state!",
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
func OptinLabel(deploymentItem g.DeploymentInfo) (bool, error) {

	return validations.OptinLabelExists(deploymentItem.Labels)
}

func StateReplicas(state states.State, deploymentItem g.DeploymentInfo) (sr.StateReplica, error) {
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

func StateReplicasList(state states.State, deployments v1.DeploymentList) ([]sr.StateReplica, error) {

	var stateReplicaList []sr.StateReplica
	var err error

	for _, deploymentItem := range deployments.Items {
		log := ctrl.Log.
			WithValues("deploymentItem", deploymentItem.Name).
			WithValues("namespace", deploymentItem.Namespace)
		stateReplicas, err := sr.NewStateReplicasFromAnnotations(deploymentItem.GetAnnotations())
		if err != nil {
			log.WithValues("deploymentItem", deploymentItem.Name).
				WithValues("namespace", deploymentItem.Namespace).
				Error(err, "Cannot calculate state replicas. Please check deploymentItem annotations. Continuing.")
			return []sr.StateReplica{}, err
		}

		optIn, err := DeploymentOptinLabel(deploymentItem)
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

func Scale(ctx context.Context, _client client.Client, deploymentItem g.DeploymentInfo, stateReplica sr.StateReplica, whereFrom string) error {

	log := ctrl.Log.
		WithValues("deploymentItem", deploymentItem.Name).
		WithValues("namespace", deploymentItem.Namespace)
	var err error

	if g.GetDenyList().IsInConcurrentList(deploymentItem) {
		log.Info("Waiting for the deploymentItem to be off the denylist.")
		for stay, timeout := true, time.After(time.Second*120); stay; {
			select {
			case <-timeout:
				log.Info("Timeout reached! The deploymentItem stayed on the denylist for too long. Couldn't reconcile this deploymentItem!")
				return nil
			default:
				time.Sleep(time.Second * 10)
				if !g.GetDenyList().IsInConcurrentList(deploymentItem) {
					// Refresh deploymentItem to get a new object to reconcile

					deploymentItem, err = GetDeploymentItem(ctx, _client, deploymentItem)
					if err != nil {
						log.Error(err, "Deployment waited to be out of denylist but couldn't get a refreshed object to Reconcile.")
						return nil
					}
					stay = false
				}
			}
		}
	}

	var oldReplicaCount int32
	oldReplicaCount = deploymentItem.SpecReplica
	desiredReplicaCount := stateReplica.Replicas

	// This might not be necessary anymore
	if oldReplicaCount == stateReplica.Replicas {
		log.Info("No Update on deploymentItem. Desired replica count already matches current.")
		return nil
	}

	var stepReplicaCount int32
	var stepCondition bool = true
	var retryErr error = nil
	rateLimitingEnabled := states.GetStepScaleSetting(ctx, _client)
	log.Info("Putting deploymentItem on denylist")
	g.GetDenyList().SetDeploymentInfoOnList(deploymentItem, false, "", desiredReplicaCount)
	if rateLimitingEnabled {
		log.WithValues("Deployment: ", deploymentItem.Name).
			WithValues("Namespace: ", deploymentItem.Namespace).
			WithValues("DesiredReplicaount: ", deploymentItem.DesiredReplicas).
			WithValues("Wherefrom: ", whereFrom).
			Info("Going into step scaler..")
		// Loop step by step until deploymentItem has reached desiredreplica count. Fail when the deploymentItem update failed too many times
		for stepCondition {

			desiredReplicaCount = int32(g.GetDenyList().GetDesiredReplicasFromList(deploymentItem))
			// decide if we need to step up or down
			oldReplicaCount = deploymentItem.SpecReplica
			if oldReplicaCount < desiredReplicaCount {
				stepReplicaCount = oldReplicaCount + 1
			} else if oldReplicaCount > desiredReplicaCount {
				stepReplicaCount = oldReplicaCount - 1
			} else if oldReplicaCount == desiredReplicaCount {
				log.Info("Finished scaling. Leaving early due to an update from another goroutine.")
				g.GetDenyList().RemoveFromList(deploymentItem)
				return nil
			}
			log.WithValues("Deployment: ", deploymentItem.Name).
				WithValues("Namespace: ", deploymentItem.Namespace).
				WithValues("DesiredReplicaount on item:  ", deploymentItem.DesiredReplicas).
				WithValues("Desiredreplicacount", desiredReplicaCount).
				WithValues("Stepreplicacount", stepReplicaCount).
				WithValues("Wherefrom: ", whereFrom).
				Info("Step Scaling!")

			retryErr = Scaler(ctx, _client, deploymentItem, stepReplicaCount)

			if retryErr != nil {
				log.Error(retryErr, "Unable to scale the deploymentItem, err: %v")
				g.GetDenyList().RemoveFromList(deploymentItem)
				return retryErr
			}

			// Wait until deploymentItem is ready for the next step
			for stay, timeout := true, time.After(time.Second*60); stay; {
				select {
				case <-timeout:
					stay = false
				default:
					time.Sleep(time.Second * 5)
					deploymentItem, err = GetDeploymentItem(ctx, _client, deploymentItem)
					if err != nil {
						log.Error(err, "Error getting refreshed deploymentItem in wait for Readiness loop")
						g.GetDenyList().RemoveFromList(deploymentItem)
						return err
					}
					if deploymentItem.ReadyReplicas == stepReplicaCount {
						stay = false
					}
				}
			}

			// check if desired is reached
			if deploymentItem.ReadyReplicas == desiredReplicaCount {
				stepCondition = false
			}
		}
	} else {
		// Rapid scale. No Step Scale
		retryErr = Scaler(ctx, _client, deploymentItem, desiredReplicaCount)

		if retryErr != nil {
			log.Error(retryErr, "Unable to scale the deploymentItem, err: %v")
			g.GetDenyList().RemoveFromList(deploymentItem)
			return retryErr
		}
	}
	log.WithValues("State", stateReplica.Name).
		WithValues("Desired Replica Count", stateReplica.Replicas).
		WithValues("Deployment Name", deploymentItem.Name).
		WithValues("Namespace", deploymentItem.Namespace).
		Info("Finished scaling deploymentItem to desired replica count")
	g.GetDenyList().RemoveFromList(deploymentItem)
	return nil
}

func LimitsNeeded(deploymentItem g.DeploymentInfo, replicas int32) corev1.ResourceList {

	return math.Mul(math.ReplicaCalc(replicas, deploymentItem.SpecReplica), deploymentItem.ResourceList)
}

func LimitsNeededList(deployments v1.DeploymentList, scaleReplicalist []sr.StateReplica) corev1.ResourceList {

	var limitsneeded corev1.ResourceList
	for i, deploymentItem := range deployments.Items {
		limitsneeded = math.Add(limitsneeded, math.Mul(math.ReplicaCalc(scaleReplicalist[i].Replicas, *deploymentItem.Spec.Replicas), deploymentItem.Spec.Template.Spec.Containers[0].Resources.Limits))
	}
	return limitsneeded
}

func GetDeploymentItem(ctx context.Context, _client client.Client, deploymentInfo g.DeploymentInfo) (g.DeploymentInfo, error) {
	var req reconcile.Request
	req.NamespacedName.Namespace = deploymentInfo.Namespace
	req.NamespacedName.Name = deploymentInfo.Name
	itemToReturn := g.DeploymentInfo{}
	if deploymentInfo.IsDeploymentConfig {
		deploymentconfig := ocv1.DeploymentConfig{}
		err := _client.Get(ctx, req.NamespacedName, &deploymentconfig)
		if err != nil {
			return g.DeploymentInfo{}, err
		}
		itemToReturn = g.ConvertDeploymentConfigToItem(deploymentconfig)
	} else {
		// deployment
		deployment := v1.Deployment{}
		err := _client.Get(ctx, req.NamespacedName, &deployment)
		if err != nil {
			return g.DeploymentInfo{}, err
		}
		itemToReturn = g.ConvertDeploymentToItem(deployment)
	}
	return itemToReturn, nil
}

// func GetDeploymentConfig(ctx context.Context, _client client.Client, req ctrl.Request) (g.DeploymentInfo, error) {
// 	if
// 	// deployment := v1.Deployment{}
// 	// err := _client.Get(ctx, req.NamespacedName, &deployment)
// 	// if err != nil {
// 	// 	return v1.Deployment{}, err
// 	// }
// 	return g.DeploymentInfo{}, nil

// }

// func GetDeployment(ctx context.Context, _client client.Client, req ctrl.Request) (g.DeploymentInfo, error) {
// 	if
// 	// deployment := v1.Deployment{}
// 	// err := _client.Get(ctx, req.NamespacedName, &deployment)
// 	// if err != nil {
// 	// 	return v1.Deployment{}, err
// 	// }
// 	return g.DeploymentInfo{}, nil

// }

func UpdateDeploymentOrDeploymentConfig(ctx context.Context, _client client.Client, deploymentItem g.DeploymentInfo) error {
	// deployment := v1.Deployment{}
	// err := _client.Get(ctx, req.NamespacedName, &deployment)
	// if err != nil {
	// 	return v1.Deployment{}, err
	// }
	return nil

}
