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

// RisingWaveUserSpec defines the desired state of RisingWaveUser.
type RisingWaveUserSpec struct {
	// ConnectionRef specifies a direct connection to an externally managed RisingWave frontend.
	// +kubebuilder:validation:Required
	ConnectionRef ConnectionRef `json:"connectionRef"`

	// User name in RisingWave. Defaults to metadata.name if empty.
	// +optional
	Name string `json:"name,omitempty"`

	// Password configuration for the user.
	// +optional
	Password *PasswordConfig `json:"password,omitempty"`

	// Authentication method configuration.
	// +optional
	Auth *AuthConfig `json:"auth,omitempty"`

	// User-level permissions (SUPERUSER, CREATEDB, CREATEUSER).
	// +optional
	Permissions []UserPermission `json:"permissions,omitempty"`

	// Privilege grants on database objects.
	// +optional
	Grants *GrantSpec `json:"grants,omitempty"`
}

// AdminCredentials holds credentials for connecting to RisingWave as an admin user.
type AdminCredentials struct {
	// Username to connect with.
	// +kubebuilder:default=root
	// +optional
	Username string `json:"username,omitempty"`

	// Password is the plaintext admin password.
	// Not recommended for production; use passwordSecretRef instead.
	// Defaults to empty string if neither password nor passwordSecretRef is set.
	// +optional
	Password string `json:"password,omitempty"`

	// PasswordSecretRef references a Kubernetes Secret containing the admin password.
	// Takes precedence over the password field when both are set.
	// +optional
	PasswordSecretRef *SecretReference `json:"passwordSecretRef,omitempty"`
}

// ConnectionRef defines a direct connection to an externally managed RisingWave frontend.
type ConnectionRef struct {
	// Host is the hostname or IP of the RisingWave frontend service.
	// +kubebuilder:validation:Required
	Host string `json:"host"`

	// Port is the PostgreSQL-compatible port of the RisingWave frontend.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default=4567
	// +optional
	Port int32 `json:"port,omitempty"`

	// Credentials for connecting to the RisingWave cluster as an admin.
	// If omitted, connects with username "root" and empty password.
	// +optional
	Credentials *AdminCredentials `json:"credentials,omitempty"`
}

// PasswordConfig defines password configuration for a user.
type PasswordConfig struct {
	// SecretRef references a secret containing the password.
	// The secret must contain a 'password' key.
	// +optional
	SecretRef *SecretReference `json:"secretRef,omitempty"`

	// Generate a random password with the specified length.
	// If not specified and no secretRef is provided, a 16-character password will be generated.
	// +kubebuilder:validation:Minimum=8
	// +kubebuilder:validation:Maximum=128
	// +optional
	GenerateRandomLength *int32 `json:"generateRandomLength,omitempty"`
}

// SecretReference references a secret in the same namespace.
type SecretReference struct {
	// Name of the secret.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace of the secret. Defaults to the same namespace as the RisingWaveUser.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Key in the secret data containing the password. Defaults to "password".
	// +optional
	Key string `json:"key,omitempty"`
}

// AuthConfig defines authentication method configuration.
type AuthConfig struct {
	// Type of authentication. Valid values are "password", "oauth", "ldap".
	// Defaults to "password" if not specified.
	// +kubebuilder:validation:Enum=password;oauth;ldap
	// +kubebuilder:default=password
	// +optional
	Type *AuthType `json:"type,omitempty"`

	// OAuth authentication configuration.
	// +optional
	OAuth *OAuthConfig `json:"oauth,omitempty"`

	// LDAP authentication configuration.
	// +optional
	LDAP *LDAPConfig `json:"ldap,omitempty"`
}

// AuthType represents the authentication type.
// +kubebuilder:validation:Enum=password;oauth;ldap
type AuthType string

