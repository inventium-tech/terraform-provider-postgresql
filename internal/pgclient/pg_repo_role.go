package pgclient

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"terraform-provider-postgresql/internal/helpers"

	"github.com/jackc/pgx/v5/pgtype"
)

// queries.
const (
	selectRoleQuery = `
	SELECT r.rolname                                    AS name,
       r.rolsuper                                       AS superuser,
       r.rolinherit                                     AS inherit,
       r.rolcreaterole                                  AS create_role,
       r.rolcreatedb                                    AS create_db,
       r.rolcanlogin                                    AS login,
       r.rolreplication                                 AS replication,
       r.rolbypassrls                                   AS bypass_rls,
       r.rolconnlimit                                   AS connection_limit,
       r.rolvaliduntil                                  AS valid_until,
       pg_catalog.shobj_description(r.oid, 'pg_authid') AS comment
	FROM pg_catalog.pg_roles r
	WHERE r.rolname = $1;`

	existsRoleQuery = `
	SELECT EXISTS (
		SELECT 1
		FROM pg_catalog.pg_roles r
		WHERE r.rolname = $1
	);`

	createRoleQuery = `
	CREATE ROLE %s
		WITH %s
		CONNECTION LIMIT %d
		VALID UNTIL '%s';`

	selectRoleMembershipQuery = `
SELECT granted.rolname AS role_name,
       m.admin_option  AS admin_option
FROM pg_catalog.pg_auth_members m
JOIN pg_catalog.pg_roles granted ON granted.oid = m.roleid
JOIN pg_catalog.pg_roles member ON member.oid = m.member
WHERE member.rolname = $1;`
)

type RoleRepo interface {
	Create(ctx context.Context, dbtx DBTX, params RoleCreateParams) error
	Exists(ctx context.Context, dbtx DBTX, name string) (bool, error)
	Drop(ctx context.Context, dbtx DBTX, name string) error
	GetOne(ctx context.Context, dbtx DBTX, name string) (*RoleModel, error)
	Update(ctx context.Context, dbtx DBTX, name string, params RoleUpdateParams) error
}

func NewRoleRepo() RoleRepo {
	return &roleRepo{}
}

type (
	RoleModel struct {
		Name            pgtype.Text `json:"name" validate:"required"`
		Superuser       pgtype.Bool `json:"superuser" validate:"boolean"`
		Inherit         pgtype.Bool `json:"inherit" validate:"boolean"`
		CreateRole      pgtype.Bool `json:"create_role" validate:"boolean"`
		CreateDB        pgtype.Bool `json:"create_db" validate:"boolean"`
		Login           pgtype.Bool `json:"login" validate:"boolean"`
		Replication     pgtype.Bool `json:"replication" validate:"boolean"`
		BypassRLS       pgtype.Bool `json:"bypass_rls" validate:"boolean"`
		ConnectionLimit pgtype.Int4 `json:"connection_limit" validate:"boolean"`
		ValidUntil      pgtype.Text `json:"valid_until" validate:"boolean"`
		Comment         pgtype.Text `json:"comment" validate:"boolean"`
		Roles           []string    `json:"roles"`
		AdminRoles      []string    `json:"admin_roles"`
	}
	RoleCreateParams struct {
		Name            string   `json:"name" validate:"required"`
		Password        *string  `json:"password"`
		Superuser       bool     `json:"superuser"`
		Inherit         bool     `json:"inherit"`
		CreateRole      bool     `json:"create_role"`
		CreateDB        bool     `json:"create_db"`
		Login           bool     `json:"login"`
		Replication     bool     `json:"replication"`
		BypassRLS       bool     `json:"bypass_rls"`
		ConnectionLimit int32    `json:"connection_limit"`
		ValidUntil      string   `json:"valid_until"`
		Comment         string   `json:"comment"`
		InRole          []string `json:"in_role"`
		Roles           []string `json:"roles"`
		AdminRoles      []string `json:"admin_roles"`
	}
	RoleUpdateParams struct {
		Name            *string  `json:"name"`
		Password        *string  `json:"password"`
		Superuser       *bool    `json:"superuser"`
		Inherit         *bool    `json:"inherit"`
		CreateRole      *bool    `json:"create_role"`
		CreateDB        *bool    `json:"create_db"`
		Login           *bool    `json:"login"`
		Replication     *bool    `json:"replication"`
		BypassRLS       *bool    `json:"bypass_rls"`
		ConnectionLimit *int32   `json:"connection_limit"`
		ValidUntil      *string  `json:"valid_until"`
		Comment         *string  `json:"comment"`
		Roles           []string `json:"roles"`
		AdminRoles      []string `json:"admin_roles"`
	}
)

type roleRepo struct{}

