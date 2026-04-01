package pgclient

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// queries.
const (
	selectExtensionQuery = `
	SELECT e.extname                                   AS name,
	       n.nspname                                   AS schema,
	       e.extversion                                AS version,
	       e.extrelocatable                            AS relocatable,
	       pg_catalog.obj_description(e.oid, 'pg_extension') AS comment
	FROM pg_catalog.pg_extension e
	LEFT JOIN pg_catalog.pg_namespace n ON e.extnamespace = n.oid
	WHERE e.extname = $1;`

	selectAllExtensionsQuery = `
	SELECT e.extname                                   AS name,
	       n.nspname                                   AS schema,
	       e.extversion                                AS version,
	       e.extrelocatable                            AS relocatable,
	       pg_catalog.obj_description(e.oid, 'pg_extension') AS comment
	FROM pg_catalog.pg_extension e
	LEFT JOIN pg_catalog.pg_namespace n ON e.extnamespace = n.oid
	ORDER BY e.extname;`

	existsExtensionQuery = `
	SELECT EXISTS (
		SELECT 1
		FROM pg_catalog.pg_extension e
		WHERE e.extname = $1
	);`

	selectAvailableExtensionsQuery = `
	SELECT name, default_version, installed_version, comment
	FROM pg_available_extensions
	WHERE name = $1;`
)

type ExtensionRepo interface {
	Create(ctx context.Context, dbtx DBTX, params ExtensionCreateParams) error
	Exists(ctx context.Context, dbtx DBTX, name string) (bool, error)
	Drop(ctx context.Context, dbtx DBTX, name string, cascade bool) error
	GetOne(ctx context.Context, dbtx DBTX, name string) (*ExtensionModel, error)
	GetAll(ctx context.Context, dbtx DBTX) ([]*ExtensionModel, error)
	Update(ctx context.Context, dbtx DBTX, name string, params ExtensionUpdateParams) error
	GetAvailable(ctx context.Context, dbtx DBTX, name string) (*AvailableExtensionModel, error)
}

func NewExtensionRepo() ExtensionRepo {
	return &extensionRepo{}
}

type (
	ExtensionModel struct {
		Name        pgtype.Text `json:"name"`
		Schema      pgtype.Text `json:"schema"`
		Version     pgtype.Text `json:"version"`
		Relocatable pgtype.Bool `json:"relocatable"`
		Comment     pgtype.Text `json:"comment"`
	}
	AvailableExtensionModel struct {
		Name             pgtype.Text `json:"name"`
		DefaultVersion   pgtype.Text `json:"default_version"`
		InstalledVersion pgtype.Text `json:"installed_version"`
		Comment          pgtype.Text `json:"comment"`
	}
	ExtensionCreateParams struct {
		Name     string `json:"name" validate:"required"`
		Schema   string `json:"schema"`
		Version  string `json:"version"`
		Cascade  bool   `json:"cascade"`
		Database string `json:"database"`
	}
	ExtensionUpdateParams struct {
		Version *string `json:"version"`
		Schema  *string `json:"schema"`
	}
)

type extensionRepo struct{}

