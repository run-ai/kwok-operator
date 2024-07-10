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

// PodPoolSpec defines the desired state of PodPool
type PodPoolSpec struct {
	PodCount    int32      `json:"podCount"`
	PodTemplate corev1.Pod `json:"podTemplate"`
}

// PodPoolStatus defines the observed state of PodPool
type PodPoolStatus struct {
	Conditions         []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty" protobuf:"varint,2,opt,name=observedGeneration"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PodPool is the Schema for the podpools API
type PodPool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PodPoolSpec   `json:"spec,omitempty"`
	Status PodPoolStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PodPoolList contains a list of PodPool
type PodPoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PodPool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PodPool{}, &PodPoolList{})
}
