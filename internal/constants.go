package internal

const (
	//LabelNotFound is the message for when the label doesn't exist in the application manifest
	LabelNotFound = "Opt-in label was not found"

	//ResourceNotFound is the message returned when the API server doesn't have the desired resource
	ResourceNotFound = "the server could not find the requested resource"

	//OpenshiftObjectGroup is the resource group and version of openshift objects
	OpenshiftObjectGroup = "apps.openshift.io/v1"

	//OpenshiftResources respresents the Openshift object to watch
	OpenshiftResources = "DeploymentConfig"

	//Key for the default replica annotation
	DefaultReplicaAnnotation = "default"
)

var (
	//OptInLabel represents the label key and value for the opted-in applications
	OptInLabel = map[string]string{"scaler/opt-in": "true"}

	//OpenshiftCluster is used to identify if the operator is running in an Openshift cluster
	OpenshiftCluster bool
)
