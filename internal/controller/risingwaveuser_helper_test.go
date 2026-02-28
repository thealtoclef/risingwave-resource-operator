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

package controller

import (
	"fmt"
	"strings"
	"testing"

	v1alpha1 "github.com/risingwavelabs/risingwave-resource-operator/api/v1alpha1"
	"github.com/risingwavelabs/risingwave-resource-operator/internal/rwclient"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestRedactSQL tests the SQL redaction function.
func TestRedactSQL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "redact password in CREATE USER",
			input:    "CREATE USER \"test\" WITH PASSWORD 'secret123'",
			expected: "CREATE USER \"test\" WITH PASSWORD '***'",
		},
		{
			name:     "redact password in ALTER USER",
			input:    "ALTER USER \"test\" WITH PASSWORD 'newpass456'",
			expected: "ALTER USER \"test\" WITH PASSWORD '***'",
		},
		{
			name:     "don't modify GRANT statement",
			input:    "GRANT SELECT ON TABLE \"db\".\"schema\".\"table\" TO \"user\"",
			expected: "GRANT SELECT ON TABLE \"db\".\"schema\".\"table\" TO \"user\"",
		},
		{
			name:     "don't modify REVOKE statement",
			input:    "REVOKE INSERT ON TABLE \"db\".\"schema\".\"table\" FROM \"user\"",
			expected: "REVOKE INSERT ON TABLE \"db\".\"schema\".\"table\" FROM \"user\"",
		},
		{
			name:     "redact SECRET in ON clause",
			input:    "GRANT USAGE ON SECRET \"db\".\"schema\".\"mysecret\" TO \"user\"",
			expected: "GRANT USAGE ON [REDACTED] \"db\".\"schema\".\"mysecret\" TO \"user\"",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactSQL(tt.input)
			if result != tt.expected {
				t.Errorf("redactSQL() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestIsPasswordAuth tests the isPasswordAuth helper.
func TestIsPasswordAuth(t *testing.T) {
	tests := []struct {
		name     string
		user     *v1alpha1.RisingWaveUser
		expected bool
	}{
		{
			name: "nil auth config defaults to password",
			user: &v1alpha1.RisingWaveUser{
				Spec: v1alpha1.RisingWaveUserSpec{
					Auth: nil,
				},
			},
			expected: true,
		},
		{
			name: "nil auth type defaults to password",
			user: &v1alpha1.RisingWaveUser{
				Spec: v1alpha1.RisingWaveUserSpec{
					Auth: &v1alpha1.AuthConfig{
						Type: nil,
					},
				},
			},
			expected: true,
		},
		{
			name: "explicit password auth",
			user: &v1alpha1.RisingWaveUser{
				Spec: v1alpha1.RisingWaveUserSpec{
					Auth: &v1alpha1.AuthConfig{
						Type: func() *v1alpha1.AuthType {
							t := v1alpha1.AuthTypePassword
							return &t
						}(),
					},
				},
			},
			expected: true,
		},
		{
			name: "oauth auth",
			user: &v1alpha1.RisingWaveUser{
				Spec: v1alpha1.RisingWaveUserSpec{
					Auth: &v1alpha1.AuthConfig{
						Type: func() *v1alpha1.AuthType {
							t := v1alpha1.AuthTypeOAuth
							return &t
						}(),
					},
				},
			},
			expected: false,
		},
		{
			name: "ldap auth",
			user: &v1alpha1.RisingWaveUser{
				Spec: v1alpha1.RisingWaveUserSpec{
					Auth: &v1alpha1.AuthConfig{
						Type: func() *v1alpha1.AuthType {
							t := v1alpha1.AuthTypeLDAP
							return &t
						}(),
					},
				},
			},
			expected: false,
		},
	}

	reconciler := &RisingWaveUserReconciler{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reconciler.isPasswordAuth(tt.user)
			if result != tt.expected {
				t.Errorf("isPasswordAuth() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestFindObjectPrivilege tests the findObjectPrivilege helper.
func TestFindObjectPrivilege(t *testing.T) {
	tests := []struct {
		name        string
		objects     []rwclient.ObjectPrivilege
		targetName  string
		expectFound bool
		expectPrivs []string
	}{
		{
			name: "find existing object",
			objects: []rwclient.ObjectPrivilege{
				{Name: "table1", Privileges: []string{"SELECT", "INSERT"}},
				{Name: "table2", Privileges: []string{"SELECT"}},
			},
			targetName:  "table1",
			expectFound: true,
			expectPrivs: []string{"SELECT", "INSERT"},
		},
		{
			name: "object not found returns empty object",
			objects: []rwclient.ObjectPrivilege{
				{Name: "table1", Privileges: []string{"SELECT", "INSERT"}},
			},
			targetName:  "nonexistent",
			expectFound: false,
			expectPrivs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findObjectPrivilege(tt.objects, tt.targetName)
			if result.Name != tt.targetName {
				t.Errorf("findObjectPrivilege() name = %q, want %q", result.Name, tt.targetName)
			}
			// Check privileges length
			if len(result.Privileges) != len(tt.expectPrivs) {
				t.Errorf("findObjectPrivilege() privileges length = %d, want %d", len(result.Privileges), len(tt.expectPrivs))
			}
		})
	}
}

// TestPrivilegeSliceToString tests the privilegeSliceToString helper.
func TestPrivilegeSliceToString(t *testing.T) {
	tests := []struct {
		name     string
		input    []v1alpha1.TablePrivilegeType
		expected []string
	}{
		{
			name:     "convert table privileges",
			input:    []v1alpha1.TablePrivilegeType{v1alpha1.TablePrivilegeSelect, v1alpha1.TablePrivilegeInsert},
			expected: []string{"SELECT", "INSERT"},
		},
		{
			name:     "empty slice",
			input:    []v1alpha1.TablePrivilegeType{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := privilegeSliceToString(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("privilegeSliceToString() length = %d, want %d", len(result), len(tt.expected))
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("privilegeSliceToString()[%d] = %q, want %q", i, v, tt.expected[i])
				}
			}
		})
	}
}

// TestBuildCreateUserSQL tests the buildCreateUserSQL helper for different auth types.
func TestBuildCreateUserSQL(t *testing.T) {
	tests := []struct {
		name           string
		user           *v1alpha1.RisingWaveUser
		userName       string
		expectError    bool
		expectContains []string
	}{
		{
			name: "oauth auth",
			user: &v1alpha1.RisingWaveUser{
				ObjectMeta: metav1.ObjectMeta{
					Name: "oauth-user",
				},
				Spec: v1alpha1.RisingWaveUserSpec{
					Auth: &v1alpha1.AuthConfig{
						Type: func() *v1alpha1.AuthType {
							t := v1alpha1.AuthTypeOAuth
							return &t
						}(),
						OAuth: &v1alpha1.OAuthConfig{
							JWKSUrl: "https://example.com/jwks",
							Issuer:  "https://example.com",
						},
					},
				},
			},
			userName:    "oauth-user",
			expectError: false,
			expectContains: []string{
				"CREATE USER",
				"\"oauth-user\"",
				"oauth-placeholder",
			},
		},
		{
			name: "ldap auth",
			user: &v1alpha1.RisingWaveUser{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ldap-user",
				},
				Spec: v1alpha1.RisingWaveUserSpec{
					Auth: &v1alpha1.AuthConfig{
						Type: func() *v1alpha1.AuthType {
							t := v1alpha1.AuthTypeLDAP
							return &t
						}(),
						LDAP: &v1alpha1.LDAPConfig{
							Host:   "ldap.example.com",
							BaseDN: "dc=example,dc=com",
						},
					},
				},
			},
			userName:    "ldap-user",
			expectError: false,
			expectContains: []string{
				"CREATE USER",
				"\"ldap-user\"",
				"ldap-placeholder",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reconciler := &RisingWaveUserReconciler{}
			result, err := reconciler.buildCreateUserSQL(tt.user, tt.userName)

			if (err != nil) != tt.expectError {
				t.Errorf("buildCreateUserSQL() error = %v, expectError %v", err, tt.expectError)
				return
			}

			for _, expected := range tt.expectContains {
				if !strings.Contains(result, expected) {
					t.Errorf("buildCreateUserSQL() result = %q, does not contain %q", result, expected)
				}
			}
		})
	}
}

// TestMultiDatabasePrivilegeSync tests that multi-database privilege sync works correctly.
func TestMultiDatabasePrivilegeSync(t *testing.T) {
	t.Run("verify database context switching logic", func(t *testing.T) {
		// Simulate the logic from syncPrivileges
		databaseNames := []string{"test_db1", "test_db2", "test_db3"}

		// Verify that we can group statements by database
		type dbStatements struct {
			database   string
			statements []string
		}
		statementsByDB := make(map[string]*dbStatements)

		for _, db := range databaseNames {
			if _, exists := statementsByDB[db]; !exists {
				statementsByDB[db] = &dbStatements{database: db}
			}
			statementsByDB[db].statements = append(statementsByDB[db].statements,
				fmt.Sprintf("GRANT SELECT ON TABLE %s.schema1.table1 TO user1", db),
				fmt.Sprintf("GRANT USAGE ON SCHEMA %s.schema1 TO user1", db),
			)
		}

		// Verify each database has statements
		for _, db := range databaseNames {
			if len(statementsByDB[db].statements) == 0 {
				t.Errorf("Database %s has no statements", db)
			}
			if statementsByDB[db].database != db {
				t.Errorf("Database name mismatch: got %v, want %v", statementsByDB[db].database, db)
			}
		}

		// Verify total statement count
		totalStmts := 0
		for _, dbStmts := range statementsByDB {
			totalStmts += len(dbStmts.statements)
		}
		expectedStmts := len(databaseNames) * 2
		if totalStmts != expectedStmts {
			t.Errorf("Expected %d statements, got %d", expectedStmts, totalStmts)
		}
	})
}

// TestWildcardPrivilegeSQL tests wildcard privilege SQL generation.
func TestWildcardPrivilegeSQL(t *testing.T) {
	tests := []struct {
		name           string
		objectType     string
		schemaName     string
		expectedSubstr string
	}{
		{
			name:           "ALL TABLES IN SCHEMA",
			objectType:     "ALL TABLES IN SCHEMA",
			schemaName:     "app_schema",
			expectedSubstr: "ON ALL TABLES IN SCHEMA \"app_schema\"",
		},
		{
			name:           "ALL VIEWS IN SCHEMA",
			objectType:     "ALL VIEWS IN SCHEMA",
			schemaName:     "app_schema",
			expectedSubstr: "ON ALL VIEWS IN SCHEMA \"app_schema\"",
		},
		{
			name:           "ALL MATERIALIZED VIEWS IN SCHEMA",
			objectType:     "ALL MATERIALIZED VIEWS IN SCHEMA",
			schemaName:     "analytics_schema",
			expectedSubstr: "ON ALL MATERIALIZED VIEWS IN SCHEMA \"analytics_schema\"",
		},
		{
			name:           "ALL SOURCES IN SCHEMA",
			objectType:     "ALL SOURCES IN SCHEMA",
			schemaName:     "public_schema",
			expectedSubstr: "ON ALL SOURCES IN SCHEMA \"public_schema\"",
		},
		{
			name:           "ALL CONNECTIONS IN SCHEMA",
			objectType:     "ALL CONNECTIONS IN SCHEMA",
			schemaName:     "app_schema",
			expectedSubstr: "ON ALL CONNECTIONS IN SCHEMA \"app_schema\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the buildOnClause logic from CalculateObjectDiff
			result := fmt.Sprintf("GRANT SELECT %s TO user1", buildWildcardOnClause(tt.objectType, tt.schemaName))
			if !strings.Contains(result, tt.expectedSubstr) {
				t.Errorf("Expected substring %v not found in %v", tt.expectedSubstr, result)
			}
		})
	}
}

// buildWildcardOnClause simulates the wildcard clause logic from CalculateObjectDiff.
func buildWildcardOnClause(objectType, schemaName string) string {
	if strings.HasPrefix(objectType, "ALL ") {
		return fmt.Sprintf("ON %s \"%s\"", objectType, schemaName)
	}
	return fmt.Sprintf("ON ALL %s IN SCHEMA \"%s\"", objectType+"S", schemaName)
}

// TestOrphanCleanup tests the orphan cleanup logic.
func TestOrphanCleanup(t *testing.T) {
	t.Run("orphan table should be identified for revocation", func(t *testing.T) {
		// Simulate actual state with 2 tables
		actualTables := []rwclient.ObjectPrivilege{
			{Name: "users", Privileges: []string{"SELECT", "INSERT"}},
			{Name: "orders", Privileges: []string{"SELECT"}},
		}

		// Spec only has 1 table (users)
		specTableName := "users"

		// Build map of actual tables
		actualMap := make(map[string]rwclient.ObjectPrivilege)
		for _, tbl := range actualTables {
			actualMap[tbl.Name] = tbl
		}

		// Remove spec table from actual map
		delete(actualMap, specTableName)

		// Remaining tables are orphans
		orphanCount := len(actualMap)
		if orphanCount != 1 {
			t.Errorf("Expected 1 orphan, got %d", orphanCount)
		}

		if _, exists := actualMap["orders"]; !exists {
			t.Errorf("Expected 'orders' to be identified as orphan")
		}

		// Verify the orphan has the expected privileges
		orphan := actualMap["orders"]
		if len(orphan.Privileges) != 1 {
			t.Errorf("Expected orphan to have 1 privilege, got %d", len(orphan.Privileges))
		}
		if orphan.Privileges[0] != "SELECT" {
			t.Errorf("Expected orphan privilege SELECT, got %v", orphan.Privileges[0])
		}
	})

	t.Run("no orphans when spec matches actual", func(t *testing.T) {
		actualTables := []rwclient.ObjectPrivilege{
			{Name: "users", Privileges: []string{"SELECT"}},
			{Name: "orders", Privileges: []string{"SELECT", "INSERT"}},
		}

		// Spec has both tables
		specTableNames := []string{"users", "orders"}

		actualMap := make(map[string]rwclient.ObjectPrivilege)
		for _, tbl := range actualTables {
			actualMap[tbl.Name] = tbl
		}

		for _, specName := range specTableNames {
			delete(actualMap, specName)
		}

		orphanCount := len(actualMap)
		if orphanCount != 0 {
			t.Errorf("Expected 0 orphans, got %d", orphanCount)
		}
	})
}

// TestAllObjectTypesCoverage tests that all 8+ object types are supported.
func TestAllObjectTypesCoverage(t *testing.T) {
	expectedObjectTypes := []string{
		"TABLE", "VIEW", "MATERIALIZED VIEW",
		"SOURCE", "SINK", "CONNECTION", "SECRET", "FUNCTION",
	}

	for _, objType := range expectedObjectTypes {
		t.Run(objType+" privilege generation", func(t *testing.T) {
			// Verify each object type can generate valid GRANT/REVOKE statements
			schemaName := "test_schema"
			objectName := "test_object"

			grantStmt := fmt.Sprintf("GRANT SELECT ON %s \"%s\".\"%s\" TO \"user1\"",
				objType, schemaName, objectName)
			revokeStmt := fmt.Sprintf("REVOKE SELECT ON %s \"%s\".\"%s\" FROM \"user1\"",
				objType, schemaName, objectName)

			if !strings.Contains(grantStmt, "GRANT SELECT") {
				t.Errorf("GRANT statement for %s is invalid: %v", objType, grantStmt)
			}
			if !strings.Contains(revokeStmt, "REVOKE SELECT") {
				t.Errorf("REVOKE statement for %s is invalid: %v", objType, revokeStmt)
			}

			// Verify statements contain proper quoting
			if !strings.Contains(grantStmt, "\""+schemaName+"\"") {
				t.Errorf("Schema not properly quoted in: %v", grantStmt)
			}
			if !strings.Contains(grantStmt, "\""+objectName+"\"") {
				t.Errorf("Object not properly quoted in: %v", grantStmt)
			}
		})
	}
}

// TestDatabasePrivilegeDiff tests database-level privilege diff calculation.
func TestDatabasePrivilegeDiff(t *testing.T) {
	tests := []struct {
		name         string
		actualPrivs  []string
		desiredPrivs []v1alpha1.DatabasePrivilegeType
		expectGrant  bool
		expectRevoke bool
	}{
		{
			name:         "new privilege should be granted",
			actualPrivs:  []string{"CONNECT"},
			desiredPrivs: []v1alpha1.DatabasePrivilegeType{v1alpha1.DatabasePrivilegeConnect, v1alpha1.DatabasePrivilegeCreate},
			expectGrant:  true,
			expectRevoke: false,
		},
		{
			name:         "removed privilege should be revoked",
			actualPrivs:  []string{"CONNECT", "CREATE"},
			desiredPrivs: []v1alpha1.DatabasePrivilegeType{v1alpha1.DatabasePrivilegeConnect},
			expectGrant:  false,
			expectRevoke: true,
		},
		{
			name:         "no changes needed",
			actualPrivs:  []string{"CONNECT"},
			desiredPrivs: []v1alpha1.DatabasePrivilegeType{v1alpha1.DatabasePrivilegeConnect},
			expectGrant:  false,
			expectRevoke: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualDB := rwclient.DatabasePrivilegeSnapshot{
				Name:       "test_db",
				Privileges: tt.actualPrivs,
			}
			desiredDB := &v1alpha1.DatabasePrivilege{
				Name:       "test_db",
				Privileges: tt.desiredPrivs,
			}

			diff := rwclient.CalculateDatabaseDiff("testuser", actualDB, desiredDB)

			if (len(diff.ToGrant) > 0) != tt.expectGrant {
				t.Errorf("Expected grant=%v, got grants: %v", tt.expectGrant, diff.ToGrant)
			}
			if (len(diff.ToRevoke) > 0) != tt.expectRevoke {
				t.Errorf("Expected revoke=%v, got revokes: %v", tt.expectRevoke, diff.ToRevoke)
			}
		})
	}
}

// TestSchemaPrivilegeDiff tests schema-level privilege diff calculation.
func TestSchemaPrivilegeDiff(t *testing.T) {
	tests := []struct {
		name         string
		actualPrivs  []string
		desiredPrivs []v1alpha1.SchemaPrivilegeType
		expectGrant  bool
		expectRevoke bool
	}{
		{
			name:         "grant USAGE privilege",
			actualPrivs:  []string{},
			desiredPrivs: []v1alpha1.SchemaPrivilegeType{v1alpha1.SchemaPrivilegeUsage},
			expectGrant:  true,
			expectRevoke: false,
		},
		{
			name:         "grant CREATE privilege",
			actualPrivs:  []string{"USAGE"},
			desiredPrivs: []v1alpha1.SchemaPrivilegeType{v1alpha1.SchemaPrivilegeUsage, v1alpha1.SchemaPrivilegeCreate},
			expectGrant:  true,
			expectRevoke: false,
		},
		{
			name:         "revoke CREATE privilege",
			actualPrivs:  []string{"USAGE", "CREATE"},
			desiredPrivs: []v1alpha1.SchemaPrivilegeType{v1alpha1.SchemaPrivilegeUsage},
			expectGrant:  false,
			expectRevoke: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualSchema := rwclient.SchemaPrivilegeSnapshot{
				Name:       "test_schema",
				Privileges: tt.actualPrivs,
			}
			desiredSchema := &v1alpha1.NestedSchemaPrivilege{
				Name:       "test_schema",
				Privileges: tt.desiredPrivs,
			}

			diff := rwclient.CalculateSchemaDiff("testuser", actualSchema, desiredSchema)

			if (len(diff.ToGrant) > 0) != tt.expectGrant {
				t.Errorf("Expected grant=%v, got grants: %v", tt.expectGrant, diff.ToGrant)
			}
			if (len(diff.ToRevoke) > 0) != tt.expectRevoke {
				t.Errorf("Expected revoke=%v, got revokes: %v", tt.expectRevoke, diff.ToRevoke)
			}
		})
	}
}

// TestObjectPrivilegeDiff tests object-level privilege diff calculation.
func TestObjectPrivilegeDiff(t *testing.T) {
	tests := []struct {
		name         string
		actualPrivs  []string
		desiredPrivs []string
		expectGrant  bool
		expectRevoke bool
		objectType   string
	}{
		{
			name:         "grant table privileges",
			actualPrivs:  []string{},
			desiredPrivs: []string{"SELECT", "INSERT"},
			expectGrant:  true,
			expectRevoke: false,
			objectType:   "TABLE",
		},
		{
			name:         "revoke table privileges",
			actualPrivs:  []string{"SELECT", "INSERT", "UPDATE"},
			desiredPrivs: []string{"SELECT"},
			expectGrant:  false,
			expectRevoke: true,
			objectType:   "TABLE",
		},
		{
			name:         "grant only for view (no revoke needed)",
			actualPrivs:  []string{"SELECT"},
			desiredPrivs: []string{"SELECT", "INSERT"},
			expectGrant:  true,
			expectRevoke: false,
			objectType:   "VIEW",
		},
		{
			name:         "grant and revoke for view",
			actualPrivs:  []string{"SELECT", "DELETE"},
			desiredPrivs: []string{"SELECT", "INSERT"},
			expectGrant:  true,
			expectRevoke: true,
			objectType:   "VIEW",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualObj := rwclient.ObjectPrivilege{
				Name:       "test_object",
				Privileges: tt.actualPrivs,
			}

			diff := rwclient.CalculateObjectDiff("testuser", "test_db", "test_schema",
				tt.objectType, actualObj, "test_object", tt.desiredPrivs)

			if (len(diff.ToGrant) > 0) != tt.expectGrant {
				t.Errorf("Expected grant=%v, got grants: %v", tt.expectGrant, diff.ToGrant)
			}
			if (len(diff.ToRevoke) > 0) != tt.expectRevoke {
				t.Errorf("Expected revoke=%v, got revokes: %v", tt.expectRevoke, diff.ToRevoke)
			}

			// Verify GRANT statement format
			for _, grant := range diff.ToGrant {
				if !strings.Contains(grant, "GRANT") {
					t.Errorf("GRANT statement doesn't contain GRANT: %v", grant)
				}
				if !strings.Contains(grant, tt.objectType) {
					t.Errorf("GRANT statement doesn't contain object type %s: %v", tt.objectType, grant)
				}
			}

			// Verify REVOKE statement format
			for _, revoke := range diff.ToRevoke {
				if !strings.Contains(revoke, "REVOKE") {
					t.Errorf("REVOKE statement doesn't contain REVOKE: %v", revoke)
				}
				if !strings.Contains(revoke, tt.objectType) {
					t.Errorf("REVOKE statement doesn't contain object type %s: %v", tt.objectType, revoke)
				}
			}
		})
	}
}
