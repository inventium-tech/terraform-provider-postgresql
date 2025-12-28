package provider

import (
	"context"
	"fmt"
	"time"

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &postgresqlDatabaseResource{}
	_ resource.ResourceWithConfigure   = &postgresqlDatabaseResource{}
	_ resource.ResourceWithImportState = &postgresqlDatabaseResource{}
)

func NewPostgresqlDatabaseResource() resource.Resource {
	return &postgresqlDatabaseResource{
		resName: "postgresql_database",
	}
}

type postgresqlDatabaseResource struct {
	resName      string
	pgClient     pgclient.PostgresqlClient
	databaseRepo pgclient.DatabaseRepo
}

func (r *postgresqlDatabaseResource) Metadata(_ context.Context, _ resource.MetadataRequest, res *resource.MetadataResponse) {
	res.TypeName = r.resName
}

func (r *postgresqlDatabaseResource) Schema(_ context.Context, _ resource.SchemaRequest, res *resource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "Creates a PostgreSQL database. [PostgreSQL documentation](https://www.postgresql.org/docs/current/sql-createdatabase.html)",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for the database",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp of the resource's last modification",
			},
			"name": schema.StringAttribute{
				Description: "The name of the database.",
				Required:    true,
				Validators: []validator.String{
					validators.PostgresqlObjectName(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"owner": schema.StringAttribute{
				Description: "The role that owns the database. Defaults to the user executing the command.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					validators.PostgresqlObjectName(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"encoding": schema.StringAttribute{
				Description: "Character set encoding to use in the new database. Default is the encoding of the template database.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"collation": schema.StringAttribute{
				Description: "Collation order (LC_COLLATE) to use in the new database. Default is the collation of the template database.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ctype": schema.StringAttribute{
				Description: "Character classification (LC_CTYPE) to use in the new database. Default is the ctype of the template database.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"template": schema.StringAttribute{
				Description: "The name of the template database from which to create the new database. Default is 'template1'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("template1"),
				Validators: []validator.String{
					validators.PostgresqlObjectName(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"connection_limit": schema.Int32Attribute{
				Description: "Maximum concurrent connections to the database. -1 means no limit. Default is -1.",
				Optional:    true,
				Computed:    true,
				Default:     int32default.StaticInt32(-1),
				Validators: []validator.Int32{
					int32validator.AtLeast(-1),
				},
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.UseStateForUnknown(),
				},
			},
			"allow_connections": schema.BoolAttribute{
				Description: "If false, no one can connect to this database. Default is true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"is_template": schema.BoolAttribute{
				Description: "If true, this database can be cloned by any user with CREATEDB privileges. Default is false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"tablespace": schema.StringAttribute{
				Description: "The tablespace for the database. Default is the template database's tablespace.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					validators.PostgresqlObjectName(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"comment": schema.StringAttribute{
				Description: "A comment for the database.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"force_drop": schema.BoolAttribute{
				Description: "If true, terminates all connections to the database before dropping it. Default is false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *postgresqlDatabaseResource) Configure(_ context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	pgClient, ok := req.ProviderData.(pgclient.PostgresqlClient)
	if !ok {
		res.Diagnostics.AddError(
			msgErrInvalidProviderData,
			fmt.Sprintf("Expected pgclient.PostgresqlClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.pgClient = pgClient
	r.databaseRepo = pgclient.NewDatabaseRepo()
}

func (r *postgresqlDatabaseResource) Create(ctx context.Context, req resource.CreateRequest, res *resource.CreateResponse) {
	tflog.Debug(ctx, fmt.Sprintf("creating '%s' resource", r.resName))

	var model postgresqlDatabaseModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	conn, err := r.pgClient.GetConnection(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	params := pgclient.DatabaseCreateParams{
		Name:             model.Name.ValueString(),
		Owner:            model.Owner.ValueString(),
		Encoding:         model.Encoding.ValueString(),
		Collation:        model.Collation.ValueString(),
		Ctype:            model.Ctype.ValueString(),
		Template:         model.Template.ValueString(),
		ConnectionLimit:  model.ConnectionLimit.ValueInt32(),
		AllowConnections: model.AllowConnections.ValueBoolPointer(),
		IsTemplate:       model.IsTemplate.ValueBoolPointer(),
		Tablespace:       model.Tablespace.ValueString(),
		Comment:          model.Comment.ValueString(),
	}

	err = r.databaseRepo.Create(ctx, conn, params)
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFCreateAction, PGDatabase), err.Error())
		return
	}

	dbModel, err := r.databaseRepo.GetOne(ctx, conn, model.Name.ValueString())
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFReadAction, PGDatabase), err.Error())
		return
	}

	model.Id = types.StringValue(dbModel.Name.String)
	model.Name = types.StringValue(dbModel.Name.String)
	model.Owner = types.StringValue(dbModel.Owner.String)
	model.Encoding = types.StringValue(dbModel.Encoding.String)
	model.Collation = types.StringValue(dbModel.Collation.String)
	model.Ctype = types.StringValue(dbModel.Ctype.String)
	model.IsTemplate = types.BoolValue(dbModel.IsTemplate.Bool)
	model.AllowConnections = types.BoolValue(dbModel.AllowConnections.Bool)
	model.ConnectionLimit = types.Int32Value(dbModel.ConnectionLimit.Int32)
	model.Tablespace = types.StringValue(dbModel.Tablespace.String)
	model.Comment = types.StringValue(dbModel.Comment.String)
	model.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	res.Diagnostics.Append(res.State.Set(ctx, &model)...)
}

func (r *postgresqlDatabaseResource) Read(ctx context.Context, req resource.ReadRequest, res *resource.ReadResponse) {
	tflog.Debug(ctx, fmt.Sprintf("reading '%s' resource", r.resName))

	var model postgresqlDatabaseModel
	res.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	conn, err := r.pgClient.GetConnection(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	exists, err := r.databaseRepo.Exists(ctx, conn, model.Name.ValueString())
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFReadAction, PGDatabase), err.Error())
		return
	}

	if !exists {
		res.State.RemoveResource(ctx)
		return
	}

	dbModel, err := r.databaseRepo.GetOne(ctx, conn, model.Name.ValueString())
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFReadAction, PGDatabase), err.Error())
		return
	}

	model.Name = types.StringValue(dbModel.Name.String)
	model.Owner = types.StringValue(dbModel.Owner.String)
	model.Encoding = types.StringValue(dbModel.Encoding.String)
	model.Collation = types.StringValue(dbModel.Collation.String)
	model.Ctype = types.StringValue(dbModel.Ctype.String)
	model.IsTemplate = types.BoolValue(dbModel.IsTemplate.Bool)
	model.AllowConnections = types.BoolValue(dbModel.AllowConnections.Bool)
	model.ConnectionLimit = types.Int32Value(dbModel.ConnectionLimit.Int32)
	model.Tablespace = types.StringValue(dbModel.Tablespace.String)
	model.Comment = types.StringValue(dbModel.Comment.String)

	res.Diagnostics.Append(res.State.Set(ctx, &model)...)
}

