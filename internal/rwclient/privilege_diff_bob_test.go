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

// TestMapCharToTablePrivilegeString tests the string mapper directly
func TestMapCharToTablePrivilegeString(t *testing.T) {
	tests := []struct {
		char     string
		expected string
	}{
		{"r", "SELECT"},
		{"a", "INSERT"},
		{"w", "UPDATE"},
		{"d", "DELETE"},
		{"D", "TRUNCATE"},
		{"x", "REFERENCES"},
		{"t", "TRIGGER"},
		{"X", ""}, // Function privilege, not table
		{"U", ""}, // Schema privilege
		{"c", ""}, // Database privilege
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.char, func(t *testing.T) {
			result := MapCharToTablePrivilegeString(tt.char)
			if result != tt.expected {
				t.Errorf("MapCharToTablePrivilegeString(%q) = %q, want %q", tt.char, result, tt.expected)
			} else {
				t.Logf("✓ '%s' -> '%s'", tt.char, tt.expected)
			}
		})
	}
}

// TestCalculateObjectDiffWithBob tests the exact scenario
func TestCalculateObjectDiffWithBob(t *testing.T) {
	userName := "bob"
	databaseName := "dev"
	schemaName := "test_schema"
	objectType := "TABLE"
	desiredName := "users"

	// Actual: bob has INSERT, SELECT, UPDATE (from ACL bob=arw/root)
	actualObj := ObjectPrivilege{
		Name:            "users",
		Privileges:      []string{"INSERT", "SELECT", "UPDATE"},
		WithGrantOption: false,
	}

	// Desired: only SELECT, INSERT
	desiredPrivs := []string{"SELECT", "INSERT"}

	diff := CalculateObjectDiff(userName, databaseName, schemaName, objectType, actualObj, desiredName, desiredPrivs)

	t.Logf("Actual privileges: %v", actualObj.Privileges)
	t.Logf("Desired privileges: %v", desiredPrivs)
	t.Logf("ToRevoke: %v", diff.ToRevoke)
	t.Logf("ToGrant: %v", diff.ToGrant)

	// Should have UPDATE in ToRevoke
	if len(diff.ToRevoke) != 1 {
		t.Errorf("Expected 1 revoke statement, got %d: %v", len(diff.ToRevoke), diff.ToRevoke)
	} else {
		if diff.ToRevoke[0] == "" {
			t.Errorf("Revoke statement is empty!")
		}
	}
}
