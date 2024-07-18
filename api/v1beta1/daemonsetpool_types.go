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

// DaemonsetPoolSpec defines the desired state of DaemonsetPool

type DaemonsetPoolSpec struct {
	DaemonsetTemplate appsv1.DaemonSet `json:"daemonsetTemplate"`
}

// DaemonsetPoolStatus defines the observed state of DaemonsetPool
type DaemonsetPoolStatus struct {
	Conditions         []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty" protobuf:"varint,2,opt,name=observedGeneration"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DaemonsetPool is the Schema for the daemonsetpools API
type DaemonsetPool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DaemonsetPoolSpec   `json:"spec,omitempty"`
	Status DaemonsetPoolStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DaemonsetPoolList contains a list of DaemonsetPool
type DaemonsetPoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DaemonsetPool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DaemonsetPool{}, &DaemonsetPoolList{})
}
