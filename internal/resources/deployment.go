package resources

import (
	"context"
	"strings"
	"time"

	c "github.com/containersol/prescale-operator/internal"
	sr "github.com/containersol/prescale-operator/internal/state_replicas"
	"github.com/containersol/prescale-operator/internal/states"
	"github.com/containersol/prescale-operator/internal/validations"
	"github.com/containersol/prescale-operator/pkg/utils/annotations"
	"github.com/containersol/prescale-operator/pkg/utils/math"
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
func DeploymentScaler(ctx context.Context, _client client.Client, deployment v1.Deployment, replicas int32) error {

	if v, found := deployment.GetAnnotations()["scaler/allow-autoscaling"]; found {
		if v == "true" {
			if replicas <= *deployment.Spec.Replicas {
				return nil
			}
		}
	}

	deployment.Spec.Replicas = &replicas
	err := _client.Update(ctx, &deployment, &client.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
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

func ScaleDeployment(ctx context.Context, _client client.Client, deployment v1.Deployment, stateReplica sr.StateReplica) error {
	log := ctrl.Log.
		WithValues("deployment", deployment.Name).
		WithValues("namespace", deployment.Namespace)

	var err error
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
	var req reconcile.Request
	req.NamespacedName.Namespace = deployment.Namespace
	req.NamespacedName.Name = deployment.Name
	// Loop step by step until deployment has reached desiredreplica count. Fail when the deployment update failed too many times
	for stepCondition && retryErr == nil {

		// decide if we need to step up or down
		oldReplicaCount = *deployment.Spec.Replicas
		if oldReplicaCount < desiredReplicaCount {
			stepReplicaCount = oldReplicaCount + 1
		} else {
			stepReplicaCount = oldReplicaCount - 1
		}


		retryErr = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			// keep/put the annotation on the deployment or remove it in last run
			if oldReplicaCount+1 == desiredReplicaCount || oldReplicaCount-1 == desiredReplicaCount {
				deployment = annotations.RemoveAnnotationFromDeployment(deployment, "scaler/step-scale-active")
			} else {
				deployment = annotations.PutAnnotationOnDeployment(deployment, "scaler/step-scale-active", "true")
			}

			// Don't spam the api in case of conflict error
			time.Sleep(time.Second * 2)

			updateErr := DeploymentScaler(ctx, _client, deployment, stepReplicaCount)

			// We need to get a newer version of the object from the client
			deployment, err = DeploymentGetter(ctx, _client, req)

			if err != nil {
				log.Error(err, "Error getting refreshed deployment in conflict resolution")
			}
			return updateErr
		})
		if retryErr != nil {
			log.Error(retryErr, "Unable to scale the deployment, err: %v")
		}

		// Wait until deployment is ready for the step
		for deployment.Status.ReadyReplicas != stepReplicaCount {
			time.Sleep(time.Second * 10)
			deployment, err = DeploymentGetter(ctx, _client, req)
			if err != nil {
				log.Error(err, "Error getting refreshed deployment in wait for Readiness loop")
			}
		}

		// check if desired is reached
		if deployment.Status.ReadyReplicas == desiredReplicaCount {
			log.WithValues("State", stateReplica.Name).
				WithValues("Desired Replica Count", stateReplica.Replicas).
				WithValues("Deployment Name", deployment.Name).
				WithValues("Namespace", deployment.Namespace).
				Info("Finished scaling deployment to desired replica count")
			stepCondition = false
		}
	}
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
