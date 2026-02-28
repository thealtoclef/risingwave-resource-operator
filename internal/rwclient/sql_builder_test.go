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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestQuoteIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "ordinary name",
			input:    "mytable",
			expected: `"mytable"`,
		},
		{
			name:     "name with double quotes",
			input:    `my"table`,
			expected: `"my""table"`,
		},
		{
			name:     "wildcard passthrough",
			input:    "*",
			expected: "*",
		},
		{
			name:     "name with special chars",
			input:    "my-table.v1",
			expected: `"my-table.v1"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := QuoteIdentifier(tt.input)
			if got != tt.expected {
				t.Errorf("QuoteIdentifier() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestQuoteUser(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "ordinary user",
			input:    "alice",
			expected: `"alice"`,
		},
		{
			name:     "user with quotes",
			input:    `alice"bob`,
			expected: `"alice""bob"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := QuoteUser(tt.input)
			if got != tt.expected {
				t.Errorf("QuoteUser() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBuildCreateUserSQL(t *testing.T) {
	tests := []struct {
		name     string
		user     *v1alpha1.RisingWaveUser
		password string
		expected string
	}{
		{
			name: "without password",
			user: &v1alpha1.RisingWaveUser{
				ObjectMeta: metav1.ObjectMeta{Name: "alice"},
				Spec: v1alpha1.RisingWaveUserSpec{
					Permissions: []v1alpha1.UserPermission{},
				},
			},
			password: "",
			expected: `CREATE USER "alice"`,
		},
		{
			name: "with password",
			user: &v1alpha1.RisingWaveUser{
				ObjectMeta: metav1.ObjectMeta{Name: "bob"},
				Spec: v1alpha1.RisingWaveUserSpec{
					Permissions: []v1alpha1.UserPermission{},
				},
			},
			password: "secret123",
			expected: `CREATE USER "bob" WITH PASSWORD 'secret123'`,
		},
		{
			name: "password with single quote escaping",
			user: &v1alpha1.RisingWaveUser{
				ObjectMeta: metav1.ObjectMeta{Name: "charlie"},
				Spec: v1alpha1.RisingWaveUserSpec{
					Permissions: []v1alpha1.UserPermission{},
				},
			},
			password: "pass'word",
			expected: `CREATE USER "charlie" WITH PASSWORD 'pass''word'`,
		},
		{
			name: "with SUPERUSER permission",
			user: &v1alpha1.RisingWaveUser{
				ObjectMeta: metav1.ObjectMeta{Name: "admin"},
				Spec: v1alpha1.RisingWaveUserSpec{
					Permissions: []v1alpha1.UserPermission{v1alpha1.UserPermission("SUPERUSER")},
				},
			},
			password: "adm1n",
			expected: `CREATE USER "admin" WITH PASSWORD 'adm1n' SUPERUSER`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildCreateUserSQL(tt.user, tt.password)
			if got != tt.expected {
				t.Errorf("BuildCreateUserSQL() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBuildAlterUserPasswordSQL(t *testing.T) {
	tests := []struct {
		name     string
		userName string
		password string
		expected string
	}{
		{
			name:     "basic password change",
			userName: "alice",
			password: "newpass",
			expected: `ALTER USER "alice" WITH PASSWORD 'newpass'`,
		},
		{
			name:     "password with single quotes",
			userName: "bob",
			password: "pass'123",
			expected: `ALTER USER "bob" WITH PASSWORD 'pass''123'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildAlterUserPasswordSQL(tt.userName, tt.password)
			if got != tt.expected {
				t.Errorf("BuildAlterUserPasswordSQL() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBuildDropUserSQL(t *testing.T) {
	tests := []struct {
		name     string
		userName string
		expected string
	}{
		{
			name:     "basic drop user",
			userName: "alice",
			expected: `DROP USER IF EXISTS "alice"`,
		},
		{
			name:     "user with quotes",
			userName: `alice"bob`,
			expected: `DROP USER IF EXISTS "alice""bob"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildDropUserSQL(tt.userName)
			if got != tt.expected {
				t.Errorf("BuildDropUserSQL() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBuildGrantStatements(t *testing.T) {
	tests := []struct {
		name     string
		userName string
		spec     *v1alpha1.RisingWaveUserSpec
		check    func([]string) bool
	}{
		{
			name:     "nil grants",
			userName: "alice",
			spec: &v1alpha1.RisingWaveUserSpec{
				Grants: nil,
			},
			check: func(stmts []string) bool {
				return len(stmts) == 0
			},
		},
		{
			name:     "empty grants",
			userName: "alice",
			spec: &v1alpha1.RisingWaveUserSpec{
				Grants: &v1alpha1.GrantSpec{
					Databases: []v1alpha1.DatabasePrivilege{},
				},
			},
			check: func(stmts []string) bool {
				return len(stmts) == 0
			},
		},
		{
			name:     "database CONNECT grant",
			userName: "alice",
			spec: &v1alpha1.RisingWaveUserSpec{
				Grants: &v1alpha1.GrantSpec{
					Databases: []v1alpha1.DatabasePrivilege{
						{
							Name: "mydb",
							Privileges: []v1alpha1.DatabasePrivilegeType{
								v1alpha1.DatabasePrivilegeConnect,
							},
						},
					},
				},
			},
			check: func(stmts []string) bool {
				return len(stmts) > 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildGrantStatements(tt.userName, tt.spec)
			if !tt.check(got) {
				t.Errorf("BuildGrantStatements() check failed for %q: %v", tt.name, got)
			}
		})
	}
}

func TestBuildAlterUserPermissionsSQL(t *testing.T) {
	tests := []struct {
		name        string
		userName    string
		permissions []v1alpha1.UserPermission
		expected    string
	}{
		{
			name:        "empty permissions",
			userName:    "alice",
			permissions: []v1alpha1.UserPermission{},
			expected:    "",
		},
		{
			name:     "single permission",
			userName: "bob",
			permissions: []v1alpha1.UserPermission{
				v1alpha1.UserPermission("SUPERUSER"),
			},
			expected: `ALTER USER "bob" SUPERUSER`,
		},
		{
			name:     "multiple permissions",
			userName: "charlie",
			permissions: []v1alpha1.UserPermission{
				v1alpha1.UserPermission("SUPERUSER"),
				v1alpha1.UserPermission("CREATEDB"),
			},
			expected: `ALTER USER "charlie" SUPERUSER CREATEDB`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildAlterUserPermissionsSQL(tt.userName, tt.permissions)
			if got != tt.expected {
				t.Errorf("BuildAlterUserPermissionsSQL() = %q, want %q", got, tt.expected)
			}
		})
	}
}
