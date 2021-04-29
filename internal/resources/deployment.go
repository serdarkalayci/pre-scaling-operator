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
	"github.com/prometheus/common/log"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

//DeploymentLister lists all deployments in a namespace
func DeploymentLister(ctx context.Context, _client client.Client, namespace string, OptInLabel map[string]string) (v1.DeploymentList, error) {

	deployments := v1.DeploymentList{}
	err := _client.List(ctx, &deployments, client.MatchingLabels(OptInLabel), client.InNamespace(namespace))
	if err != nil {
		return v1.DeploymentList{}, err
	}
	return deployments, nil

}

//DeploymentGetter returns the specific deployment data given a reconciliation request
func DeploymentGetter(ctx context.Context, _client client.Client, req ctrl.Request) (v1.Deployment, error) {

	deployment := v1.Deployment{}
	err := _client.Get(ctx, req.NamespacedName, &deployment)
	if err != nil {
		return v1.Deployment{}, err
	}
	return deployment, nil

}

//DeploymentScaler scales the deployment to the desired replica number
func DeploymentScaler(ctx context.Context, _client client.Client, deployment v1.Deployment, replicas int32, req reconcile.Request) error {

	if v, found := deployment.GetAnnotations()["scaler/allow-autoscaling"]; found {
		if v == "true" {
			if replicas <= *deployment.Spec.Replicas {
				return nil
			}
		}
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Don't spam the api in case of conflict error
		time.Sleep(time.Second * 1)

		// We need to get a newer version of the object from the client
		deployment, err := DeploymentGetter(ctx, _client, req)
		_ = deployment
		if err != nil {
			log.Error(err, "Error getting refreshed deployment in conflict resolution")
			return err
		}

		// Skip if we couldn't get the deployment
		if err == nil {
			deployment.Spec.Replicas = &replicas

			updateErr := _client.Update(ctx, &deployment, &client.UpdateOptions{})
			if updateErr != nil {
				return updateErr
			}
		}
		return err
	})
}

//DeploymentOptinLabel returns true if the optin-label is found and is true for the deployment
func DeploymentOptinLabel(deployment v1.Deployment) (bool, error) {

	return validations.OptinLabelExists(deployment.GetLabels())
}

