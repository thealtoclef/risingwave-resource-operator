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
	"strings"
	"testing"

	v1alpha1 "github.com/risingwavelabs/risingwave-resource-operator/api/v1alpha1"
)

func TestCalculateDatabaseDiff(t *testing.T) {
	tests := []struct {
		name          string
		userName      string
		actual        DatabasePrivilegeSnapshot
		desired       *v1alpha1.DatabasePrivilege
		checkToGrant  func([]string) bool
		checkToRevoke func([]string) bool
	}{
		{
			name:     "desired is subset of actual",
			userName: "alice",
			actual: DatabasePrivilegeSnapshot{
				Name:            "mydb",
				Privileges:      []string{"CONNECT", "CREATE"},
				WithGrantOption: false,
			},
			desired: &v1alpha1.DatabasePrivilege{
				Name:       "mydb",
				Privileges: []v1alpha1.DatabasePrivilegeType{v1alpha1.DatabasePrivilegeConnect},
			},
			checkToGrant: func(stmts []string) bool {
				return len(stmts) == 0
			},
			checkToRevoke: func(stmts []string) bool {
				return len(stmts) == 1 && strings.Contains(stmts[0], "REVOKE")
			},
		},
		{
			name:     "actual is subset of desired",
			userName: "alice",
			actual: DatabasePrivilegeSnapshot{
				Name:            "mydb",
				Privileges:      []string{"CONNECT"},
				WithGrantOption: false,
			},
			desired: &v1alpha1.DatabasePrivilege{
				Name: "mydb",
				Privileges: []v1alpha1.DatabasePrivilegeType{
					v1alpha1.DatabasePrivilegeConnect,
					v1alpha1.DatabasePrivilegeCreate,
				},
			},
			checkToGrant: func(stmts []string) bool {
				return len(stmts) == 1 && strings.Contains(stmts[0], "GRANT")
			},
			checkToRevoke: func(stmts []string) bool {
				return len(stmts) == 0
			},
		},
		{
			name:     "equal privileges",
			userName: "alice",
			actual: DatabasePrivilegeSnapshot{
				Name:            "mydb",
				Privileges:      []string{"CONNECT"},
				WithGrantOption: false,
			},
			desired: &v1alpha1.DatabasePrivilege{
				Name:       "mydb",
				Privileges: []v1alpha1.DatabasePrivilegeType{v1alpha1.DatabasePrivilegeConnect},
			},
			checkToGrant: func(stmts []string) bool {
				return len(stmts) == 0
			},
			checkToRevoke: func(stmts []string) bool {
				return len(stmts) == 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := CalculateDatabaseDiff(tt.userName, tt.actual, tt.desired)
			if !tt.checkToGrant(diff.ToGrant) {
				t.Errorf("CalculateDatabaseDiff() ToGrant check failed: %v", diff.ToGrant)
			}
			if !tt.checkToRevoke(diff.ToRevoke) {
				t.Errorf("CalculateDatabaseDiff() ToRevoke check failed: %v", diff.ToRevoke)
			}
		})
	}
}

