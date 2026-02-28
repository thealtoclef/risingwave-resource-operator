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

package rwclient

import (
	"fmt"
	"strings"

	v1alpha1 "github.com/risingwavelabs/risingwave-resource-operator/api/v1alpha1"
)

// QuoteIdentifier quotes an SQL identifier (table name, column name, etc.).
func QuoteIdentifier(name string) string {
	if name == "*" {
		return name
	}
	return fmt.Sprintf("\"%s\"", strings.ReplaceAll(name, "\"", "\"\""))
}

// QuoteUser quotes a user name for SQL.
func QuoteUser(name string) string {
	return fmt.Sprintf("\"%s\"", strings.ReplaceAll(name, "\"", "\"\""))
}

// BuildCreateUserSQL builds a CREATE USER statement.
func BuildCreateUserSQL(user *v1alpha1.RisingWaveUser, password string) string {
	var sb strings.Builder

	sb.WriteString("CREATE USER ")
	sb.WriteString(QuoteUser(getUserName(user)))

	if password != "" {
		sb.WriteString(" WITH PASSWORD '")
		sb.WriteString(escapeStringLiteral(password))
		sb.WriteString("'")
	}

	// Add user permissions
	for _, perm := range user.Spec.Permissions {
		sb.WriteString(" ")
		sb.WriteString(string(perm))
	}

	return sb.String()
}

// BuildAlterUserPasswordSQL builds an ALTER USER ... WITH PASSWORD statement.
func BuildAlterUserPasswordSQL(userName string, password string) string {
	return fmt.Sprintf("ALTER USER %s WITH PASSWORD '%s'",
		QuoteUser(userName),
		escapeStringLiteral(password))
}

// BuildDropUserSQL builds a DROP USER statement.
func BuildDropUserSQL(userName string) string {
	return fmt.Sprintf("DROP USER IF EXISTS %s", QuoteUser(userName))
}

// BuildCreateUserWithOAuthSQL builds a CREATE USER statement with OAuth authentication.
// Note: OAuth/JWT authentication is configured at the cluster level in RisingWave.
// The user is created with a placeholder password that won't be used when OAuth is enabled.
func BuildCreateUserWithOAuthSQL(userName string, _ *v1alpha1.OAuthConfig) string {
	var sb strings.Builder

	sb.WriteString("CREATE USER ")
	sb.WriteString(QuoteUser(userName))
	sb.WriteString(" WITH PASSWORD 'oauth-placeholder-change-in-cluster-config'")

	return sb.String()
}

// BuildCreateUserWithLDAPSQL builds a CREATE USER statement with LDAP authentication.
// Note: LDAP authentication is configured at the cluster level in RisingWave.
// The user is created with a placeholder password that won't be used when LDAP is enabled.
func BuildCreateUserWithLDAPSQL(userName string, _ *v1alpha1.LDAPConfig) string {
	var sb strings.Builder

	sb.WriteString("CREATE USER ")
	sb.WriteString(QuoteUser(userName))
	sb.WriteString(" WITH PASSWORD 'ldap-placeholder-change-in-cluster-config'")

	return sb.String()
}

// BuildGrantStatements builds GRANT statements for all grants in spec.
func BuildGrantStatements(userName string, spec *v1alpha1.RisingWaveUserSpec) []string {
	var statements []string

	if spec.Grants == nil {
		return statements
	}

	// Process hierarchical structure (DatabasePrivilege with nested Schemas)
	for _, dbPriv := range spec.Grants.Databases {
		statements = append(statements, buildDatabasePrivilegesHierarchical(userName, &dbPriv)...)
	}

	return statements
}

// buildDatabasePrivilegesHierarchical recursively builds GRANT statements for hierarchical structure.
func buildDatabasePrivilegesHierarchical(userName string, dbPriv *v1alpha1.DatabasePrivilege) []string {
	var statements []string

	// Database-level privileges
	if len(dbPriv.Privileges) > 0 {
		stmt := buildGrantDatabasePrivilege(userName, dbPriv)
		statements = append(statements, stmt)
	}

	// Nested schema-level privileges
	for _, schemaPriv := range dbPriv.Schemas {
		statements = append(statements, buildSchemaPrivilegesHierarchical(userName, dbPriv.Name, &schemaPriv)...)
	}

	return statements
}