func DeploymentStateReplicas(state states.State, deployment v1.Deployment) (sr.StateReplica, error) {
	log := ctrl.Log.
		WithValues("deployment", deployment.Name).
		WithValues("namespace", deployment.Namespace)
	stateReplicas, err := sr.NewStateReplicasFromAnnotations(deployment.GetAnnotations())
	if err != nil {
		log.WithValues("deployment", deployment.Name).
			WithValues("namespace", deployment.Namespace).
			Error(err, "Cannot calculate state replicas. Please check deployment annotations. Continuing.")
		return sr.StateReplica{}, err
	}
	// Now we have all the state settings, we can set the replicas for the deployment accordingly
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

func DeploymentStateReplicasList(state states.State, deployments v1.DeploymentList) ([]sr.StateReplica, error) {

	var stateReplicaList []sr.StateReplica
	var err error

	for _, deployment := range deployments.Items {
		log := ctrl.Log.
			WithValues("deployment", deployment.Name).
			WithValues("namespace", deployment.Namespace)
		stateReplicas, err := sr.NewStateReplicasFromAnnotations(deployment.GetAnnotations())
		if err != nil {
			log.WithValues("deployment", deployment.Name).
				WithValues("namespace", deployment.Namespace).
				Error(err, "Cannot calculate state replicas. Please check deployment annotations. Continuing.")
			return []sr.StateReplica{}, err
		}

		optIn, err := DeploymentOptinLabel(deployment)
		if err != nil {
			if strings.Contains(err.Error(), c.LabelNotFound) {
				return []sr.StateReplica{}, nil
			}
			log.Error(err, "Failed to validate the opt-in label")
			return []sr.StateReplica{}, err
		}

		// Now we have all the state settings, we can set the replicas for the deployment accordingly
		if !optIn {
			// the deployment opted out. We need to set back to default.
			log.Info("The deployment opted out. Will scale back to default")
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

func ScaleDeployment(ctx context.Context, _client client.Client, deployment v1.Deployment, stateReplica sr.StateReplica, rateLimitingEnabled bool) error {

	log := ctrl.Log.
		WithValues("deployment", deployment.Name).
		WithValues("namespace", deployment.Namespace)
	var req reconcile.Request
	deploymentItem := g.ConvertDeploymentToItem(deployment)
	req.NamespacedName.Namespace = deployment.Namespace
	req.NamespacedName.Name = deployment.Name
	var err error

	if g.GetDenyList().IsInConcurrentDenyList(deploymentItem) {
		log.Info("Waiting for the deployment to be off the denylist.")
		for stay, timeout := true, time.After(time.Second*120); stay; {
			select {
			case <-timeout:
				log.Info("Timeout reached! The deployment stayed on the denylist for too long. Couldn't reconcile this deployment!")
				return nil
			default:
				time.Sleep(time.Second * 10)
				if !g.GetDenyList().IsInConcurrentDenyList(deploymentItem) {
					// Refresh deployment to get a new object to reconcile

					deployment, err = DeploymentGetter(ctx, _client, req)
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
	oldReplicaCount = *deployment.Spec.Replicas
	desiredReplicaCount := stateReplica.Replicas

	if oldReplicaCount == stateReplica.Replicas {
		log.Info("No Update on deployment. Desired replica count already matches current.")
		return nil
	}

	var stepReplicaCount int32
	var stepCondition bool = true
	var retryErr error = nil
	log.Info("Putting deployment on denylist")
	g.GetDenyList().UpdateOrAppend(deploymentItem)
	if rateLimitingEnabled {
		// Loop step by step until deployment has reached desiredreplica count. Fail when the deployment update failed too many times
		for stepCondition && retryErr == nil {

			// decide if we need to step up or down
			oldReplicaCount = *deployment.Spec.Replicas
			if oldReplicaCount < desiredReplicaCount {
				stepReplicaCount = oldReplicaCount + 1
			} else {
				stepReplicaCount = oldReplicaCount - 1
			}

			retryErr = DeploymentScaler(ctx, _client, deployment, stepReplicaCount, req)

			if retryErr != nil {
				log.Error(retryErr, "Unable to scale the deployment, err: %v")
				g.GetDenyList().RemoveFromDenyList(deploymentItem)
				return retryErr
			}

			// Wait until deployment is ready for the next step
			for stay, timeout := true, time.After(time.Second*60); stay; {
				select {
				case <-timeout:
					stay = false
				default:
					time.Sleep(time.Second * 5)
					deployment, err = DeploymentGetter(ctx, _client, req)
					if err != nil {
						log.Error(err, "Error getting refreshed deployment in wait for Readiness loop")
						g.GetDenyList().RemoveFromDenyList(deploymentItem)
						return err
					}
					if deployment.Status.ReadyReplicas == stepReplicaCount {
						stay = false
					}
				}
			}

			// check if desired is reached
			if deployment.Status.ReadyReplicas == desiredReplicaCount {
				stepCondition = false
			}
		}
	} else {
		// Rapid scale. No Step Scale
		retryErr = DeploymentScaler(ctx, _client, deployment, desiredReplicaCount, req)

		if retryErr != nil {
			log.Error(retryErr, "Unable to scale the deployment, err: %v")
			g.GetDenyList().RemoveFromDenyList(deploymentItem)
			return retryErr
		}
	}
	log.WithValues("State", stateReplica.Name).
		WithValues("Desired Replica Count", stateReplica.Replicas).
		WithValues("Deployment Name", deployment.Name).
		WithValues("Namespace", deployment.Namespace).
		Info("Finished scaling deployment to desired replica count")
	g.GetDenyList().RemoveFromDenyList(deploymentItem)
	return nil
}

func LimitsNeededDeployment(deployment v1.Deployment, replicas int32) corev1.ResourceList {

	return math.Mul(math.ReplicaCalc(replicas, *deployment.Spec.Replicas), deployment.Spec.Template.Spec.Containers[0].Resources.Limits)
}

func LimitsNeededDeploymentList(deployments v1.DeploymentList, scaleReplicalist []sr.StateReplica) corev1.ResourceList {

	var limitsneeded corev1.ResourceList
	for i, deployment := range deployments.Items {
		limitsneeded = math.Add(limitsneeded, math.Mul(math.ReplicaCalc(scaleReplicalist[i].Replicas, *deployment.Spec.Replicas), deployment.Spec.Template.Spec.Containers[0].Resources.Limits))
	}
	return limitsneeded
}
