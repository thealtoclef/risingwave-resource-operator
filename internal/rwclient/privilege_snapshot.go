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
	"context"
	"database/sql"
	"fmt"
)

// ObjectPrivilege represents privileges on a specific object.
type ObjectPrivilege struct {
	Name            string
	Privileges      []string
	WithGrantOption bool
}

// SchemaPrivilegeSnapshot represents privileges within a schema.
type SchemaPrivilegeSnapshot struct {
	Name              string
	Privileges        []string
	WithGrantOption   bool
	Tables            []ObjectPrivilege
	Views             []ObjectPrivilege
	MaterializedViews []ObjectPrivilege
	Sources           []ObjectPrivilege
	Sinks             []ObjectPrivilege
	Secrets           []ObjectPrivilege
	Functions         []ObjectPrivilege
}

// DatabasePrivilegeSnapshot represents privileges on a database.
type DatabasePrivilegeSnapshot struct {
	Name            string
	Privileges      []string
	WithGrantOption bool
	Schemas         []SchemaPrivilegeSnapshot
}

// UserPrivilegeSnapshot represents all privileges a user has in RisingWave.
type UserPrivilegeSnapshot struct {
	UserName  string
	Databases []DatabasePrivilegeSnapshot
}

// Rows is an interface for sql.Rows.
type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Close() error
	Err() error
}

// DatabaseAccessor provides methods to execute queries on a database.
type DatabaseAccessor interface {
	QueryContext(ctx context.Context, query string, args ...any) (Rows, error)
}

// FetchUserPrivilegeSnapshot fetches all privileges for a user.
func FetchUserPrivilegeSnapshot(ctx context.Context, db DatabaseAccessor, userName string) (*UserPrivilegeSnapshot, error) {
	snapshot := &UserPrivilegeSnapshot{
		UserName: userName,
	}

	// 1. Fetch Database Privileges
	dbPrivs, err := fetchDatabasePrivileges(ctx, db, userName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch database privileges: %w", err)
	}

	for dbName, privs := range dbPrivs {
		dbSnapshot := DatabasePrivilegeSnapshot{
			Name:            dbName,
			Privileges:      privs.Privileges,
			WithGrantOption: privs.WithGrantOption,
		}

		// 2. Fetch Schema and Object Privileges for this database
		// Note: This fetches schemas from the CURRENT database context
		// We need to execute USE <database> first, or use fully qualified queries
		schemas, err := FetchSchemaPrivileges(ctx, db, userName)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch schema privileges for database %s: %w", dbName, err)
		}

		// Only add schemas if we found any (matches the database we're querying)
		dbSnapshot.Schemas = schemas
		snapshot.Databases = append(snapshot.Databases, dbSnapshot)
	}

	return snapshot, nil
}

// FetchSchemaPrivileges fetches all schema and object privileges in the current database.
func FetchSchemaPrivileges(ctx context.Context, db DatabaseAccessor, userName string) ([]SchemaPrivilegeSnapshot, error) {
	// 1. Fetch Schemas
	schemas, err := fetchSchemaPrivs(ctx, db, userName)
	if err != nil {
		return nil, err
	}

	for i := range schemas {
		// 2. Fetch Tables
		tables, err := fetchTablePrivs(ctx, db, schemas[i].Name, userName)
		if err != nil {
			return nil, err
		}
		schemas[i].Tables = tables

		// 3. Fetch Views
		views, err := fetchViewPrivs(ctx, db, schemas[i].Name, userName)
		if err != nil {
			return nil, err
		}
		schemas[i].Views = views

		// 4. Fetch Materialized Views
		mvs, err := fetchMaterializedViewPrivs(ctx, db, schemas[i].Name, userName)
		if err != nil {
			return nil, err
		}
		schemas[i].MaterializedViews = mvs

		// 5. Fetch Sources
		sources, err := fetchSourcePrivs(ctx, db, schemas[i].Name, userName)
		if err != nil {
			return nil, err
		}
		schemas[i].Sources = sources

		// 6. Fetch Sinks
		sinks, err := fetchSinkPrivs(ctx, db, schemas[i].Name, userName)
		if err != nil {
			return nil, err
		}
		schemas[i].Sinks = sinks

		// 7. Fetch Secrets
		secrets, err := fetchSecretPrivs(ctx, db, schemas[i].Name, userName)
		if err != nil {
			return nil, err
		}
		schemas[i].Secrets = secrets

		// 9. Fetch Functions
		functions, err := fetchFunctionPrivs(ctx, db, schemas[i].Name, userName)
		if err != nil {
			return nil, err
		}
		schemas[i].Functions = functions
	}

	return schemas, nil
}

func fetchSchemaPrivs(ctx context.Context, db DatabaseAccessor, userName string) ([]SchemaPrivilegeSnapshot, error) {
	query := "SELECT name, acl FROM rw_catalog.rw_schemas"
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	var results []SchemaPrivilegeSnapshot
	for rows.Next() {
		var name string
		var acl sql.NullString
		if err := rows.Scan(&name, &acl); err != nil {
			return nil, err
		}

		if !acl.Valid {
			continue
		}

		userPrivs := ParseACL(acl.String)
		for _, up := range userPrivs {
			if up.User == userName {
				var privs []string
				grantOption := false
				for _, p := range up.Privileges {
					privType := string(MapCharToSchemaPrivilege(p.Privilege))
					if privType != "" {
						privs = append(privs, privType)
						if p.WithGrantOption {
							grantOption = true
						}
					}
				}
				if len(privs) > 0 {
					results = append(results, SchemaPrivilegeSnapshot{
						Name:            name,
						Privileges:      privs,
						WithGrantOption: grantOption,
					})
				}
			}
		}
	}
	return results, nil
}

