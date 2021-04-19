# Operator goals

There are a few key features the operator looks to solve,
and these should always be catered for in some manner. 

## Should allow for 3 types of applications

### Don’t scale

These applications should not be scaled automatically and often require manual configuration changes. 

eg. Data Stores

### Scale once

These applications are allowed to scale up in anticipation of load, but should not scale thereafter, as they might be unstable for the first couple minutes after scaling to reset quorum, or distribute processing.

eg. Queue workers, Sharded caches

### Auto-scale

Applications can scale freely at any point in time.

eg. Microservices

## Should be Kubernetes Native

It should support Kubernetes Distributions such as Openshift as well. 

It's architecture should never really be concerned with the internal working of the distribution,
but should support the distributions resources which use the `.spec.replicas` field.

`DeploymentConfigs` in Openshift for example.

Immediate Support
* Deployment
* DeploymentConfig

To be added in the future:
* Statefulset
* ReplicaSet

## Should be able to scale and entire cluster, or just a single namespace

This allows a system to be pre-scaled for a load spike, or a single namespace for testing

## Should be able to rate-limit scaling up of clusters

Should be able to rate-limit it’s scaling up procedure to prevent all applications in a cluster scaling at the same time. 

It should have some mechanism to specify applications-at-a-time or applications-per-minute type of rate limit

## Should have a simple switch from one state to another

It should have a simple switch between states, in our case, a CRD update
