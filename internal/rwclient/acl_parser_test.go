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

	v1alpha1 "github.com/risingwavelabs/risingwave-resource-operator/api/v1alpha1"
)

func TestParseACL(t *testing.T) {
	tests := []struct {
		name     string
		aclStr   string
		expected []UserPrivileges
	}{
		{
			name:     "empty string",
			aclStr:   "",
			expected: nil,
		},
		{
			name:     "empty braces",
			aclStr:   "{}",
			expected: nil,
		},
		{
			name:   "single entry",
			aclStr: "{user1=arwd/root}",
			expected: []UserPrivileges{
				{
					User: "user1",
					Privileges: []PrivilegeGrant{
						{Privilege: "a", WithGrantOption: false},
						{Privilege: "r", WithGrantOption: false},
						{Privilege: "w", WithGrantOption: false},
						{Privilege: "d", WithGrantOption: false},
					},
				},
			},
		},
		{
			name:   "grant option entry",
			aclStr: "{user1=r*/root}",
			expected: []UserPrivileges{
				{
					User: "user1",
					Privileges: []PrivilegeGrant{
						{Privilege: "r", WithGrantOption: true},
					},
				},
			},
		},
		{
			name:   "multiple entries",
			aclStr: "{user1=arwd/root,user2=r/root}",
			expected: []UserPrivileges{
				{
					User: "user1",
					Privileges: []PrivilegeGrant{
						{Privilege: "a", WithGrantOption: false},
						{Privilege: "r", WithGrantOption: false},
						{Privilege: "w", WithGrantOption: false},
						{Privilege: "d", WithGrantOption: false},
					},
				},
				{
					User: "user2",
					Privileges: []PrivilegeGrant{
						{Privilege: "r", WithGrantOption: false},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseACL(tt.aclStr)
			if len(got) != len(tt.expected) {
				t.Errorf("ParseACL() returned %d results, want %d", len(got), len(tt.expected))
				return
			}

			for i, userPriv := range got {
				if userPriv.User != tt.expected[i].User {
					t.Errorf("ParseACL()[%d].User = %q, want %q", i, userPriv.User, tt.expected[i].User)
				}
				if len(userPriv.Privileges) != len(tt.expected[i].Privileges) {
					t.Errorf("ParseACL()[%d] has %d privileges, want %d", i, len(userPriv.Privileges), len(tt.expected[i].Privileges))
				}
				for j, priv := range userPriv.Privileges {
					if priv.Privilege != tt.expected[i].Privileges[j].Privilege {
						t.Errorf("ParseACL()[%d].Privileges[%d].Privilege = %q, want %q", i, j, priv.Privilege, tt.expected[i].Privileges[j].Privilege)
					}
					if priv.WithGrantOption != tt.expected[i].Privileges[j].WithGrantOption {
						t.Errorf("ParseACL()[%d].Privileges[%d].WithGrantOption = %v, want %v", i, j, priv.WithGrantOption, tt.expected[i].Privileges[j].WithGrantOption)
					}
				}
			}
		})
	}
}

func TestMapCharToTablePrivilege(t *testing.T) {
	tests := []struct {
		name     string
		char     string
		expected v1alpha1.TablePrivilegeType
	}{
		{name: "r to SELECT", char: "r", expected: v1alpha1.TablePrivilegeSelect},
		{name: "a to INSERT", char: "a", expected: v1alpha1.TablePrivilegeInsert},
		{name: "w to UPDATE", char: "w", expected: v1alpha1.TablePrivilegeUpdate},
		{name: "d to DELETE", char: "d", expected: v1alpha1.TablePrivilegeDelete},
		{name: "D to TRUNCATE", char: "D", expected: v1alpha1.TablePrivilegeTruncate},
		{name: "x to REFERENCES", char: "x", expected: v1alpha1.TablePrivilegeReferences},
		{name: "t to TRIGGER", char: "t", expected: v1alpha1.TablePrivilegeTrigger},
		{name: "unknown char", char: "z", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapCharToTablePrivilege(tt.char)
			if got != tt.expected {
				t.Errorf("MapCharToTablePrivilege(%q) = %q, want %q", tt.char, got, tt.expected)
			}
		})
	}
}

func TestMapCharToDatabasePrivilege(t *testing.T) {
	tests := []struct {
		name     string
		char     string
		expected v1alpha1.DatabasePrivilegeType
	}{
		{name: "c to CONNECT", char: "c", expected: v1alpha1.DatabasePrivilegeConnect},
		{name: "C to CREATE", char: "C", expected: v1alpha1.DatabasePrivilegeCreate},
		{name: "unknown char", char: "z", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapCharToDatabasePrivilege(tt.char)
			if got != tt.expected {
				t.Errorf("MapCharToDatabasePrivilege(%q) = %q, want %q", tt.char, got, tt.expected)
			}
		})
	}
}

func TestMapCharToSchemaPrivilege(t *testing.T) {
	tests := []struct {
		name     string
		char     string
		expected v1alpha1.SchemaPrivilegeType
	}{
		{name: "U to USAGE", char: "U", expected: v1alpha1.SchemaPrivilegeUsage},
		{name: "C to CREATE", char: "C", expected: v1alpha1.SchemaPrivilegeCreate},
		{name: "unknown char", char: "z", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapCharToSchemaPrivilege(tt.char)
			if got != tt.expected {
				t.Errorf("MapCharToSchemaPrivilege(%q) = %q, want %q", tt.char, got, tt.expected)
			}
		})
	}
}
