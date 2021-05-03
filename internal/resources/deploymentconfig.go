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
	v1 "github.com/openshift/api/apps/v1"
	"github.com/prometheus/common/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

//DeploymentConfigLister lists all deploymentconfigs in a namespace
func DeploymentConfigLister(ctx context.Context, _client client.Client, namespace string, OptInLabel map[string]string) (v1.DeploymentConfigList, error) {

	deploymentconfigs := v1.DeploymentConfigList{}
	err := _client.List(ctx, &deploymentconfigs, client.MatchingLabels(OptInLabel), client.InNamespace(namespace))
	if err != nil {
		return v1.DeploymentConfigList{}, err
	}
	return deploymentconfigs, nil

}

//DeploymentConfigGetter returns the specific deploymentconfig data given a reconciliation request
func DeploymentConfigGetter(ctx context.Context, _client client.Client, req ctrl.Request) (v1.DeploymentConfig, error) {

	deploymentconfig := v1.DeploymentConfig{}
	err := _client.Get(ctx, req.NamespacedName, &deploymentconfig)
	if err != nil {
		return v1.DeploymentConfig{}, err
	}
	return deploymentconfig, nil

}

//DeploymentConfigScaler scales the deploymentconfig to the desired replica number
func DeploymentConfigScaler(ctx context.Context, _client client.Client, deploymentConfig v1.DeploymentConfig, replicas int32, req reconcile.Request) error {

	if v, found := deploymentConfig.GetAnnotations()["scaler/allow-autoscaling"]; found {
		if v == "true" {
			if replicas <= deploymentConfig.Spec.Replicas {
				return nil
			}
		}
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Don't spam the api in case of conflict error
		time.Sleep(time.Second * 1)

		// We need to get a newer version of the object from the client
		deploymentConfig, err := DeploymentConfigGetter(ctx, _client, req)
		if err != nil {
			log.Error(err, "Error getting refreshed deploymentconfig in conflict resolution")
		}

		// Skip if we couldn't get the deploymentconfig
		if err == nil {
			deploymentConfig.Spec.Replicas = replicas

			updateErr := _client.Update(ctx, &deploymentConfig, &client.UpdateOptions{})
			if updateErr != nil {
				return updateErr
			}
		}
		return err
	})
}

//DeploymentConfigOptinLabel returns true if the optin-label is found and is true for the deploymentconfig
func DeploymentConfigOptinLabel(deploymentConfig v1.DeploymentConfig) (bool, error) {

	return validations.OptinLabelExists(deploymentConfig.GetLabels())
}

func DeploymentConfigStateReplicas(state states.State, deploymentconfig v1.DeploymentConfig) (sr.StateReplica, error) {
	log := ctrl.Log.
		WithValues("deploymentconfig", deploymentconfig.Name).
		WithValues("namespace", deploymentconfig.Namespace)
	stateReplicas, err := sr.NewStateReplicasFromAnnotations(deploymentconfig.GetAnnotations())
	if err != nil {
		log.WithValues("deploymentconfig", deploymentconfig.Name).
			WithValues("namespace", deploymentconfig.Namespace).
			Error(err, "Cannot calculate state replicas. Please check deploymentconfig annotations. Continuing.")
		return sr.StateReplica{}, err
	}
	// Now we have all the state settings, we can set the replicas for the deploymentconfig accordingly

	stateReplica, err := stateReplicas.GetState(state.Name)
	if err != nil {
		// TODO here we should do priority filtering, and go down one level of priority to find the lowest set one.
		// We will ignore any that are not set
		// log.WithValues("set states", stateReplicas).
		// 	WithValues("namespace state", state.Name).
		// 	Info("State could not be found")
		return sr.StateReplica{}, err
	}
	return stateReplica, nil
}

func DeploymentConfigStateReplicasList(state states.State, deploymentconfigs v1.DeploymentConfigList) ([]sr.StateReplica, error) {

	var stateReplicaList []sr.StateReplica
	var err error

	for _, deploymentconfig := range deploymentconfigs.Items {
		log := ctrl.Log.
			WithValues("deploymentconfig", deploymentconfig.Name).
			WithValues("namespace", deploymentconfig.Namespace)
		stateReplicas, err := sr.NewStateReplicasFromAnnotations(deploymentconfig.GetAnnotations())
		if err != nil {
			log.WithValues("deploymentconfig", deploymentconfig.Name).
				WithValues("namespace", deploymentconfig.Namespace).
				Error(err, "Cannot calculate state replicas. Please check deploymentconfig annotations. Continuing.")
			return []sr.StateReplica{}, err
		}

		optIn, err := DeploymentConfigOptinLabel(deploymentconfig)
		if err != nil {
			if strings.Contains(err.Error(), c.LabelNotFound) {
				return []sr.StateReplica{}, nil
			}
			log.Error(err, "Failed to validate the opt-in label")
			return []sr.StateReplica{}, err
		}

		// Now we have all the state settings, we can set the replicas for the deploymentconfig accordingly
		if !optIn {
			// the deploymentconfig opted out. We need to set back to default.
			log.Info("The deploymentconfig opted out. Will scale back to default")
			state.Name = c.DefaultReplicaAnnotation
		}

		stateReplica, err := stateReplicas.GetState(state.Name)
		if err != nil {
			// TODO here we should do priority filtering, and go down one level of priority to find the lowest set one.
			// We will ignore any that are not set
			// log.WithValues("set states", stateReplicas).
			// 	WithValues("namespace state", state.Name).
			// 	Info("State could not be found")
			return []sr.StateReplica{}, err
		}

		stateReplicaList = append(stateReplicaList, stateReplica)

	}

	return stateReplicaList, err
}

