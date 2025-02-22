package reconciler

import (
	"context"
	"errors"
	"fmt"

	"github.com/containersol/prescale-operator/api/v1alpha1"
	constants "github.com/containersol/prescale-operator/internal"
	"github.com/containersol/prescale-operator/internal/quotas"
	"github.com/containersol/prescale-operator/internal/resources"
	"github.com/containersol/prescale-operator/internal/states"
	g "github.com/containersol/prescale-operator/pkg/utils/global"

	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NamespaceInfo struct {
	NSEvents       resources.NamespaceEvents
	AppliedState   string
	Error          error
	RetriggerMe    bool
	ScaleNamespace bool
}

type ReconcilerError struct {
	msg string
}

func (err ReconcilerError) Error() string {
	return err.msg
}

func PrepareForNamespaceReconcile(ctx context.Context, _client client.Client, namespace string, stateDefinitions states.States, clusterState states.State, recorder record.EventRecorder, dryRun bool) (map[string]NamespaceInfo, bool, error) {
	log := ctrl.Log
	var err error
	var scalingobjects []g.ScalingInfo
	var reTrigger bool = false

	nsInfoMap := make(map[string]NamespaceInfo)

	if namespace == "" {
		scalingobjects, err = resources.ScalingItemNamespaceLister(ctx, _client, "", constants.OptInLabel)
		if err != nil {
			log.Error(err, "error listing ScalingObjects")
			return nil, true, err
		}
	} else {
		scalingobjects, err = resources.ScalingItemNamespaceLister(ctx, _client, namespace, constants.OptInLabel)
		if err != nil {
			log.Error(err, fmt.Sprintf("error listing ScalingObjects in namespace %s", namespace))
			return nil, true, err
		}
	}

	if len(scalingobjects) == 0 {
		log.Info("nothing to reconcile. No opted in objects found.")
		return nil, false, nil
	}

	scalingObjectGrouped := resources.GroupScalingItemByNamespace(scalingobjects)

	overallNsInformation, err := resources.MakeNamespacesScaleDecisions(ctx, _client, scalingObjectGrouped, stateDefinitions, clusterState, dryRun)
	if err != nil {
		return nil, false, err
	}

	for namespaceKey, value := range overallNsInformation.NSScaleInfo {
		if overallNsInformation.NSScaleInfo[namespaceKey].ScaleNameSpace && !dryRun {
			overallNsInformation.NumberofNsToScale--
			ReconcileNamespace(ctx, _client, namespaceKey, overallNsInformation.NSScaleInfo[namespaceKey].ScalingItems, overallNsInformation.NSScaleInfo[namespaceKey].FinalNamespaceState, recorder, dryRun)
		}

		nsInfoMap[namespaceKey] = NamespaceInfo{
			NSEvents:       value.NamespaceEvents,
			AppliedState:   value.FinalNamespaceState.Name,
			ScaleNamespace: value.ScaleNameSpace,
		}

		// Accumulate the information to return to the controller
	}
	if overallNsInformation.NumberofNsToScale > 0 && !dryRun {
		reTrigger = true
	}
	return nsInfoMap, reTrigger, nil

}

func ReconcileNamespace(ctx context.Context, _client client.Client, namespace string, scalingItems []g.ScalingInfo, finalState states.State, recorder record.EventRecorder, dryRun bool) {

	//	var objectsToReconcile int

	log := ctrl.Log.
		WithValues("namespace", namespace)

	for _, scalingItem := range scalingItems {
		// Don't scale if we don't need to
		if scalingItem.SpecReplica == scalingItem.DesiredReplicas || scalingItem.DesiredReplicas == -1 {
			continue
		}

		scalingItemFresh, notFoundErr := g.GetDenyList().GetDeploymentInfoFromList(scalingItem)
		if notFoundErr == nil {
			if scalingItemFresh.DesiredReplicas != scalingItem.DesiredReplicas {
				g.GetDenyList().SetScalingItemOnList(scalingItemFresh, scalingItemFresh.Failure, scalingItemFresh.FailureMessage, scalingItem.DesiredReplicas)
				log.WithValues("Name: ", scalingItemFresh.Name).
					WithValues("Namespace: ", scalingItemFresh.Namespace).
					WithValues("Object: ", scalingItemFresh.ScalingItemType.ItemTypeName).
					WithValues("DesiredReplica count on item: ", scalingItemFresh.DesiredReplicas).
					WithValues("New replica count:", scalingItem.DesiredReplicas).
					WithValues("Failure: ", scalingItemFresh.Failure).
					WithValues("Failure message: ", scalingItemFresh.FailureMessage).
					Info("Deployment is already being scaled at the moment. Updated desired replica count with new replica count")
			}
			continue
		}
		if !g.GetDenyList().IsDeploymentInFailureState(scalingItem) {
			go resources.ScaleOrStepScale(ctx, _client, scalingItem, "NSSCALER", recorder)
		}

	}
}

func ReconcileScalingItem(ctx context.Context, _client client.Client, scalingItem g.ScalingInfo, forceReconcile bool, recorder record.EventRecorder, whereFromScalingItem string) error {
	log := ctrl.Log.
		WithValues("deploymentItem", scalingItem.Name).
		WithValues("namespace", scalingItem.Namespace)

		// Get all necessary information
	stateDefinitions, err := states.GetClusterScalingStates(ctx, _client)
	if err != nil {
		log.Error(err, "Failed to get ClusterStateDefinitions")
		return err
	}
	namespaceState, nsStateErr := states.FetchNameSpaceState(ctx, _client, stateDefinitions, scalingItem.Namespace)
	if err != nsStateErr {
		return nsStateErr
	}
	// get all css
	clusterScalingStates := v1alpha1.ClusterScalingStateList{}
	cssErr := _client.List(ctx, &clusterScalingStates, &client.ListOptions{})
	if cssErr != nil {
		return cssErr
	}

	// The state and replica determination functions are using lists.
	deploymentItems := []g.ScalingInfo{}
	deploymentItems = append(deploymentItems, scalingItem)
	deploymentItems = states.GetAppliedStatesOnItems(scalingItem.Namespace, namespaceState, clusterScalingStates, stateDefinitions, deploymentItems)
	deploymentItems, _ = resources.DetermineDesiredReplicas(deploymentItems)

	if len(deploymentItems) == 0 {
		return nil
	} else {
		scalingItem = deploymentItems[0]
	}

	if scalingItem.DesiredReplicas == -1 {
		return errors.New(fmt.Sprintf("Desired replica count could not be determined! State: %s | Class: %s ", scalingItem.State, scalingItem.ScalingClass))
	}

	// Don't scale if we don't need to
	if scalingItem.ReadyReplicas == scalingItem.DesiredReplicas && scalingItem.SpecReplica == scalingItem.DesiredReplicas {
		return nil
	}

	_, _, allowed, err := quotas.ResourceQuotaCheck(ctx, scalingItem.Namespace, resources.LimitsNeeded(scalingItem, scalingItem.DesiredReplicas))
	if err != nil {
		log.Error(err, "Cannot calculate the resource quotas")
		return err
	}

	if allowed {
		scalingItemNew, notFoundErr := g.GetDenyList().GetDeploymentInfoFromList(scalingItem)
		if notFoundErr == nil && !scalingItemNew.Failure {
			if scalingItemNew.DesiredReplicas != scalingItem.DesiredReplicas {
				g.GetDenyList().SetScalingItemOnList(scalingItemNew, scalingItemNew.Failure, scalingItemNew.FailureMessage, scalingItem.DesiredReplicas)

				log.WithValues("Name: ", scalingItemNew.Name).
					WithValues("Namespace: ", scalingItemNew.Namespace).
					WithValues("Object: ", scalingItemNew.ScalingItemType.ItemTypeName).
					WithValues("DesiredReplicacount on item: ", scalingItemNew.DesiredReplicas).
					WithValues("New desired replica count:", scalingItem.DesiredReplicas).
					WithValues("Failure: ", scalingItemNew.Failure).
					WithValues("Failure message: ", scalingItemNew.FailureMessage).
					Info("Deployment is already being scaled at the moment. Updated desired replica count with new replica count")
			}
		} else {
			err = resources.ScaleOrStepScale(ctx, _client, scalingItemNew, "deployScaler", recorder)
			if err != nil {
				log.Error(err, "Error scaling object!")
			}
			return err
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
