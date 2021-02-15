# Current Operator Limitations

## One ClusterScalingStateDefinition per Cluster
More than one definition would need to be merged by the Operator, and an easier solution is to only allow one.

Webhook validation should be used to reject any attempt to create more than one

We would like to extend this to have an Ingress-like Scaler "class", 
in order to support multiple environments in the same cluster,
and each operator only managing a subset of resources.
 

## One ClusterScalingState per Cluster
More than one definition would need to be merged by the Operator, and an easier solution is to only allow one.

Webhook validation should be used to reject any attempt to create more than one

Solved by Ingress-like scaler mentioned above

## One ScalingState per Namespace
More than one definition would need to be merged by the Operator, and an easier solution is to only allow one.

Webhook validation should be used to reject any attempt to create more than one

We'd also like to support resources which can target multiple namespaces for applications which stretch multiple namespaces
