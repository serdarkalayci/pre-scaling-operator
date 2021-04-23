package global

import (
	"sync"

	v1 "k8s.io/api/apps/v1"
)

type DeploymentInfo struct {
	Namespace string
	Name      string
}

var blackListSlice *ConcurrentSlice

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

func GetBlackListSlice() *ConcurrentSlice {
	if blackListSlice == nil {
		blackListSlice = NewConcurrentSlice()
		return blackListSlice
	}
	return blackListSlice
}

// NewConcurrentSlice creates a new concurrent slice
func NewConcurrentSlice() *ConcurrentSlice {
	cs := &ConcurrentSlice{
		items: make([]DeploymentInfo, 0),
	}

	return cs
}

// Append adds an item to the concurrent slice
func (cs *ConcurrentSlice) Append(item DeploymentInfo) {
	if !cs.IsInConcurrentBlackList(item) {
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

func (cs *ConcurrentSlice) DeleteFromBlackList(item DeploymentInfo) {
	for inList := range cs.Iter() {
		if item == inList.Value {
			cs.items = RemoveIndex(cs.items, inList.Index)
		}
	}
}

func (cs *ConcurrentSlice) PurgeBlackList() {
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

func (cs *ConcurrentSlice) IsInConcurrentBlackList(item DeploymentInfo) bool {
	for inList := range cs.Iter() {
		if item == inList.Value {
			return true
		}
	}
	return false
}

func ConvertDeploymentToItem(deployment v1.Deployment) DeploymentInfo {
	return DeploymentInfo{
		Name:      deployment.Name,
		Namespace: deployment.Namespace,
	}
}
