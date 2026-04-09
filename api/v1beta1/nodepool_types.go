/*
Copyright 2024.

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

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NodePoolSpec defines the desired state of NodePool
type NodePoolSpec struct {
	NodeCount    int32       `json:"nodeCount"`
	NodeTemplate corev1.Node `json:"nodeTemplate"`

	// GenerateUniqueSystemUUID, when true, sets a distinct status.nodeInfo.systemUUID on each
	// created node when the node template leaves systemUUID empty (SMBIOS-style uppercase UUID).
	// If the template already sets systemUUID, that value is copied unchanged to every node.
	// Default false preserves previous behavior (https://github.com/run-ai/kwok-operator/issues/20).
	// +optional
	// +kubebuilder:default=false
	GenerateUniqueSystemUUID bool `json:"generateUniqueSystemUUID,omitempty"`
}

// NodePoolStatus defines the observed state of NodePool
type NodePoolStatus struct {
	Conditions         []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty" protobuf:"varint,2,opt,name=observedGeneration"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// NodePool is the Schema for the nodepools API
type NodePool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodePoolSpec   `json:"spec,omitempty"`
	Status NodePoolStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NodePoolList contains a list of NodePool
type NodePoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodePool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodePool{}, &NodePoolList{})
}
