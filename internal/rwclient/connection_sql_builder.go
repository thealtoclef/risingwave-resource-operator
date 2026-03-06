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
	"sort"
	"strings"

	v1alpha1 "github.com/risingwavelabs/risingwave-resource-operator/api/v1alpha1"
)

// BuildCreateConnectionSQL builds a CREATE CONNECTION statement.
func BuildCreateConnectionSQL(conn *v1alpha1.RisingWaveConnection) string {
	var sb strings.Builder

	sb.WriteString("CREATE CONNECTION ")
	sb.WriteString(QuoteIdentifier(conn.GetConnectionName()))
	sb.WriteString("\nWITH (\n")

	// Always include type first
	sb.WriteString("    type = '")
	sb.WriteString(escapeStringLiteral(string(conn.Spec.Type)))
	sb.WriteString("'")

	// Add remaining properties in sorted order for deterministic output
	if len(conn.Spec.Properties) > 0 {
		keys := make([]string, 0, len(conn.Spec.Properties))
		for k := range conn.Spec.Properties {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, key := range keys {
			value := conn.Spec.Properties[key]
			sb.WriteString(",\n    ")
			sb.WriteString(key) // Property names don't need quoting in RisingWave
			sb.WriteString(" = ")

			// Check if value is a secret reference (prefix: "SECRET ")
			if strings.HasPrefix(value, "SECRET ") {
				secretName := strings.TrimPrefix(value, "SECRET ")
				sb.WriteString("SECRET ")
				sb.WriteString(QuoteIdentifier(secretName))
			} else {
				sb.WriteString("'")
				sb.WriteString(escapeStringLiteral(value))
				sb.WriteString("'")
			}
		}
	}

	sb.WriteString("\n)")

	return sb.String()
}

// BuildDropConnectionSQL builds a DROP CONNECTION IF EXISTS statement.
func BuildDropConnectionSQL(connName string) string {
	return fmt.Sprintf("DROP CONNECTION IF EXISTS %s", QuoteIdentifier(connName))
}

// BuildAlterConnectionSQL builds an ALTER CONNECTION statement for updating properties.
func BuildAlterConnectionSQL(conn *v1alpha1.RisingWaveConnection) string {
	var sb strings.Builder

	sb.WriteString("ALTER CONNECTION ")
	sb.WriteString(QuoteIdentifier(conn.GetConnectionName()))
	sb.WriteString("\nCONNECTOR WITH (\n")

	// Add properties in sorted order
	if len(conn.Spec.Properties) > 0 {
		keys := make([]string, 0, len(conn.Spec.Properties))
		for k := range conn.Spec.Properties {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for i, key := range keys {
			value := conn.Spec.Properties[key]
			if i > 0 {
				sb.WriteString(",\n")
			}
			sb.WriteString("    ")
			sb.WriteString(key)
			sb.WriteString(" = ")

			// Check if value is a secret reference (prefix: "SECRET ")
			if strings.HasPrefix(value, "SECRET ") {
				secretName := strings.TrimPrefix(value, "SECRET ")
				sb.WriteString("SECRET ")
				sb.WriteString(QuoteIdentifier(secretName))
			} else {
				sb.WriteString("'")
				sb.WriteString(escapeStringLiteral(value))
				sb.WriteString("'")
			}
		}
	}

	sb.WriteString("\n)")

	return sb.String()
}

// BuildAlterConnectionOwnerSQL builds an ALTER CONNECTION OWNER statement.
func BuildAlterConnectionOwnerSQL(connName, owner string) string {
	return fmt.Sprintf("ALTER CONNECTION %s OWNER TO %s",
		QuoteIdentifier(connName),
		QuoteUser(owner))
}

// CheckConnectionExists checks if a connection exists in the current database schema.
func CheckConnectionExists(ctx context.Context, db *sql.DB, connName string) (bool, error) {
	row := db.QueryRowContext(ctx,
		"SELECT name FROM rw_catalog.rw_connections WHERE name = $1", connName)
	var name string
	if err := row.Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetConnectionOwner returns the owner of a connection in the current database schema.
func GetConnectionOwner(ctx context.Context, db *sql.DB, connName string) (string, error) {
	row := db.QueryRowContext(ctx,
		"SELECT owner FROM rw_catalog.rw_connections WHERE name = $1", connName)
	var owner string
	if err := row.Scan(&owner); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("connection %s not found", connName)
		}
		return "", err
	}
	return owner, nil
}
