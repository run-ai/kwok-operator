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

// KwokControllerSpec defines the desired state of KwokController
type KwokControllerSpec struct {
	// Version is the KWOK image tag to deploy (e.g. "v0.6.0").
	// +kubebuilder:validation:Required
	Version string `json:"version"`

	// Namespace is the target namespace for the KWOK controller Deployment and ServiceAccount.
	// +optional
	// +kubebuilder:default="kube-system"
	Namespace string `json:"namespace,omitempty"`

	// Replicas is the desired replica count for the KWOK controller Deployment.
	// +optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=0
	Replicas *int32 `json:"replicas,omitempty"`

	// Resources specifies CPU and memory resource requirements for the KWOK container.
	// Setting a memory limit is the primary mechanism to address issue #1494 (~300MB per instance).
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// EnableCRDs lists specific KWOK CRD kinds to watch (e.g. ["Stage"]).
	// When non-empty, KWOK's informer cache is limited to only those CRDs,
	// which reduces memory consumption significantly.
	// +optional
	EnableCRDs []string `json:"enableCRDs,omitempty"`

	// ManageAllNodes, when true, passes --manage-all-nodes=true to the KWOK controller,
	// causing it to manage every node regardless of annotation.
	// Default false means only nodes matching NodeAnnotationSelector are managed.
	// +optional
	// +kubebuilder:default=false
	ManageAllNodes bool `json:"manageAllNodes,omitempty"`

	// NodeAnnotationSelector is the annotation selector used to identify fake nodes
	// when ManageAllNodes is false.
	// +optional
	// +kubebuilder:default="kwok.x-k8s.io/node=fake"
	NodeAnnotationSelector string `json:"nodeAnnotationSelector,omitempty"`

	// AdditionalFlags are extra command-line flags appended to the KWOK controller container args.
	// +optional
	AdditionalFlags []string `json:"additionalFlags,omitempty"`
}

// KwokControllerStatus defines the observed state of KwokController
type KwokControllerStatus struct {
	Conditions         []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty" protobuf:"varint,2,opt,name=observedGeneration"`
	// ReadyReplicas is the number of ready replicas in the managed Deployment.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`
	// DeployedVersion is the KWOK image tag currently deployed.
	// +optional
	DeployedVersion string `json:"deployedVersion,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// KwokController is the Schema for the kwokcontrollers API
type KwokController struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KwokControllerSpec   `json:"spec,omitempty"`
	Status KwokControllerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KwokControllerList contains a list of KwokController
type KwokControllerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KwokController `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KwokController{}, &KwokControllerList{})
}
