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
	"sort"
	"strings"

	v1alpha1 "github.com/risingwavelabs/risingwave-resource-operator/api/v1alpha1"
)

// PrivilegeDiff represents the difference between actual and desired privileges.
type PrivilegeDiff struct {
	ToGrant  []string
	ToRevoke []string
}

// CalculateDatabaseDiff calculates the diff for database-level privileges.
func CalculateDatabaseDiff(userName string, actual DatabasePrivilegeSnapshot, desired *v1alpha1.DatabasePrivilege) PrivilegeDiff {
	var diff PrivilegeDiff

	desiredPrivs := make(map[string]bool)
	for _, p := range desired.Privileges {
		desiredPrivs[string(p)] = true
	}

	actualPrivs := make(map[string]bool)
	for _, p := range actual.Privileges {
		actualPrivs[p] = true
	}

	// Privileges to GRANT
	var grantPrivs []string
	for p := range desiredPrivs {
		if !actualPrivs[p] {
			grantPrivs = append(grantPrivs, p)
		}
	}
	if len(grantPrivs) > 0 {
		diff.ToGrant = append(diff.ToGrant, fmt.Sprintf("GRANT %s ON DATABASE %s TO %s",
			strings.Join(grantPrivs, ", "),
			QuoteIdentifier(desired.Name),
			QuoteUser(userName)))
	}

	// Privileges to REVOKE
	var revokePrivs []string
	for p := range actualPrivs {
		if !desiredPrivs[p] {
			revokePrivs = append(revokePrivs, p)
		}
	}
	if len(revokePrivs) > 0 {
		diff.ToRevoke = append(diff.ToRevoke, fmt.Sprintf("REVOKE %s ON DATABASE %s FROM %s",
			strings.Join(revokePrivs, ", "),
			QuoteIdentifier(desired.Name),
			QuoteUser(userName)))
	}

	return diff
}

// CalculateSchemaDiff calculates the diff for schema-level privileges.
func CalculateSchemaDiff(userName string, actual SchemaPrivilegeSnapshot, desired *v1alpha1.NestedSchemaPrivilege) PrivilegeDiff {
	var diff PrivilegeDiff

	desiredPrivs := make(map[string]bool)
	for _, p := range desired.Privileges {
		desiredPrivs[string(p)] = true
	}

	actualPrivs := make(map[string]bool)
	for _, p := range actual.Privileges {
		actualPrivs[p] = true
	}

	// GRANT
	var grantPrivs []string
	for p := range desiredPrivs {
		if !actualPrivs[p] {
			grantPrivs = append(grantPrivs, p)
		}
	}
	if len(grantPrivs) > 0 {
		diff.ToGrant = append(diff.ToGrant, fmt.Sprintf("GRANT %s ON SCHEMA %s TO %s",
			strings.Join(grantPrivs, ", "),
			QuoteIdentifier(desired.Name),
			QuoteUser(userName)))
	}

	// REVOKE
	var revokePrivs []string
	for p := range actualPrivs {
		if !desiredPrivs[p] {
			revokePrivs = append(revokePrivs, p)
		}
	}
	if len(revokePrivs) > 0 {
		diff.ToRevoke = append(diff.ToRevoke, fmt.Sprintf("REVOKE %s ON SCHEMA %s FROM %s",
			strings.Join(revokePrivs, ", "),
			QuoteIdentifier(desired.Name),
			QuoteUser(userName)))
	}

	return diff
}

// CalculateObjectDiff calculates the diff for object-level privileges (Tables, Views, MVs, etc.).
func CalculateObjectDiff(userName string, databaseName string, schemaName string, objectType string, actual ObjectPrivilege, desiredName string, desiredPrivs []string) PrivilegeDiff {
	var diff PrivilegeDiff

	dPrivs := make(map[string]bool)
	for _, p := range desiredPrivs {
		dPrivs[p] = true
	}

	aPrivs := make(map[string]bool)
	for _, p := range actual.Privileges {
		aPrivs[p] = true
	}

	// Helper to build ON clause
	buildOnClause := func(targetName string) string {
		if targetName == "*" {
			// Check if objectType already contains the "ALL ... IN SCHEMA" prefix
			if strings.HasPrefix(objectType, "ALL ") {
				// objectType is already "ALL TABLES IN SCHEMA" format, use it directly
				return fmt.Sprintf("ON %s %s", objectType, QuoteIdentifier(schemaName))
			}
			// Build wildcard format from simple type (e.g., "TABLE" -> "ALL TABLES IN SCHEMA")
			pluralType := objectType + "S"
			if objectType == "MATERIALIZED VIEW" {
				pluralType = "MATERIALIZED VIEWS"
			}
			return fmt.Sprintf("ON ALL %s IN SCHEMA %s", pluralType, QuoteIdentifier(schemaName))
		}
		// For specific objects, use schema.object format
		// Note: RisingWave does not support cross-database references (database.schema.object)
		// The connection must be to the target database before granting privileges
		return fmt.Sprintf("ON %s %s.%s", objectType, QuoteIdentifier(schemaName), QuoteIdentifier(targetName))
	}

	// GRANT
	var grantPrivs []string
	for p := range dPrivs {
		if !aPrivs[p] {
			grantPrivs = append(grantPrivs, p)
		}
	}
	if len(grantPrivs) > 0 {
		sort.Strings(grantPrivs)
		diff.ToGrant = append(diff.ToGrant, fmt.Sprintf("GRANT %s %s TO %s",
			strings.Join(grantPrivs, ", "),
			buildOnClause(desiredName),
			QuoteUser(userName)))
	}

	// REVOKE
	var revokePrivs []string
	for p := range aPrivs {
		if !dPrivs[p] {
			revokePrivs = append(revokePrivs, p)
		}
	}
	if len(revokePrivs) > 0 {
		sort.Strings(revokePrivs)
		diff.ToRevoke = append(diff.ToRevoke, fmt.Sprintf("REVOKE %s %s FROM %s",
			strings.Join(revokePrivs, ", "),
			buildOnClause(desiredName),
			QuoteUser(userName)))
	}

	return diff
}