const (
	// AuthTypePassword uses password authentication (default).
	AuthTypePassword AuthType = "password"
	// AuthTypeOAuth uses OAuth/JWT authentication.
	AuthTypeOAuth AuthType = "oauth"
	// AuthTypeLDAP uses LDAP authentication.
	AuthTypeLDAP AuthType = "ldap"
)

// OAuthConfig defines OAuth/JWT authentication configuration.
type OAuthConfig struct {
	// The JWKS (JSON Web Key Set) URL for verifying JWT tokens.
	// +kubebuilder:validation:Required
	JWKSUrl string `json:"jwksUrl"`

	// The issuer claim to verify in the JWT token.
	// +kubebuilder:validation:Required
	Issuer string `json:"issuer"`

	// The audience claim to verify in the JWT token.
	// +optional
	Audience []string `json:"audience,omitempty"`
}

// LDAPConfig defines LDAP authentication configuration.
type LDAPConfig struct {
	// LDAP server host.
	// +kubebuilder:validation:Required
	Host string `json:"host"`

	// LDAP server port.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default=389
	// +optional
	Port int32 `json:"port,omitempty"`

	// Base DN for LDAP searches.
	// +kubebuilder:validation:Required
	BaseDN string `json:"baseDN"`

	// Use SSL/TLS for LDAP connections.
	// +optional
	UseSSL *bool `json:"useSSL,omitempty"`

	// Skip TLS certificate verification.
	// +optional
	InsecureSkipVerify *bool `json:"insecureSkipVerify,omitempty"`
}

// UserPermission represents user-level permissions.
// Permission value validation is delegated to RisingWave during SQL execution.
type UserPermission string

// GrantSpec defines privilege grants on database objects using hierarchical structure.
// Grants are nested: databases -> schemas -> objects (tables, views, etc.)
// This allows database and schema context to be specified once at the parent level.
type GrantSpec struct {
	// Database-level privileges. Each database can contain nested schema privileges.
	// +optional
	Databases []DatabasePrivilege `json:"databases,omitempty"`
}

// DatabasePrivilege defines privileges on a database and its nested objects.
type DatabasePrivilege struct {
	// Database name.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Privileges to grant. Valid values are CONNECT, CREATE, ALL.
	// +optional
	Privileges []DatabasePrivilegeType `json:"privileges,omitempty"`

	// Grant option allows the grantee to grant these privileges to others.
	// +optional
	WithGrantOption bool `json:"withGrantOption,omitempty"`

	// Nested schema-level privileges within this database.
	// +optional
	Schemas []NestedSchemaPrivilege `json:"schemas,omitempty"`
}

// DatabasePrivilegeType represents database privilege types.
// Privilege value validation is delegated to RisingWave during SQL execution.
type DatabasePrivilegeType string

const (
	DatabasePrivilegeConnect DatabasePrivilegeType = "CONNECT"
	DatabasePrivilegeCreate  DatabasePrivilegeType = "CREATE"
	DatabasePrivilegeAll     DatabasePrivilegeType = "ALL"
)

// NestedSchemaPrivilege defines privileges on a schema within a database.
// The database name is inherited from the parent DatabasePrivilege.
type NestedSchemaPrivilege struct {
	// Schema name.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Privileges to grant. Valid values are USAGE, CREATE, ALL.
	// +optional
	Privileges []SchemaPrivilegeType `json:"privileges,omitempty"`

	// Grant option allows the grantee to grant these privileges to others.
	// +optional
	WithGrantOption bool `json:"withGrantOption,omitempty"`

	// Nested object privileges within this schema.
	// +optional
	Tables []NestedTablePrivilege `json:"tables,omitempty"`

	// +optional
	Views []NestedViewPrivilege `json:"views,omitempty"`

	// +optional
	MaterializedViews []NestedMaterializedViewPrivilege `json:"materializedViews,omitempty"`

	// +optional
	Sources []NestedSourcePrivilege `json:"sources,omitempty"`

	// +optional
	Sinks []NestedSinkPrivilege `json:"sinks,omitempty"`

	// +optional
	Connections []NestedConnectionPrivilege `json:"connections,omitempty"`

	// +optional
	Secrets []NestedSecretPrivilege `json:"secrets,omitempty"`

	// +optional
	Functions []NestedFunctionPrivilege `json:"functions,omitempty"`
}