func (r *roleRepo) Create(ctx context.Context, dbtx DBTX, params RoleCreateParams) error {
	err := helpers.GetSafeValidator().Struct(params)
	if err != nil {
		return err
	}

	roleName, err := sanitizeInput(params.Name, SanitizeIdentifier)
	if err != nil {
		return fmt.Errorf("invalid role name %q. error: %w", params.Name, err)
	}

	booleanOpts := []string{
		r.useBooleanOption("SUPERUSER", &params.Superuser),
		r.useBooleanOption("INHERIT", &params.Inherit),
		r.useBooleanOption("CREATEROLE", &params.CreateRole),
		r.useBooleanOption("CREATEDB", &params.CreateDB),
		r.useBooleanOption("LOGIN", &params.Login),
		r.useBooleanOption("REPLICATION", &params.Replication),
		r.useBooleanOption("BYPASSRLS", &params.BypassRLS),
	}

	if params.Password != nil && *params.Password != "" {
		booleanOpts = append(booleanOpts, fmt.Sprintf("PASSWORD '%s'", *params.Password))
	}

	var connectionLimitOption int32 = -1
	if params.ConnectionLimit != 0 {
		connectionLimitOption = params.ConnectionLimit
	}

	validUntilOption := "infinity"
	if params.ValidUntil != "" {
		validUntilOption = params.ValidUntil
	}

	booleanOpts = slices.DeleteFunc(booleanOpts, func(v string) bool { return v == "" })
	booleanOptsString := strings.Join(booleanOpts, " ")

	query := fmt.Sprintf(createRoleQuery, roleName, booleanOptsString, connectionLimitOption, validUntilOption)

	_, err = dbtx.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create role %s: %w", params.Name, err)
	}

	err = CreateComment(ctx, dbtx, PgObjRole, params.Name, params.Comment)
	if err != nil {
		return err
	}

	allRoles := append([]string{}, params.InRole...)
	allRoles = append(allRoles, params.Roles...)

	err = r.syncMembership(ctx, dbtx, params.Name, allRoles, params.AdminRoles)
	if err != nil {
		return err
	}

	return nil
}

func (r *roleRepo) syncMembership(ctx context.Context, dbtx DBTX, member string, roles []string, adminRoles []string) error {
	if roles == nil && adminRoles == nil {
		return nil
	}

	memberName, err := sanitizeInput(member, SanitizeIdentifier)
	if err != nil {
		return fmt.Errorf("invalid member role name %q. error: %w", member, err)
	}

	desired, err := buildDesiredMembershipMap(roles, adminRoles)
	if err != nil {
		return err
	}

	current, err := r.getMembership(ctx, dbtx, memberName)
	if err != nil {
		return err
	}

	for roleName, currentAdmin := range current {
		desiredAdmin, ok := desired[roleName]
		switch {
		case !ok:
			if err := r.revokeMembership(ctx, dbtx, roleName, memberName); err != nil {
				return err
			}
		case desiredAdmin != currentAdmin:
			if err := r.revokeMembership(ctx, dbtx, roleName, memberName); err != nil {
				return err
			}

			if err := r.grantMembership(ctx, dbtx, roleName, memberName, desiredAdmin); err != nil {
				return err
			}
		}
	}

	for roleName, desiredAdmin := range desired {
		if _, ok := current[roleName]; ok {
			continue
		}

		if err := r.grantMembership(ctx, dbtx, roleName, memberName, desiredAdmin); err != nil {
			return err
		}
	}

	return nil
}

func (r *roleRepo) grantMembership(ctx context.Context, dbtx DBTX, roleName, memberName string, adminOption bool) error {
	query := fmt.Sprintf("GRANT %s TO %s", roleName, memberName)
	if adminOption {
		query += " WITH ADMIN OPTION"
	}

	_, err := dbtx.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to grant role %s to %s: %w", roleName, memberName, err)
	}

	return nil
}

func (r *roleRepo) revokeMembership(ctx context.Context, dbtx DBTX, roleName, memberName string) error {
	query := fmt.Sprintf("REVOKE %s FROM %s", roleName, memberName)
	_, err := dbtx.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to revoke role %s from %s: %w", roleName, memberName, err)
	}

	return nil
}

