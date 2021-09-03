#!/bin/bash

# Delete e2e namespaces
kubectl get namespaces --no-headers=true -o custom-columns=:metadata.name | grep e2e > temp/namespaces.txt
while read p; do 
    kubectl delete namespace "$p"
done < temp/namespaces.txt

# Delete ClusterScalingStates
kubectl get ClusterScalingStates --no-headers=true -o custom-columns=:metadata.name > temp/clusterScalingStates.txt
while read p; do 
    kubectl delete ClusterScalingState "$p"
done < temp/clusterScalingStates.txt

# Delete ClusterScalingStateDefinitions
kubectl get ClusterScalingStateDefinition --no-headers=true -o custom-columns=:metadata.name > temp/clusterScalingStateDefinitions.txt
while read p; do 
    kubectl delete ClusterScalingStateDefinition "$p"
done < temp/clusterScalingStateDefinitions.txt

# If there is a prescaler operator running in any namespace, scale it to 0
kubectl get deployments -A -l "operator=pre-scaling-operator" --no-headers=true -o custom-columns=:metadata.namespace,:metadata.name > temp/operators.txt
while read p; do 
    line=($p)
    kubectl -n "${line[0]}" scale deployment "${line[1]}" --replicas=0
done < temp/operators.txt
