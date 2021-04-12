package resources

import (
	"context"
	"strings"

	c "github.com/containersol/prescale-operator/internal"
	sr "github.com/containersol/prescale-operator/internal/state_replicas"
	"github.com/containersol/prescale-operator/internal/states"
	"github.com/containersol/prescale-operator/internal/validations"
	"github.com/containersol/prescale-operator/pkg/utils/math"
	v1 "github.com/openshift/api/apps/v1"
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
func DeploymentConfigScaler(ctx context.Context, _client client.Client, deploymentConfig v1.DeploymentConfig, replicas int32) error {

	// if v, found := deploymentConfig.GetAnnotations()["scaler/allow-autoscaling"]; found {
	// 	if v == "true" {
	// 		if replicas <= deploymentConfig.Spec.Replicas {
	// 			return nil
	// 		}
	// 	}
	// }

	deploymentConfig.Spec.Replicas = replicas
	err := _client.Update(ctx, &deploymentConfig, &client.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

//DeploymentConfigOptinLabel returns true if the optin-label is found and is true for the deploymentconfig
func DeploymentConfigOptinLabel(deploymentConfig v1.DeploymentConfig) (bool, error) {

	return validations.OptinLabelExists(deploymentConfig.GetLabels())
}

func DeploymentConfigStateReplicas(state states.State, deployment v1.DeploymentConfig, optIn bool) (sr.StateReplica, error) {
	log := ctrl.Log.
		WithValues("deploymentconfig", deployment.Name).
		WithValues("namespace", deployment.Namespace)
	stateReplicas, err := sr.NewStateReplicasFromAnnotations(deployment.GetAnnotations())
	if err != nil {
		log.WithValues("deploymentconfig", deployment.Name).
			WithValues("namespace", deployment.Namespace).
			Error(err, "Cannot calculate state replicas. Please check deploymentconfig annotations. Continuing.")
		return sr.StateReplica{}, err
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
		log.WithValues("set states", stateReplicas).
			WithValues("namespace state", state.Name).
			Info("State could not be found")
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

		// Now we have all the state settings, we can set the replicas for the deployment accordingly
		if !optIn {
			// the deployment opted out. We need to set back to default.
			log.Info("The deploymentconfig opted out. Will scale back to default")
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

func ScaleDeploymentConfig(ctx context.Context, _client client.Client, deploymentconfig v1.DeploymentConfig, stateReplica sr.StateReplica) error {
	log := ctrl.Log.
		WithValues("deploymentconfig", deploymentconfig.Name).
		WithValues("namespace", deploymentconfig.Namespace)

	var err error
	var oldReplicaCount int32
	oldReplicaCount = deploymentconfig.Spec.Replicas
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if oldReplicaCount == stateReplica.Replicas {
			log.Info("No Update on deploymentconfig. Desired replica count already matches current.")
			return nil
		}
		log.Info("Updating deploymentconfig replicas for state", "replicas", stateReplica.Replicas)
		updateErr := DeploymentConfigScaler(ctx, _client, deploymentconfig, stateReplica.Replicas)
		if updateErr == nil {
			log.WithValues("Deploymentconfig", deploymentconfig.Name).
				WithValues("StateReplica mode", stateReplica.Name).
				WithValues("Old Replica count", oldReplicaCount).
				WithValues("New Replica count", stateReplica.Replicas).
				Info("Deployment succesfully updated")
			return nil
		}
		log.Info("Updating deploymentconfig failed due to a conflict! Retrying..")
		// We need to get a newer version of the object from the client
		var req reconcile.Request
		req.NamespacedName.Namespace = deploymentconfig.Namespace
		req.NamespacedName.Name = deploymentconfig.Name
		deploymentconfig, err = DeploymentConfigGetter(ctx, _client, req)
		if err != nil {
			log.Error(err, "Error getting refreshed deploymentconfig in conflict resolution")
		}
		return updateErr

	})
	if retryErr != nil {
		log.Error(retryErr, "Unable to scale the deploymentconfig, err: %v")
	}

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
