package labels_test

import (
	"testing"

	"github.com/containersol/prescale-operator/pkg/utils/labels"
)

func TestOptInLabelRetrievalFromLabelsMap(t *testing.T) {
	var key string = "the/key"

	// Present and true
	labelPresentTrue := map[string]string{
		"something": "else",
		key:         "true",
	}

	// Present but false
	labelPresentFalse := map[string]string{
		"something": "else",
		key:         "false",
	}

	// No label present
	labelNotPresent := map[string]string{
		"something": "else",
	}

	gotPresentTrue := labels.GetLabelValue(labelPresentTrue, key)
	gotPresentFalse := labels.GetLabelValue(labelPresentFalse, key)
	gotNotPresent := labels.GetLabelValue(labelNotPresent, key)

	if !gotPresentTrue {
		t.Errorf("The label should be true as the label is present!")
	}

	if gotPresentFalse {
		t.Errorf("The result should be false, as the key is false!")
	}

	if gotNotPresent {
		t.Errorf("The result should be false as the label does not exist!")
	}

}
