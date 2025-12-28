package pgclient

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

// queries.
const (
	selectDatabaseQuery = `
	SELECT d.datname                                       AS name,
	       pg_catalog.pg_get_userbyid(d.datdba)           AS owner,
	       pg_catalog.pg_encoding_to_char(d.encoding)     AS encoding,
	       d.datcollate                                   AS collation,
	       d.datctype                                     AS ctype,
	       d.datistemplate                                AS is_template,
	       d.datallowconn                                 AS allow_connections,
	       d.datconnlimit                                 AS connection_limit,
	       t.spcname                                      AS tablespace,
	       pg_catalog.shobj_description(d.oid, 'pg_database') AS comment
	FROM pg_catalog.pg_database d
	LEFT JOIN pg_catalog.pg_tablespace t ON d.dattablespace = t.oid
	WHERE d.datname = $1;`

	existsDatabaseQuery = `
	SELECT EXISTS (
		SELECT 1
		FROM pg_catalog.pg_database d
		WHERE d.datname = $1
	);`
)

type DatabaseRepo interface {
	Create(ctx context.Context, dbtx DBTX, params DatabaseCreateParams) error
	Exists(ctx context.Context, dbtx DBTX, name string) (bool, error)
	Drop(ctx context.Context, dbtx DBTX, name string, forceDrop bool) error
	GetOne(ctx context.Context, dbtx DBTX, name string) (*DatabaseModel, error)
	Update(ctx context.Context, dbtx DBTX, name string, params DatabaseUpdateParams) error
	TerminateConnections(ctx context.Context, dbtx DBTX, name string) error
}

func NewDatabaseRepo() DatabaseRepo {
	return &databaseRepo{}
}

type (
	DatabaseModel struct {
		Name             pgtype.Text `json:"name"`
		Owner            pgtype.Text `json:"owner"`
		Encoding         pgtype.Text `json:"encoding"`
		Collation        pgtype.Text `json:"collation"`
		Ctype            pgtype.Text `json:"ctype"`
		IsTemplate       pgtype.Bool `json:"is_template"`
		AllowConnections pgtype.Bool `json:"allow_connections"`
		ConnectionLimit  pgtype.Int4 `json:"connection_limit"`
		Tablespace       pgtype.Text `json:"tablespace"`
		Comment          pgtype.Text `json:"comment"`
	}
	DatabaseCreateParams struct {
		Name             string `json:"name" validate:"required"`
		Owner            string `json:"owner"`
		Encoding         string `json:"encoding"`
		Collation        string `json:"collation"`
		Ctype            string `json:"ctype"`
		Template         string `json:"template"`
		ConnectionLimit  int32  `json:"connection_limit"`
		AllowConnections *bool  `json:"allow_connections"`
		IsTemplate       *bool  `json:"is_template"`
		Tablespace       string `json:"tablespace"`
		Comment          string `json:"comment"`
	}
	DatabaseUpdateParams struct {
		Owner            *string `json:"owner"`
		ConnectionLimit  *int32  `json:"connection_limit"`
		AllowConnections *bool   `json:"allow_connections"`
		IsTemplate       *bool   `json:"is_template"`
		Tablespace       *string `json:"tablespace"`
		Comment          *string `json:"comment"`
	}
)

type databaseRepo struct{}

func (d databaseRepo) Create(ctx context.Context, dbtx DBTX, params DatabaseCreateParams) error {
	var err error

	if params.Name, err = sanitizeInput(params.Name, SanitizeIdentifier); err != nil {
		return fmt.Errorf("invalid database name %q: %w", params.Name, err)
	}

	var queryParts []string
	queryParts = append(queryParts, fmt.Sprintf("CREATE DATABASE %s", params.Name))

	if params.Owner != "" {
		if params.Owner, err = sanitizeInput(params.Owner, SanitizeIdentifier); err != nil {
			return fmt.Errorf("invalid owner %q: %w", params.Owner, err)
		}
		queryParts = append(queryParts, fmt.Sprintf("OWNER = %s", params.Owner))
	}

	if params.Template != "" {
		if params.Template, err = sanitizeInput(params.Template, SanitizeIdentifier); err != nil {
			return fmt.Errorf("invalid template %q: %w", params.Template, err)
		}
		queryParts = append(queryParts, fmt.Sprintf("TEMPLATE = %s", params.Template))
	}

	if params.Encoding != "" {
		queryParts = append(queryParts, fmt.Sprintf("ENCODING = '%s'", params.Encoding))
	}

	if params.Collation != "" {
		queryParts = append(queryParts, fmt.Sprintf("LC_COLLATE = '%s'", params.Collation))
	}

	if params.Ctype != "" {
		queryParts = append(queryParts, fmt.Sprintf("LC_CTYPE = '%s'", params.Ctype))
	}

	if params.Tablespace != "" {
		if params.Tablespace, err = sanitizeInput(params.Tablespace, SanitizeIdentifier); err != nil {
			return fmt.Errorf("invalid tablespace %q: %w", params.Tablespace, err)
		}
		queryParts = append(queryParts, fmt.Sprintf("TABLESPACE = %s", params.Tablespace))
	}

	if params.AllowConnections != nil {
		queryParts = append(queryParts, fmt.Sprintf("ALLOW_CONNECTIONS = %t", *params.AllowConnections))
	}

	queryParts = append(queryParts, fmt.Sprintf("CONNECTION LIMIT = %d", params.ConnectionLimit))

	if params.IsTemplate != nil {
		queryParts = append(queryParts, fmt.Sprintf("IS_TEMPLATE = %t", *params.IsTemplate))
	}

	query := strings.Join(queryParts, " WITH ")
	if !strings.Contains(query, " WITH ") {
		query = queryParts[0]
	} else {
		query = queryParts[0] + " WITH " + strings.Join(queryParts[1:], " ")
	}

	_, err = dbtx.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create database %s: %w", params.Name, err)
	}

	if params.Comment != "" {
		commentQuery := fmt.Sprintf("COMMENT ON DATABASE %s IS '%s'", params.Name, params.Comment)
		_, err = dbtx.Exec(ctx, commentQuery)
		if err != nil {
			return fmt.Errorf("failed to set comment on database %s: %w", params.Name, err)
		}
	}

	return nil
}

