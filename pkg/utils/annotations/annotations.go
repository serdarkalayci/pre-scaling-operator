package annotations

import (
	"strings"

	v1 "k8s.io/api/apps/v1"
)

func FilterByKeyPrefix(prefix string, annotations map[string]string) map[string]string {
	matches := make(map[string]string)
	for annotation, value := range annotations {
		if strings.HasPrefix(annotation, prefix) {
			matches[annotation] = value
		}
	}
	return matches
}

func PutAnnotationOnDeployment(deployment v1.Deployment, key string, value string) v1.Deployment {

	annotations := deployment.Annotations
	// make sure it's not in the map already
	delete(annotations, key)

	annotations[key] = value
	deployment.Annotations = annotations

	return deployment
}

func RemoveAnnotationFromDeployment(deployment v1.Deployment, key string) v1.Deployment {
	annotations := deployment.Annotations
	delete(annotations, key)
	deployment.Annotations = annotations

	return deployment
}
