package provider

import (
	"context"
	"errors"
	"fmt"
	"terraform-provider-postgresql/internal/helpers"
	"terraform-provider-postgresql/internal/pgclient"
	"terraform-provider-postgresql/internal/provider/validators"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jackc/pgx/v5"
)

var (
	_ resource.Resource                = &postgresqlRoleResource{}
	_ resource.ResourceWithConfigure   = &postgresqlRoleResource{}
	_ resource.ResourceWithImportState = &postgresqlRoleResource{}
)

func NewPostgresqlRoleResource() resource.Resource {
	return &postgresqlRoleResource{
		resName: "postgresql_role",
	}
}

type postgresqlRoleResource struct {
	resName  string
	pgClient pgclient.PostgresqlClient
	roleRepo pgclient.RoleRepo
}

func (r *postgresqlRoleResource) Metadata(_ context.Context, _ resource.MetadataRequest, res *resource.MetadataResponse) {
	res.TypeName = r.resName
}

func (r *postgresqlRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, res *resource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "Creates a Postgresql role. [Postgresql documentation](https://www.postgresql.org/docs/current/sql-createrole.html)",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for the role",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp of the resource's last modification",
			},
			"name": schema.StringAttribute{
				Description: "The name of the Postgresql role.",
				Required:    true,
				Validators: []validator.String{
					validators.PostgresqlObjectName(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"password_wo": schema.StringAttribute{
				Description: "The password of the Postgresql role. This is a write-only attribute.",
				Optional:    true,
				Sensitive:   true,
				WriteOnly:   true,
			},
			"password_wo_version": schema.Int32Attribute{
				Description: "Increment this value to force a password update.",
				Optional:    true,
				Computed:    true,
				Default:     int32default.StaticInt32(0),
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int32{
					int32validator.AtLeast(1),
				},
			},
			"superuser": schema.BoolAttribute{
				Description: "Determines whether the role is a superuser who can override all access restrictions within the database. Default is `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"inherit": schema.BoolAttribute{
				Description: "Determines whether the role inherits the privileges of roles it is a member of. Default is `true`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"create_role": schema.BoolAttribute{
				Description: "Determines whether the role can create new roles. Default is `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"create_db": schema.BoolAttribute{
				Description: "Determines whether the role can create new databases. Default is `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"login": schema.BoolAttribute{
				Description: "Determines whether the role can log in. Default is `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"replication": schema.BoolAttribute{
				Description: "Determines whether the role can initiate streaming replication or put the system in and out of backup mode. Default is `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"bypass_rls": schema.BoolAttribute{
				Description: "Determines whether the role bypasses every row-level security (RLS) policy. Default is `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"connection_limit": schema.Int32Attribute{
				Description: "The maximum number of concurrent connections the role can make. -1 means no limit. Default is `-1`.",
				Optional:    true,
				Computed:    true,
				Default:     int32default.StaticInt32(-1),
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int32{
					int32validator.AtLeast(-1),
				},
			},
			"in_role": schema.SetAttribute{
				Description: "Roles to grant membership during creation (one-time). Changes force recreation.",
				ElementType: types.StringType,
				Optional:    true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
					setplanmodifier.RequiresReplace(),
				},
			},
			"role": schema.SetAttribute{
				Description: "Roles this role belongs to (membership without admin option).",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"admin": schema.SetAttribute{
				Description: "Roles this role can administer (membership with admin option).",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"valid_until": schema.StringAttribute{
				Description: "The date and time after which the role's password is no longer valid. Default is 'infinity'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("infinity"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"comment": schema.StringAttribute{
				Optional:    true,
				Description: "Comment associated with the role",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *postgresqlRoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	tflog.Debug(ctx, fmt.Sprintf("configuring '%s' resource", r.resName))

	if req.ProviderData == nil {
		return
	}

	res.Diagnostics.Append(parsePgClientFromRequest(req, &r.pgClient))
	if res.Diagnostics.HasError() {
		return
	}

	r.roleRepo = pgclient.NewRoleRepo()

}

func (r *postgresqlRoleResource) Create(ctx context.Context, req resource.CreateRequest, res *resource.CreateResponse) {
	tflog.Debug(ctx, fmt.Sprintf("creating '%s' resource", r.resName))

	var model postgresqlRoleModel

	var passwdAttr types.String
	res.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	res.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("password_wo"), &passwdAttr)...)

	if res.Diagnostics.HasError() {
		return
	}

	inRole := make([]string, 0)
	roleMembership := make([]string, 0)
	adminMembership := make([]string, 0)

	res.Diagnostics.Append(parseSetIntoSlice(ctx, &inRole, model.InRole)...)
	res.Diagnostics.Append(parseSetIntoSlice(ctx, &roleMembership, model.Role)...)
	res.Diagnostics.Append(parseSetIntoSlice(ctx, &adminMembership, model.Admin)...)

	if res.Diagnostics.HasError() {
		return
	}

	// Validate that role and admin don't have overlapping memberships
	if overlaps := helpers.SliceIntersection(roleMembership, adminMembership); len(overlaps) > 0 {
		res.Diagnostics.AddError(
			"Conflicting role membership",
			fmt.Sprintf("The following roles cannot be in both 'role' and 'admin' attributes: %v", overlaps),
		)
		return
	}

	pool, err := r.pgClient.GetPool(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrStartPgTransaction, err.Error())
		return
	}

	defer pgclient.DeferredRollback(ctx, tx)

	params := pgclient.RoleCreateParams{
		Name:            model.Name.ValueString(),
		Password:        passwdAttr.ValueStringPointer(),
		Superuser:       model.Superuser.ValueBool(),
		Inherit:         model.Inherit.ValueBool(),
		CreateRole:      model.CreateRole.ValueBool(),
		CreateDB:        model.CreateDB.ValueBool(),
		Login:           model.Login.ValueBool(),
		Replication:     model.Replication.ValueBool(),
		BypassRLS:       model.BypassRLS.ValueBool(),
		ConnectionLimit: model.ConnectionLimit.ValueInt32(),
		ValidUntil:      model.ValidUntil.ValueString(),
		Comment:         model.Comment.ValueString(),
		InRole:          inRole,
		Roles:           roleMembership,
		AdminRoles:      adminMembership,
	}

	err = r.roleRepo.Create(ctx, tx, params)
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFCreateAction, PGRole), err.Error())
		return
	}

	err = tx.Commit(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrCommitPgTransaction, err.Error())
		return
	}

	// Read back the role to get the actual state including computed membership fields
	conn, err := r.pgClient.GetConnection(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	pgRole, err := r.roleRepo.GetOne(ctx, conn, model.Name.ValueString())
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFReadAction, PGRole), err.Error())
		return
	}

	// Filter out inRole memberships from the role list for the "role" attribute
	filteredRoles := pgRole.Roles
	if len(inRole) > 0 {
		filteredRoles = helpers.SliceDifference(pgRole.Roles, inRole)
	}

	model.Id = model.Name
	// Set membership fields based on whether they were configured
	if !model.InRole.IsNull() && !model.InRole.IsUnknown() {
		res.Diagnostics.Append(parseSliceIntoStringSet(ctx, &model.InRole, inRole)...)
	}
	// Role and Admin are computed, so set them from actual database state
	res.Diagnostics.Append(parseSliceIntoStringSet(ctx, &model.Role, filteredRoles)...)
	res.Diagnostics.Append(parseSliceIntoStringSet(ctx, &model.Admin, pgRole.AdminRoles)...)

	if res.Diagnostics.HasError() {
		return
	}

	setLastUpdatedFieldValue(&model.LastUpdated)
	res.Diagnostics.Append(res.State.Set(ctx, &model)...)
}

