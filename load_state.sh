#!/bin/bash

# Restore namespaces
FILES="./temp/namespace/*"
for f in $FILES
do
    kubectl apply -f $f
done

# Restore ClusterScalingStates
FILES="./temp/clusterScalingState/*"
for f in $FILES
do
    kubectl apply -f $f
done

# Restore ClusterScalingStateDefinitions
FILES="./temp/clusterScalingStateDefinition/*"
for f in $FILES
do
    kubectl apply -f $f
done

# Scale operators back to 1
while read p; do 
    line=($p)
    kubectl -n "${line[0]}" scale deployment "${line[1]}" --replicas=1
done < temp/operators.txt