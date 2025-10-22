package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"terraform-provider-postgresql/internal/pgclient"
	"terraform-provider-postgresql/internal/provider/validators"
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

	conn, err := r.pgClient.GetConnection(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	tx, err := conn.Begin(ctx)
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
	}

	err = r.roleRepo.Create(ctx, conn, params)
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFCreateAction, PGRole), err.Error())
		return
	}

	err = tx.Commit(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrCommitPgTransaction, err.Error())
		return
	}

	model.Id = model.Name
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
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFReadAction, PGRole), err.Error())
		return
	}

	if pgRole == nil {
		res.Diagnostics.AddAttributeError(
			path.Root("id"),
			fmt.Sprintf(msgErrorPgObjectNotFund, PGRole),
			fmt.Sprintf(msgErrorPgObjectNotFundDetail, PGRole, model.Id.ValueString()),
		)
		return
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
	model.Comment = types.StringValue(pgRole.Comment.String)

	setLastUpdatedFieldValue(&model.LastUpdated)

	res.Diagnostics.Append(res.State.Set(ctx, &model)...)
}

func (r *postgresqlRoleResource) Update(ctx context.Context, req resource.UpdateRequest, res *resource.UpdateResponse) {
	tflog.Debug(ctx, fmt.Sprintf("updating '%s' resource", r.resName))

	var planModel postgresqlRoleModel
	var stateModel postgresqlRoleModel

	var planPasswd types.String
	res.Diagnostics.Append(req.Plan.Get(ctx, &planModel)...)
	res.Diagnostics.Append(req.State.Get(ctx, &stateModel)...)
	res.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("password_wo"), &planPasswd)...)

	if res.Diagnostics.HasError() {
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

	conn, err := r.pgClient.GetConnection(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrStartPgTransaction, err.Error())
		return
	}

	defer pgclient.DeferredRollback(ctx, tx)

	updateParams := planModel.buildPgRoleUpdateParams(&stateModel)
	updateParams.Password = planPasswd.ValueStringPointer()

	err = r.roleRepo.Update(ctx, conn, stateModel.Name.ValueString(), updateParams)
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFUpdateAction, PGRole), err.Error())
		return
	}

	err = tx.Commit(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrCommitPgTransaction, err.Error())
		return
	}

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
