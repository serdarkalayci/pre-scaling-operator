package reconciler

import (
	"context"
	"errors"
	"fmt"

	c "github.com/containersol/prescale-operator/internal"
	"github.com/containersol/prescale-operator/internal/quotas"
	"github.com/containersol/prescale-operator/internal/resources"
	"github.com/containersol/prescale-operator/internal/states"
	g "github.com/containersol/prescale-operator/pkg/utils/global"

	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NamespaceInfo struct {
	NSEvents     resources.NamespaceEvents
	AppliedState string
	Error        error
	RetriggerMe  bool
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
		scalingobjects, err = resources.ScalingItemNamespaceLister(ctx, _client, "", c.OptInLabel)
		if err != nil {
			log.Error(err, "error listing ScalingObjects")
			return nil, true, err
		}
	} else {
		scalingobjects, err = resources.ScalingItemNamespaceLister(ctx, _client, namespace, c.OptInLabel)
		if err != nil {
			log.Error(err, fmt.Sprintf("error listing ScalingObjects in namespace %s", namespace))
			return nil, true, err
		}
	}

	if len(scalingobjects) == 0 {
		return nil, false, errors.New("no opted-in scalingobjects found")
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
		// Accumulate the information to return to the controller
		nsInfoMap[namespaceKey] = NamespaceInfo{
			NSEvents:     value.NamespaceEvents,
			AppliedState: value.FinalNamespaceState.Name,
		}
	}
	if overallNsInformation.NumberofNsToScale > 0 {
		reTrigger = true
	}
	return nsInfoMap, reTrigger, nil

}

func ReconcileNamespace(ctx context.Context, _client client.Client, namespace string, scalingItems []g.ScalingInfo, finalState states.State, recorder record.EventRecorder, dryRun bool) {

	//	var objectsToReconcile int

	log := ctrl.Log.
		WithValues("namespace", namespace)

		// //Here we calculate the resource limits we need from all deployments combined
		// limitsneeded = resources.LimitsNeededList(scalingItems, scaleReplicalist)

		// // After we have calculated the resources needed from all workloads in a given namespace, we can determine if the scaling should be allowed to go through
		// finalLimitsCPU, finalLimitsMemory, allowed, err := quotas.ResourceQuotaCheck(ctx, namespace, limitsneeded)
		// if err != nil {
		// 	log.Error(err, "Cannot calculate the resource quotas")
		// 	return nsEvents, finalState.Name, err
		// }

	for _, scalingItem := range scalingItems {
		// Don't scale if we don't need to
		if scalingItem.SpecReplica == scalingItem.DesiredReplicas {
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
			scalingItem.DesiredReplicas = stateReplica.Replicas
			err = resources.ScaleOrStepScale(ctx, _client, scalingItem, "deployScaler", recorder)
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
