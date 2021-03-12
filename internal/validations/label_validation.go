package validations

import (
	"errors"
	"strconv"

	c "github.com/containersol/prescale-operator/internal"
)

// OptinLabelExists checks if the opt-in label exists in the target object and returns its value
func OptinLabelExists(labels map[string]string) (bool, error) {

	var optinlabel bool

	for k := range c.OptInLabel {
		if v, found := labels[k]; found {
			optinlabel, err := strconv.ParseBool(v)
			return optinlabel, err
		}
	}
	return optinlabel, errors.New(c.LabelNotFound)
}
