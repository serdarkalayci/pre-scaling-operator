package global

import (
	"sync"

	ocv1 "github.com/openshift/api/apps/v1"
	v1 "k8s.io/api/apps/v1"
)

type DeploymentInfo struct {
	Namespace       string
	Name            string
	Failure         bool
	FailureMessage  string
	DesiredReplicas int
}

// Global DenyList to check if the deployment is currently reconciles/step scaled
var denylist *ConcurrentSlice

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
	if cs.IsInConcurrentDenyList(item) {
		cs.RemoveFromDenyList(item)

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

func (cs *ConcurrentSlice) RemoveFromDenyList(item DeploymentInfo) {
	i := 0
	for inList := range cs.Iter() {
		if item.Name == inList.Value.Name && item.Namespace == inList.Value.Namespace {
			denylist.items = RemoveIndex(cs.items, i)
		}
		i++
	}
}

func (cs *ConcurrentSlice) PurgeDenyList() {
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

func (cs *ConcurrentSlice) IsInConcurrentDenyList(item DeploymentInfo) bool {
	result := false
	for inList := range cs.Iter() {
		if item.Name == inList.Value.Name && item.Namespace == inList.Value.Namespace {
			result = true
		}
	}
	return result
}

func (cs *ConcurrentSlice) SetDeploymentInfoOnDenyList(item DeploymentInfo, failure bool, failureMessage string, desiredReplicas int) {
	item.Failure = failure
	item.FailureMessage = failureMessage
	item.DesiredReplicas = desiredReplicas
	cs.UpdateOrAppend(item)
}

func (cs *ConcurrentSlice) GetDeploymentInfoFromDenyList(item DeploymentInfo) (DeploymentInfo, error) {
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

func (cs *ConcurrentSlice) IsDeploymentInFailureState(item DeploymentInfo) (bool, string) {
	itemToReturn, _ := cs.GetDeploymentInfoFromDenyList(item)
	return itemToReturn.Failure, itemToReturn.FailureMessage
}

func (cs *ConcurrentSlice) GetDesiredReplicasFromDenyList(item DeploymentInfo) int {
	itemToReturn, _ := cs.GetDeploymentInfoFromDenyList(item)
	return itemToReturn.DesiredReplicas
}

func ConvertDeploymentToItem(deployment v1.Deployment) DeploymentInfo {
	return DeploymentInfo{
		Name:            deployment.Name,
		Namespace:       deployment.Namespace,
		Failure:         false,
		FailureMessage:  "",
		DesiredReplicas: -1,
	}
}

func ConvertDeploymentConfigToItem(deploymentconfig ocv1.DeploymentConfig) DeploymentInfo {
	return DeploymentInfo{
		Name:            deploymentconfig.Name,
		Namespace:       deploymentconfig.Namespace,
		Failure:         false,
		FailureMessage:  "",
		DesiredReplicas: -1,
	}
}
