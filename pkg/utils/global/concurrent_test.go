package global

import (
	"testing"

	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPutOnBlackListAndIsFound(t *testing.T) {
	type args struct {
		deployment []v1.Deployment
	}
	tests := []struct {
		name   string
		args   args
		result bool
	}{
		{
			name: "TestPutOneOnBlackListAndIsFound",
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
			cs := GetBlackList()
			for _, item := range tt.args.deployment {
				cs.Append(ConvertDeploymentToItem(item))
			}

			for _, item := range tt.args.deployment {
				if cs.IsInConcurrentBlackList(ConvertDeploymentToItem(item)) != true {
					t.Errorf("The item is not in the Blacklist! Got  %v, Want %v", cs.IsInConcurrentBlackList(ConvertDeploymentToItem(item)), tt.result)
				}
			}
		})
	}

}

func TestBlackList(t *testing.T) {
	type args struct {
		deployment []v1.Deployment
	}
	tests := []struct {
		name   string
		args   args
		length int
	}{
		{
			name: "TestPutOneOnBlackList",
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
			name: "TestPutTwoOnBlackList",
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
			name: "TestPutDuplicateOnBlackList",
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
	cs := GetBlackList()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			for _, item := range tt.args.deployment {
				cs.Append(ConvertDeploymentToItem(item))
			}
			listle := cs.Length()
			if listle != tt.length {
				t.Errorf("The length is not correct! Got  %v, Want %v", listle, tt.length)
			}

			cs.PurgeBlackList()
		})
	}

}

func TestAddDuplicateAndPurge(t *testing.T) {
	cs := GetBlackList()
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

	cs.Append(deploymentItem)
	cs.Append(deploymentItemDuplicate)
	cs.Append(secondDeploymentItem)

	if cs.Length() != 2 {
		t.Errorf("Failed to put item on slice! Got  %v, Want %v", cs.Length(), 2)
	}

	cs.PurgeBlackList()

	if cs.Length() != 0 {
		t.Errorf("The slice didn't get purged correctly! Got  %v, Want %v", cs.Length(), 0)
	}
}
