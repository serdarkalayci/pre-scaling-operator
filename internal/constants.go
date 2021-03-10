package internal

const LabelNotFound = "Opt-in label was not found"
const ResourceNotFound = "the server could not find the requested resource"
const OpenshiftObjectGroup = "apps.openshift.io/v1"
const OpenshiftResources = "DeploymentConfig"

var (
	OptInLabel = map[string]string{"scaler/opt-in": "true"}

	OpenshiftCluster bool
)
