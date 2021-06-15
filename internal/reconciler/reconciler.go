package reconciler

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	c "github.com/containersol/prescale-operator/internal"
	"github.com/containersol/prescale-operator/internal/quotas"
	"github.com/containersol/prescale-operator/internal/resources"
	"github.com/containersol/prescale-operator/internal/states"
	g "github.com/containersol/prescale-operator/pkg/utils/global"
	"github.com/olekukonko/tablewriter"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NamespaceEvents struct {
	QuotaExceeded    string
	ReconcileSuccess []string
	ReconcileFailure []string
	DryRunInfo       string
}

type ReconcilerError struct {
	msg string
}

func (err ReconcilerError) Error() string {
	return err.msg
}

func ReconcileNamespace(ctx context.Context, _client client.Client, namespace string, stateDefinitions states.States, clusterState states.State, recorder record.EventRecorder, dryRun bool) (NamespaceEvents, string, error) {

	var objectsToReconcile int
	var nsEvents NamespaceEvents

	var limitsneeded corev1.ResourceList
	var finalLimitsCPU, finalLimitsMemory string

	log := ctrl.Log.
		WithValues("namespace", namespace)

	finalState, err := GetAppliedState(ctx, _client, namespace, stateDefinitions, clusterState)
	if err != nil {
		return nsEvents, finalState.Name, err
	}

	// We now need to look for objects (currently supported deployments and deploymentConfigs) which are opted in,
	// then use their annotations to determine the correct scale
	deployments, err := resources.ScalingItemLister(ctx, _client, namespace, c.OptInLabel)
	if err != nil {
		log.Error(err, "Cannot list deployments in namespace")
		return nsEvents, finalState.Name, err
	}
	objectsToReconcile = objectsToReconcile + len(deployments)

	if objectsToReconcile != 0 {
		log.Info("Found opted in objects in Namespace. Reconciling namespace")
	} else {
		return nsEvents, finalState.Name, err
	}

	scaleReplicalist, err := resources.StateReplicasList(finalState, deployments)
	if err != nil {
		log.Error(err, "Cannot fetch replicas of all opted-in deployments")
		return nsEvents, finalState.Name, err
	}

	//Here we calculate the resource limits we need from all deployments combined
	limitsneeded = resources.LimitsNeededList(deployments, scaleReplicalist)

	// After we have calculated the resources needed from all workloads in a given namespace, we can determine if the scaling should be allowed to go through
	finalLimitsCPU, finalLimitsMemory, allowed, err := quotas.ResourceQuotaCheck(ctx, namespace, limitsneeded)
	if err != nil {
		log.Error(err, "Cannot calculate the resource quotas")
		return nsEvents, finalState.Name, err
	}

	if !dryRun {
		if allowed {
			for i, deployment := range deployments {
				// Don't scale if we don't need to
				if deployment.SpecReplica == scaleReplicalist[i].Replicas {
					continue
				}

				scalingItem, notFoundErr := g.GetDenyList().GetDeploymentInfoFromList(deployment)
				if notFoundErr == nil {
					if scalingItem.DesiredReplicas != scaleReplicalist[i].Replicas {
						g.GetDenyList().SetScalingItemOnList(scalingItem, scalingItem.Failure, scalingItem.FailureMessage, scaleReplicalist[i].Replicas)
						log.WithValues("Name: ", scalingItem.Name).
							WithValues("Namespace: ", scalingItem.Namespace).
							WithValues("Object: ", scalingItem.ScalingItemType.ItemTypeName).
							WithValues("DesiredReplicacount on item: ", scalingItem.DesiredReplicas).
							WithValues("New replica count:", scaleReplicalist[i].Replicas).
							WithValues("Failure: ", scalingItem.Failure).
							WithValues("Failure message: ", scalingItem.FailureMessage).
							Info("Deployment is already being scaled at the moment. Updated desired replica count with new replica count")
					}
					continue
				}
				if !g.GetDenyList().IsDeploymentInFailureState(deployment) {
					go resources.ScaleOrStepScale(ctx, _client, deployment, scaleReplicalist[i], "NSSCALER", recorder)
				}

			}
		} else {
			nsEvents.QuotaExceeded = namespace
		}
	} else {

		tableString := &strings.Builder{}
		table := tablewriter.NewWriter(tableString)
		table.SetHeader([]string{"Namespace", "Quotas enough", "Cpu left after scaling", "Memory left after scaling"})
		table.Append([]string{namespace, strconv.FormatBool(allowed), finalLimitsCPU, finalLimitsMemory})
		table.Render()

		nsEvents.DryRunInfo = tableString.String()

		var applicationData [][]string
		tableString = &strings.Builder{}
		table = tablewriter.NewWriter(tableString)
		table.SetHeader([]string{"Application", "Current replicas", "New state", "New replicas", "Rapid Scaling"})

		for i, deployment := range deployments {

			applicationData = append(applicationData, []string{deployment.Name, fmt.Sprint(deployment.ReadyReplicas), scaleReplicalist[i].Name, fmt.Sprint(scaleReplicalist[i].Replicas), strconv.FormatBool(states.GetRapidScalingSetting(deployment))})

		}

		for _, v := range applicationData {
			table.Append(v)
		}

		table.Render()

		nsEvents.DryRunInfo = nsEvents.DryRunInfo + tableString.String()
	}

	return nsEvents, finalState.Name, err
}

