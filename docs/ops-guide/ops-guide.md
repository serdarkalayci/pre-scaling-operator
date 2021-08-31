### Ops-Guide

!Important! Please read the Developer-guide before reading the ops-guide in order to understand ScalingStates!

## Deployment

Coming soon

## Permissions and RBAC

The ClusterRole the Operator-SA needs:


```yaml
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  creationTimestamp: null
  name: pre-scaling-operator-role
rules:
- apiGroups:
  - ""
  resources:
  - events
  verbs:
  - create
  - patch
- apiGroups:
  - ""
  resources:
  - namespaces
  verbs:
  - list
  - watch
- apiGroups:
  - ""
  resources:
  - resourcequotas
  verbs:
  - list
- apiGroups:
  - apps
  resources:
  - deployments
  verbs:
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - apps.openshift.io
  resources:
  - deploymentconfigs
  verbs:
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - scaling.prescale.com
  resources:
  - clusterscalingstatedefinitions
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - scaling.prescale.com
  resources:
  - clusterscalingstatedefinitions/finalizers
  verbs:
  - update
- apiGroups:
  - scaling.prescale.com
  resources:
  - clusterscalingstatedefinitions/status
  verbs:
  - get
  - patch
  - update
- apiGroups:
  - scaling.prescale.com
  resources:
  - clusterscalingstates
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - scaling.prescale.com
  resources:
  - clusterscalingstates/finalizers
  verbs:
  - update
- apiGroups:
  - scaling.prescale.com
  resources:
  - clusterscalingstates/status
  verbs:
  - get
  - patch
  - update
- apiGroups:
  - scaling.prescale.com
  resources:
  - scalingstates
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - scaling.prescale.com
  resources:
  - scalingstates/finalizers
  verbs:
  - update
- apiGroups:
  - scaling.prescale.com
  resources:
  - scalingstates/status
  verbs:
  - get
  - patch
  - update
```

## (Optional) Roles to view/edit Custom Resources

Additionally there can be roles for team-members to edit the CustomResources.
Types of Custom Resources that could be subject to roles are:

- ClusterScalingStateDefinition (Cluster-wide)
  - Resource-Name: `clusterscalingstatedefinitions`
- ClusterScalingState (Cluster-wide)
  - Resource-Name: `clusterscalingstates`
- ScalingState (Namespaced)
  - Resource-Name: `scalingstates` _(See examples below)_

 For example to change the ScalingStates in all namespaces:

Editor:
```yaml
# permissions for end users to edit scalingstates.
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: scalingstate-editor-role
rules:
- apiGroups:
  - scaling.prescale.com
  resources:
  - scalingstates
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - scaling.prescale.com
  resources:
  - scalingstates/status
  verbs:
  - get
```

For team-members to be able to view ScalingStates

```yaml
# permissions for end users to view scalingstates.
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: scalingstate-viewer-role
rules:
- apiGroups:
  - scaling.prescale.com
  resources:
  - scalingstates
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - scaling.prescale.com
  resources:
  - scalingstates/status
  verbs:
  - get
```


## Configuration

There is no ConfigMap or Environment variables for the Operator.

All behaviour is determined out of a combination of:

- ClusterScalingStateDefinition:
    - Priority of the defined states
- ClusterScalingState
    - The cluster-wide state
- ScalingState
    - The namespace-wide state 
- Deployments/DeploymentConfig
    - Optin-Label
        - Example: <br>
        ```yaml
          labels:
            scaler/opt-in: "true" 
        ```
    - State-Annotations
        - Example: <br>
        ```yaml
        annotations:
            scaler/state-peak-replicas: "5"
            scaler/state-bau-replicas: "3"
            scaler/state-default-replicas: "1"
        ```
    - Rapid-Scaling Annotation
        - Example: <br>
        ```yaml
        annotations:
            scaler/rapid-scaling: "false"
        ```
