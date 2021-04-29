package global

import (
	"reflect"
	"testing"

	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPutOnDenyListAndIsFound(t *testing.T) {
	type args struct {
		deployment []v1.Deployment
	}
	tests := []struct {
		name   string
		args   args
		result bool
	}{
		{
			name: "TestPutOneOnDenyListAndIsFound",
			args: args{
				[]v1.Deployment{
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "foo",
							Namespace: "bar",
						},
						Spec: v1.DeploymentSpec{
							Replicas: new(int32),
						},
						Status: v1.DeploymentStatus{
							Replicas: 5,
						},
					},
				},
			},
			result: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, item := range tt.args.deployment {
				GetDenyList().UpdateOrAppend(ConvertDeploymentToItem(item))
			}

			for _, item := range tt.args.deployment {
				if GetDenyList().IsInConcurrentDenyList(ConvertDeploymentToItem(item)) != true {
					t.Errorf("The item is not in the DenyList! Got  %v, Want %v", GetDenyList().IsInConcurrentDenyList(ConvertDeploymentToItem(item)), tt.result)
				}
			}
		})
	}

}

func TestDenyList(t *testing.T) {
	type args struct {
		deployment []v1.Deployment
	}
	tests := []struct {
		name   string
		args   args
		length int
	}{
		{
			name: "TestPutOneOnDenyList",
			args: args{
				[]v1.Deployment{
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "foo",
							Namespace: "bar",
						},
						Spec: v1.DeploymentSpec{
							Replicas: new(int32),
						},
						Status: v1.DeploymentStatus{
							Replicas: 5,
						},
					},
				},
			},
			length: 1,
		},
		{
			name: "TestPutTwoOnDenyList",
			args: args{
				[]v1.Deployment{
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "foo",
							Namespace: "bar",
						},
						Spec: v1.DeploymentSpec{
							Replicas: new(int32),
						},
						Status: v1.DeploymentStatus{
							Replicas: 5,
						},
					},
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "another",
							Namespace: "one",
						},
						Spec: v1.DeploymentSpec{
							Replicas: new(int32),
						},
						Status: v1.DeploymentStatus{
							Replicas: 5,
						},
					},
				},
			},
			length: 2,
		},
		{
			name: "TestPutDuplicateOnDenyList",
			args: args{
				[]v1.Deployment{
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "foo",
							Namespace: "bar",
						},
						Spec: v1.DeploymentSpec{
							Replicas: new(int32),
						},
						Status: v1.DeploymentStatus{
							Replicas: 5,
						},
					},
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "foo",
							Namespace: "bar",
						},
						Spec: v1.DeploymentSpec{
							Replicas: new(int32),
						},
						Status: v1.DeploymentStatus{
							Replicas: 5,
						},
					},
				},
			},
			length: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			for _, item := range tt.args.deployment {
				GetDenyList().UpdateOrAppend(ConvertDeploymentToItem(item))
			}
			listle := GetDenyList().Length()
			if listle != tt.length {
				t.Errorf("The length is not correct! Got  %v, Want %v", listle, tt.length)
			}

			GetDenyList().PurgeDenyList()
		})
	}

}

func TestAddDuplicateAndPurge(t *testing.T) {
	deploymentItem := DeploymentInfo{
		Name:      "foo",
		Namespace: "bar",
	}

	deploymentItemDuplicate := DeploymentInfo{
		Name:      "foo",
		Namespace: "bar",
	}

	secondDeploymentItem := DeploymentInfo{
		Name:      "woop",
		Namespace: "wob",
	}

	GetDenyList().UpdateOrAppend(deploymentItem)
	GetDenyList().UpdateOrAppend(deploymentItemDuplicate)
	GetDenyList().UpdateOrAppend(secondDeploymentItem)

	if GetDenyList().Length() != 2 {
		t.Errorf("Failed to put item on slice! Got  %v, Want %v", GetDenyList().Length(), 2)
	}

	GetDenyList().PurgeDenyList()

	if GetDenyList().Length() != 0 {
		t.Errorf("The slice didn't get purged correctly! Got  %v, Want %v", GetDenyList().Length(), 0)
	}
}

