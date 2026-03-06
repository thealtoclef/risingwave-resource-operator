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

// Database SQL Builders

// BuildCreateDatabaseSQL builds a CREATE DATABASE statement.
func BuildCreateDatabaseSQL(dbName string) string {
	return fmt.Sprintf("CREATE DATABASE %s", QuoteIdentifier(dbName))
}

// BuildDropDatabaseSQL builds a DROP DATABASE IF EXISTS statement.
func BuildDropDatabaseSQL(dbName string) string {
	return fmt.Sprintf("DROP DATABASE IF EXISTS %s", QuoteIdentifier(dbName))
}

// BuildAlterDatabaseOwnerSQL builds an ALTER DATABASE OWNER TO statement.
func BuildAlterDatabaseOwnerSQL(dbName, owner string) string {
	return fmt.Sprintf("ALTER DATABASE %s OWNER TO %s", QuoteIdentifier(dbName), QuoteUser(owner))
}

// Schema SQL Builders

// BuildCreateSchemaSQL builds a CREATE SCHEMA statement.
func BuildCreateSchemaSQL(schemaName string) string {
	return fmt.Sprintf("CREATE SCHEMA %s", QuoteIdentifier(schemaName))
}

// BuildCreateSchemaWithOwnerSQL builds a CREATE SCHEMA AUTHORIZATION statement.
func BuildCreateSchemaWithOwnerSQL(schemaName, owner string) string {
	return fmt.Sprintf("CREATE SCHEMA %s AUTHORIZATION %s", QuoteIdentifier(schemaName), QuoteUser(owner))
}

// BuildDropSchemaSQL builds a DROP SCHEMA IF EXISTS CASCADE statement.
func BuildDropSchemaSQL(schemaName string) string {
	return fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", QuoteIdentifier(schemaName))
}

// BuildAlterSchemaOwnerSQL builds an ALTER SCHEMA OWNER TO statement.
func BuildAlterSchemaOwnerSQL(schemaName, owner string) string {
	return fmt.Sprintf("ALTER SCHEMA %s OWNER TO %s", QuoteIdentifier(schemaName), QuoteUser(owner))
}

// Utility

// BuildUseDatabaseSQL builds a USE database statement.
func BuildUseDatabaseSQL(dbName string) string {
	return fmt.Sprintf("USE %s", QuoteIdentifier(dbName))
}

// Snapshot Functions

// CheckDatabaseExists checks if a database exists in RisingWave.
func CheckDatabaseExists(ctx context.Context, db *sql.DB, dbName string) (bool, error) {
	row := db.QueryRowContext(ctx,
		"SELECT name FROM rw_catalog.rw_databases WHERE name = $1", dbName)
	var name string
	if err := row.Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CheckSchemaExists checks if a schema exists in the current database.
func CheckSchemaExists(ctx context.Context, db *sql.DB, schemaName string) (bool, error) {
	row := db.QueryRowContext(ctx,
		"SELECT name FROM rw_catalog.rw_schemas WHERE name = $1", schemaName)
	var name string
	if err := row.Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetDatabaseOwner returns the owner of a database.
func GetDatabaseOwner(ctx context.Context, db *sql.DB, dbName string) (string, error) {
	row := db.QueryRowContext(ctx,
		"SELECT owner FROM rw_catalog.rw_databases WHERE name = $1", dbName)
	var owner string
	if err := row.Scan(&owner); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("database %s not found", dbName)
		}
		return "", err
	}
	return owner, nil
}

// GetSchemaOwner returns the owner of a schema in the current database.
func GetSchemaOwner(ctx context.Context, db *sql.DB, schemaName string) (string, error) {
	row := db.QueryRowContext(ctx,
		"SELECT owner FROM rw_catalog.rw_schemas WHERE name = $1", schemaName)
	var owner string
	if err := row.Scan(&owner); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("schema %s not found", schemaName)
		}
		return "", err
	}
	return owner, nil
}

// ListUserSchemas returns all user schemas (excluding system schemas) in the current database.
func ListUserSchemas(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx,
		"SELECT name FROM rw_catalog.rw_schemas WHERE name NOT IN ('public', 'rw_catalog', 'information_schema')")
	if err != nil {
		return nil, fmt.Errorf("failed to list schemas: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("failed to close rows: %w", err)
	}

	var schemas []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		schemas = append(schemas, name)
	}
	return schemas, nil
}
