/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// States defines the of desired states fields of ClusterScalingStateDefinition
type States struct {
	// Use name to define the cluster state name
	Name string `json:"name"`
	// Use description to describe the state
	Description string `json:"description,omitempty"`
}

// ClusterScalingStateDefinitionStatus defines the observed state of ClusterScalingStateDefinition
type ClusterScalingStateDefinitionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=clusterscalingstatedefinitions,scope=Cluster

// ClusterScalingStateDefinition is the Schema for the clusterscalingstatedefinitions API
type ClusterScalingStateDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   []States                            `json:"spec,omitempty"`
	Status ClusterScalingStateDefinitionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterScalingStateDefinitionList contains a list of ClusterScalingStateDefinition
type ClusterScalingStateDefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterScalingStateDefinition `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterScalingStateDefinition{}, &ClusterScalingStateDefinitionList{})
}