// buildSchemaPrivilegesHierarchical recursively builds GRANT statements for nested schema objects.
func buildSchemaPrivilegesHierarchical(userName string, database string, schemaPriv *v1alpha1.NestedSchemaPrivilege) []string {
	var statements []string

	// Schema-level privileges
	if len(schemaPriv.Privileges) > 0 {
		stmt := buildGrantNestedSchemaPrivilege(userName, schemaPriv)
		statements = append(statements, stmt)
	}

	// Nested table privileges
	for _, tablePriv := range schemaPriv.Tables {
		statements = append(statements, buildGrantNestedTablePrivilege(userName, database, schemaPriv.Name, &tablePriv))
	}

	// Nested view privileges
	for _, viewPriv := range schemaPriv.Views {
		statements = append(statements, buildGrantNestedViewPrivilege(userName, database, schemaPriv.Name, &viewPriv))
	}

	// Nested materialized view privileges
	for _, mvPriv := range schemaPriv.MaterializedViews {
		statements = append(statements, buildGrantNestedMaterializedViewPrivilege(userName, database, schemaPriv.Name, &mvPriv))
	}

	// Nested source privileges
	for _, sourcePriv := range schemaPriv.Sources {
		statements = append(statements, buildGrantNestedSourcePrivilege(userName, database, schemaPriv.Name, &sourcePriv))
	}

	// Nested sink privileges
	for _, sinkPriv := range schemaPriv.Sinks {
		statements = append(statements, buildGrantNestedSinkPrivilege(userName, database, schemaPriv.Name, &sinkPriv))
	}

	// Nested connection privileges
	for _, connPriv := range schemaPriv.Connections {
		statements = append(statements, buildGrantNestedConnectionPrivilege(userName, database, schemaPriv.Name, &connPriv))
	}

	// Nested secret privileges
	for _, secretPriv := range schemaPriv.Secrets {
		statements = append(statements, buildGrantNestedSecretPrivilege(userName, database, schemaPriv.Name, &secretPriv))
	}

	// Nested function privileges
	for _, funcPriv := range schemaPriv.Functions {
		statements = append(statements, buildGrantNestedFunctionPrivilege(userName, database, schemaPriv.Name, &funcPriv))
	}

	return statements
}

func buildGrantDatabasePrivilege(userName string, priv *v1alpha1.DatabasePrivilege) string {
	return fmt.Sprintf("GRANT %s ON DATABASE %s TO %s%s",
		normalizePrivileges(priv.Privileges),
		QuoteIdentifier(priv.Name),
		QuoteUser(userName),
		buildGrantOption(priv.WithGrantOption))
}

func buildRevokeDatabasePrivilege(userName string, priv *v1alpha1.DatabasePrivilege) string {
	return fmt.Sprintf("REVOKE %s ON DATABASE %s FROM %s",
		normalizePrivileges(priv.Privileges),
		QuoteIdentifier(priv.Name),
		QuoteUser(userName))
}

func buildGrantNestedSchemaPrivilege(userName string, priv *v1alpha1.NestedSchemaPrivilege) string {
	return fmt.Sprintf("GRANT %s ON SCHEMA %s TO %s%s",
		normalizePrivileges(priv.Privileges),
		QuoteIdentifier(priv.Name),
		QuoteUser(userName),
		buildGrantOption(priv.WithGrantOption))
}

func buildRevokeNestedSchemaPrivilege(userName string, priv *v1alpha1.NestedSchemaPrivilege) string {
	return fmt.Sprintf("REVOKE %s ON SCHEMA %s FROM %s",
		normalizePrivileges(priv.Privileges),
		QuoteIdentifier(priv.Name),
		QuoteUser(userName))
}

func buildRevokeNestedTablePrivilege(userName string, database, schema string, priv *v1alpha1.NestedTablePrivilege) string {
	if priv.Name == "*" {
		return fmt.Sprintf("REVOKE %s ON ALL TABLES IN SCHEMA %s FROM %s",
			normalizePrivileges(priv.Privileges),
			QuoteIdentifier(schema),
			QuoteUser(userName))
	}
	return fmt.Sprintf("REVOKE %s ON TABLE %s.%s.%s FROM %s",
		normalizePrivileges(priv.Privileges),
		QuoteIdentifier(database),
		QuoteIdentifier(schema),
		QuoteIdentifier(priv.Name),
		QuoteUser(userName))
}

