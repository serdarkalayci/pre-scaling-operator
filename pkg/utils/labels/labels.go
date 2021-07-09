package labels

import (
	"strconv"
)

func GetLabelValueBool(labels map[string]string, optin string) bool {
	var detect bool
	if v, found := labels[optin]; found {

		detect, _ = strconv.ParseBool(v)
		if detect {
			return true
		}
	}
	return false
}

func GetLabelValueString(labels map[string]string, key string) string {
	if v, found := labels[key]; found {
		return v
	}
	return ""
}
