package validations

import (
	"errors"
	"github.com/containersol/prescale-operator/internal/reconciler"
	v1 "k8s.io/api/apps/v1"
	"strconv"
)

const LabelNotFound = "Opt-in label was not found"

// OptinLabelExists checks if the opt-in label exists in the target object and returns its value
func OptinLabelExists(deployment v1.Deployment) (bool, error) {

	var optinlabel bool

	labels := deployment.GetLabels()
	for k := range reconciler.OptInLabel {
		if v, found := labels[k]; found {
			optinlabel, err := strconv.ParseBool(v)
			return optinlabel, err
		}
	}
	return optinlabel, errors.New(LabelNotFound)
}
