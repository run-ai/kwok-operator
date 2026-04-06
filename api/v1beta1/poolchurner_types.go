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

// PoolChurnerWorkloadType identifies which fake workload the churner manages.
// Only Pod is implemented today; others may be added for issue #18 extensions.
type PoolChurnerWorkloadType string

const (
	PoolChurnerWorkloadPod PoolChurnerWorkloadType = "Pod"
)

// PoolChurnerSpec defines fake workload churn: keep a steady pod count while
// periodically deleting a batch so controllers re-create them (API churn for tests).
type PoolChurnerSpec struct {
	// WorkloadType is the kind of workload to churn. Only "Pod" is supported; empty defaults to Pod.
	// +optional
	WorkloadType PoolChurnerWorkloadType `json:"workloadType,omitempty"`

	// PodCount is the steady-state number of Pods to keep.
	PodCount int32 `json:"podCount"`

	// IntervalSeconds is the period t between churn cycles (delete-then-recreate batch).
	// Set to 0 to disable churn and only maintain PodCount (behaves like a static pool).
	// +kubebuilder:validation:Minimum=0
	IntervalSeconds int64 `json:"intervalSeconds"`

	// ChurnCount is how many Pods to delete each cycle (immediately re-created on next reconcile).
	// +kubebuilder:validation:Minimum=0
	ChurnCount int32 `json:"churnCount"`

	// PodTemplate is used to create fake Pods (same pattern as PodPool).
	PodTemplate corev1.Pod `json:"podTemplate"`
}

// PoolChurnerStatus is observed churn state.
type PoolChurnerStatus struct {
	Conditions         []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty" protobuf:"varint,2,opt,name=observedGeneration"`
	// LastChurnTime is when the last churn batch was applied (deletes issued).
	// +optional
	LastChurnTime *metav1.Time `json:"lastChurnTime,omitempty"`
	// ChurnCycles is the number of completed churn batches.
	// +optional
	ChurnCycles int64 `json:"churnCycles,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PoolChurner creates a pool of KWOK-schedulable Pods and periodically churns a subset
// for load tests (e.g. network policies) without real workloads.
type PoolChurner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PoolChurnerSpec   `json:"spec,omitempty"`
	Status PoolChurnerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PoolChurnerList contains a list of PoolChurner.
type PoolChurnerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PoolChurner `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PoolChurner{}, &PoolChurnerList{})
}