func (r *postgresqlDatabaseResource) Update(ctx context.Context, req resource.UpdateRequest, res *resource.UpdateResponse) {
	tflog.Debug(ctx, fmt.Sprintf("updating '%s' resource", r.resName))

	var model, stateModel postgresqlDatabaseModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	res.Diagnostics.Append(req.State.Get(ctx, &stateModel)...)
	if res.Diagnostics.HasError() {
		return
	}

	conn, err := r.pgClient.GetConnection(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	params := model.buildPgDatabaseUpdateParams(&stateModel)
	err = r.databaseRepo.Update(ctx, conn, model.Name.ValueString(), params)
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFUpdateAction, PGDatabase), err.Error())
		return
	}

	dbModel, err := r.databaseRepo.GetOne(ctx, conn, model.Name.ValueString())
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFReadAction, PGDatabase), err.Error())
		return
	}

	model.Owner = types.StringValue(dbModel.Owner.String)
	model.IsTemplate = types.BoolValue(dbModel.IsTemplate.Bool)
	model.AllowConnections = types.BoolValue(dbModel.AllowConnections.Bool)
	model.ConnectionLimit = types.Int32Value(dbModel.ConnectionLimit.Int32)
	model.Tablespace = types.StringValue(dbModel.Tablespace.String)
	model.Comment = types.StringValue(dbModel.Comment.String)
	model.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	res.Diagnostics.Append(res.State.Set(ctx, &model)...)
}

func (r *postgresqlDatabaseResource) Delete(ctx context.Context, req resource.DeleteRequest, res *resource.DeleteResponse) {
	tflog.Debug(ctx, fmt.Sprintf("deleting '%s' resource", r.resName))

	var model postgresqlDatabaseModel
	res.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	conn, err := r.pgClient.GetConnection(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	err = r.databaseRepo.Drop(ctx, conn, model.Name.ValueString(), model.ForceDrop.ValueBool())
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFDeleteAction, PGDatabase), err.Error())
		return
	}
}

func (r *postgresqlDatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, res *resource.ImportStateResponse) {
	// Import by name
	res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
	res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