func (r *postgresqlRoleResource) Read(ctx context.Context, req resource.ReadRequest, res *resource.ReadResponse) {
	tflog.Debug(ctx, fmt.Sprintf("reading '%s' resource", r.resName))

	var model postgresqlRoleModel

	// retrieve values from the state
	res.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	if model.Id.IsUnknown() || model.Id.IsNull() {
		res.Diagnostics.AddError(msgErrorMissingResId, fmt.Sprintf(msgRequiredField, "Id", r.resName))
		return
	}

	conn, err := r.pgClient.GetConnection(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	pgRole, err := r.roleRepo.GetOne(ctx, conn, model.Id.ValueString())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Role was deleted outside of Terraform — remove from state.
			res.State.RemoveResource(ctx)
			return
		}
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFReadAction, PGRole), err.Error())
		return
	}

	var inRoleMembership []string
	res.Diagnostics.Append(parseSetIntoSlice(ctx, &inRoleMembership, model.InRole)...)
	if res.Diagnostics.HasError() {
		return
	}

	// Filter out inRole memberships from the role list
	filteredRoles := pgRole.Roles
	if len(inRoleMembership) > 0 {
		filteredRoles = helpers.SliceDifference(pgRole.Roles, inRoleMembership)
	}

	model.Id = types.StringValue(pgRole.Name.String)
	model.Name = types.StringValue(pgRole.Name.String)
	model.Superuser = types.BoolValue(pgRole.Superuser.Bool)
	model.Inherit = types.BoolValue(pgRole.Inherit.Bool)
	model.CreateRole = types.BoolValue(pgRole.CreateRole.Bool)
	model.CreateDB = types.BoolValue(pgRole.CreateDB.Bool)
	model.Login = types.BoolValue(pgRole.Login.Bool)
	model.Replication = types.BoolValue(pgRole.Replication.Bool)
	model.BypassRLS = types.BoolValue(pgRole.BypassRLS.Bool)
	model.ConnectionLimit = types.Int32Value(pgRole.ConnectionLimit.Int32)
	model.ValidUntil = types.StringValue(pgRole.ValidUntil.String)
	if pgRole.Comment.Valid {
		model.Comment = types.StringValue(pgRole.Comment.String)
	} else {
		model.Comment = types.StringNull()
	}
	res.Diagnostics.Append(parseSliceIntoStringSet(ctx, &model.Role, filteredRoles)...)
	res.Diagnostics.Append(parseSliceIntoStringSet(ctx, &model.Admin, pgRole.AdminRoles)...)

	res.Diagnostics.Append(res.State.Set(ctx, &model)...)
}

