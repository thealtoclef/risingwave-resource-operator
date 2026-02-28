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

func TestDatabasePrivilegeSnapshot_Structure(t *testing.T) {
	snapshot := DatabasePrivilegeSnapshot{
		Name:            "testdb",
		Privileges:      []string{"CONNECT", "CREATE"},
		WithGrantOption: true,
		Schemas: []SchemaPrivilegeSnapshot{
			{
				Name:            "public",
				Privileges:      []string{"USAGE"},
				WithGrantOption: false,
			},
		},
	}

	if snapshot.Name != "testdb" {
		t.Errorf("Name = %q, want %q", snapshot.Name, "testdb")
	}

	if len(snapshot.Privileges) != 2 {
		t.Errorf("Privileges length = %d, want 2", len(snapshot.Privileges))
	}

	if !snapshot.WithGrantOption {
		t.Error("WithGrantOption should be true")
	}

	if len(snapshot.Schemas) != 1 {
		t.Errorf("Schemas length = %d, want 1", len(snapshot.Schemas))
	}
}

func TestSchemaPrivilegeSnapshot_Structure(t *testing.T) {
	snapshot := SchemaPrivilegeSnapshot{
		Name:            "public",
		Privileges:      []string{"USAGE"},
		WithGrantOption: false,
		Tables: []ObjectPrivilege{
			{
				Name:            "users",
				Privileges:      []string{"SELECT", "INSERT"},
				WithGrantOption: false,
			},
		},
		Views: []ObjectPrivilege{
			{
				Name:            "user_view",
				Privileges:      []string{"SELECT"},
				WithGrantOption: true,
			},
		},
		MaterializedViews: []ObjectPrivilege{
			{
				Name:            "user_mv",
				Privileges:      []string{"SELECT"},
				WithGrantOption: false,
			},
		},
	}

	if snapshot.Name != "public" {
		t.Errorf("Name = %q, want %q", snapshot.Name, "public")
	}

	if len(snapshot.Tables) != 1 {
		t.Errorf("Tables length = %d, want 1", len(snapshot.Tables))
	}

	if len(snapshot.Views) != 1 {
		t.Errorf("Views length = %d, want 1", len(snapshot.Views))
	}

	if len(snapshot.MaterializedViews) != 1 {
		t.Errorf("MaterializedViews length = %d, want 1", len(snapshot.MaterializedViews))
	}
}

func TestObjectPrivilege_Structure(t *testing.T) {
	obj := ObjectPrivilege{
		Name:            "users",
		Privileges:      []string{"SELECT", "INSERT", "UPDATE", "DELETE"},
		WithGrantOption: true,
	}

	if obj.Name != "users" {
		t.Errorf("Name = %q, want %q", obj.Name, "users")
	}

	if len(obj.Privileges) != 4 {
		t.Errorf("Privileges length = %d, want 4", len(obj.Privileges))
	}

	if !obj.WithGrantOption {
		t.Error("WithGrantOption should be true")
	}
}

func TestUserPrivilegeSnapshot_Structure(t *testing.T) {
	snapshot := UserPrivilegeSnapshot{
		UserName: "alice",
		Databases: []DatabasePrivilegeSnapshot{
			{
				Name:            "db1",
				Privileges:      []string{"CONNECT"},
				WithGrantOption: false,
			},
			{
				Name:            "db2",
				Privileges:      []string{"CONNECT", "CREATE"},
				WithGrantOption: true,
			},
		},
	}

	if snapshot.UserName != "alice" {
		t.Errorf("UserName = %q, want %q", snapshot.UserName, "alice")
	}

	if len(snapshot.Databases) != 2 {
		t.Errorf("Databases length = %d, want 2", len(snapshot.Databases))
	}
}

func TestPrivilegeSnapshot_EmptyStructures(t *testing.T) {
	// Test empty database snapshot
	dbSnapshot := DatabasePrivilegeSnapshot{
		Name:       "emptydb",
		Privileges: []string{},
		Schemas:    []SchemaPrivilegeSnapshot{},
	}

	if len(dbSnapshot.Privileges) != 0 {
		t.Errorf("Empty privileges length = %d, want 0", len(dbSnapshot.Privileges))
	}

	if len(dbSnapshot.Schemas) != 0 {
		t.Errorf("Empty schemas length = %d, want 0", len(dbSnapshot.Schemas))
	}

	// Test empty user snapshot
	userSnapshot := UserPrivilegeSnapshot{
		UserName:  "bob",
		Databases: []DatabasePrivilegeSnapshot{},
	}

	if len(userSnapshot.Databases) != 0 {
		t.Errorf("Empty databases length = %d, want 0", len(userSnapshot.Databases))
	}
}
