#!/bin/bash

# Delete e2e namespaces
kubectl get namespaces --no-headers=true -o custom-columns=:metadata.name | grep e2e > namespace.txt
while read p; do 
    kubectl delete namespace "$p"
done < namespace.txt

# Delete all ClusterScalingStates
kubectl get ClusterScalingStates --no-headers=true -o custom-columns=:metadata.name > clusterScalingStates.txt
while read p; do 
    kubectl delete ClusterScalingState "$p"
done < clusterScalingStates.txt

# Delete all ClusterScalingStateDefinitions
kubectl get ClusterScalingStateDefinition --no-headers=true -o custom-columns=:metadata.name > clusterScalingStateDefinitions.txt
while read p; do 
    kubectl delete ClusterScalingStateDefinition "$p"
done < clusterScalingStateDefinitions.txt

# If there is a prescaler operator running in any namespace, scale it to 0
kubectl get deployments -A -l "operator=pre-scaling-operator" --no-headers=true -o custom-columns=:metadata.namespace,:metadata.name > operators.txt
while read p; do 
    line=($p)
    kubectl -n "${line[0]}" scale deployment "${line[1]}" --replicas=0
done < operators.txt

# Remove txt files
rm clusterScalingStateDefinitions.txt
rm namespace.txt
rm clusterScalingStates.txt
rm operators.txt