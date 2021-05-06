package global

import (
	"sync"

	ocv1 "github.com/openshift/api/apps/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type DeploymentInfo struct {
	Namespace          string
	Name               string
	Annotations        map[string]string
	Labels             map[string]string
	IsDeploymentConfig bool
	Failure            bool
	FailureMessage     string
	SpecReplica        int32
	ReadyReplicas      int32
	DesiredReplicas    int32
	ResourceList       corev1.ResourceList
}

// Global DenyList to check if the deployment is currently reconciles/step scaled
var denylist *ConcurrentSlice

// Global ReconcileList hold deployment/deploymenconfig information to make reconcile decisions
var reconcileList *ConcurrentSlice

// ConcurrentSlice type that can be safely shared between goroutines
type ConcurrentSlice struct {
	sync.RWMutex
	items []DeploymentInfo
}

// ConcurrentSliceItem contains the index/value pair of an item in a
// concurrent slice
type ConcurrentSliceItem struct {
	Index int
	Value DeploymentInfo
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
		items: make([]DeploymentInfo, 0),
	}

	denylist = cs
	return cs
}

func (cs *ConcurrentSlice) UpdateOrAppend(item DeploymentInfo) {
	if cs.IsInConcurrentList(item) {
		cs.RemoveFromList(item)

		cs.Lock()
		defer cs.Unlock()

		cs.items = append(cs.items, item)
	} else {
		cs.Lock()
		defer cs.Unlock()

		cs.items = append(cs.items, item)
	}
}

// Iter iterates over the items in the concurrent slice
// Each item is sent over a channel, so that
// we can iterate over the slice using the builin range keyword
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

func (cs *ConcurrentSlice) RemoveFromList(item DeploymentInfo) {
	i := 0
	for inList := range cs.Iter() {
		if item.Name == inList.Value.Name && item.Namespace == inList.Value.Namespace {
			denylist.items = RemoveIndex(cs.items, i)
		}
		i++
	}
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

func RemoveIndex(s []DeploymentInfo, index int) []DeploymentInfo {
	return append(s[:index], s[index+1:]...)
}

func (cs *ConcurrentSlice) IsInConcurrentList(item DeploymentInfo) bool {
	result := false
	for inList := range cs.Iter() {
		if item.Name == inList.Value.Name && item.Namespace == inList.Value.Namespace {
			result = true
		}
	}
	return result
}

func (cs *ConcurrentSlice) SetDeploymentInfoOnList(item DeploymentInfo, failure bool, failureMessage string, desiredReplicas int32) {
	item.Failure = failure
	item.FailureMessage = failureMessage
	item.DesiredReplicas = desiredReplicas
	cs.UpdateOrAppend(item)
}

func (cs *ConcurrentSlice) GetDeploymentInfoFromList(item DeploymentInfo) (DeploymentInfo, error) {
	result := DeploymentInfo{}
	for inList := range cs.Iter() {
		if item.Name == inList.Value.Name && item.Namespace == inList.Value.Namespace {
			result = inList.Value
		}
	}
	if result.Name != "" && result.Namespace != "" {
		return result, nil
	}
	return DeploymentInfo{}, NotFound{
		msg: "No deploymentInfo found!",
	}
}

func (cs *ConcurrentSlice) IsDeploymentInFailureState(item DeploymentInfo) bool {
	itemToReturn, _ := cs.GetDeploymentInfoFromList(item)
	return itemToReturn.Failure
}

func (cs *ConcurrentSlice) GetDesiredReplicasFromList(item DeploymentInfo) int32 {
	itemToReturn, _ := cs.GetDeploymentInfoFromList(item)
	return itemToReturn.DesiredReplicas
}

func ConvertDeploymentToItem(deployment v1.Deployment) DeploymentInfo {

	if deployment.Spec.Replicas == nil {
		// We are in a test. Return dummy object.
		return DeploymentInfo{
			Name:               deployment.Name,
			Namespace:          deployment.Namespace,
			Labels:             deployment.Labels,
			IsDeploymentConfig: false,
			Failure:            false,
			FailureMessage:     "",
			DesiredReplicas:    0,
		}
	}

	return DeploymentInfo{
		Name:               deployment.Name,
		Namespace:          deployment.Namespace,
		Annotations:        deployment.Annotations,
		Labels:             deployment.Labels,
		IsDeploymentConfig: false,
		Failure:            false,
		FailureMessage:     "",
		SpecReplica:        *deployment.Spec.Replicas,
		ReadyReplicas:      deployment.Status.AvailableReplicas,
		DesiredReplicas:    -1,
		ResourceList:       deployment.Spec.Template.Spec.Containers[0].Resources.Limits,
	}
}

func ConvertDeploymentConfigToItem(deploymentConfig ocv1.DeploymentConfig) DeploymentInfo {
	return DeploymentInfo{
		Name:               deploymentConfig.Name,
		Namespace:          deploymentConfig.Namespace,
		Annotations:        deploymentConfig.Annotations,
		Labels:             deploymentConfig.Labels,
		IsDeploymentConfig: true,
		Failure:            false,
		FailureMessage:     "",
		SpecReplica:        deploymentConfig.Spec.Replicas,
		ReadyReplicas:      deploymentConfig.Status.AvailableReplicas,
		DesiredReplicas:    -1,
		ResourceList:       deploymentConfig.Spec.Template.Spec.Containers[0].Resources.Limits,
	}
}
