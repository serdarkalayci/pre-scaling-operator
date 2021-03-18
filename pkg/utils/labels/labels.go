package labels

import (
	"strconv"
)

func GetLabelValue(labels map[string]string, optin string) bool {
	var detect bool
	if v, found := labels[optin]; found {

		detect, _ = strconv.ParseBool(v)
		if detect {
			return true
		}
	}
	return false
}
