package internal

const LabelNotFound = "Opt-in label was not found"

var (
	OptInLabel = map[string]string{"scaler/opt-in": "true"}

	OpenshiftCluster bool
)
