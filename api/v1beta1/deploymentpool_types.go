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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DeploymentPoolSpec defines the desired state of DeploymentPool
type DeploymentPoolSpec struct {
	DeploymentTemplate appsv1.Deployment `json:"deploymentTemplate"`
}

// DeepCopyObject implements client.Object.
func (in DeploymentPoolSpec) DeepCopyObject() runtime.Object {
	panic("unimplemented")
}

// GetAnnotations implements client.Object.
func (in DeploymentPoolSpec) GetAnnotations() map[string]string {
	panic("unimplemented")
}

// GetCreationTimestamp implements client.Object.
func (in DeploymentPoolSpec) GetCreationTimestamp() metav1.Time {
	panic("unimplemented")
}

// GetDeletionGracePeriodSeconds implements client.Object.
func (in DeploymentPoolSpec) GetDeletionGracePeriodSeconds() *int64 {
	panic("unimplemented")
}

// GetDeletionTimestamp implements client.Object.
func (in DeploymentPoolSpec) GetDeletionTimestamp() *metav1.Time {
	panic("unimplemented")
}

// GetFinalizers implements client.Object.
func (in DeploymentPoolSpec) GetFinalizers() []string {
	panic("unimplemented")
}

// GetGenerateName implements client.Object.
func (in DeploymentPoolSpec) GetGenerateName() string {
	panic("unimplemented")
}

// GetGeneration implements client.Object.
func (in DeploymentPoolSpec) GetGeneration() int64 {
	panic("unimplemented")
}

// GetLabels implements client.Object.
func (in DeploymentPoolSpec) GetLabels() map[string]string {
	panic("unimplemented")
}

// GetManagedFields implements client.Object.
func (in DeploymentPoolSpec) GetManagedFields() []metav1.ManagedFieldsEntry {
	panic("unimplemented")
}

// GetName implements client.Object.
func (in DeploymentPoolSpec) GetName() string {
	panic("unimplemented")
}

// GetNamespace implements client.Object.
func (in DeploymentPoolSpec) GetNamespace() string {
	panic("unimplemented")
}

// GetObjectKind implements client.Object.
func (in DeploymentPoolSpec) GetObjectKind() schema.ObjectKind {
	panic("unimplemented")
}

// GetOwnerReferences implements client.Object.
func (in DeploymentPoolSpec) GetOwnerReferences() []metav1.OwnerReference {
	panic("unimplemented")
}

// GetResourceVersion implements client.Object.
func (in DeploymentPoolSpec) GetResourceVersion() string {
	panic("unimplemented")
}

// GetSelfLink implements client.Object.
func (in DeploymentPoolSpec) GetSelfLink() string {
	panic("unimplemented")
}

// GetUID implements client.Object.
func (in DeploymentPoolSpec) GetUID() types.UID {
	panic("unimplemented")
}

// SetAnnotations implements client.Object.
func (in DeploymentPoolSpec) SetAnnotations(annotations map[string]string) {
	panic("unimplemented")
}

// SetCreationTimestamp implements client.Object.
func (in DeploymentPoolSpec) SetCreationTimestamp(timestamp metav1.Time) {
	panic("unimplemented")
}

// SetDeletionGracePeriodSeconds implements client.Object.
func (in DeploymentPoolSpec) SetDeletionGracePeriodSeconds(*int64) {
	panic("unimplemented")
}

// SetDeletionTimestamp implements client.Object.
func (in DeploymentPoolSpec) SetDeletionTimestamp(timestamp *metav1.Time) {
	panic("unimplemented")
}

// SetFinalizers implements client.Object.
func (in DeploymentPoolSpec) SetFinalizers(finalizers []string) {
	panic("unimplemented")
}

// SetGenerateName implements client.Object.
func (in DeploymentPoolSpec) SetGenerateName(name string) {
	panic("unimplemented")
}

// SetGeneration implements client.Object.
func (in DeploymentPoolSpec) SetGeneration(generation int64) {
	panic("unimplemented")
}

// SetLabels implements client.Object.
func (in DeploymentPoolSpec) SetLabels(labels map[string]string) {
	panic("unimplemented")
}

// SetManagedFields implements client.Object.
func (in DeploymentPoolSpec) SetManagedFields(managedFields []metav1.ManagedFieldsEntry) {
	panic("unimplemented")
}

// SetName implements client.Object.
func (in DeploymentPoolSpec) SetName(name string) {
	panic("unimplemented")
}

// SetNamespace implements client.Object.
func (in DeploymentPoolSpec) SetNamespace(namespace string) {
	panic("unimplemented")
}

// SetOwnerReferences implements client.Object.
func (in DeploymentPoolSpec) SetOwnerReferences([]metav1.OwnerReference) {
	panic("unimplemented")
}

// SetResourceVersion implements client.Object.
func (in DeploymentPoolSpec) SetResourceVersion(version string) {
	panic("unimplemented")
}

// SetSelfLink implements client.Object.
func (in DeploymentPoolSpec) SetSelfLink(selfLink string) {
	panic("unimplemented")
}

// SetUID implements client.Object.
func (in DeploymentPoolSpec) SetUID(uid types.UID) {
	panic("unimplemented")
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