func buildRevokeNestedViewPrivilege(userName string, database, schema string, priv *v1alpha1.NestedViewPrivilege) string {
	if priv.Name == "*" {
		return fmt.Sprintf("REVOKE %s ON ALL VIEWS IN SCHEMA %s FROM %s",
			normalizePrivileges(priv.Privileges),
			QuoteIdentifier(schema),
			QuoteUser(userName))
	}
	return fmt.Sprintf("REVOKE %s ON VIEW %s.%s.%s FROM %s",
		normalizePrivileges(priv.Privileges),
		QuoteIdentifier(database),
		QuoteIdentifier(schema),
		QuoteIdentifier(priv.Name),
		QuoteUser(userName))
}

func buildRevokeNestedMaterializedViewPrivilege(userName string, database, schema string, priv *v1alpha1.NestedMaterializedViewPrivilege) string {
	if priv.Name == "*" {
		return fmt.Sprintf("REVOKE %s ON ALL MATERIALIZED VIEWS IN SCHEMA %s FROM %s",
			normalizePrivileges(priv.Privileges),
			QuoteIdentifier(schema),
			QuoteUser(userName))
	}
	return fmt.Sprintf("REVOKE %s ON MATERIALIZED VIEW %s.%s.%s FROM %s",
		normalizePrivileges(priv.Privileges),
		QuoteIdentifier(database),
		QuoteIdentifier(schema),
		QuoteIdentifier(priv.Name),
		QuoteUser(userName))
}

func buildRevokeNestedSourcePrivilege(userName string, database, schema string, priv *v1alpha1.NestedSourcePrivilege) string {
	if priv.Name == "*" {
		return fmt.Sprintf("REVOKE %s ON ALL SOURCES IN SCHEMA %s FROM %s",
			normalizePrivileges(priv.Privileges),
			QuoteIdentifier(schema),
			QuoteUser(userName))
	}
	return fmt.Sprintf("REVOKE %s ON SOURCE %s.%s.%s FROM %s",
		normalizePrivileges(priv.Privileges),
		QuoteIdentifier(database),
		QuoteIdentifier(schema),
		QuoteIdentifier(priv.Name),
		QuoteUser(userName))
}

func buildRevokeNestedSinkPrivilege(userName string, database, schema string, priv *v1alpha1.NestedSinkPrivilege) string {
	if priv.Name == "*" {
		return fmt.Sprintf("REVOKE %s ON ALL SINKS IN SCHEMA %s FROM %s",
			normalizePrivileges(priv.Privileges),
			QuoteIdentifier(schema),
			QuoteUser(userName))
	}
	return fmt.Sprintf("REVOKE %s ON SINK %s.%s.%s FROM %s",
		normalizePrivileges(priv.Privileges),
		QuoteIdentifier(database),
		QuoteIdentifier(schema),
		QuoteIdentifier(priv.Name),
		QuoteUser(userName))
}

func buildRevokeNestedConnectionPrivilege(userName string, database, schema string, priv *v1alpha1.NestedConnectionPrivilege) string {
	if priv.Name == "*" {
		return fmt.Sprintf("REVOKE %s ON ALL CONNECTIONS IN SCHEMA %s FROM %s",
			normalizePrivileges(priv.Privileges),
			QuoteIdentifier(schema),
			QuoteUser(userName))
	}
	return fmt.Sprintf("REVOKE %s ON CONNECTION %s.%s.%s FROM %s",
		normalizePrivileges(priv.Privileges),
		QuoteIdentifier(database),
		QuoteIdentifier(schema),
		QuoteIdentifier(priv.Name),
		QuoteUser(userName))
}

