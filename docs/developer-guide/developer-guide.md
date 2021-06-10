# Operator Usage/ScalingStates

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
 - name: marketing-runs
    description: "Use when running marketing campaigns, and higher-than-normal load is expected"
    priority: 5
  - name: peak
    description: "Maximum scale settings."
    priority: 1
config:
  dryRun: false
```

### ClusterScalingState

Defines the current state that a cluster is set to. 

Updating this CRD, will trigger the operator to scale all opted-in applications, 
to their desired replicas for that particular state

```yaml
kind: ClusterScalingState
spec:
  state: merketing-runs
config:
  dryRun: false
```

### ScalingState

Can be used to override the state for a particular namespace.

```yaml
kind: ScalingState
metadata:
  namespace: product
spec:
  state: peak
config:
  dryRun: false
```

### 
```yaml
config:
  dryRun: false
```
DryRun explained in the CustomResources above: If dryRun is set to `true` the operator will create an event with an elaborate message what _would_ happen if the CustomResource is applied for the specific state in that CustomResource. 
For example:
```
Events:
  Type    Reason  Age              From                            Message
  ----    ------  ----             ----                            -------
  Normal  DryRun  1s (x2 over 5s)  clusterscalingstate-controller  DryRun: +-----------+---------------+------------------------+---------------------------+
| NAMESPACE | QUOTAS ENOUGH | CPU LEFT AFTER SCALING | MEMORY LEFT AFTER SCALING |
+-----------+---------------+------------------------+---------------------------+
| default   | true          | 3300m                  | 3550Mi                    |
+-----------+---------------+------------------------+---------------------------+
+--------------------+------------------+-----------+--------------+---------------+
|    APPLICATION     | CURRENT REPLICAS | NEW STATE | NEW REPLICAS | RAPID SCALING |
+--------------------+------------------+-----------+--------------+---------------+
| random-generator-1 |                1 | peak      |            5 | false         |
| random-generator-2 |                1 | peak      |            5 | false         |
+--------------------+------------------+-----------+--------------+---------------+

```
The Operator will *not* make changes on the cluster based on the applied CustomResource. It'll simply report back in terms of what would happen. <br>

*Important note*: DryRun set to `true` on a custom resource will not disable the operator alltogether. Changes from another CustomResource, or annotation on a deployment for example, would still lead to the operator reflecting that change on the cluster.

### Scaling state priority
Each scaling state has a priority setting.

This is used when deciding whether to use the ClusterScalingState or ScalingState as the current state for an application.

If the cluster is set to business-as-usual, and the namespace is set to peak, peak should be selected, and in reverse is the namespace is only set  to business-as-usual and the cluster to peak, peak setting should be used to prevent under provisioning of applications.

The priority settings here, delimits the ranking of a state over others.

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
    scaler/rapid-scaling: "false"
    scaler/state-peak-replicas: 50
    scaler/state-bau-replicas: 15
    scaler/state-default-replicas: 15
```

### Allow Autoscaling

`scaler/allow-autoscaling` <br>
Autoscaling will be defaulted to false, to protect applications which are not ready, or mature enough to leverage autoscaling. 

When autoscaling is enabled, the application will scale freely using metrics, and be capable of using custom metrics with the normal HPA underlying. We will in this case only manage the minimum replica count.


### Rapid Scaling/StepScaling:

If the annotation `scaler/rapid-scaling:` is set to `true`, the operator will scale the Deployment or DeploymentConfig in a single Step. e.g. From 5 -> 10. 
By default, if the value is set to `false`, or if the annotation is missing, the stepScaler will scale to Deployment or DeploymentConfig towards the intended replicacount step by step and will check for the readiness for each pod along the way.


### Default Replica Count

An application should define a default replica count using scaler/state-default-replicas. This is treated as a regular state and can be used to direct the application to scale back to the user-defined default state.
