package pgclient

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// GrantRepo defines operations for managing PostgreSQL privileges (GRANT/REVOKE).
type GrantRepo interface {
	Grant(ctx context.Context, dbtx DBTX, params GrantParams) error
	Revoke(ctx context.Context, dbtx DBTX, params RevokeParams) error
	GetGrant(ctx context.Context, dbtx DBTX, params GetGrantParams) (*GrantModel, error)
}

// NewGrantRepo creates a new grant repository instance.
func NewGrantRepo() GrantRepo {
	return &grantRepo{}
}

type grantRepo struct{}

// validPrivilegeNames is the set of recognised PostgreSQL privilege keywords.
var validPrivilegeNames = map[string]struct{}{
	"SELECT": {}, "INSERT": {}, "UPDATE": {}, "DELETE": {},
	"TRUNCATE": {}, "REFERENCES": {}, "TRIGGER": {},
	"USAGE": {}, "CONNECT": {}, "TEMPORARY": {}, "TEMP": {},
	"CREATE": {}, "EXECUTE": {}, "ALL": {},
}

// validatePrivileges returns an error if any privilege name is not in the
// recognised allowlist, preventing free-form strings from reaching SQL.
func validatePrivileges(privileges []string) error {
	for _, priv := range privileges {
		if _, ok := validPrivilegeNames[strings.ToUpper(priv)]; !ok {
			return fmt.Errorf("unrecognized privilege %q: must be one of SELECT, INSERT, UPDATE, DELETE, TRUNCATE, REFERENCES, TRIGGER, USAGE, CONNECT, TEMPORARY, CREATE, EXECUTE, or ALL", priv)
		}
	}
	return nil
}

// GrantModel represents privileges granted on a database object.
type GrantModel struct {
	ObjectType      string   `json:"object_type"`
	ObjectName      string   `json:"object_name"`
	Schema          string   `json:"schema,omitempty"`
	Role            string   `json:"role"`
	Privileges      []string `json:"privileges"`
	WithGrantOption bool     `json:"with_grant_option"`
}

// GrantParams contains parameters for granting privileges.
type GrantParams struct {
	ObjectType      string   `json:"object_type" validate:"required"`
	ObjectName      string   `json:"object_name" validate:"required"`
	Schema          string   `json:"schema,omitempty"`
	Role            string   `json:"role" validate:"required"`
	Privileges      []string `json:"privileges" validate:"required"`
	WithGrantOption bool     `json:"with_grant_option"`
}

// RevokeParams contains parameters for revoking privileges.
type RevokeParams struct {
	ObjectType string   `json:"object_type" validate:"required"`
	ObjectName string   `json:"object_name" validate:"required"`
	Schema     string   `json:"schema,omitempty"`
	Role       string   `json:"role" validate:"required"`
	Privileges []string `json:"privileges" validate:"required"`
	Cascade    bool     `json:"cascade"`
}

// GetGrantParams contains parameters for querying granted privileges.
type GetGrantParams struct {
	ObjectType string `json:"object_type" validate:"required"`
	ObjectName string `json:"object_name" validate:"required"`
	Schema     string `json:"schema,omitempty"`
	Role       string `json:"role" validate:"required"`
}

// Grant executes a GRANT statement to assign privileges.
func (r *grantRepo) Grant(ctx context.Context, dbtx DBTX, params GrantParams) error {
	if err := validateGrantParams(params); err != nil {
		return fmt.Errorf("invalid grant parameters: %w", err)
	}

	query := buildGrantQuery(params)
	_, err := dbtx.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to grant privileges: %w", err)
	}

	return nil
}

// Revoke executes a REVOKE statement to remove privileges.
func (r *grantRepo) Revoke(ctx context.Context, dbtx DBTX, params RevokeParams) error {
	if err := validateRevokeParams(params); err != nil {
		return fmt.Errorf("invalid revoke parameters: %w", err)
	}

	query := buildRevokeQuery(params)
	_, err := dbtx.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to revoke privileges: %w", err)
	}

	return nil
}

