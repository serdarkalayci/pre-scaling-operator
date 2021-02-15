# Pre-Scaling Operator Docs

Welcome to the Operators docs. 

In here you will find all the information you need to contribute, 
as well as the architecture, and understanding how the Operator works.

It's a good idea to read the [Goals page](goals.md), to understand the reasoning and constraints behind the operator,
before setting off on big changes to it.

## Table of Contents
1. [Architecture](architecture.md)
2. [Limitations](limitations.md)

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
