package validations

import (
	"reflect"
	"testing"

	c "github.com/containersol/prescale-operator/internal"
)

func TestOptinlabelWhenLabelIsSetCorrectly(t *testing.T) {

	optinLabel := map[string]string{"scaler/opt-in": "true"}

	expected := true
	got, err := OptinLabelExists(optinLabel)
	if err != nil {
		t.Errorf("Failed to get label")
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Opt-in label not identified correctly. Expected %v, got %v", expected, got)
	}
}

func TestOptinlabelWhenLabelIsAbsent(t *testing.T) {

	optinLabel := map[string]string{"foo": "bar"}

	expected := false
	got, err := OptinLabelExists(optinLabel)
	if !reflect.DeepEqual(err.Error(), c.LabelNotFound) {
		t.Errorf("Error was not identified correctly. Expected %s, got %s", c.LabelNotFound, err.Error())
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Opt-in label not identified correctly. Expected %v, got %v", expected, got)
	}
}