// GetGrant retrieves current privileges for a role on an object.
func (r *grantRepo) GetGrant(ctx context.Context, dbtx DBTX, params GetGrantParams) (*GrantModel, error) {
	if err := validateGetGrantParams(params); err != nil {
		return nil, fmt.Errorf("invalid get grant parameters: %w", err)
	}

	query, args := buildGetGrantQuery(params)
	rows, err := dbtx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query privileges: %w", err)
	}
	defer rows.Close()

	var privileges []string
	var withGrantOption bool

	for rows.Next() {
		var privilege pgtype.Text
		var grantOption pgtype.Text // Changed from pgtype.Bool to match query result

		if err := rows.Scan(&privilege, &grantOption); err != nil {
			return nil, fmt.Errorf("failed to scan privilege row: %w", err)
		}

		if privilege.Valid {
			privileges = append(privileges, privilege.String)
		}
		if grantOption.Valid && grantOption.String == "YES" {
			withGrantOption = true
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating privilege rows: %w", err)
	}

	if len(privileges) == 0 {
		// No privileges found — the grant does not exist (it may have been revoked outside Terraform).
		return nil, nil
	}

	return &GrantModel{
		ObjectType:      params.ObjectType,
		ObjectName:      params.ObjectName,
		Schema:          params.Schema,
		Role:            params.Role,
		Privileges:      privileges,
		WithGrantOption: withGrantOption,
	}, nil
}

// buildGrantQuery constructs a GRANT SQL statement.
func buildGrantQuery(params GrantParams) string {
	// Normalise to uppercase so the query is deterministic regardless of input casing.
	normalized := make([]string, len(params.Privileges))
	for i, p := range params.Privileges {
		normalized[i] = strings.ToUpper(p)
	}
	privList := strings.Join(normalized, ", ")
	objectRef := buildObjectReference(params.ObjectType, params.ObjectName, params.Schema)
	role := pgx.Identifier{params.Role}.Sanitize()

	var query string
	if len(normalized) == 1 && normalized[0] == "ALL" {
		query = fmt.Sprintf("GRANT ALL PRIVILEGES ON %s TO %s", objectRef, role)
	} else {
		query = fmt.Sprintf("GRANT %s ON %s TO %s", privList, objectRef, role)
	}

	if params.WithGrantOption {
		query += " WITH GRANT OPTION"
	}

	return query
}

// buildRevokeQuery constructs a REVOKE SQL statement.
func buildRevokeQuery(params RevokeParams) string {
	// Normalise to uppercase.
	normalized := make([]string, len(params.Privileges))
	for i, p := range params.Privileges {
		normalized[i] = strings.ToUpper(p)
	}
	privList := strings.Join(normalized, ", ")
	objectRef := buildObjectReference(params.ObjectType, params.ObjectName, params.Schema)
	role := pgx.Identifier{params.Role}.Sanitize()

	var query string
	if len(normalized) == 1 && normalized[0] == "ALL" {
		query = fmt.Sprintf("REVOKE ALL PRIVILEGES ON %s FROM %s", objectRef, role)
	} else {
		query = fmt.Sprintf("REVOKE %s ON %s FROM %s", privList, objectRef, role)
	}

	if params.Cascade {
		query += " CASCADE"
	}

	return query
}

// buildObjectReference constructs an object reference string (e.g., "DATABASE mydb", "TABLE schema.table").
// Only object types that are validated and supported (database, schema, table, sequence) are handled.
func buildObjectReference(objectType, objectName, schema string) string {
	objType := strings.ToUpper(objectType)
	name := pgx.Identifier{objectName}.Sanitize()

	switch objType {
	case "DATABASE":
		return fmt.Sprintf("DATABASE %s", name)
	case "SCHEMA":
		return fmt.Sprintf("SCHEMA %s", name)
	case "TABLE":
		if schema != "" {
			schemaIdent := pgx.Identifier{schema}.Sanitize()
			return fmt.Sprintf("TABLE %s.%s", schemaIdent, name)
		}
		return fmt.Sprintf("TABLE %s", name)
	case "SEQUENCE":
		if schema != "" {
			schemaIdent := pgx.Identifier{schema}.Sanitize()
			return fmt.Sprintf("SEQUENCE %s.%s", schemaIdent, name)
		}
		return fmt.Sprintf("SEQUENCE %s", name)
	default:
		// This should never be reached because object_type is validated at the schema level.
		return fmt.Sprintf("%s %s", objType, name)
	}
}

