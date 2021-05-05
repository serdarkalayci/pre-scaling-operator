package annotations_test

import (
	"reflect"
	"testing"

	annotations "github.com/containersol/prescale-operator/pkg/utils/annotations"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFilterByKeyPrefixFiltersCorrectly(t *testing.T) {
	deploymentAnnotations := map[string]string{
		"deployment.kubernetes.io/revision": "1",
		"scaler/opt-in":                     "true",
		"scaler/state-bau-replicas":         "5",
	}

	expected := map[string]string{
		"scaler/opt-in":             "true",
		"scaler/state-bau-replicas": "5",
	}
	got := annotations.FilterByKeyPrefix("scaler", deploymentAnnotations)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Annotations not filtered correctly. Expected %s, got %s", expected, got)
	}
}

func TestFilterByKeyPrefixReturnsEmptyMapIfNothingMatches(t *testing.T) {
	deploymentAnnotations := map[string]string{
		"deployment.kubernetes.io/revision": "1",
		"scaler/opt-in":                     "true",
		"scaler/state-bau-replicas":         "5",
	}

	expected := map[string]string{}
	got := annotations.FilterByKeyPrefix("foobar", deploymentAnnotations)
	if !reflect.DeepEqual(got, map[string]string{}) {
		t.Errorf("Annotations not filtered correctly. Expected %s, got %s", expected, got)
	}
}

func TestAnnotationIsDeletedCorrectly(t *testing.T) {
	deploymentAnnotations := map[string]string{
		"foo":     "bar",
		"some":    "content",
		"another": "annotation",
	}

	depl := v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "foo",
			Namespace:   "bar",
			Annotations: deploymentAnnotations,
		},
		Spec: v1.DeploymentSpec{
			Replicas: new(int32),
		},
		Status: v1.DeploymentStatus{
			Replicas: 5,
		},
	}
	expected := map[string]string{
		"some":    "content",
		"another": "annotation",
	}

	got := annotations.RemoveAnnotationFromDeployment(depl, "foo").Annotations
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Annotation not deleted. Expected %s, got %s", expected, got)
	}
}

func TestPutOnDeployment(t *testing.T) {
	deploymentAnnotations := map[string]string{
		"some":    "content",
		"another": "annotation",
	}

	depl := v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "foo",
			Namespace:   "bar",
			Annotations: deploymentAnnotations,
		},
		Spec: v1.DeploymentSpec{
			Replicas: new(int32),
		},
		Status: v1.DeploymentStatus{
			Replicas: 5,
		},
	}

	expected := map[string]string{
		"some":    "content",
		"another": "annotation",
		"foo":     "bar",
	}

	got := annotations.PutAnnotationOnDeployment(depl, "foo", "bar").Annotations
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Annotation not found on deployment. Expected %s, got %s", expected, got)
	}
}