func TestCalculateSchemaDiff(t *testing.T) {
	tests := []struct {
		name          string
		userName      string
		actual        SchemaPrivilegeSnapshot
		desired       *v1alpha1.NestedSchemaPrivilege
		checkToGrant  func([]string) bool
		checkToRevoke func([]string) bool
	}{
		{
			name:     "desired is subset of actual",
			userName: "alice",
			actual: SchemaPrivilegeSnapshot{
				Name:            "myschema",
				Privileges:      []string{"USAGE", "CREATE"},
				WithGrantOption: false,
			},
			desired: &v1alpha1.NestedSchemaPrivilege{
				Name:       "myschema",
				Privileges: []v1alpha1.SchemaPrivilegeType{v1alpha1.SchemaPrivilegeUsage},
			},
			checkToGrant: func(stmts []string) bool {
				return len(stmts) == 0
			},
			checkToRevoke: func(stmts []string) bool {
				return len(stmts) == 1 && strings.Contains(stmts[0], "REVOKE")
			},
		},
		{
			name:     "equal privileges",
			userName: "alice",
			actual: SchemaPrivilegeSnapshot{
				Name:            "myschema",
				Privileges:      []string{"USAGE"},
				WithGrantOption: false,
			},
			desired: &v1alpha1.NestedSchemaPrivilege{
				Name:       "myschema",
				Privileges: []v1alpha1.SchemaPrivilegeType{v1alpha1.SchemaPrivilegeUsage},
			},
			checkToGrant: func(stmts []string) bool {
				return len(stmts) == 0
			},
			checkToRevoke: func(stmts []string) bool {
				return len(stmts) == 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := CalculateSchemaDiff(tt.userName, tt.actual, tt.desired)
			if !tt.checkToGrant(diff.ToGrant) {
				t.Errorf("CalculateSchemaDiff() ToGrant check failed: %v", diff.ToGrant)
			}
			if !tt.checkToRevoke(diff.ToRevoke) {
				t.Errorf("CalculateSchemaDiff() ToRevoke check failed: %v", diff.ToRevoke)
			}
		})
	}
}

func TestCalculateObjectDiff(t *testing.T) {
	tests := []struct {
		name          string
		userName      string
		databaseName  string
		schemaName    string
		objectType    string
		actual        ObjectPrivilege
		desiredName   string
		desiredPrivs  []string
		checkToGrant  func([]string) bool
		checkToRevoke func([]string) bool
	}{
		{
			name:         "specific table name",
			userName:     "alice",
			databaseName: "dev",
			schemaName:   "myschema",
			objectType:   "TABLE",
			actual: ObjectPrivilege{
				Name:            "mytable",
				Privileges:      []string{"SELECT"},
				WithGrantOption: false,
			},
			desiredName:  "mytable",
			desiredPrivs: []string{"SELECT"},
			checkToGrant: func(stmts []string) bool {
				return len(stmts) == 0
			},
			checkToRevoke: func(stmts []string) bool {
				return len(stmts) == 0
			},
		},
		{
			name:         "wildcard table privileges",
			userName:     "alice",
			databaseName: "dev",
			schemaName:   "myschema",
			objectType:   "TABLE",
			actual: ObjectPrivilege{
				Name:            "*",
				Privileges:      []string{},
				WithGrantOption: false,
			},
			desiredName:  "*",
			desiredPrivs: []string{"SELECT"},
			checkToGrant: func(stmts []string) bool {
				return len(stmts) == 1 && strings.Contains(stmts[0], "ON ALL TABLES")
			},
			checkToRevoke: func(stmts []string) bool {
				return len(stmts) == 0
			},
		},
		{
			name:         "materialized view plural form",
			userName:     "alice",
			databaseName: "dev",
			schemaName:   "myschema",
			objectType:   "MATERIALIZED VIEW",
			actual: ObjectPrivilege{
				Name:            "*",
				Privileges:      []string{},
				WithGrantOption: false,
			},
			desiredName:  "*",
			desiredPrivs: []string{"SELECT"},
			checkToGrant: func(stmts []string) bool {
				return len(stmts) == 1 && strings.Contains(stmts[0], "ON ALL MATERIALIZED VIEWS")
			},
			checkToRevoke: func(stmts []string) bool {
				return len(stmts) == 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := CalculateObjectDiff(tt.userName, tt.databaseName, tt.schemaName, tt.objectType, tt.actual, tt.desiredName, tt.desiredPrivs)
			if !tt.checkToGrant(diff.ToGrant) {
				t.Errorf("CalculateObjectDiff() ToGrant check failed: %v", diff.ToGrant)
			}
			if !tt.checkToRevoke(diff.ToRevoke) {
				t.Errorf("CalculateObjectDiff() ToRevoke check failed: %v", diff.ToRevoke)
			}
		})
	}
}