// SchemaPrivilegeType represents schema privilege types.
// Privilege value validation is delegated to RisingWave during SQL execution.
type SchemaPrivilegeType string

const (
	SchemaPrivilegeUsage  SchemaPrivilegeType = "USAGE"
	SchemaPrivilegeCreate SchemaPrivilegeType = "CREATE"
	SchemaPrivilegeAll    SchemaPrivilegeType = "ALL"
)

// NestedTablePrivilege defines privileges on a table within a schema.
// The database and schema names are inherited from parent levels.
type NestedTablePrivilege struct {
	// Table name. Use "*" for all tables in the schema.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Privileges to grant. Valid values are SELECT, INSERT, UPDATE, DELETE, TRUNCATE, REFERENCES, TRIGGER, ALL.
	// +kubebuilder:validation:Required
	Privileges []TablePrivilegeType `json:"privileges"`

	// Grant option allows the grantee to grant these privileges to others.
	// +optional
	WithGrantOption bool `json:"withGrantOption,omitempty"`
}

// TablePrivilegeType represents table privilege types.
// Privilege value validation is delegated to RisingWave during SQL execution.
type TablePrivilegeType string

const (
	TablePrivilegeSelect     TablePrivilegeType = "SELECT"
	TablePrivilegeInsert     TablePrivilegeType = "INSERT"
	TablePrivilegeUpdate     TablePrivilegeType = "UPDATE"
	TablePrivilegeDelete     TablePrivilegeType = "DELETE"
	TablePrivilegeTruncate   TablePrivilegeType = "TRUNCATE"
	TablePrivilegeReferences TablePrivilegeType = "REFERENCES"
	TablePrivilegeTrigger    TablePrivilegeType = "TRIGGER"
	TablePrivilegeAll        TablePrivilegeType = "ALL"
)

// NestedViewPrivilege defines privileges on a view within a schema.
// The database and schema names are inherited from parent levels.
type NestedViewPrivilege struct {
	// View name. Use "*" for all views in the schema.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Privileges to grant. Valid values are SELECT, INSERT, DELETE, UPDATE, TRIGGER, ALL.
	// +kubebuilder:validation:Required
	Privileges []ViewPrivilegeType `json:"privileges"`

	// Grant option allows the grantee to grant these privileges to others.
	// +optional
	WithGrantOption bool `json:"withGrantOption,omitempty"`
}

// ViewPrivilegeType represents view privilege types.
// Privilege value validation is delegated to RisingWave during SQL execution.
type ViewPrivilegeType string

const (
	ViewPrivilegeSelect  ViewPrivilegeType = "SELECT"
	ViewPrivilegeInsert  ViewPrivilegeType = "INSERT"
	ViewPrivilegeDelete  ViewPrivilegeType = "DELETE"
	ViewPrivilegeUpdate  ViewPrivilegeType = "UPDATE"
	ViewPrivilegeTrigger ViewPrivilegeType = "TRIGGER"
	ViewPrivilegeAll     ViewPrivilegeType = "ALL"
)

// NestedMaterializedViewPrivilege defines privileges on a materialized view within a schema.
// The database and schema names are inherited from parent levels.
type NestedMaterializedViewPrivilege struct {
	// Materialized view name. Use "*" for all materialized views in the schema.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Privileges to grant. Valid values are SELECT, ALL.
	// +kubebuilder:validation:Required
	Privileges []MaterializedViewPrivilegeType `json:"privileges"`

	// Grant option allows the grantee to grant these privileges to others.
	// +optional
	WithGrantOption bool `json:"withGrantOption,omitempty"`
}

// MaterializedViewPrivilegeType represents materialized view privilege types.
// Privilege value validation is delegated to RisingWave during SQL execution.
type MaterializedViewPrivilegeType string

