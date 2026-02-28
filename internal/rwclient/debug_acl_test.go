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
	"testing"
)

// TestParseACLForBob tests the exact ACL from RisingWave
func TestParseACLForBob(t *testing.T) {
	aclStr := "bob=arw/root" // From RisingWave: a=INSERT, r=SELECT, w=UPDATE

	result := ParseACL(aclStr)

	if len(result) != 1 {
		t.Fatalf("Expected 1 user, got %d", len(result))
	}

	bobPrivs := result[0]
	if bobPrivs.User != "bob" {
		t.Errorf("Expected user 'bob', got '%s'", bobPrivs.User)
	}

	expectedPrivs := map[string]bool{"a": true, "r": true, "w": true}
	if len(bobPrivs.Privileges) != 3 {
		t.Errorf("Expected 3 privileges, got %d: %v", len(bobPrivs.Privileges), bobPrivs.Privileges)
	}

	for _, p := range bobPrivs.Privileges {
		if !expectedPrivs[p.Privilege] {
			t.Errorf("Unexpected privilege: %s", p.Privilege)
		}
		fmt.Printf("✓ Privilege '%s' found\n", p.Privilege)
	}

	// Test privilege mapping
	fmt.Println("\nPrivilege mappings:")
	for _, p := range bobPrivs.Privileges {
		mapped := MapCharToTablePrivilegeString(p.Privilege)
		fmt.Printf("  '%s' -> '%s'\n", p.Privilege, mapped)
		if mapped == "" {
			t.Errorf("Mapping for '%s' returned empty string", p.Privilege)
		}
	}
}

// TestFetchObjectPrivsCustomWithBob tests fetchObjectPrivsCustom with bob's ACL
func TestFetchObjectPrivsCustomWithBob(t *testing.T) {
	// Simulate the ACL parsing result
	aclStr := "bob=arw/root"
	userPrivs := ParseACL(aclStr)

	// This simulates what fetchObjectPrivsCustom does
	var privs []string
	for _, up := range userPrivs {
		if up.User == "bob" {
			for _, p := range up.Privileges {
				privType := MapCharToTablePrivilegeString(p.Privilege)
				if privType != "" {
					privs = append(privs, privType)
				}
			}
		}
	}

	fmt.Printf("\nMapped privileges for bob: %v\n", privs)

	// Check we have all three privileges
	if len(privs) != 3 {
		t.Errorf("Expected 3 privileges (INSERT, SELECT, UPDATE), got %d: %v", len(privs), privs)
	}

	expected := map[string]bool{"INSERT": true, "SELECT": true, "UPDATE": true}
	for _, priv := range privs {
		if !expected[priv] {
			t.Errorf("Unexpected privilege: %s", priv)
		}
		fmt.Printf("✓ Privilege '%s' correctly mapped\n", priv)
	}
}
