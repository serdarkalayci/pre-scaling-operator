package annotations_test

import (
	annotations "github.com/containersol/prescale-operator/pkg/utils"
	"reflect"
	"testing"
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
