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

// RisingWaveSchemaSpec defines the desired state of RisingWaveSchema.
type RisingWaveSchemaSpec struct {
	// ConnectionRef specifies a direct connection to an externally managed RisingWave frontend.
	// +kubebuilder:validation:Required
	ConnectionRef ConnectionRef `json:"connectionRef"`

	// DatabaseRef specifies the database this schema belongs to.
	// +kubebuilder:validation:Required
	DatabaseRef DatabaseRef `json:"databaseRef"`

	// Schema name in RisingWave. Defaults to metadata.name if empty.
	// +optional
	Name string `json:"name,omitempty"`

	// Owner specifies the user who should own this schema.
	// If not specified, the schema owner will be the database owner.
	// +optional
	Owner string `json:"owner,omitempty"`
}

// DatabaseRef specifies which database the schema belongs to.
type DatabaseRef struct {
	// Name of the database in RisingWave.
	// +kubebuilder:validation:Required
	Name string `json:"name"`
}

// RisingWaveSchemaStatus defines the observed state of RisingWaveSchema.
type RisingWaveSchemaStatus struct {
	// Conditions represent the latest available observations.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// ObservedGeneration is the generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// SchemaCreated indicates whether the schema has been created.
	// +optional
	SchemaCreated bool `json:"schemaCreated,omitempty"`

	// Phase is a high-level summary of the lifecycle.
	// +optional
	Phase string `json:"phase,omitempty"`

	// Reason provides more detail about the current phase.
	// +optional
	Reason string `json:"reason,omitempty"`
}

// RisingWaveSchemaPhase constants.
const (
	RisingWaveSchemaPhasePending  = "Pending"
	RisingWaveSchemaPhaseCreating = "Creating"
	RisingWaveSchemaPhaseReady    = "Ready"
	RisingWaveSchemaPhaseUpdating = "Updating"
	RisingWaveSchemaPhaseDeleting = "Deleting"
	RisingWaveSchemaPhaseFailed   = "Failed"
	RisingWaveSchemaPhaseUnknown  = "Unknown"
)

// RisingWaveSchemaConditionType defines the condition types for RisingWaveSchema.
type RisingWaveSchemaConditionType string

const (
	// RisingWaveSchemaConditionReady indicates the schema is ready.
	RisingWaveSchemaConditionReady RisingWaveSchemaConditionType = "Ready"
	// RisingWaveSchemaConditionSchemaCreated indicates the schema has been created.
	RisingWaveSchemaConditionSchemaCreated RisingWaveSchemaConditionType = "SchemaCreated"
	// RisingWaveSchemaConditionConnectionError indicates a connection error.
	RisingWaveSchemaConditionConnectionError RisingWaveSchemaConditionType = "ConnectionError"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=rws,categories=all;streaming
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Schema",type=boolean,JSONPath=`.status.schemaCreated`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type RisingWaveSchema struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RisingWaveSchemaSpec   `json:"spec,omitempty"`
	Status RisingWaveSchemaStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RisingWaveSchemaList contains a list of RisingWaveSchema.
type RisingWaveSchemaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RisingWaveSchema `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RisingWaveSchema{}, &RisingWaveSchemaList{})
}

// GetSchemaName returns the effective schema name.
func (r *RisingWaveSchema) GetSchemaName() string {
	if r.Spec.Name != "" {
		return r.Spec.Name
	}
	return r.Name
}

// GetConditions returns the conditions of RisingWaveSchema.
func (r *RisingWaveSchema) GetConditions() []metav1.Condition {
	return r.Status.Conditions
}

// SetConditions sets the conditions of RisingWaveSchema.
func (r *RisingWaveSchema) SetConditions(conditions []metav1.Condition) {
	r.Status.Conditions = conditions
}
