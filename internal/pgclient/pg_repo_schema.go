package pgclient

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

// queries.
const (
	selectSchemaQuery = `
	SELECT n.nspname                               AS name,
	       pg_catalog.pg_get_userbyid(n.nspowner)  AS owner
	FROM pg_catalog.pg_namespace n
	WHERE n.nspname = $1;`

	existsSchemaQuery = `
	SELECT EXISTS (
		SELECT 1
		FROM information_schema.schemata
		WHERE schema_name = $1
	);`
)

type SchemaRepo interface {
	Create(ctx context.Context, dbtx DBTX, params SchemaCreateParams) error
	Exists(ctx context.Context, dbtx DBTX, name string) (bool, error)
	Drop(ctx context.Context, dbtx DBTX, name string, cascade bool) error
	GetOne(ctx context.Context, dbtx DBTX, name string) (*SchemaModel, error)
	Update(ctx context.Context, dbtx DBTX, name string, params SchemaUpdateParams) error
	List(ctx context.Context, dbtx DBTX, params SchemaListParams) ([]SchemaModel, error)
}

func NewSchemaRepo() SchemaRepo {
	return &schemaRepo{}
}

type (
	SchemaModel struct {
		Name  pgtype.Text `json:"name"`
		Owner pgtype.Text `json:"owner"`
	}
	SchemaCreateParams struct {
		Name        string  `json:"name" validate:"required"`
		Owner       string  `json:"owner"`
		IfNotExists bool    `json:"if_not_exists"`
		Policy      *string `json:"policy"`
	}
	SchemaUpdateParams struct {
		Owner *string `json:"owner"`
	}
	SchemaListParams struct {
		IncludeSystemSchemas bool     `json:"include_system_schemas"`
		LikeAnyPatterns      []string `json:"like_any_patterns"`
		LikeAllPatterns      []string `json:"like_all_patterns"`
		NotLikeAllPatterns   []string `json:"not_like_all_patterns"`
		RegexPattern         string   `json:"regex_pattern"`
	}
)

type schemaRepo struct{}

func (s schemaRepo) Create(ctx context.Context, dbtx DBTX, params SchemaCreateParams) error {
	var err error

	if params.Name, err = sanitizeInput(params.Name, SanitizeIdentifier); err != nil {
		return fmt.Errorf("invalid schema name %q: %w", params.Name, err)
	}

	var queryParts []string

	if params.IfNotExists {
		queryParts = append(queryParts, "CREATE SCHEMA IF NOT EXISTS")
	} else {
		queryParts = append(queryParts, "CREATE SCHEMA")
	}

	queryParts = append(queryParts, params.Name)

	if params.Owner != "" {
		if params.Owner, err = sanitizeInput(params.Owner, SanitizeIdentifier); err != nil {
			return fmt.Errorf("invalid owner %q: %w", params.Owner, err)
		}
		queryParts = append(queryParts, fmt.Sprintf("AUTHORIZATION %s", params.Owner))
	}

	query := strings.Join(queryParts, " ")
	_, err = dbtx.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create schema %s: %w", params.Name, err)
	}

	return nil
}

func (s schemaRepo) Exists(ctx context.Context, dbtx DBTX, name string) (bool, error) {
	var err error

	if name, err = sanitizeInput(name, SanitizeIdentifier); err != nil {
		return false, fmt.Errorf("invalid schema name %q: %w", name, err)
	}

	var exists bool
	err = dbtx.QueryRow(ctx, existsSchemaQuery, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if schema %s exists: %w", name, err)
	}

	return exists, nil
}

func (s schemaRepo) Drop(ctx context.Context, dbtx DBTX, name string, cascade bool) error {
	var err error

	if name, err = sanitizeInput(name, SanitizeIdentifier); err != nil {
		return fmt.Errorf("invalid schema name %q: %w", name, err)
	}

	// Prevent dropping the public schema
	if name == "public" {
		return fmt.Errorf("cannot drop the public schema")
	}

	query := fmt.Sprintf("DROP SCHEMA %s", name)
	if cascade {
		query += " CASCADE"
	}

	_, err = dbtx.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to drop schema %s: %w", name, err)
	}

	return nil
}

