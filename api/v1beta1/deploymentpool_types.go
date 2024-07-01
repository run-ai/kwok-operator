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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DeploymentPoolSpec defines the desired state of DeploymentPool
type DeploymentPoolSpec struct {
	DeploymentTemplate appsv1.Deployment `json:"deploymentTemplate"`
}

// DeploymentPoolStatus defines the observed state of DeploymentPool
type DeploymentPoolStatus struct {
	Conditions         []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty" protobuf:"varint,2,opt,name=observedGeneration"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DeploymentPool is the Schema for the deploymentpools API
type DeploymentPool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeploymentPoolSpec   `json:"spec,omitempty"`
	Status DeploymentPoolStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DeploymentPoolList contains a list of DeploymentPool
type DeploymentPoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeploymentPool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DeploymentPool{}, &DeploymentPoolList{})
}