func (r *postgresqlRoleResource) Update(ctx context.Context, req resource.UpdateRequest, res *resource.UpdateResponse) {
	tflog.Debug(ctx, fmt.Sprintf("updating '%s' resource", r.resName))

	var (
		planModel      postgresqlRoleModel
		stateModel     postgresqlRoleModel
		planRoles      []string
		planAdminRoles []string
	)

	var planPasswd types.String
	res.Diagnostics.Append(req.Plan.Get(ctx, &planModel)...)
	res.Diagnostics.Append(req.State.Get(ctx, &stateModel)...)
	res.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("password_wo"), &planPasswd)...)

	if res.Diagnostics.HasError() {
		return
	}

	res.Diagnostics.Append(parseSetIntoSlice(ctx, &planRoles, planModel.Role)...)
	res.Diagnostics.Append(parseSetIntoSlice(ctx, &planAdminRoles, planModel.Admin)...)

	if res.Diagnostics.HasError() {
		return
	}

	// Validate that role and admin don't have overlapping memberships
	if overlaps := helpers.SliceIntersection(planRoles, planAdminRoles); len(overlaps) > 0 {
		res.Diagnostics.AddError(
			"Conflicting role membership",
			fmt.Sprintf("The following roles cannot be in both 'role' and 'admin' attributes: %v", overlaps),
		)
		return
	}

	// If the password_wo is empty but password_wo_version has changed, throw an error
	if planPasswd.IsNull() && !stateModel.PasswordVersion.Equal(planModel.PasswordVersion) {
		res.Diagnostics.AddError(
			"The attribute 'password_wo' is required when 'password_wo_version' changes",
			"You must provide a password when changing 'password_wo_version'",
		)
		return
	}

	pool, err := r.pgClient.GetPool(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrStartPgTransaction, err.Error())
		return
	}

	defer pgclient.DeferredRollback(ctx, tx)

	updateParams := planModel.buildPgRoleUpdateParams(&stateModel)
	updateParams.Password = planPasswd.ValueStringPointer()

	// Parse in_role from state: these memberships were set at creation and are immutable
	// (RequiresReplace). They must be included in every syncMembership call so they are
	// never accidentally revoked when role or admin changes.
	stateInRole := make([]string, 0)
	res.Diagnostics.Append(parseSetIntoSlice(ctx, &stateInRole, stateModel.InRole)...)
	if res.Diagnostics.HasError() {
		return
	}

	roleChanged := !planModel.Role.IsUnknown() && !planModel.Role.Equal(stateModel.Role)
	adminChanged := !planModel.Admin.IsUnknown() && !planModel.Admin.Equal(stateModel.Admin)

	if roleChanged || adminChanged {
		// Combine plan roles with immutable in_role memberships so syncMembership
		// sees the full desired state and does not revoke in_role memberships.
		updateParams.Roles = append(planRoles, stateInRole...)
		updateParams.AdminRoles = planAdminRoles
	}

	err = r.roleRepo.Update(ctx, tx, stateModel.Name.ValueString(), updateParams)
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFUpdateAction, PGRole), err.Error())
		return
	}

	err = tx.Commit(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrCommitPgTransaction, err.Error())
		return
	}

	if planModel.Role.IsUnknown() {
		planModel.Role = stateModel.Role
	} else {
		res.Diagnostics.Append(parseSliceIntoStringSet(ctx, &planModel.Role, planRoles)...)
	}

	if planModel.Admin.IsUnknown() {
		planModel.Admin = stateModel.Admin
	} else {
		res.Diagnostics.Append(parseSliceIntoStringSet(ctx, &planModel.Admin, planAdminRoles)...)
	}

	// InRole is immutable (has RequiresReplace), so it should already be in planModel
	// No need to copy from state

	planModel.Id = planModel.Name
	setLastUpdatedFieldValue(&planModel.LastUpdated)
	res.Diagnostics.Append(res.State.Set(ctx, &planModel)...)
}

func (r *postgresqlRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, res *resource.DeleteResponse) {
	tflog.Debug(ctx, fmt.Sprintf("deleting '%s' resource", r.resName))

	var model postgresqlRoleModel

	// retrieve values from the state
	res.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	if model.Id.IsUnknown() || model.Id.IsNull() {
		res.Diagnostics.AddError(msgErrorMissingResId, fmt.Sprintf(msgRequiredField, "Id", r.resName))
		return
	}

	conn, err := r.pgClient.GetConnection(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	err = r.roleRepo.Drop(ctx, conn, model.Id.ValueString())
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFDeleteAction, PGRole), err.Error())
	}
}

func (r *postgresqlRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, res *resource.ImportStateResponse) {
	tflog.Debug(ctx, fmt.Sprintf("importing '%s' resource", r.resName))
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, res)
}
