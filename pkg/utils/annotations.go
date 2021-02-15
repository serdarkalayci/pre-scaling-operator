package annotations

import "strings"

func FilterByKeyPrefix(prefix string, annotations map[string]string) map[string]string {
	matches := make(map[string]string)
	for annotation, value := range annotations {
		if strings.HasPrefix(annotation, prefix) {
			matches[annotation] = value
		}
	}
	return matches
}