func ScaleDeploymentConfig(ctx context.Context, _client client.Client, deploymentconfig v1.DeploymentConfig, stateReplica sr.StateReplica, rateLimitingEnabled bool) error {
	log := ctrl.Log.
		WithValues("deploymentconfig", deploymentconfig.Name).
		WithValues("namespace", deploymentconfig.Namespace)

	var req reconcile.Request
	deploymentItem := g.ConvertDeploymentConfigToItem(deploymentconfig)
	req.NamespacedName.Namespace = deploymentconfig.Namespace
	req.NamespacedName.Name = deploymentconfig.Name
	var err error

	if g.GetDenyList().IsInConcurrentDenyList(deploymentItem) {
		log.Info("Waiting for the deploymentconfig to be off the denylist.")
		for stay, timeout := true, time.After(time.Second*120); stay; {
			select {
			case <-timeout:
				log.Info("Timeout reached! The deploymentconfig stayed on the denylist for too long. Couldn't reconcile this deploymentconfig!")
				return nil
			default:
				time.Sleep(time.Second * 10)
				if !g.GetDenyList().IsInConcurrentDenyList(deploymentItem) {
					// Refresh deploymentconfig to get a new object to reconcile

					deploymentconfig, err = DeploymentConfigGetter(ctx, _client, req)
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
	oldReplicaCount = deploymentconfig.Spec.Replicas
	desiredReplicaCount := stateReplica.Replicas

	if oldReplicaCount == stateReplica.Replicas {
		log.Info("No Update on deploymentconfig. Desired replica count already matches current.")
		return nil
	}

	var stepReplicaCount int32
	var stepCondition bool = true
	var retryErr error = nil
	log.Info("Adding to denylist")
	g.GetDenyList().SetDeploymentInfoOnDenyList(deploymentItem, false, "", int(desiredReplicaCount))
	if rateLimitingEnabled {
		for stepCondition {

			desiredReplicaCount = int32(g.GetDenyList().GetDesiredReplicasFromDenyList(deploymentItem))
			// decide if we need to step up or down
			oldReplicaCount = deploymentconfig.Spec.Replicas
			if oldReplicaCount < desiredReplicaCount {
				stepReplicaCount = oldReplicaCount + 1
			} else if oldReplicaCount > desiredReplicaCount {
				stepReplicaCount = oldReplicaCount - 1
			} else if oldReplicaCount == desiredReplicaCount {
				log.Info("Finished scaling. Leaving early due to an update from another goroutine.")
				g.GetDenyList().RemoveFromDenyList(deploymentItem)
				return nil
			}
			// Do the scaling
			retryErr = DeploymentConfigScaler(ctx, _client, deploymentconfig, stepReplicaCount, req)
			if retryErr != nil {
				log.Error(retryErr, "Unable to scale the deploymentconfig, err: %v")
				g.GetDenyList().RemoveFromDenyList(deploymentItem)
				return retryErr
			}

			// Wait until deploymentconfig is ready for the next step
			for stay, timeout := true, time.After(time.Second*60); stay; {
				select {
				case <-timeout:
					stay = false
				default:
					time.Sleep(time.Second * 5)
					deploymentconfig, err = DeploymentConfigGetter(ctx, _client, req)
					if err != nil {
						log.Error(err, "Error getting refreshed deploymentconfig in wait for Readiness loop")
						g.GetDenyList().RemoveFromDenyList(deploymentItem)
						return err
					}
					if deploymentconfig.Status.ReadyReplicas == stepReplicaCount {
						stay = false
					}
				}
			}

			// check if desired is reached
			if deploymentconfig.Status.ReadyReplicas == desiredReplicaCount {
				stepCondition = false
			}
		}
	} else {
		// Rapid scale. No Step Scale
		retryErr = DeploymentConfigScaler(ctx, _client, deploymentconfig, desiredReplicaCount, req)

		if retryErr != nil {
			log.Error(retryErr, "Unable to scale the deploymentconfig, err: %v")
			g.GetDenyList().RemoveFromDenyList(deploymentItem)
			return retryErr
		}
	}

	log.WithValues("State", stateReplica.Name).
		WithValues("Desired Replica Count", stateReplica.Replicas).
		WithValues("Deployment Name", deploymentconfig.Name).
		WithValues("Namespace", deploymentconfig.Namespace).
		Info("Finished scaling deploymentconfig to desired replica count")

	g.GetDenyList().RemoveFromDenyList(deploymentItem)
	return nil
}

func LimitsNeededDeploymentConfig(deploymentConfig v1.DeploymentConfig, replicas int32) corev1.ResourceList {

	return math.Mul(math.ReplicaCalc(replicas, deploymentConfig.Spec.Replicas), deploymentConfig.Spec.Template.Spec.Containers[0].Resources.Limits)
}

func LimitsNeededDeploymentConfigList(deploymentConfigs v1.DeploymentConfigList, scaleReplicalist []sr.StateReplica) corev1.ResourceList {

	var limitsneeded corev1.ResourceList
	for i, deploymentConfig := range deploymentConfigs.Items {
		limitsneeded = math.Add(limitsneeded, math.Mul(math.ReplicaCalc(scaleReplicalist[i].Replicas, deploymentConfig.Spec.Replicas), deploymentConfig.Spec.Template.Spec.Containers[0].Resources.Limits))
	}

	return limitsneeded
}
