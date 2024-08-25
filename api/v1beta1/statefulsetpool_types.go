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
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StatefulsetPoolSpec defines the desired state of StatefulsetPool
type StatefulsetPoolSpec struct {
	CreatePV            bool               `json:"createPV,omitempty"`
	StatefulsetTemplate appsv1.StatefulSet `json:"StatefulsetTemplate"`
}

// StatefulsetPoolStatus defines the observed state of StatefulsetPool
type StatefulsetPoolStatus struct {
	Conditions         []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty" protobuf:"varint,2,opt,name=observedGeneration"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// StatefulsetPool is the Schema for the statefulsetpools API
type StatefulsetPool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StatefulsetPoolSpec   `json:"spec,omitempty"`
	Status StatefulsetPoolStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// StatefulsetPoolList contains a list of StatefulsetPool
type StatefulsetPoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StatefulsetPool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StatefulsetPool{}, &StatefulsetPoolList{})
}