func (r *roleRepo) getMembership(ctx context.Context, dbtx DBTX, memberName string) (map[string]bool, error) {
	memberships := make(map[string]bool)

	rows, err := dbtx.Query(ctx, selectRoleMembershipQuery, memberName)
	if err != nil {
		return nil, fmt.Errorf("failed to list memberships for %s: %w", memberName, err)
	}

	defer rows.Close()

	for rows.Next() {
		var roleName string
		var adminOption bool

		if err := rows.Scan(&roleName, &adminOption); err != nil {
			return nil, fmt.Errorf("failed to scan membership for %s: %w", memberName, err)
		}

		memberships[roleName] = adminOption
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return memberships, nil
}

func buildDesiredMembershipMap(roles []string, adminRoles []string) (map[string]bool, error) {
	desired := make(map[string]bool)

	for _, roleName := range roles {
		if roleName == "" {
			continue
		}

		sanitized, err := sanitizeInput(roleName, SanitizeIdentifier)
		if err != nil {
			return nil, fmt.Errorf("invalid role membership %q. error: %w", roleName, err)
		}

		desired[sanitized] = false
	}

	for _, roleName := range adminRoles {
		if roleName == "" {
			continue
		}

		sanitized, err := sanitizeInput(roleName, SanitizeIdentifier)
		if err != nil {
			return nil, fmt.Errorf("invalid admin membership %q. error: %w", roleName, err)
		}

		desired[sanitized] = true
	}

	return desired, nil
}

func (r *roleRepo) Update(ctx context.Context, dbtx DBTX, name string, params RoleUpdateParams) error {
	booleanOpts := []string{
		r.useBooleanOption("SUPERUSER", params.Superuser),
		r.useBooleanOption("INHERIT", params.Inherit),
		r.useBooleanOption("CREATEROLE", params.CreateRole),
		r.useBooleanOption("CREATEDB", params.CreateDB),
		r.useBooleanOption("LOGIN", params.Login),
		r.useBooleanOption("REPLICATION", params.Replication),
		r.useBooleanOption("BYPASSRLS", params.BypassRLS),
	}
	booleanOpts = slices.DeleteFunc(booleanOpts, func(v string) bool { return v == "" })

	configOpts := make([]string, 0)

	if params.ConnectionLimit != nil {
		configOpts = append(configOpts, fmt.Sprintf("CONNECTION LIMIT %d", *params.ConnectionLimit))
	}

	if params.ValidUntil != nil {
		configOpts = append(configOpts, fmt.Sprintf("VALID UNTIL '%s'", *params.ValidUntil))
	}

	if len(booleanOpts) > 0 || len(configOpts) > 0 {
		booleanOptsString := strings.Join(booleanOpts, " ")
		configOptsString := strings.Join(configOpts, " ")

		queryParts := []string{
			fmt.Sprintf("ALTER ROLE %s WITH", name),
			booleanOptsString,
			configOptsString,
		}
		query := strings.Join(queryParts, "\n\t")
		_, err := dbtx.Exec(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to update role %s: %w", name, err)
		}
	}

	if params.Comment != nil {
		err := CreateComment(ctx, dbtx, PgObjRole, name, *params.Comment)
		if err != nil {
			return err
		}
	}

	targetName := name

	if params.Name != nil && *params.Name != "" {
		roleName, err := sanitizeInput(*params.Name, SanitizeIdentifier)
		if err != nil {
			return fmt.Errorf("invalid role name %q. error: %w", *params.Name, err)
		}
		err = RenameObject(ctx, dbtx, PgObjRole, name, roleName)
		if err != nil {
			return err
		}

		targetName = roleName
	}

	if params.Roles != nil || params.AdminRoles != nil {
		if err := r.syncMembership(ctx, dbtx, targetName, params.Roles, params.AdminRoles); err != nil {
			return err
		}
	}

	return nil
}

func (r *roleRepo) Exists(ctx context.Context, dbtx DBTX, name string) (bool, error) {
	var err error

	if name, err = sanitizeInput(name, SanitizeIdentifier); err != nil {
		return false, fmt.Errorf("invalid role name %q. error: %w", name, err)
	}

	var exists bool
	err = dbtx.QueryRow(ctx, existsRoleQuery, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if role %s exists: %w", name, err)
	}

	return exists, nil
}

func (r *roleRepo) GetOne(ctx context.Context, dbtx DBTX, name string) (*RoleModel, error) {
	var err error

	if name, err = sanitizeInput(name, SanitizeIdentifier); err != nil {
		return nil, fmt.Errorf("invalid role name %q. error: %w", name, err)
	}

	// Define a structure to hold the result
	var model RoleModel

	row := dbtx.QueryRow(ctx, selectRoleQuery, name)
	err = row.Scan(
		&model.Name,
		&model.Superuser,
		&model.Inherit,
		&model.CreateRole,
		&model.CreateDB,
		&model.Login,
		&model.Replication,
		&model.BypassRLS,
		&model.ConnectionLimit,
		&model.ValidUntil,
		&model.Comment,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get role %s: %w", name, err)
	}

	memberships, err := r.getMembership(ctx, dbtx, model.Name.String)
	if err != nil {
		return nil, err
	}

	for roleName, adminOption := range memberships {
		if adminOption {
			model.AdminRoles = append(model.AdminRoles, roleName)
			continue
		}

		model.Roles = append(model.Roles, roleName)
	}

	slices.Sort(model.Roles)
	slices.Sort(model.AdminRoles)

	return &model, nil
}

func (r *roleRepo) Drop(ctx context.Context, dbtx DBTX, name string) error {
	err := DropObject(ctx, dbtx, PgObjRole, name)
	if err != nil {
		return fmt.Errorf("failed to drop role with name %s: %w", name, err)
	}

	return nil
}

func (r *roleRepo) useBooleanOption(option string, enabled *bool) string {
	if enabled == nil {
		return ""
	}

	var prefix string
	if !*enabled {
		prefix = "NO"
	}

	return fmt.Sprintf("%s%s", prefix, option)

}