func buildRevokeNestedSecretPrivilege(userName string, database, schema string, priv *v1alpha1.NestedSecretPrivilege) string {
	if priv.Name == "*" {
		return fmt.Sprintf("REVOKE %s ON ALL SECRETS IN SCHEMA %s FROM %s",
			normalizePrivileges(priv.Privileges),
			QuoteIdentifier(schema),
			QuoteUser(userName))
	}
	return fmt.Sprintf("REVOKE %s ON SECRET %s.%s.%s FROM %s",
		normalizePrivileges(priv.Privileges),
		QuoteIdentifier(database),
		QuoteIdentifier(schema),
		QuoteIdentifier(priv.Name),
		QuoteUser(userName))
}

func buildRevokeNestedFunctionPrivilege(userName string, database, schema string, priv *v1alpha1.NestedFunctionPrivilege) string {
	if priv.Name == "*" {
		return fmt.Sprintf("REVOKE %s ON ALL FUNCTIONS IN SCHEMA %s FROM %s",
			normalizePrivileges(priv.Privileges),
			QuoteIdentifier(schema),
			QuoteUser(userName))
	}
	return fmt.Sprintf("REVOKE %s ON FUNCTION %s.%s(%s) FROM %s",
		normalizePrivileges(priv.Privileges),
		QuoteIdentifier(database),
		QuoteIdentifier(schema),
		QuoteIdentifier(priv.Name),
		QuoteUser(userName))
}

func buildGrantNestedTablePrivilege(userName string, database, schema string, priv *v1alpha1.NestedTablePrivilege) string {
	if priv.Name == "*" {
		return fmt.Sprintf("GRANT %s ON ALL TABLES IN SCHEMA %s TO %s%s",
			normalizePrivileges(priv.Privileges),
			QuoteIdentifier(schema),
			QuoteUser(userName),
			buildGrantOption(priv.WithGrantOption))
	}
	return fmt.Sprintf("GRANT %s ON TABLE %s.%s.%s TO %s%s",
		normalizePrivileges(priv.Privileges),
		QuoteIdentifier(database),
		QuoteIdentifier(schema),
		QuoteIdentifier(priv.Name),
		QuoteUser(userName),
		buildGrantOption(priv.WithGrantOption))
}

func buildGrantNestedViewPrivilege(userName string, database, schema string, priv *v1alpha1.NestedViewPrivilege) string {
	if priv.Name == "*" {
		return fmt.Sprintf("GRANT %s ON ALL VIEWS IN SCHEMA %s TO %s%s",
			normalizePrivileges(priv.Privileges),
			QuoteIdentifier(schema),
			QuoteUser(userName),
			buildGrantOption(priv.WithGrantOption))
	}
	return fmt.Sprintf("GRANT %s ON VIEW %s.%s.%s TO %s%s",
		normalizePrivileges(priv.Privileges),
		QuoteIdentifier(database),
		QuoteIdentifier(schema),
		QuoteIdentifier(priv.Name),
		QuoteUser(userName),
		buildGrantOption(priv.WithGrantOption))
}

func buildGrantNestedMaterializedViewPrivilege(userName string, database, schema string, priv *v1alpha1.NestedMaterializedViewPrivilege) string {
	if priv.Name == "*" {
		return fmt.Sprintf("GRANT %s ON ALL MATERIALIZED VIEWS IN SCHEMA %s TO %s%s",
			normalizePrivileges(priv.Privileges),
			QuoteIdentifier(schema),
			QuoteUser(userName),
			buildGrantOption(priv.WithGrantOption))
	}
	return fmt.Sprintf("GRANT %s ON MATERIALIZED VIEW %s.%s.%s TO %s%s",
		normalizePrivileges(priv.Privileges),
		QuoteIdentifier(database),
		QuoteIdentifier(schema),
		QuoteIdentifier(priv.Name),
		QuoteUser(userName),
		buildGrantOption(priv.WithGrantOption))
}

func buildGrantNestedSourcePrivilege(userName string, database, schema string, priv *v1alpha1.NestedSourcePrivilege) string {
	if priv.Name == "*" {
		return fmt.Sprintf("GRANT %s ON ALL SOURCES IN SCHEMA %s TO %s%s",
			normalizePrivileges(priv.Privileges),
			QuoteIdentifier(schema),
			QuoteUser(userName),
			buildGrantOption(priv.WithGrantOption))
	}
	return fmt.Sprintf("GRANT %s ON SOURCE %s.%s.%s TO %s%s",
		normalizePrivileges(priv.Privileges),
		QuoteIdentifier(database),
		QuoteIdentifier(schema),
		QuoteIdentifier(priv.Name),
		QuoteUser(userName),
		buildGrantOption(priv.WithGrantOption))
}

