# Operator Architecture

The Operator can be used as a layer on top of the 
(Horizontal Pod Autoscaler)[https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/],
leveraging the pod autoscaler for auto-scaled applications, and setting replica counts for other types of applications

## Three CRDs

### ClusterScalingStateDefinition

Defines all of the possible states that a cluster and application can be set to, 
including each states configuration and priority.

This is primarily used to set your business specific states.

```yaml
kind: ClusterScalingStateDefinition
spec:
  states:
  - name: bau
    description: "Business as usual. Scaling state of normal everyday operations"
    priority: 10
    rate-limit: {}
  - name: marketing-runs
    description: "Use when running marketing campaigns, and higher-than-normal load is expected"
    priority: 5
    rate-limit: {}
  - name: peak
    description: "Maximum scale settings."
    priority: 1
    rate-limit: {}
```

### ClusterScalingState

Defines the current state that a cluster is set to. 

Updating this CRD, will trigger the operator to scale all opted-in applications, 
to their desired replicas for that particular state

```yaml
kind: ClusterScalingState
spec:
  state: merketing-runs
```

### ScalingState

Can be used to override the state for a particular namespace.

```yaml
kind: ScalingState
metadata:
  namespace: product
spec:
  state: peak
```

### Scaling state priority
Each scaling state has a priority setting.

This is used when deciding whether to use the ClusterScalingState or ScalingState as the current state for an application.

If the cluster is set to business-as-usual, and the namespace is set to peak, peak should be selected, and in reverse is the namespace is only set  to business-as-usual and the cluster to peak, peak setting should be used to prevent under provisioning of applications.

The priority settings here, delimits the ranking of a state over others.

### Scaling state rate limit
Each scaling state has rate-limiting settings which are applied during scale-ups. Scale downs are done at full speed, as scale-down is usually the safest of the two.

These rate limits can be defined as applications-per-minute (Only X kubectl scale executions per minute)  or applications-at-once (Only x deployments may have ready !== desired at any one time)

## Operator is Opt-in only
In order to protect the applications, and enable a gradual rollout of the Scaler in our platforms, the Operator is strictly opt-in.

Opt-in uses Kubernetes labels to opt-in to the scaling mechanism. 

```yaml
kind: Deployment
metdata:
  labels:
    scaler/opt-in: true
```

This also allows easier searching for applications to the operator using Kubernetes label selection.

## Application configuration

Applications define their scaling and scale requirements using annotations
For applications to setup their application for different states, they need to use annotations on the deployment resources to define which type of scaling is required, and how many replicas are required for different states

```yaml
kind: Deployment
metadata:
  labels: 
    scaler/opt-in: true
  annotations:
    scaler/allow-autoscaling: true # true | false 
    scaler/state-peak-replicas: 50
    scaler/state-bau-replicas: 15
    scaler/state-default-replicas: 15
```

### Allow Autoscaling

Autoscaling will be defaulted to false, to protect applications which are not ready, or mature enough to leverage autoscaling. 

When autoscaling is enabled, the application will scale freely using metrics, and be capable of using custom metrics with the normal HPA underlying. We will in this case only manage the minimum replica count.

### Default Replica Count

An application should define a default replica count using scaler/state-default-replicas to return to, when no specific state can be determined. 

The default annotation will also automatically be set by the Scaler when the application is opted in and scaled for the first time, if no default exists in annotations. This protects applications which have not set a default from returning to 1 replica.