func fetchDatabasePrivileges(ctx context.Context, db DatabaseAccessor, userName string) (map[string]struct {
	Privileges      []string
	WithGrantOption bool
}, error) {
	query := "SELECT name, acl FROM rw_catalog.rw_databases"
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	results := make(map[string]struct {
		Privileges      []string
		WithGrantOption bool
	})

	for rows.Next() {
		var name string
		var acl sql.NullString
		if err := rows.Scan(&name, &acl); err != nil {
			return nil, err
		}

		if !acl.Valid {
			continue
		}

		userPrivs := ParseACL(acl.String)
		for _, up := range userPrivs {
			if up.User == userName {
				var privs []string
				grantOption := false
				for _, p := range up.Privileges {
					privType := string(MapCharToDatabasePrivilege(p.Privilege))
					if privType != "" {
						privs = append(privs, privType)
						if p.WithGrantOption {
							grantOption = true
						}
					}
				}
				results[name] = struct {
					Privileges      []string
					WithGrantOption bool
				}{Privileges: privs, WithGrantOption: grantOption}
			}
		}
	}

	return results, nil
}

// fetchTablePrivs fetches table privileges for a user in a schema.
func fetchTablePrivs(ctx context.Context, db DatabaseAccessor, schemaName, userName string) ([]ObjectPrivilege, error) {
	return fetchObjectPrivsCustom(ctx, db, "rw_tables", schemaName, userName, MapCharToTablePrivilegeString)
}

// fetchViewPrivs fetches view privileges for a user in a schema.
func fetchViewPrivs(ctx context.Context, db DatabaseAccessor, schemaName, userName string) ([]ObjectPrivilege, error) {
	return fetchObjectPrivsCustom(ctx, db, "rw_views", schemaName, userName, MapCharToTablePrivilegeString)
}

// fetchMaterializedViewPrivs fetches materialized view privileges for a user in a schema.
func fetchMaterializedViewPrivs(ctx context.Context, db DatabaseAccessor, schemaName, userName string) ([]ObjectPrivilege, error) {
	return fetchObjectPrivsCustom(ctx, db, "rw_materialized_views", schemaName, userName, MapCharToTablePrivilegeString)
}

// fetchSourcePrivs fetches source privileges for a user in a schema.
func fetchSourcePrivs(ctx context.Context, db DatabaseAccessor, schemaName, userName string) ([]ObjectPrivilege, error) {
	return fetchObjectPrivsCustom(ctx, db, "rw_sources", schemaName, userName, MapCharToSourcePrivilegeString)
}

// fetchSinkPrivs fetches sink privileges for a user in a schema.
func fetchSinkPrivs(ctx context.Context, db DatabaseAccessor, schemaName, userName string) ([]ObjectPrivilege, error) {
	return fetchObjectPrivsCustom(ctx, db, "rw_sinks", schemaName, userName, MapCharToSinkPrivilegeString)
}

// fetchSecretPrivs fetches secret privileges for a user in a schema.
func fetchSecretPrivs(ctx context.Context, db DatabaseAccessor, schemaName, userName string) ([]ObjectPrivilege, error) {
	return fetchObjectPrivsCustom(ctx, db, "rw_secrets", schemaName, userName, MapCharToSecretPrivilegeString)
}

// fetchFunctionPrivs fetches function privileges for a user in a schema.
func fetchFunctionPrivs(ctx context.Context, db DatabaseAccessor, schemaName, userName string) ([]ObjectPrivilege, error) {
	return fetchObjectPrivsCustom(ctx, db, "rw_functions", schemaName, userName, MapCharToFunctionPrivilegeString)
}

// fetchObjectPrivsCustom fetches object privileges using a custom privilege mapper.
func fetchObjectPrivsCustom(ctx context.Context, db DatabaseAccessor, catalogTable, schemaName, userName string, mapper func(char string) string) ([]ObjectPrivilege, error) {
	query := fmt.Sprintf(`
		SELECT t.name, t.acl
		FROM rw_catalog.%s t
		JOIN rw_catalog.rw_schemas s ON t.schema_id = s.id
		WHERE s.name = $1`, catalogTable)

	rows, err := db.QueryContext(ctx, query, schemaName)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	var results []ObjectPrivilege
	for rows.Next() {
		var name string
		var acl sql.NullString
		if err := rows.Scan(&name, &acl); err != nil {
			return nil, err
		}

		if !acl.Valid {
			continue
		}

		userPrivs := ParseACL(acl.String)
		for _, up := range userPrivs {
			if up.User == userName {
				var privs []string
				grantOption := false
				for _, p := range up.Privileges {
					privType := mapper(p.Privilege)
					if privType != "" {
						privs = append(privs, privType)
						if p.WithGrantOption {
							grantOption = true
						}
					}
				}
				if len(privs) > 0 {
					results = append(results, ObjectPrivilege{
						Name:            name,
						Privileges:      privs,
						WithGrantOption: grantOption,
					})
				}
			}
		}
	}
	return results, nil
}