func (s schemaRepo) GetOne(ctx context.Context, dbtx DBTX, name string) (*SchemaModel, error) {
	var err error

	if name, err = sanitizeInput(name, SanitizeIdentifier); err != nil {
		return nil, fmt.Errorf("invalid schema name %q: %w", name, err)
	}

	var model SchemaModel
	err = dbtx.QueryRow(ctx, selectSchemaQuery, name).Scan(
		&model.Name,
		&model.Owner,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema %s: %w", name, err)
	}

	return &model, nil
}

func (s schemaRepo) Update(ctx context.Context, dbtx DBTX, name string, params SchemaUpdateParams) error {
	var err error

	if name, err = sanitizeInput(name, SanitizeIdentifier); err != nil {
		return fmt.Errorf("invalid schema name %q: %w", name, err)
	}

	if params.Owner != nil {
		if *params.Owner, err = sanitizeInput(*params.Owner, SanitizeIdentifier); err != nil {
			return fmt.Errorf("invalid owner %q: %w", *params.Owner, err)
		}
		query := fmt.Sprintf("ALTER SCHEMA %s OWNER TO %s", name, *params.Owner)
		if _, err := dbtx.Exec(ctx, query); err != nil {
			return fmt.Errorf("failed to change owner of schema %s: %w", name, err)
		}
	}

	return nil
}

func (s schemaRepo) List(ctx context.Context, dbtx DBTX, params SchemaListParams) ([]SchemaModel, error) {
	var queryParts []string
	var args []interface{}
	argCount := 0

	queryParts = append(queryParts, `
		SELECT n.nspname AS name,
		       pg_catalog.pg_get_userbyid(n.nspowner) AS owner
		FROM pg_catalog.pg_namespace n
		WHERE 1=1`)

	// Filter out system schemas if requested
	if !params.IncludeSystemSchemas {
		queryParts = append(queryParts, `
			AND n.nspname NOT LIKE 'pg_%'
			AND n.nspname != 'information_schema'`)
	}

	// Apply LIKE ANY patterns
	if len(params.LikeAnyPatterns) > 0 {
		var conditions []string
		for _, pattern := range params.LikeAnyPatterns {
			argCount++
			conditions = append(conditions, fmt.Sprintf("n.nspname LIKE $%d", argCount))
			args = append(args, pattern)
		}
		queryParts = append(queryParts, fmt.Sprintf("AND (%s)", strings.Join(conditions, " OR ")))
	}

	// Apply LIKE ALL patterns
	if len(params.LikeAllPatterns) > 0 {
		for _, pattern := range params.LikeAllPatterns {
			argCount++
			queryParts = append(queryParts, fmt.Sprintf("AND n.nspname LIKE $%d", argCount))
			args = append(args, pattern)
		}
	}

	// Apply NOT LIKE ALL patterns
	if len(params.NotLikeAllPatterns) > 0 {
		for _, pattern := range params.NotLikeAllPatterns {
			argCount++
			queryParts = append(queryParts, fmt.Sprintf("AND n.nspname NOT LIKE $%d", argCount))
			args = append(args, pattern)
		}
	}

	// Apply regex pattern
	if params.RegexPattern != "" {
		argCount++
		queryParts = append(queryParts, fmt.Sprintf("AND n.nspname ~ $%d", argCount))
		args = append(args, params.RegexPattern)
	}

	queryParts = append(queryParts, "ORDER BY n.nspname")

	query := strings.Join(queryParts, " ")
	rows, err := dbtx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list schemas: %w", err)
	}
	defer rows.Close()

	var schemas []SchemaModel
	for rows.Next() {
		var schema SchemaModel
		if err := rows.Scan(&schema.Name, &schema.Owner); err != nil {
			return nil, fmt.Errorf("failed to scan schema row: %w", err)
		}
		schemas = append(schemas, schema)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating schema rows: %w", err)
	}

	return schemas, nil
}
