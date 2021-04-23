package global

import v1 "k8s.io/api/apps/v1"

var (
	// Deployments in this list shall not be reconciled
	ReconcileBlackList []DeploymentInfo
)

// type DeploymentInfo struct {
// 	Namespace string
// 	Name      string
// }

// type ConcurrentBlackList struct {
// 	sync.RWMutex
// 	items []ConcurrentDeploymentInfoItem
// }

// type ConcurrentDeploymentInfoItem struct {
// 	Index      int
// 	DeployInfo DeploymentInfo
// }

// func (cbl *ConcurrentBlackList) Append(item ConcurrentDeploymentInfoItem) {
// 	cbl.Lock()
// 	defer cbl.Unlock()

// 	cbl.items = append(cbl.items, item)
// }

// func (cbl *ConcurrentBlackList) Iter() <-chan ConcurrentDeploymentInfoItem {
// 	c := make(chan ConcurrentDeploymentInfoItem)

// 	f := func() {
// 		cbl.Lock()
// 		defer cbl.Unlock()
// 		for index, value := range cbl.items {
// 			c <- ConcurrentDeploymentInfoItem{index, value.DeployInfo}
// 		}
// 		close(c)
// 	}
// 	go f()

// 	return c
// }

func RemoveDeploymentFromGlobalBlackList(deployment v1.Deployment) {
	removeItem := DeploymentInfo{
		Namespace: deployment.Namespace,
		Name:      deployment.Name,
	}
	RemoveFromBlackList(removeItem)
}

func RemoveFromBlackList(removeItem DeploymentInfo) {
	temp := ReconcileBlackList
	var i int = 0
	for _, item := range temp {
		if item == removeItem {
			ReconcileBlackList = RemoveIndex(temp, i)
		}
		i++
	}
}

// Puts unique deployment on the Blacklist
func PutDeploymentOnGlobalBlackList(deployment v1.Deployment) {
	newItem := DeploymentInfo{
		Namespace: deployment.Namespace,
		Name:      deployment.Name,
	}
	PutOnBlackList(newItem)
}

func PutOnBlackList(newItem DeploymentInfo) {
	if !IsOnBlackList(newItem) {
		ReconcileBlackList = append(ReconcileBlackList, newItem)
	}
}

func IsDeploymentOnBlackList(deployment v1.Deployment) bool {
	checkItem := DeploymentInfo{
		Namespace: deployment.Namespace,
		Name:      deployment.Name,
	}
	return IsOnBlackList(checkItem)
}

func IsOnBlackList(checkItem DeploymentInfo) bool {
	for _, item := range ReconcileBlackList {
		if item == checkItem {
			return true
		}
	}
	return false
}
