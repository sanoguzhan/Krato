/*
Copyright 2026.

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

// ResourceAttributionSpec defines the desired state of ResourceAttribution
type ResourceAttributionSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// Selector matches pods to attribute
	// +required
	Selector *metav1.LabelSelector `json:"selector,omitempty"`

	// Namespace to scope the attribution
	// +optional
	Namespace *string `json:"namespace,omitempty"`

	// Interval at which to update the attribution status (e.g., "30s", "1m")
	// defaults to "30s" if not specified
	// +optional
	UpdateInterval *string `json:"updateInterval,omitempty"`

}

// ResourceAttributionStatus defines the observed state of ResourceAttribution.
type ResourceAttributionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the ResourceAttribution resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Resource usage metrics observed for the selected pods.
	// +optional
	MemoryUsage *string `json:"memoryUsage,omitempty"`

	// CPU usage observed for the selected pods.
	// +optional
	CPUUsage *string `json:"cpuUsage,omitempty"`

	// Number of pods matching the selector.
	// +optional
	PodCount *int32 `json:"podCount,omitempty"`

	// Last time the status was updated.
	// +optional
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`

	// MaxConcurrentReconciles indicates the maximum number of concurrent reconciles allowed for this resource.
	// +optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	MaxConcurrentReconciles *int32 `json:"maxConcurrentReconciles,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ResourceAttribution is the Schema for the resourceattributions API
type ResourceAttribution struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of ResourceAttribution
	// +required
	Spec ResourceAttributionSpec `json:"spec"`

	// status defines the observed state of ResourceAttribution
	// +optional
	Status ResourceAttributionStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// ResourceAttributionList contains a list of ResourceAttribution
type ResourceAttributionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []ResourceAttribution `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ResourceAttribution{}, &ResourceAttributionList{})
}
