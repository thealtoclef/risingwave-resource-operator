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
	"testing"
)

func TestBuildCreateDatabaseSQL(t *testing.T) {
	tests := []struct {
		name     string
		dbName   string
		expected string
	}{
		{
			name:     "simple database name",
			dbName:   "mydb",
			expected: `CREATE DATABASE "mydb"`,
		},
		{
			name:     "database name with underscores",
			dbName:   "my_database",
			expected: `CREATE DATABASE "my_database"`,
		},
		{
			name:     "database name with special chars",
			dbName:   "my-db",
			expected: `CREATE DATABASE "my-db"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildCreateDatabaseSQL(tt.dbName)
			if got != tt.expected {
				t.Errorf("BuildCreateDatabaseSQL() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBuildDropDatabaseSQL(t *testing.T) {
	tests := []struct {
		name     string
		dbName   string
		expected string
	}{
		{
			name:     "simple database name",
			dbName:   "mydb",
			expected: `DROP DATABASE IF EXISTS "mydb"`,
		},
		{
			name:     "database name with underscores",
			dbName:   "my_database",
			expected: `DROP DATABASE IF EXISTS "my_database"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildDropDatabaseSQL(tt.dbName)
			if got != tt.expected {
				t.Errorf("BuildDropDatabaseSQL() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBuildAlterDatabaseOwnerSQL(t *testing.T) {
	tests := []struct {
		name     string
		dbName   string
		owner    string
		expected string
	}{
		{
			name:     "simple names",
			dbName:   "mydb",
			owner:    "alice",
			expected: `ALTER DATABASE "mydb" OWNER TO "alice"`,
		},
		{
			name:     "names with special chars",
			dbName:   "my-db",
			owner:    "admin-user",
			expected: `ALTER DATABASE "my-db" OWNER TO "admin-user"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildAlterDatabaseOwnerSQL(tt.dbName, tt.owner)
			if got != tt.expected {
				t.Errorf("BuildAlterDatabaseOwnerSQL() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBuildCreateSchemaSQL(t *testing.T) {
	tests := []struct {
		name       string
		schemaName string
		expected   string
	}{
		{
			name:       "simple schema name",
			schemaName: "myschema",
			expected:   `CREATE SCHEMA "myschema"`,
		},
		{
			name:       "schema name with underscores",
			schemaName: "my_schema",
			expected:   `CREATE SCHEMA "my_schema"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildCreateSchemaSQL(tt.schemaName)
			if got != tt.expected {
				t.Errorf("BuildCreateSchemaSQL() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBuildCreateSchemaWithOwnerSQL(t *testing.T) {
	tests := []struct {
		name       string
		schemaName string
		owner      string
		expected   string
	}{
		{
			name:       "simple names",
			schemaName: "myschema",
			owner:      "alice",
			expected:   `CREATE SCHEMA "myschema" AUTHORIZATION "alice"`,
		},
		{
			name:       "names with special chars",
			schemaName: "my-schema",
			owner:      "admin-user",
			expected:   `CREATE SCHEMA "my-schema" AUTHORIZATION "admin-user"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildCreateSchemaWithOwnerSQL(tt.schemaName, tt.owner)
			if got != tt.expected {
				t.Errorf("BuildCreateSchemaWithOwnerSQL() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBuildDropSchemaSQL(t *testing.T) {
	tests := []struct {
		name       string
		schemaName string
		expected   string
	}{
		{
			name:       "simple schema name",
			schemaName: "myschema",
			expected:   `DROP SCHEMA IF EXISTS "myschema" CASCADE`,
		},
		{
			name:       "schema name with underscores",
			schemaName: "my_schema",
			expected:   `DROP SCHEMA IF EXISTS "my_schema" CASCADE`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildDropSchemaSQL(tt.schemaName)
			if got != tt.expected {
				t.Errorf("BuildDropSchemaSQL() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBuildAlterSchemaOwnerSQL(t *testing.T) {
	tests := []struct {
		name       string
		schemaName string
		owner      string
		expected   string
	}{
		{
			name:       "simple names",
			schemaName: "myschema",
			owner:      "alice",
			expected:   `ALTER SCHEMA "myschema" OWNER TO "alice"`,
		},
		{
			name:       "names with special chars",
			schemaName: "my-schema",
			owner:      "admin-user",
			expected:   `ALTER SCHEMA "my-schema" OWNER TO "admin-user"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildAlterSchemaOwnerSQL(tt.schemaName, tt.owner)
			if got != tt.expected {
				t.Errorf("BuildAlterSchemaOwnerSQL() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBuildUseDatabaseSQL(t *testing.T) {
	tests := []struct {
		name     string
		dbName   string
		expected string
	}{
		{
			name:     "simple database name",
			dbName:   "mydb",
			expected: `USE "mydb"`,
		},
		{
			name:     "database name with underscores",
			dbName:   "my_database",
			expected: `USE "my_database"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildUseDatabaseSQL(tt.dbName)
			if got != tt.expected {
				t.Errorf("BuildUseDatabaseSQL() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDatabaseSQLIdentifierEscaping(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "identifier with double quotes gets escaped in database",
			input:    `db"name`,
			contains: `"db""name"`,
		},
		{
			name:     "identifier with double quotes gets escaped in schema",
			input:    `schema"name`,
			contains: `"schema""name"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test database builders
			dbSQL := BuildCreateDatabaseSQL(tt.input)
			if !containsStr(dbSQL, tt.contains) {
				t.Errorf("BuildCreateDatabaseSQL() = %q, should contain %q", dbSQL, tt.contains)
			}

			// Test schema builders
			schemaSQL := BuildCreateSchemaSQL(tt.input)
			if !containsStr(schemaSQL, tt.contains) {
				t.Errorf("BuildCreateSchemaSQL() = %q, should contain %q", schemaSQL, tt.contains)
			}
		})
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