func TestAddFiveAndDeleteMutiple(t *testing.T) {
	deploymentItem := DeploymentInfo{
		Name:      "first",
		Namespace: "bar",
	}
	secondDeploymentItem := DeploymentInfo{
		Name:      "second",
		Namespace: "wob",
	}

	thirdDeploymentItem := DeploymentInfo{
		Name:      "third",
		Namespace: "woooob",
	}

	fourthdeploymentItem := DeploymentInfo{
		Name:      "fourth",
		Namespace: "baaar",
	}
	fithDeploymentItem := DeploymentInfo{
		Name:      "fifth",
		Namespace: "wfob",
	}

	GetDenyList().UpdateOrAppend(deploymentItem)
	GetDenyList().UpdateOrAppend(secondDeploymentItem)
	GetDenyList().UpdateOrAppend(thirdDeploymentItem)
	GetDenyList().UpdateOrAppend(fourthdeploymentItem)
	GetDenyList().UpdateOrAppend(fithDeploymentItem)

	if GetDenyList().Length() != 5 {
		t.Errorf("Failed to put item on slice! Got  %v, Want %v", GetDenyList().Length(), 5)
	}

	GetDenyList().RemoveFromDenyList(secondDeploymentItem)
	isSecondItemInList := GetDenyList().IsInConcurrentDenyList(secondDeploymentItem)
	if isSecondItemInList {
		t.Errorf("The item is still on the list!! Got  %v, Want %v", isSecondItemInList, false)
	}

	if GetDenyList().Length() != 4 {
		t.Errorf("The item didn't get removed properly!! Got  %v, Want %v", GetDenyList().Length(), 4)
	}

	// delete one that isn't on the list anymore, and that is
	GetDenyList().RemoveFromDenyList(secondDeploymentItem)
	GetDenyList().RemoveFromDenyList(thirdDeploymentItem)
	isSecondItemInList = GetDenyList().IsInConcurrentDenyList(secondDeploymentItem)
	if isSecondItemInList {
		t.Errorf("The item is still on the list!! Got  %v, Want %v", isSecondItemInList, false)
	}

	isThirdItemInList := GetDenyList().IsInConcurrentDenyList(thirdDeploymentItem)
	if isThirdItemInList {
		t.Errorf("The item is still on the list!! Got  %v, Want %v", isThirdItemInList, false)
	}

	if GetDenyList().Length() != 3 {
		t.Errorf("The item didn't get removed properly!! Got  %v, Want %v", GetDenyList().Length(), 3)
	}

	//delete two at once
	GetDenyList().RemoveFromDenyList(fourthdeploymentItem)
	GetDenyList().RemoveFromDenyList(fithDeploymentItem)

	isFourthItemInList := GetDenyList().IsInConcurrentDenyList(fourthdeploymentItem)
	isFifthItemInList := GetDenyList().IsInConcurrentDenyList(fithDeploymentItem)
	if isFourthItemInList {
		t.Errorf("The item is still on the list!! Got  %v, Want %v", isFourthItemInList, false)
	}
	if isFifthItemInList {
		t.Errorf("The item is still on the list!! Got  %v, Want %v", isFifthItemInList, false)
	}

	println("After third delete")
	if GetDenyList().Length() != 1 {
		t.Errorf("The item didn't get removed properly!! Got  %v, Want %v", GetDenyList().Length(), 1)
	}

	GetDenyList().PurgeDenyList()
}

func TestIsInList(t *testing.T) {
	theItemInList := DeploymentInfo{
		Name:      "foo",
		Namespace: "bar",
	}

	someOther := DeploymentInfo{
		Name:      "some",
		Namespace: "other",
	}

	GetDenyList().UpdateOrAppend(theItemInList)

	// Testing IsInConcurrenyDenyList (false case)
	isSomeOtherInList := GetDenyList().IsInConcurrentDenyList(someOther)
	if isSomeOtherInList {
		t.Errorf("! Got  %v, Want %v", isSomeOtherInList, false)
	}

	isTheItemInList := GetDenyList().IsInConcurrentDenyList(theItemInList)
	if !isTheItemInList {
		t.Errorf("! Got  %v, Want %v", !isTheItemInList, true)
	}

	GetDenyList().PurgeDenyList()

}

func TestUpdateAndAppend(t *testing.T) {
	theItemInList := DeploymentInfo{
		Name:      "foo",
		Namespace: "bar",
	}

	theUpdateItem := DeploymentInfo{
		Name:            "foo",
		Namespace:       "bar",
		Failure:         true,
		FailureMessage:  "A message",
		DesiredReplicas: 1,
	}

	GetDenyList().UpdateOrAppend(theItemInList)
	GetDenyList().UpdateOrAppend(theUpdateItem)
	// Check if it updated and didn't add a new one to the list
	if GetDenyList().Length() != 1 {
		t.Errorf("! Got  %v, Want %v", GetDenyList().Length(), 1)
	}

	comparison1, _ := GetDenyList().GetDeploymentInfoFromDenyList(theItemInList)
	comparison2, _ := GetDenyList().GetDeploymentInfoFromDenyList(theUpdateItem)

	isEqual := reflect.DeepEqual(comparison1, comparison2)
	if !isEqual {
		t.Errorf("! Got  %v, Want %v", isEqual, true)
	}

	if comparison1.Name != "foo" || comparison1.Namespace != "bar" || comparison1.Failure != true || comparison1.FailureMessage != "A message" || comparison1.DesiredReplicas != 1 {
		t.Errorf("! Got  %v, Want %v", true, false)
	}

	GetDenyList().PurgeDenyList()

}

func TestUpdateItemInList(t *testing.T) {
	theItemInList := DeploymentInfo{
		Name:      "foo",
		Namespace: "bar",
	}

	notInList := DeploymentInfo{
		Name:      "not",
		Namespace: "there",
	}

	GetDenyList().UpdateOrAppend(theItemInList)
	GetDenyList().SetDeploymentInfoOnDenyList(theItemInList, true, "A failure", 2)
	// Check if it updated and didn't add a new one to the list
	if GetDenyList().Length() != 1 {
		t.Errorf("! Got  %v, Want %v", GetDenyList().Length(), 1)
	}

	comparison1, _ := GetDenyList().GetDeploymentInfoFromDenyList(theItemInList)

	if comparison1.Name != "foo" || comparison1.Namespace != "bar" || comparison1.Failure != true || comparison1.FailureMessage != "A failure" || comparison1.DesiredReplicas != 2 {
		t.Errorf("! Got  %v, Want %v", true, false)
	}

	failure, msg := GetDenyList().IsDeploymentInFailureState(theItemInList)
	if failure == false || msg != "A failure" {
		t.Errorf("! Got  %v, Want %v", failure, true)
	}

	desiredReplicas := GetDenyList().GetDesiredReplicasFromDenyList(theItemInList)
	if desiredReplicas != 2 {
		t.Errorf("! Got  %v, Want %v", desiredReplicas, 2)
	}

	item, err := GetDenyList().GetDeploymentInfoFromDenyList(notInList)
	if item.Name != "" || item.Namespace != "" {
		t.Errorf("! Got  %v, Want %v", "", item.Name)
	}
	if err != "Not Found" {
		t.Errorf("! Got  %v, Want %v", "Not Found", err)
	}

	GetDenyList().PurgeDenyList()

}