// buildGetGrantQuery constructs a query to retrieve privileges for a role on an object.
// Only object types that are validated and supported (database, schema, table, sequence) are handled.
func buildGetGrantQuery(params GetGrantParams) (string, []interface{}) {
	objType := strings.ToUpper(params.ObjectType)

	switch objType {
	case "DATABASE":
		// Use LATERAL aclexplode in FROM clause — set-returning functions are not
		// allowed in SELECT or JOIN ON clauses in modern PostgreSQL.
		return `
			SELECT acl.privilege_type,
			       CASE WHEN acl.is_grantable THEN 'YES' ELSE 'NO' END
			FROM pg_database d,
			     LATERAL aclexplode(d.datacl) AS acl
			WHERE d.datname = $1
			  AND acl.grantee = (SELECT oid FROM pg_roles WHERE rolname = $2)
		`, []interface{}{params.ObjectName, params.Role}

	case "SCHEMA":
		return `
			SELECT acl.privilege_type,
			       CASE WHEN acl.is_grantable THEN 'YES' ELSE 'NO' END
			FROM pg_namespace n,
			     LATERAL aclexplode(n.nspacl) AS acl
			WHERE n.nspname = $1
			  AND acl.grantee = (SELECT oid FROM pg_roles WHERE rolname = $2)
		`, []interface{}{params.ObjectName, params.Role}

	case "TABLE":
		schema := params.Schema
		if schema == "" {
			schema = "public"
		}
		return `
			SELECT privilege_type, is_grantable
			FROM information_schema.role_table_grants
			WHERE grantee = $1 AND table_schema = $2 AND table_name = $3
		`, []interface{}{params.Role, schema, params.ObjectName}

	case "SEQUENCE":
		schema := params.Schema
		if schema == "" {
			schema = "public"
		}
		return `
			SELECT acl.privilege_type,
			       CASE WHEN acl.is_grantable THEN 'YES' ELSE 'NO' END
			FROM pg_class c
			JOIN pg_namespace n ON n.oid = c.relnamespace
			CROSS JOIN LATERAL aclexplode(c.relacl) AS acl
			WHERE c.relkind = 'S'
			  AND n.nspname = $1
			  AND c.relname = $2
			  AND acl.grantee = (SELECT oid FROM pg_roles WHERE rolname = $3)
		`, []interface{}{schema, params.ObjectName, params.Role}

	default:
		// This should never be reached because object_type is validated at the schema level.
		// Return a query that will return no rows.
		return `SELECT NULL::text, NULL::text WHERE false`, nil
	}
}

// validateGrantParams validates grant parameters.
func validateGrantParams(params GrantParams) error {
	if params.ObjectType == "" {
		return fmt.Errorf("object_type is required")
	}
	if params.ObjectName == "" {
		return fmt.Errorf("object_name is required")
	}
	if params.Role == "" {
		return fmt.Errorf("role is required")
	}
	if len(params.Privileges) == 0 {
		return fmt.Errorf("at least one privilege is required")
	}
	if err := validatePrivileges(params.Privileges); err != nil {
		return err
	}
	return nil
}

// validateRevokeParams validates revoke parameters.
func validateRevokeParams(params RevokeParams) error {
	if params.ObjectType == "" {
		return fmt.Errorf("object_type is required")
	}
	if params.ObjectName == "" {
		return fmt.Errorf("object_name is required")
	}
	if params.Role == "" {
		return fmt.Errorf("role is required")
	}
	if len(params.Privileges) == 0 {
		return fmt.Errorf("at least one privilege is required")
	}
	if err := validatePrivileges(params.Privileges); err != nil {
		return err
	}
	return nil
}

// validateGetGrantParams validates get grant parameters.
func validateGetGrantParams(params GetGrantParams) error {
	if params.ObjectType == "" {
		return fmt.Errorf("object_type is required")
	}
	if params.ObjectName == "" {
		return fmt.Errorf("object_name is required")
	}
	if params.Role == "" {
		return fmt.Errorf("role is required")
	}
	return nil
}