const (
	MaterializedViewPrivilegeSelect MaterializedViewPrivilegeType = "SELECT"
	MaterializedViewPrivilegeAll    MaterializedViewPrivilegeType = "ALL"
)

// NestedSourcePrivilege defines privileges on a source within a schema.
// The database and schema names are inherited from parent levels.
type NestedSourcePrivilege struct {
	// Source name. Use "*" for all sources in the schema.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Privileges to grant. Valid values are SELECT, ALL.
	// +kubebuilder:validation:Required
	Privileges []SourcePrivilegeType `json:"privileges"`

	// Grant option allows the grantee to grant these privileges to others.
	// +optional
	WithGrantOption bool `json:"withGrantOption,omitempty"`
}

// SourcePrivilegeType represents source privilege types.
// Privilege value validation is delegated to RisingWave during SQL execution.
type SourcePrivilegeType string

const (
	SourcePrivilegeSelect SourcePrivilegeType = "SELECT"
	SourcePrivilegeAll    SourcePrivilegeType = "ALL"
)

// NestedSinkPrivilege defines privileges on a sink within a schema.
// The database and schema names are inherited from parent levels.
type NestedSinkPrivilege struct {
	// Sink name. Use "*" for all sinks in the schema.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Privileges to grant. Valid values are SELECT, ALL.
	// +kubebuilder:validation:Required
	Privileges []SinkPrivilegeType `json:"privileges"`

	// Grant option allows the grantee to grant these privileges to others.
	// +optional
	WithGrantOption bool `json:"withGrantOption,omitempty"`
}

// SinkPrivilegeType represents sink privilege types.
// Privilege value validation is delegated to RisingWave during SQL execution.
type SinkPrivilegeType string

const (
	SinkPrivilegeSelect SinkPrivilegeType = "SELECT"
	SinkPrivilegeAll    SinkPrivilegeType = "ALL"
)

// NestedConnectionPrivilege defines privileges on a connection within a schema.
// The database and schema names are inherited from parent levels.
type NestedConnectionPrivilege struct {
	// Connection name. Use "*" for all connections in the schema.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Privileges to grant. Valid values are USAGE, ALL.
	// +kubebuilder:validation:Required
	Privileges []ConnectionPrivilegeType `json:"privileges"`

	// Grant option allows the grantee to grant these privileges to others.
	// +optional
	WithGrantOption bool `json:"withGrantOption,omitempty"`
}

// ConnectionPrivilegeType represents connection privilege types.
// Privilege value validation is delegated to RisingWave during SQL execution.
type ConnectionPrivilegeType string

const (
	ConnectionPrivilegeUsage ConnectionPrivilegeType = "USAGE"
	ConnectionPrivilegeAll   ConnectionPrivilegeType = "ALL"
)

// NestedSecretPrivilege defines privileges on a secret within a schema.
// The database and schema names are inherited from parent levels.
type NestedSecretPrivilege struct {
	// Secret name. Use "*" for all secrets in the schema.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Privileges to grant. Valid values are USAGE, ALL.
	// +kubebuilder:validation:Required
	Privileges []SecretPrivilegeType `json:"privileges"`

	// Grant option allows the grantee to grant these privileges to others.
	// +optional
	WithGrantOption bool `json:"withGrantOption,omitempty"`
}

// SecretPrivilegeType represents secret privilege types.
// Privilege value validation is delegated to RisingWave during SQL execution.
type SecretPrivilegeType string

const (
	SecretPrivilegeUsage SecretPrivilegeType = "USAGE"
	SecretPrivilegeAll   SecretPrivilegeType = "ALL"
)

