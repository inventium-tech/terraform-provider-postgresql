package pgclient

import (
	"context"
	"fmt"
)

// queries.
const (
	existsDatabaseQuery = `
	SELECT EXISTS (
		SELECT 1
		FROM pg_catalog.pg_database d
		WHERE d.datname = $1
	);`
)

type DatabaseRepo interface {
	Exists(ctx context.Context, dbtx DBTX, name string) (bool, error)
}

func NewDatabaseRepo() DatabaseRepo {
	return &databaseRepo{}
}

type databaseRepo struct{}

func (d databaseRepo) Exists(ctx context.Context, dbtx DBTX, name string) (bool, error) {
	var err error

	if name, err = sanitizeInput(name, SanitizeIdentifier); err != nil {
		return false, fmt.Errorf("invalid database name %q. error: %w", name, err)
	}

	var exists bool
	err = dbtx.QueryRow(ctx, existsDatabaseQuery, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if database %s exists: %w", name, err)
	}

	return exists, err
}