func buildGrantNestedSinkPrivilege(userName string, database, schema string, priv *v1alpha1.NestedSinkPrivilege) string {
	if priv.Name == "*" {
		return fmt.Sprintf("GRANT %s ON ALL SINKS IN SCHEMA %s TO %s%s",
			normalizePrivileges(priv.Privileges),
			QuoteIdentifier(schema),
			QuoteUser(userName),
			buildGrantOption(priv.WithGrantOption))
	}
	return fmt.Sprintf("GRANT %s ON SINK %s.%s.%s TO %s%s",
		normalizePrivileges(priv.Privileges),
		QuoteIdentifier(database),
		QuoteIdentifier(schema),
		QuoteIdentifier(priv.Name),
		QuoteUser(userName),
		buildGrantOption(priv.WithGrantOption))
}

func buildGrantNestedConnectionPrivilege(userName string, database, schema string, priv *v1alpha1.NestedConnectionPrivilege) string {
	if priv.Name == "*" {
		return fmt.Sprintf("GRANT %s ON ALL CONNECTIONS IN SCHEMA %s TO %s%s",
			normalizePrivileges(priv.Privileges),
			QuoteIdentifier(schema),
			QuoteUser(userName),
			buildGrantOption(priv.WithGrantOption))
	}
	return fmt.Sprintf("GRANT %s ON CONNECTION %s.%s.%s TO %s%s",
		normalizePrivileges(priv.Privileges),
		QuoteIdentifier(database),
		QuoteIdentifier(schema),
		QuoteIdentifier(priv.Name),
		QuoteUser(userName),
		buildGrantOption(priv.WithGrantOption))
}

func buildGrantNestedSecretPrivilege(userName string, database, schema string, priv *v1alpha1.NestedSecretPrivilege) string {
	if priv.Name == "*" {
		return fmt.Sprintf("GRANT %s ON ALL SECRETS IN SCHEMA %s TO %s%s",
			normalizePrivileges(priv.Privileges),
			QuoteIdentifier(schema),
			QuoteUser(userName),
			buildGrantOption(priv.WithGrantOption))
	}
	return fmt.Sprintf("GRANT %s ON SECRET %s.%s.%s TO %s%s",
		normalizePrivileges(priv.Privileges),
		QuoteIdentifier(database),
		QuoteIdentifier(schema),
		QuoteIdentifier(priv.Name),
		QuoteUser(userName),
		buildGrantOption(priv.WithGrantOption))
}

func buildGrantNestedFunctionPrivilege(userName string, database, schema string, priv *v1alpha1.NestedFunctionPrivilege) string {
	if priv.Name == "*" {
		return fmt.Sprintf("GRANT %s ON ALL FUNCTIONS IN SCHEMA %s TO %s%s",
			normalizePrivileges(priv.Privileges),
			QuoteIdentifier(schema),
			QuoteUser(userName),
			buildGrantOption(priv.WithGrantOption))
	}
	return fmt.Sprintf("GRANT %s ON FUNCTION %s.%s.%s TO %s%s",
		normalizePrivileges(priv.Privileges),
		QuoteIdentifier(database),
		QuoteIdentifier(schema),
		QuoteIdentifier(priv.Name),
		QuoteUser(userName),
		buildGrantOption(priv.WithGrantOption))
}