func (d databaseRepo) Exists(ctx context.Context, dbtx DBTX, name string) (bool, error) {
	var err error

	if name, err = sanitizeInput(name, SanitizeIdentifier); err != nil {
		return false, fmt.Errorf("invalid database name %q: %w", name, err)
	}

	var exists bool
	err = dbtx.QueryRow(ctx, existsDatabaseQuery, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if database %s exists: %w", name, err)
	}

	return exists, nil
}

func (d databaseRepo) Drop(ctx context.Context, dbtx DBTX, name string, forceDrop bool) error {
	var err error

	if name, err = sanitizeInput(name, SanitizeIdentifier); err != nil {
		return fmt.Errorf("invalid database name %q: %w", name, err)
	}

	if forceDrop {
		if err := d.TerminateConnections(ctx, dbtx, name); err != nil {
			return fmt.Errorf("failed to terminate connections to database %s: %w", name, err)
		}
	}

	query := fmt.Sprintf("DROP DATABASE %s", name)
	_, err = dbtx.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to drop database %s: %w", name, err)
	}

	return nil
}

func (d databaseRepo) GetOne(ctx context.Context, dbtx DBTX, name string) (*DatabaseModel, error) {
	var err error

	if name, err = sanitizeInput(name, SanitizeIdentifier); err != nil {
		return nil, fmt.Errorf("invalid database name %q: %w", name, err)
	}

	var model DatabaseModel
	err = dbtx.QueryRow(ctx, selectDatabaseQuery, name).Scan(
		&model.Name,
		&model.Owner,
		&model.Encoding,
		&model.Collation,
		&model.Ctype,
		&model.IsTemplate,
		&model.AllowConnections,
		&model.ConnectionLimit,
		&model.Tablespace,
		&model.Comment,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get database %s: %w", name, err)
	}

	return &model, nil
}

func (d databaseRepo) Update(ctx context.Context, dbtx DBTX, name string, params DatabaseUpdateParams) error {
	var err error

	if name, err = sanitizeInput(name, SanitizeIdentifier); err != nil {
		return fmt.Errorf("invalid database name %q: %w", name, err)
	}

	if params.Owner != nil {
		if *params.Owner, err = sanitizeInput(*params.Owner, SanitizeIdentifier); err != nil {
			return fmt.Errorf("invalid owner %q: %w", *params.Owner, err)
		}
		query := fmt.Sprintf("ALTER DATABASE %s OWNER TO %s", name, *params.Owner)
		if _, err := dbtx.Exec(ctx, query); err != nil {
			return fmt.Errorf("failed to change owner of database %s: %w", name, err)
		}
	}

	if params.ConnectionLimit != nil {
		query := fmt.Sprintf("ALTER DATABASE %s WITH CONNECTION LIMIT %d", name, *params.ConnectionLimit)
		if _, err := dbtx.Exec(ctx, query); err != nil {
			return fmt.Errorf("failed to update connection limit of database %s: %w", name, err)
		}
	}

	if params.AllowConnections != nil {
		query := fmt.Sprintf("ALTER DATABASE %s WITH ALLOW_CONNECTIONS %t", name, *params.AllowConnections)
		if _, err := dbtx.Exec(ctx, query); err != nil {
			return fmt.Errorf("failed to update allow_connections of database %s: %w", name, err)
		}
	}

	if params.IsTemplate != nil {
		query := fmt.Sprintf("ALTER DATABASE %s WITH IS_TEMPLATE %t", name, *params.IsTemplate)
		if _, err := dbtx.Exec(ctx, query); err != nil {
			return fmt.Errorf("failed to update is_template of database %s: %w", name, err)
		}
	}

	if params.Tablespace != nil {
		if *params.Tablespace, err = sanitizeInput(*params.Tablespace, SanitizeIdentifier); err != nil {
			return fmt.Errorf("invalid tablespace %q: %w", *params.Tablespace, err)
		}
		query := fmt.Sprintf("ALTER DATABASE %s SET TABLESPACE %s", name, *params.Tablespace)
		if _, err := dbtx.Exec(ctx, query); err != nil {
			return fmt.Errorf("failed to update tablespace of database %s: %w", name, err)
		}
	}

	if params.Comment != nil {
		var query string
		if *params.Comment == "" {
			query = fmt.Sprintf("COMMENT ON DATABASE %s IS NULL", name)
		} else {
			query = fmt.Sprintf("COMMENT ON DATABASE %s IS '%s'", name, *params.Comment)
		}
		if _, err := dbtx.Exec(ctx, query); err != nil {
			return fmt.Errorf("failed to update comment of database %s: %w", name, err)
		}
	}

	return nil
}

func (d databaseRepo) TerminateConnections(ctx context.Context, dbtx DBTX, name string) error {
	var err error

	if name, err = sanitizeInput(name, SanitizeIdentifier); err != nil {
		return fmt.Errorf("invalid database name %q: %w", name, err)
	}

	query := `
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE datname = $1
		  AND pid <> pg_backend_pid()
	`

	_, err = dbtx.Exec(ctx, query, name)
	if err != nil {
		return fmt.Errorf("failed to terminate connections to database %s: %w", name, err)
	}

	return nil
}
