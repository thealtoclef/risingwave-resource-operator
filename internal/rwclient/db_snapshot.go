package rwclient

import (
	"context"
	"database/sql"
)

// DatabaseAccessorWrapper wraps sql.DB to implement DatabaseAccessor interface
type DatabaseAccessorWrapper struct {
	DB *sql.DB
}

func (w *DatabaseAccessorWrapper) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	rows, err := w.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlRowsWrapper{rows: rows}, nil
}

type sqlRowsWrapper struct {
	rows *sql.Rows
}

func (w *sqlRowsWrapper) Next() bool {
	return w.rows.Next()
}

func (w *sqlRowsWrapper) Scan(dest ...any) error {
	return w.rows.Scan(dest...)
}

func (w *sqlRowsWrapper) Close() error {
	return w.rows.Close()
}

func (w *sqlRowsWrapper) Err() error {
	return w.rows.Err()
}

// FetchPrivilegeSnapshotForUser fetches complete privilege snapshot for a user
func FetchPrivilegeSnapshotForUser(ctx context.Context, db *sql.DB, userName string) (*UserPrivilegeSnapshot, error) {
	wrapper := &DatabaseAccessorWrapper{DB: db}
	return FetchUserPrivilegeSnapshot(ctx, wrapper, userName)
}

// FetchSchemaPrivilegesForUser fetches schema and object privileges for a user
func FetchSchemaPrivilegesForUser(ctx context.Context, db *sql.DB, userName string) ([]SchemaPrivilegeSnapshot, error) {
	wrapper := &DatabaseAccessorWrapper{DB: db}
	return FetchSchemaPrivileges(ctx, wrapper, userName)
}
