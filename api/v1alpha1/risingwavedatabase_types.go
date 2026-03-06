/*
Copyright 2025 RisingWave Labs.

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

// RisingWaveDatabaseSpec defines the desired state of RisingWaveDatabase.
type RisingWaveDatabaseSpec struct {
	// ConnectionRef specifies a direct connection to an externally managed RisingWave frontend.
	// +kubebuilder:validation:Required
	ConnectionRef ConnectionRef `json:"connectionRef"`

	// Database name in RisingWave. Defaults to metadata.name if empty.
	// +optional
	Name string `json:"name,omitempty"`

	// Owner specifies the user who should own this database.
	// If not specified, the database owner will be the admin user creating it.
	// +optional
	Owner string `json:"owner,omitempty"`
}

// RisingWaveDatabaseStatus defines the observed state of RisingWaveDatabase.
type RisingWaveDatabaseStatus struct {
	// Conditions represent the latest available observations.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// ObservedGeneration is the generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// DatabaseCreated indicates whether the database has been created.
	// +optional
	DatabaseCreated bool `json:"databaseCreated,omitempty"`

	// Phase is a high-level summary of the lifecycle.
	// +optional
	Phase string `json:"phase,omitempty"`

	// Reason provides more detail about the current phase.
	// +optional
	Reason string `json:"reason,omitempty"`
}

// RisingWaveDatabasePhase constants.
const (
	RisingWaveDatabasePhasePending  = "Pending"
	RisingWaveDatabasePhaseCreating = "Creating"
	RisingWaveDatabasePhaseReady    = "Ready"
	RisingWaveDatabasePhaseUpdating = "Updating"
	RisingWaveDatabasePhaseDeleting = "Deleting"
	RisingWaveDatabasePhaseFailed   = "Failed"
	RisingWaveDatabasePhaseUnknown  = "Unknown"
)

// RisingWaveDatabaseConditionType defines the condition types for RisingWaveDatabase.
type RisingWaveDatabaseConditionType string

const (
	// RisingWaveDatabaseConditionReady indicates the database is ready.
	RisingWaveDatabaseConditionReady RisingWaveDatabaseConditionType = "Ready"
	// RisingWaveDatabaseConditionDatabaseCreated indicates the database has been created.
	RisingWaveDatabaseConditionDatabaseCreated RisingWaveDatabaseConditionType = "DatabaseCreated"
	// RisingWaveDatabaseConditionConnectionError indicates a connection error.
	RisingWaveDatabaseConditionConnectionError RisingWaveDatabaseConditionType = "ConnectionError"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=rwd,categories=all;streaming
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Database",type=boolean,JSONPath=`.status.databaseCreated`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type RisingWaveDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RisingWaveDatabaseSpec   `json:"spec,omitempty"`
	Status RisingWaveDatabaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RisingWaveDatabaseList contains a list of RisingWaveDatabase.
type RisingWaveDatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RisingWaveDatabase `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RisingWaveDatabase{}, &RisingWaveDatabaseList{})
}

// GetDatabaseName returns the effective database name.
func (r *RisingWaveDatabase) GetDatabaseName() string {
	if r.Spec.Name != "" {
		return r.Spec.Name
	}
	return r.Name
}

// GetConditions returns the conditions of RisingWaveDatabase.
func (r *RisingWaveDatabase) GetConditions() []metav1.Condition {
	return r.Status.Conditions
}

// SetConditions sets the conditions of RisingWaveDatabase.
func (r *RisingWaveDatabase) SetConditions(conditions []metav1.Condition) {
	r.Status.Conditions = conditions
}