// BuildRevokeStatements builds REVOKE statements for all grants.
func BuildRevokeStatements(userName string, spec *v1alpha1.RisingWaveUserSpec) []string {
	var statements []string

	if spec.Grants == nil {
		return statements
	}

	// Process hierarchical structure
	for _, dbPriv := range spec.Grants.Databases {
		// Database level - only if there are database privileges
		if len(dbPriv.Privileges) > 0 {
			stmt := buildRevokeDatabasePrivilege(userName, &dbPriv)
			statements = append(statements, stmt)
		}

		// Nested schemas
		for _, schemaPriv := range dbPriv.Schemas {
			// Schema level - only if there are schema privileges
			if len(schemaPriv.Privileges) > 0 {
				stmt := buildRevokeNestedSchemaPrivilege(userName, &schemaPriv)
				statements = append(statements, stmt)
			}

			// Nested objects (all object types)
			if len(schemaPriv.Tables) > 0 {
				for _, objPriv := range schemaPriv.Tables {
					stmt := buildRevokeNestedTablePrivilege(userName, dbPriv.Name, schemaPriv.Name, &objPriv)
					statements = append(statements, stmt)
				}
			}
			if len(schemaPriv.Views) > 0 {
				for _, objPriv := range schemaPriv.Views {
					stmt := buildRevokeNestedViewPrivilege(userName, dbPriv.Name, schemaPriv.Name, &objPriv)
					statements = append(statements, stmt)
				}
			}
			if len(schemaPriv.MaterializedViews) > 0 {
				for _, objPriv := range schemaPriv.MaterializedViews {
					stmt := buildRevokeNestedMaterializedViewPrivilege(userName, dbPriv.Name, schemaPriv.Name, &objPriv)
					statements = append(statements, stmt)
				}
			}
			if len(schemaPriv.Sources) > 0 {
				for _, objPriv := range schemaPriv.Sources {
					stmt := buildRevokeNestedSourcePrivilege(userName, dbPriv.Name, schemaPriv.Name, &objPriv)
					statements = append(statements, stmt)
				}
			}
			if len(schemaPriv.Sinks) > 0 {
				for _, objPriv := range schemaPriv.Sinks {
					stmt := buildRevokeNestedSinkPrivilege(userName, dbPriv.Name, schemaPriv.Name, &objPriv)
					statements = append(statements, stmt)
				}
			}
			if len(schemaPriv.Connections) > 0 {
				for _, objPriv := range schemaPriv.Connections {
					stmt := buildRevokeNestedConnectionPrivilege(userName, dbPriv.Name, schemaPriv.Name, &objPriv)
					statements = append(statements, stmt)
				}
			}
			if len(schemaPriv.Secrets) > 0 {
				for _, objPriv := range schemaPriv.Secrets {
					stmt := buildRevokeNestedSecretPrivilege(userName, dbPriv.Name, schemaPriv.Name, &objPriv)
					statements = append(statements, stmt)
				}
			}
			if len(schemaPriv.Functions) > 0 {
				for _, objPriv := range schemaPriv.Functions {
					stmt := buildRevokeNestedFunctionPrivilege(userName, dbPriv.Name, schemaPriv.Name, &objPriv)
					statements = append(statements, stmt)
				}
			}
		}
	}

	return statements
}

// BuildAlterUserPermissionsSQL builds ALTER USER statements for permission changes.
func BuildAlterUserPermissionsSQL(userName string, permissions []v1alpha1.UserPermission) string {
	if len(permissions) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("ALTER USER ")
	sb.WriteString(QuoteUser(userName))

	for _, perm := range permissions {
		sb.WriteString(" ")
		sb.WriteString(string(perm))
	}

	return sb.String()
}

// NormalizePrivileges normalizes privilege list to a comma-separated string.
func NormalizePrivileges[T ~string](privs []T) string {
	if len(privs) == 0 {
		return "USAGE"
	}

	var normalized []string
	for _, p := range privs {
		ps := string(p)
		if ps == "ALL PRIVILEGES" {
			ps = "ALL"
		}
		normalized = append(normalized, ps)
	}

	return strings.Join(normalized, ", ")
}

// normalizePrivileges is a lowercase alias for internal use.
func normalizePrivileges[T ~string](privs []T) string {
	return NormalizePrivileges(privs)
}

// buildGrantOption returns the "WITH GRANT OPTION" clause if enabled.
func buildGrantOption(withGrant bool) string {
	if withGrant {
		return " WITH GRANT OPTION"
	}
	return ""
}

// escapeStringLiteral escapes a string for use in SQL string literals.
func escapeStringLiteral(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// getUserName returns the actual user name from spec, defaulting to metadata.name.
func getUserName(user *v1alpha1.RisingWaveUser) string {
	if user.Spec.Name != "" {
		return user.Spec.Name
	}
	return user.Name
}
