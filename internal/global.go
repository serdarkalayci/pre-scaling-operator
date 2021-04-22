package internal

import v1 "k8s.io/api/apps/v1"

var (
	// Deployments in this list shall not be reconciled
	ReconcileBlackList []DeploymentInfo
)

type DeploymentInfo struct {
	Namespace string
	Name      string
}

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

func RemoveIndex(s []DeploymentInfo, index int) []DeploymentInfo {
	return append(s[:index], s[index+1:]...)
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
