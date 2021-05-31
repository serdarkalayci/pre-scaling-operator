package global

import (
	"sync"

	ocv1 "github.com/openshift/api/apps/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// Deployment or deploymentconfig supported for now
type ScalingItemType struct {
	ItemTypeName string
}

type ScalingInfo struct {
	Namespace   string
	Name        string
	Annotations map[string]string
	Labels      map[string]string
	ScalingItemType
	IsBeingScaled    bool
	Failure          bool
	FailureMessage   string
	SpecReplica      int32
	ReadyReplicas    int32
	DesiredReplicas  int32
	ProgressDeadline int32
	ResourceList     corev1.ResourceList
	ConditionReason  string
}

// Global DenyList to check if the deployment is currently reconciles/step scaled
var denylist *ConcurrentSlice

// Global ReconcileList hold deployment/deploymenconfig information to make reconcile decisions
var reconcileList *ConcurrentSlice

// ConcurrentSlice type that can be safely shared between goroutines
type ConcurrentSlice struct {
	sync.RWMutex
	items []ScalingInfo
}

// ConcurrentSliceItem contains the index/value pair of an item in a
// concurrent slice
type ConcurrentSliceItem struct {
	Index int
	Value ScalingInfo
}

func GetReconcileList() *ConcurrentSlice {
	if reconcileList == nil {
		reconcileList = NewConcurrentSlice()
		return reconcileList
	}
	return reconcileList
}

func GetDenyList() *ConcurrentSlice {
	if denylist == nil {
		denylist = NewConcurrentSlice()
		return denylist
	}
	return denylist
}

// NewConcurrentSlice creates a new concurrent slice
func NewConcurrentSlice() *ConcurrentSlice {
	cs := &ConcurrentSlice{
		items: make([]ScalingInfo, 0),
	}

	denylist = cs
	return cs
}

func (cs *ConcurrentSlice) UpdateOrAppend(item ScalingInfo) {
	found := false
	i := 0
	for inList := range cs.Iter() {
		if item.Name == inList.Value.Name && item.Namespace == inList.Value.Namespace {
			cs.items[i] = item
			found = true
		}
		i++
	}
	if !found {
		cs.Lock()
		defer cs.Unlock()

		cs.items = append(cs.items, item)
	}
}

// Iter iterates over the items in the concurrent slice
// Each item is sent over a channel, so that
// we can iterate over the slice using the builin range keyword
// Iter() is locking the slice. Therefore we can safely use Iter() to modify it
func (cs *ConcurrentSlice) Iter() <-chan ConcurrentSliceItem {
	c := make(chan ConcurrentSliceItem)

	f := func() {
		cs.Lock()
		defer cs.Unlock()
		for index, value := range cs.items {
			c <- ConcurrentSliceItem{index, value}
		}
		close(c)
	}
	go f()

	return c
}

func (cs *ConcurrentSlice) RemoveFromList(item ScalingInfo) {
	i := 0
	for inList := range cs.Iter() {
		if item.Name == inList.Value.Name && item.Namespace == inList.Value.Namespace {
			denylist.items = RemoveIndex(cs.items, i)
		}
		i++
	}
}

func (cs *ConcurrentSlice) Update(item ScalingInfo) {
	i := 0
	for inList := range cs.Iter() {
		if item.Name == inList.Value.Name && item.Namespace == inList.Value.Namespace {
			cs.items[i] = item
		}
		i++
	}
}

func (cs *ConcurrentSlice) IterOverItemsInFailureState() <-chan ConcurrentSliceItem {
	c := make(chan ConcurrentSliceItem)

	f := func() {
		cs.Lock()
		defer cs.Unlock()
		for index, value := range cs.items {
			if value.Failure {
				c <- ConcurrentSliceItem{index, value}
			}
		}
		close(c)
	}
	go f()

	return c
}

func (cs *ConcurrentSlice) PurgeList() {
	for range cs.Iter() {
		cs.items = RemoveIndex(cs.items, 0)
	}
}

func (cs *ConcurrentSlice) Length() int {
	i := 0
	for range cs.Iter() {
		i++
	}

	return i
}

func RemoveIndex(s []ScalingInfo, index int) []ScalingInfo {
	return append(s[:index], s[index+1:]...)
}

func (cs *ConcurrentSlice) IsInConcurrentList(item ScalingInfo) bool {
	result := false
	for inList := range cs.Iter() {
		if item.Name == inList.Value.Name && item.Namespace == inList.Value.Namespace {
			result = true
		}
	}
	return result
}

func (cs *ConcurrentSlice) IsBeingScaled(item ScalingInfo) bool {
	result := false
	for inList := range cs.Iter() {
		if item.Name == inList.Value.Name && item.Namespace == inList.Value.Namespace && inList.Value.IsBeingScaled {
			result = true
		}
	}
	return result
}

func (cs *ConcurrentSlice) SetScalingItemOnList(item ScalingInfo, failure bool, failureMessage string, desiredReplicas int32) ScalingInfo {
	item.Failure = failure
	item.FailureMessage = failureMessage
	item.DesiredReplicas = desiredReplicas
	cs.UpdateOrAppend(item)
	return item
}

func (cs *ConcurrentSlice) SetProgressDeadline(item ScalingInfo, progressDeadline int32) {
	item.ProgressDeadline = progressDeadline
	cs.UpdateOrAppend(item)
}

func (cs *ConcurrentSlice) GetDeploymentInfoFromList(item ScalingInfo) (ScalingInfo, error) {
	result := ScalingInfo{}
	for inList := range cs.Iter() {
		if item.Name == inList.Value.Name && item.Namespace == inList.Value.Namespace {
			result = inList.Value
		}
	}
	if result.Name != "" && result.Namespace != "" {
		return result, nil
	} else {
		// Returning the item we passed in because there was none on the list
		return item, NotFound{
			msg: "No deploymentInfo found!",
		}
	}
}

func (cs *ConcurrentSlice) IsDeploymentInFailureState(item ScalingInfo) bool {
	itemToReturn, err := cs.GetDeploymentInfoFromList(item)
	if err != nil {
		return false
	}
	return itemToReturn.Failure
}

func (cs *ConcurrentSlice) GetDesiredReplicasFromList(item ScalingInfo) int32 {
	itemToReturn, _ := cs.GetDeploymentInfoFromList(item)
	return itemToReturn.DesiredReplicas
}

func ConvertDeploymentToItem(deployment v1.Deployment) ScalingInfo {

	// In some cases containers[] and conditions are empty[] that would lead to nullpointer exceptions.
	var conditionReason = ""
	if len(deployment.Status.Conditions) != 0 {
		// get the latest condition reason in case there is one.
		conditionReason = deployment.Status.Conditions[len(deployment.Status.Conditions)-1].Reason
	}

	var resourceList corev1.ResourceList = corev1.ResourceList{}
	if len(deployment.Spec.Template.Spec.Containers) != 0 {
		// get the latest condition reason in case there is one.
		resourceList = deployment.Spec.Template.Spec.Containers[0].Resources.Limits
	}

	failure := false
	failureMessage := ""
	if conditionReason == "ProgressDeadlineExceeded" {
		failure = true
		failureMessage = "Can't scale. ProgressDeadlineExceeded on the cluster!"
	}

	return ScalingInfo{
		Name:             deployment.Name,
		Namespace:        deployment.Namespace,
		Annotations:      deployment.Annotations,
		Labels:           deployment.Labels,
		ScalingItemType:  ScalingItemType{ItemTypeName: "Deployment"},
		Failure:          failure,
		FailureMessage:   failureMessage,
		SpecReplica:      *deployment.Spec.Replicas,
		ReadyReplicas:    deployment.Status.AvailableReplicas,
		DesiredReplicas:  -1,
		ResourceList:     resourceList,
		ConditionReason:  conditionReason,
		ProgressDeadline: *deployment.Spec.ProgressDeadlineSeconds,
	}
}

func ConvertDeploymentConfigToItem(deploymentConfig ocv1.DeploymentConfig) ScalingInfo {

	// In some cases containers[] and conditions are empty[] that would lead to nullpointer exceptions.
	var conditionReason = ""
	if len(deploymentConfig.Status.Conditions) != 0 {
		// get the latest condition reason in case there is one.
		conditionReason = deploymentConfig.Status.Conditions[len(deploymentConfig.Status.Conditions)-1].Reason
	}

	var resourceList corev1.ResourceList = corev1.ResourceList{}
	if len(deploymentConfig.Spec.Template.Spec.Containers) != 0 {
		// get the latest condition reason in case there is one.
		resourceList = deploymentConfig.Spec.Template.Spec.Containers[0].Resources.Limits
	}
	var progressDeadLine int32 = 600
	if deploymentConfig.Spec.Strategy.Type == "Rolling" {
		progressDeadLine = int32(*deploymentConfig.Spec.Strategy.RollingParams.TimeoutSeconds)
	} else if deploymentConfig.Spec.Strategy.Type == "Recreate" {
		progressDeadLine = int32(*deploymentConfig.Spec.Strategy.RecreateParams.TimeoutSeconds)
	}

	failure := false
	failureMessage := ""
	if conditionReason == "ProgressDeadlineExceeded" {
		failure = true
		failureMessage = "Can't scale. ProgressDeadlineExceeded on the cluster!"
	}
	return ScalingInfo{
		Name:             deploymentConfig.Name,
		Namespace:        deploymentConfig.Namespace,
		Annotations:      deploymentConfig.Annotations,
		Labels:           deploymentConfig.Labels,
		ScalingItemType:  ScalingItemType{ItemTypeName: "DeploymentConfig"},
		Failure:          failure,
		FailureMessage:   failureMessage,
		SpecReplica:      deploymentConfig.Spec.Replicas,
		ReadyReplicas:    deploymentConfig.Status.AvailableReplicas,
		DesiredReplicas:  -1,
		ResourceList:     resourceList,
		ConditionReason:  conditionReason,
		ProgressDeadline: progressDeadLine,
	}
}