// extensionVersionRe matches safe extension version strings.
// Versions must start with an alphanumeric character and may contain
// alphanumerics, dots, dashes, and underscores only.
var extensionVersionRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._\-]*$`)

// validateExtensionVersion returns an error if version contains characters
// that would be unsafe to interpolate into a SQL string literal.
func validateExtensionVersion(version string) error {
	if !extensionVersionRe.MatchString(version) {
		return fmt.Errorf("invalid extension version %q: must start with alphanumeric and contain only alphanumeric, dots, dashes, or underscores", version)
	}
	return nil
}

// quoteStringLiteral returns a safely-quoted PostgreSQL string literal.
// It doubles any embedded single-quote characters to prevent SQL injection.
// This provides defense-in-depth on top of regex validation.
func quoteStringLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func (e extensionRepo) Create(ctx context.Context, dbtx DBTX, params ExtensionCreateParams) error {
	var err error

	// Extension names can contain dashes, so we don't sanitize them
	// Instead, we use proper quoting
	if params.Name == "" {
		return fmt.Errorf("extension name cannot be empty")
	}

	var queryParts []string
	queryParts = append(queryParts, fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s", pgx.Identifier{params.Name}.Sanitize()))

	if params.Schema != "" {
		if params.Schema, err = sanitizeInput(params.Schema, SanitizeIdentifier); err != nil {
			return fmt.Errorf("invalid schema %q: %w", params.Schema, err)
		}
		queryParts = append(queryParts, fmt.Sprintf("SCHEMA %s", params.Schema))
	}

	if params.Version != "" {
		if err = validateExtensionVersion(params.Version); err != nil {
			return fmt.Errorf("invalid version for extension %s: %w", params.Name, err)
		}
		queryParts = append(queryParts, "VERSION "+quoteStringLiteral(params.Version))
	}

	if params.Cascade {
		queryParts = append(queryParts, "CASCADE")
	}

	query := strings.Join(queryParts, " ")

	_, err = dbtx.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create extension %s: %w", params.Name, err)
	}

	return nil
}

func (e extensionRepo) Exists(ctx context.Context, dbtx DBTX, name string) (bool, error) {
	var exists bool
	err := dbtx.QueryRow(ctx, existsExtensionQuery, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if extension %s exists: %w", name, err)
	}
	return exists, nil
}

func (e extensionRepo) Drop(ctx context.Context, dbtx DBTX, name string, cascade bool) error {
	if name == "" {
		return fmt.Errorf("extension name cannot be empty")
	}

	query := fmt.Sprintf("DROP EXTENSION IF EXISTS %s", pgx.Identifier{name}.Sanitize())
	if cascade {
		query += " CASCADE"
	}

	_, err := dbtx.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to drop extension %s: %w", name, err)
	}

	return nil
}

func (e extensionRepo) GetOne(ctx context.Context, dbtx DBTX, name string) (*ExtensionModel, error) {
	var ext ExtensionModel

	err := dbtx.QueryRow(ctx, selectExtensionQuery, name).Scan(
		&ext.Name,
		&ext.Schema,
		&ext.Version,
		&ext.Relocatable,
		&ext.Comment,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve extension %s: %w", name, err)
	}

	return &ext, nil
}

func (e extensionRepo) GetAll(ctx context.Context, dbtx DBTX) ([]*ExtensionModel, error) {
	rows, err := dbtx.Query(ctx, selectAllExtensionsQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve extensions: %w", err)
	}
	defer rows.Close()

	var extensions []*ExtensionModel
	for rows.Next() {
		var ext ExtensionModel
		err := rows.Scan(
			&ext.Name,
			&ext.Schema,
			&ext.Version,
			&ext.Relocatable,
			&ext.Comment,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan extension: %w", err)
		}
		extensions = append(extensions, &ext)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating extensions: %w", err)
	}

	return extensions, nil
}

func (e extensionRepo) Update(ctx context.Context, dbtx DBTX, name string, params ExtensionUpdateParams) error {
	if name == "" {
		return fmt.Errorf("extension name cannot be empty")
	}

	var err error
	quotedName := pgx.Identifier{name}.Sanitize()

	// Update version if provided
	if params.Version != nil && *params.Version != "" {
		if err = validateExtensionVersion(*params.Version); err != nil {
			return fmt.Errorf("invalid version for extension %s: %w", name, err)
		}
		query := fmt.Sprintf("ALTER EXTENSION %s UPDATE TO %s", quotedName, quoteStringLiteral(*params.Version))
		_, err = dbtx.Exec(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to update extension %s to version %s: %w", name, *params.Version, err)
		}
	} else if params.Version != nil {
		// Empty version means update to latest
		query := fmt.Sprintf("ALTER EXTENSION %s UPDATE", quotedName)
		_, err = dbtx.Exec(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to update extension %s to latest version: %w", name, err)
		}
	}

	// Update schema if provided
	if params.Schema != nil {
		schema := *params.Schema
		if schema, err = sanitizeInput(schema, SanitizeIdentifier); err != nil {
			return fmt.Errorf("invalid schema %q: %w", schema, err)
		}
		query := fmt.Sprintf("ALTER EXTENSION %s SET SCHEMA %s", quotedName, schema)
		_, err = dbtx.Exec(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to set extension %s schema to %s: %w", name, schema, err)
		}
	}

	return nil
}

func (e extensionRepo) GetAvailable(ctx context.Context, dbtx DBTX, name string) (*AvailableExtensionModel, error) {
	var ext AvailableExtensionModel

	err := dbtx.QueryRow(ctx, selectAvailableExtensionsQuery, name).Scan(
		&ext.Name,
		&ext.DefaultVersion,
		&ext.InstalledVersion,
		&ext.Comment,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve available extension %s: %w", name, err)
	}

	return &ext, nil
}
