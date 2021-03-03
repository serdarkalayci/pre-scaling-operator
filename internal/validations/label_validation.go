package validations

import (
	"errors"
	"strconv"
)

const LabelNotFound = "Opt-in label was not found"

var (
	OptInLabel = map[string]string{"scaler/opt-in": "true"}
)

// OptinLabelExists checks if the opt-in label exists in the target object and returns its value
func OptinLabelExists(labels map[string]string) (bool, error) {

	var optinlabel bool

	for k := range OptInLabel {
		if v, found := labels[k]; found {
			optinlabel, err := strconv.ParseBool(v)
			return optinlabel, err
		}
	}
	return optinlabel, errors.New(LabelNotFound)
}
