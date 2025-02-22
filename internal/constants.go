package internal

import (
	"time"
)

const (
	//LabelNotFound is the message for when the label doesn't exist in the application manifest
	LabelNotFound = "opt-in label was not found"

	RQNotFound = "No resource quotas found"

	//ResourceNotFound is the message returned when the API server doesn't have the desired resource
	ResourceNotFound = "the server could not find the requested resource"

	//OpenshiftObjectGroup is the resource group and version of openshift objects
	OpenshiftObjectGroup = "apps.openshift.io/v1"

	//RedisClusterObjectGroup is the resource group and version of rediscluster objects
	RedisClusterObjectGroup = "redis.containersolutions.com/v1alpha1"

	//OpenshiftResources respresents the Openshift object to watch
	OpenshiftResources = "DeploymentConfig"

	//Key for the default replica annotation
	DefaultReplicaAnnotation = "default"

	EnvMaxConcurrentNamespaceReconciles = "MaxConcurrentNamespaceReconciles"

	RetriggerControllerSeconds = 15
)

type ScalingClass struct {
	Name string
}

var (
	//OptInLabel represents the label key and value for the opted-in applications
	OptInLabel = map[string]string{"scaler/opt-in": "true"}

	//OpenshiftCluster is used to identify if the operator is running in an Openshift cluster
	OpenshiftCluster bool

	//RedisCluster is used to identify if there might be RedisCluster resources present in the cluster
	RedisCluster bool
	
	StartTime time.Time

	DefaultScalingClass = ScalingClass{
		Name: "default",
	}
)
