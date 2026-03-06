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

// ConnectionType defines the type of RisingWave connection.
// +kubebuilder:validation:Enum=kafka;schema_registry;iceberg
type ConnectionType string

const (
	ConnectionTypeKafka          ConnectionType = "kafka"
	ConnectionTypeSchemaRegistry ConnectionType = "schema_registry"
	ConnectionTypeIceberg        ConnectionType = "iceberg"
)

// RisingWaveConnectionSpec defines the desired state of RisingWaveConnection.
type RisingWaveConnectionSpec struct {
	// ConnectionRef specifies a direct connection to an externally managed RisingWave frontend.
	// +kubebuilder:validation:Required
	ConnectionRef ConnectionRef `json:"connectionRef"`

	// DatabaseRef specifies the database this connection belongs to.
	// +kubebuilder:validation:Required
	DatabaseRef DatabaseRef `json:"databaseRef"`

	// SchemaRef specifies the schema this connection belongs to.
	// Defaults to "public" if not specified.
	// +optional
	SchemaRef *SchemaRef `json:"schemaRef,omitempty"`

	// Connection name in RisingWave. Defaults to metadata.name if empty.
	// +optional
	Name string `json:"name,omitempty"`

	// Type of connection (kafka, schema_registry, iceberg).
	// +kubebuilder:validation:Required
	Type ConnectionType `json:"type"`

	// Properties are the connection properties as key-value pairs.
	// Values are literal strings. To reference a RisingWave secret, prefix with "SECRET ".
	// Example: "SECRET my_secret_name" will render as SECRET my_secret_name in SQL.
	// +optional
	Properties map[string]string `json:"properties,omitempty"`

	// Owner specifies the user who should own this connection.
	// +optional
	Owner string `json:"owner,omitempty"`
}

// SchemaRef specifies which schema the connection belongs to.
type SchemaRef struct {
	// Name of the schema in RisingWave.
	// +kubebuilder:validation:Required
	Name string `json:"name"`
}

// RisingWaveConnectionStatus defines the observed state of RisingWaveConnection.
type RisingWaveConnectionStatus struct {
	// Conditions represent the latest available observations.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// ObservedGeneration is the generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// ConnectionCreated indicates whether the connection has been created.
	// +optional
	ConnectionCreated bool `json:"connectionCreated,omitempty"`

	// Phase is a high-level summary of the lifecycle.
	// +optional
	Phase string `json:"phase,omitempty"`

	// Reason provides more detail about the current phase.
	// +optional
	Reason string `json:"reason,omitempty"`
}

// RisingWaveConnectionPhase constants.
const (
	RisingWaveConnectionPhasePending  = "Pending"
	RisingWaveConnectionPhaseCreating = "Creating"
	RisingWaveConnectionPhaseReady    = "Ready"
	RisingWaveConnectionPhaseUpdating = "Updating"
	RisingWaveConnectionPhaseDeleting = "Deleting"
	RisingWaveConnectionPhaseFailed   = "Failed"
	RisingWaveConnectionPhaseUnknown  = "Unknown"
)

// RisingWaveConnectionConditionType defines the condition types for RisingWaveConnection.
type RisingWaveConnectionConditionType string

const (
	// RisingWaveConnectionConditionReady indicates the connection is ready.
	RisingWaveConnectionConditionReady RisingWaveConnectionConditionType = "Ready"
	// RisingWaveConnectionConditionConnectionCreated indicates the connection has been created.
	RisingWaveConnectionConditionConnectionCreated RisingWaveConnectionConditionType = "ConnectionCreated"
	// RisingWaveConnectionConditionConnectionError indicates a connection error.
	RisingWaveConnectionConditionConnectionError RisingWaveConnectionConditionType = "ConnectionError"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=rwc,categories=all;streaming
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="Created",type=boolean,JSONPath=`.status.connectionCreated`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type RisingWaveConnection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RisingWaveConnectionSpec   `json:"spec,omitempty"`
	Status RisingWaveConnectionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RisingWaveConnectionList contains a list of RisingWaveConnection.
type RisingWaveConnectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RisingWaveConnection `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RisingWaveConnection{}, &RisingWaveConnectionList{})
}

// GetConnectionName returns the effective connection name.
func (r *RisingWaveConnection) GetConnectionName() string {
	if r.Spec.Name != "" {
		return r.Spec.Name
	}
	return r.Name
}

// GetSchemaName returns the schema name, defaulting to "public".
func (r *RisingWaveConnection) GetSchemaName() string {
	if r.Spec.SchemaRef != nil && r.Spec.SchemaRef.Name != "" {
		return r.Spec.SchemaRef.Name
	}
	return "public"
}

// GetConditions returns the conditions of RisingWaveConnection.
func (r *RisingWaveConnection) GetConditions() []metav1.Condition {
	return r.Status.Conditions
}

// SetConditions sets the conditions of RisingWaveConnection.
func (r *RisingWaveConnection) SetConditions(conditions []metav1.Condition) {
	r.Status.Conditions = conditions
}
