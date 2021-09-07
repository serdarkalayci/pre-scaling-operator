#!/bin/bash

mkdir -p temp/clusterScalingState
mkdir  -p temp/clusterScalingStateDefinition
mkdir -p temp/namespace

# Save e2e namespaces to file
kubectl get namespaces --no-headers=true -o custom-columns=:metadata.name | grep e2e > temp/namespaces.txt
while read p; do 
    kubectl get namespace "$p" -o yaml > temp/namespace/"$p".yaml
done < temp/namespaces.txt

# Save ClusterScalingStates to file
kubectl get ClusterScalingStates --no-headers=true -o custom-columns=:metadata.name > temp/clusterScalingStates.txt
while read p; do 
    kubectl get ClusterScalingState "$p" -o yaml > temp/clusterScalingState/"$p".yaml
done < temp/clusterScalingStates.txt

# Save ClusterScalingStateDefinition to file
kubectl get ClusterScalingStateDefinition --no-headers=true -o custom-columns=:metadata.name > temp/clusterScalingStateDefinitions.txt
while read p; do 
    kubectl get ClusterScalingStateDefinition "$p" -o yaml > temp/clusterScalingStateDefinition/"$p".yaml
done < temp/clusterScalingStateDefinitions.txt