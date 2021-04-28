package global

import (
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
				GetDenyList().Append(ConvertDeploymentToItem(item))
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
				GetDenyList().Append(ConvertDeploymentToItem(item))
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

	GetDenyList().Append(deploymentItem)
	GetDenyList().Append(deploymentItemDuplicate)
	GetDenyList().Append(secondDeploymentItem)

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

	someOther := DeploymentInfo{
		Name:      "some",
		Namespace: "other",
	}

	GetDenyList().Append(deploymentItem)
	GetDenyList().Append(secondDeploymentItem)
	GetDenyList().Append(thirdDeploymentItem)
	GetDenyList().Append(fourthdeploymentItem)
	GetDenyList().Append(fithDeploymentItem)

	if GetDenyList().Length() != 5 {
		t.Errorf("Failed to put item on slice! Got  %v, Want %v", GetDenyList().Length(), 5)
	}

	if GetDenyList().IsInConcurrentDenyList(someOther) {
		t.Errorf("The item is still on the list!! Got  %v, Want %v", GetDenyList().IsInConcurrentDenyList(someOther), false)
	}

	GetDenyList().RemoveFromDenyList(secondDeploymentItem)
	if GetDenyList().IsInConcurrentDenyList(secondDeploymentItem) {
		t.Errorf("The item is still on the list!! Got  %v, Want %v", GetDenyList().IsInConcurrentDenyList(secondDeploymentItem), false)
	}
	if !GetDenyList().IsInConcurrentDenyList(thirdDeploymentItem) {
		t.Errorf("The item is still on the list!! Got  %v, Want %v", GetDenyList().IsInConcurrentDenyList(thirdDeploymentItem), false)
	}
	if GetDenyList().Length() != 4 {
		t.Errorf("The item didn't get removed properly!! Got  %v, Want %v", GetDenyList().Length(), 4)
	}

	// delete onne, and one that isn't on it anymore
	GetDenyList().RemoveFromDenyList(thirdDeploymentItem)
	GetDenyList().RemoveFromDenyList(secondDeploymentItem)
	if GetDenyList().IsInConcurrentDenyList(thirdDeploymentItem) {
		t.Errorf("The item is still on the list!! Got  %v, Want %v", GetDenyList().IsInConcurrentDenyList(thirdDeploymentItem), false)
	}
	if GetDenyList().Length() != 3 {
		t.Errorf("The item didn't get removed properly!! Got  %v, Want %v", GetDenyList().Length(), 3)
	}

	//delete two at once
	GetDenyList().RemoveFromDenyList(fourthdeploymentItem)
	GetDenyList().RemoveFromDenyList(fithDeploymentItem)
	if GetDenyList().IsInConcurrentDenyList(fourthdeploymentItem) {
		t.Errorf("The item is still on the list!! Got  %v, Want %v", GetDenyList().IsInConcurrentDenyList(fourthdeploymentItem), false)
	}
	if GetDenyList().IsInConcurrentDenyList(fithDeploymentItem) {
		t.Errorf("The item is still on the list!! Got  %v, Want %v", GetDenyList().IsInConcurrentDenyList(fithDeploymentItem), false)
	}
	println("After third delete")
	if GetDenyList().Length() != 1 {
		t.Errorf("The item didn't get removed properly!! Got  %v, Want %v", GetDenyList().Length(), 1)
	}
}