// NestedFunctionPrivilege defines privileges on a function within a schema.
// The database and schema names are inherited from parent levels.
type NestedFunctionPrivilege struct {
	// Function name. Use "*" for all functions in the schema.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Privileges to grant. Valid values are EXECUTE, ALL.
	// +kubebuilder:validation:Required
	Privileges []FunctionPrivilegeType `json:"privileges"`

	// Grant option allows the grantee to grant these privileges to others.
	// +optional
	WithGrantOption bool `json:"withGrantOption,omitempty"`
}

// FunctionPrivilegeType represents function privilege types.
// Privilege value validation is delegated to RisingWave during SQL execution.
type FunctionPrivilegeType string

const (
	FunctionPrivilegeExecute FunctionPrivilegeType = "EXECUTE"
	FunctionPrivilegeAll     FunctionPrivilegeType = "ALL"
)

// RisingWaveUserStatus defines the observed state of RisingWaveUser.
type RisingWaveUserStatus struct {
	// Conditions represent the latest available observations of the RisingWaveUser's current state.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// ObservedGeneration is the generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// UserCreated indicates whether the user has been created in RisingWave.
	// +optional
	UserCreated bool `json:"userCreated,omitempty"`

	// SecretCreated indicates whether the password secret has been created.
	// +optional
	SecretCreated bool `json:"secretCreated,omitempty"`

	// SecretName is the name of the secret containing the user's password.
	// +optional
	SecretName string `json:"secretName,omitempty"`

	// PrivilegesSynced indicates whether privileges have been synced.
	// +optional
	PrivilegesSynced bool `json:"privilegesSynced,omitempty"`

	// Phase is a high-level summary of where the RisingWaveUser is in its lifecycle.
	// +optional
	Phase string `json:"phase,omitempty"`

	// Reason provides more detail about the current phase.
	// +optional
	Reason string `json:"reason,omitempty"`
}

// RisingWaveUserPhase constants.
const (
	RisingWaveUserPhasePending  = "Pending"
	RisingWaveUserPhaseCreating = "Creating"
	RisingWaveUserPhaseReady    = "Ready"
	RisingWaveUserPhaseUpdating = "Updating"
	RisingWaveUserPhaseDeleting = "Deleting"
	RisingWaveUserPhaseFailed   = "Failed"
	RisingWaveUserPhaseUnknown  = "Unknown"
)

// RisingWaveUserConditionType defines the condition types for RisingWaveUser.
type RisingWaveUserConditionType string

const (
	// RisingWaveUserConditionReady indicates the user is ready.
	RisingWaveUserConditionReady RisingWaveUserConditionType = "Ready"
	// RisingWaveUserConditionUserCreated indicates the user has been created.
	RisingWaveUserConditionUserCreated RisingWaveUserConditionType = "UserCreated"
	// RisingWaveUserConditionSecretCreated indicates the secret has been created.
	RisingWaveUserConditionSecretCreated RisingWaveUserConditionType = "SecretCreated"
	// RisingWaveUserConditionPrivilegesSynced indicates privileges have been synced.
	RisingWaveUserConditionPrivilegesSynced RisingWaveUserConditionType = "PrivilegesSynced"
	// RisingWaveUserConditionConnectionError indicates a connection error.
	RisingWaveUserConditionConnectionError RisingWaveUserConditionType = "ConnectionError"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=rwu,categories=all;streaming
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="User",type=boolean,JSONPath=`.status.userCreated`
// +kubebuilder:printcolumn:name="Secret",type=boolean,JSONPath=`.status.secretCreated`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type RisingWaveUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RisingWaveUserSpec   `json:"spec,omitempty"`
	Status RisingWaveUserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RisingWaveUserList contains a list of RisingWaveUser.
type RisingWaveUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RisingWaveUser `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RisingWaveUser{}, &RisingWaveUserList{})
}

// GetConditions returns the conditions of RisingWaveUser.
func (r *RisingWaveUser) GetConditions() []metav1.Condition {
	return r.Status.Conditions
}

// SetConditions sets the conditions of RisingWaveUser.
func (r *RisingWaveUser) SetConditions(conditions []metav1.Condition) {
	r.Status.Conditions = conditions
}
