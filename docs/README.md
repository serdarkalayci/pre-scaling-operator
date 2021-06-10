# Pre-Scaling Operator Docs

## Structure

* Folder ./architecture Information about the software architecture and inner working of the operator
* Folder ./ops-guide: Information about the operations of the operator
* Folder ./developer-guide: Information how to interact with the operator as a developer
* [Goals page](goals.md) Goals/Objectives the Operator tries to accomplish
* [Limitations](limitations.md) Current limitation the operator is operating under
* [Roadmap](ROADMAP.md) Future roadmap of the operator. Where we want to go

<span class="sidebar"></span>

### Basics

The Operator helps pre-scale applications in anticipation of load.

At its core, it manages a cluster / namespace "state", 
which triggers scaling of applications to the desired replicas for that specific state. 

For example, a cluster could have 2 states, `business-as-usual`, and `peak`.
Applications can configure the replicas needed for each state. When a cluster is switched from one state to the other, the operator will take care of scaling all of the applications to the necessary replicas, as set by developers in the `platform.yaml`.

From that point forward, the operator is controlling the size of all the applications that have chosen to opt-in. This is possible through a label which is specified in the `platform.yaml` as well.


#### Use cases

This operator aims to provide a solution to problems around massive and fast scaling needs. The Kubernetes Horicontal Pod Autoscaler cannot handle quick bursts of scale and that is the main reason why in these use cases it is not preferred. What happens then is that people pre-scale manually their applications to ensure that the anticipated load will be handled well.

Therefore, the operator can be very useful in situations with one or more of the following requirements:
* Anticipation of significantly higher than average traffic
* Multi-tenant or very diverse clusters that groups of applications need to scale together
* Same cluster applications with different scaling needs (for example, some applications should not scale, some should scale only one and some should be allowed to autoscale)
* Ability to prepare multiple applications together for an expected situation using an easy and declarative switch

#### Behaviour-Overview

The Operator has the following behaviour:

* The Operator will not scale an application if the application is not opted in. 

* The Operator will not scale if there's no scalingstate present. (Cluster-wide or namespace-wide)

* **Scaling an aplication:** If a scalingstate is active, the application is opted in, and the application has that scalingstate defined, the operator will scale the deployment to the specified replica count as per the scaling state.

* **Modification of `spec.replicas`:** If an application is opted in, and the application replica count is manipulated from elsewhere, the operator will set the replica count back to the replica count of the specific active scaling state.

* **Modification of a state:** If there's a change to the replica-count for a state, the operator will take that change into account as long as the application is opted in. There is also an optional `admission-controller` that can prevent that modification all-together. You can find its repository link [here](https://github.com/ContainerSolutions/pre-scaling-operator-admission-controller)

* **Default State:** All applications include a default state. The DevOps team can change the scalingstate to that default state at any time. The replica count for the default state is always what the deployment process would output into the `spec.replicas` field.

* **Resource Quota:** If there are resource requests on the application, and a resource quota in the namespace, the operator will take these into consideration. If the operator is requested to scale to a number that would violate the ResourceQuota on the namespace, the operator will not scale the application.

* **Allow autoscaling:** Developers can specify an `allow-autoscaling` flag (true|false) to allow the Horizontal Pod Autoscaler to scale above a certain state. The Pre-Scaler Operator makes sure that the application is on a certain base level of replicas, while the Horizontal Pod Autoscaler can take care of fine adjustments based on load. The Horizontal Pod Autoscaler can't scale below the given scalerstate.


## Future enhancements we'd like to see

### Rate limiting using stepped increase
Instead of jumping from 2 replicas to 50, step through replica counts over a period of time (eg. 4 replicas per minute)

Useful for teams uncomfortable with scaling too fast, and wanting to validate as the scaling proceeds.

### Scheduled Scaling
Schedule a state change for a specified datetime. 

Useful for scaling down after peak periods, instead of having to apply it manually, or scaling up after hours.

### Mechanism to target multiple namespaces
Instead of limiting a ScalingState to one namespace, a possible mechanism to allow selection of a set of namespaces by label.

Useful for applications spanning multiple namespaces

### Ingress-like Scaler 
Using class in the same fashion as an Ingress controller, whereby different subsets of Deployments can be scaled using different Scalers.

Useful for cluster with multiple environments. One Scaler for each environment, and scaling handled for each using different ClusterScalingState.