func ReconcileScalingItem(ctx context.Context, _client client.Client, scalingItem g.ScalingInfo, state states.State, forceReconcile bool, recorder record.EventRecorder, whereFromScalingItem string) error {
	log := ctrl.Log.
		WithValues("deploymentItem", scalingItem.Name).
		WithValues("namespace", scalingItem.Namespace)

	stateReplica, err := resources.StateReplicas(state, scalingItem)
	if err != nil {
		log.Error(err, "Error getting the state replicas")
		return err
	}

	// Don't scale if we don't need to
	if scalingItem.ReadyReplicas == stateReplica.Replicas && scalingItem.SpecReplica == stateReplica.Replicas {
		return nil
	}

	_, _, allowed, err := quotas.ResourceQuotaCheck(ctx, scalingItem.Namespace, resources.LimitsNeeded(scalingItem, stateReplica.Replicas))
	if err != nil {
		log.Error(err, "Cannot calculate the resource quotas")
		return err
	}

	if allowed {
		scalingItem, notFoundErr := g.GetDenyList().GetDeploymentInfoFromList(scalingItem)
		if notFoundErr == nil {
			if scalingItem.DesiredReplicas != stateReplica.Replicas {
				g.GetDenyList().SetScalingItemOnList(scalingItem, scalingItem.Failure, scalingItem.FailureMessage, stateReplica.Replicas)

				log.WithValues("Name: ", scalingItem.Name).
					WithValues("Namespace: ", scalingItem.Namespace).
					WithValues("Object: ", scalingItem.ScalingItemType.ItemTypeName).
					WithValues("DesiredReplicacount on item: ", scalingItem.DesiredReplicas).
					WithValues("New replica count:", stateReplica.Replicas).
					WithValues("Failure: ", scalingItem.Failure).
					WithValues("Failure message: ", scalingItem.FailureMessage).
					Info("Deployment is already being scaled at the moment. Updated desired replica count with new replica count")
			}
		} else {
			err = resources.ScaleOrStepScale(ctx, _client, scalingItem, stateReplica, "deployScaler", recorder)
			if err != nil {
				log.Error(err, "Error scaling object!")
			}
		}

	} else {
		log = ctrl.Log
		log.Info(fmt.Sprintf("Quota check didn't pass in namespace %s for object %s", scalingItem.Namespace, scalingItem.Name))
		return ReconcilerError{
			msg: "Can't scale due to ResourceQuota violation!",
		}
	}

	return nil
}

func GetAppliedState(ctx context.Context, _client client.Client, namespace string, stateDefinitions states.States, clusterState states.State) (states.State, error) {
	// Here we allow overriding the cluster state by passing it in.
	// This allows us to not recall the client when looping namespaces
	if clusterState == (states.State{}) {
		var err error
		clusterState, err = fetchClusterState(ctx, _client, stateDefinitions)
		if err != nil {
			return states.State{}, err
		}
	}

	// If we receive an error here, we cannot handle it and should return
	namespaceState, err := fetchNameSpaceState(ctx, _client, stateDefinitions, namespace)
	if err != nil {
		return states.State{}, err
	}

	if namespaceState == (states.State{}) && clusterState == (states.State{}) {
		return states.State{}, err
	}

	finalState := stateDefinitions.FindPriorityState(namespaceState, clusterState)
	return finalState, nil
}

func fetchClusterState(ctx context.Context, _client client.Client, stateDefinitions states.States) (states.State, error) {
	clusterStateName, err := states.GetClusterScalingState(ctx, _client)
	if err != nil {
		switch err.(type) {
		case states.NotFound:
		case states.TooMany:
			ctrl.Log.V(3).Info("Could not process cluster state, but continuing safely.")
		default:
			// For the moment, we cannot deal with any other error.
			return states.State{}, errors.New("could not retrieve cluster states")
		}
	}
	clusterState := states.State{}
	if clusterStateName != "" {
		err = stateDefinitions.FindState(clusterStateName, &clusterState)
		if err != nil {
			ctrl.Log.
				V(3).
				WithValues("state name", clusterStateName).
				Error(err, "Could not find ClusterScalingState within ClusterStateDefinitions. Continuing without considering ClusterScalingState.")
		}
	}
	return clusterState, nil
}

func fetchNameSpaceState(ctx context.Context, _client client.Client, stateDefinitions states.States, namespace string) (states.State, error) {
	namespaceStateName, err := states.GetNamespaceScalingStateName(ctx, _client, namespace)
	if err != nil {
		switch err.(type) {
		case states.NotFound:
		case states.TooMany:
			ctrl.Log.V(3).Info("Could not process namespaced state, but continuing safely.")
		default:
			return states.State{}, err
		}
	}
	namespaceState := states.State{}
	if namespaceStateName != "" {
		err = stateDefinitions.FindState(namespaceStateName, &namespaceState)
		if err != nil {
			ctrl.Log.
				V(3).
				WithValues("state name", namespaceStateName).
				Error(err, fmt.Sprintf("Could not find ScalingState %s within ClusterStateDefinitions. Continuing without considering ScalingState.", namespaceStateName))
			return states.State{}, err
		}
	}
	return namespaceState, nil
}
