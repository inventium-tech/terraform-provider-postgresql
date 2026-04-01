package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"terraform-provider-postgresql/internal/pgclient"
	"terraform-provider-postgresql/internal/provider/validators"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &postgresqlExtensionResource{}
	_ resource.ResourceWithConfigure   = &postgresqlExtensionResource{}
	_ resource.ResourceWithImportState = &postgresqlExtensionResource{}
)

func NewPostgresqlExtensionResource() resource.Resource {
	return &postgresqlExtensionResource{
		resName: "postgresql_extension",
	}
}

type postgresqlExtensionResource struct {
	resName       string
	pgClient      pgclient.PostgresqlClient
	extensionRepo pgclient.ExtensionRepo
}

func (r *postgresqlExtensionResource) Metadata(_ context.Context, _ resource.MetadataRequest, res *resource.MetadataResponse) {
	res.TypeName = r.resName
}

func (r *postgresqlExtensionResource) Schema(_ context.Context, _ resource.SchemaRequest, res *resource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "Manages a PostgreSQL extension. Extensions provide additional functionality like PostGIS, uuid-ossp, pg_trgm, and more. [PostgreSQL documentation](https://www.postgresql.org/docs/current/sql-createextension.html)",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for the extension",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp of the resource's last modification",
			},
			"name": schema.StringAttribute{
				Description: "The name of the extension to install (e.g., 'uuid-ossp', 'pg_trgm', 'postgis').",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"schema": schema.StringAttribute{
				Description: "The schema in which to install the extension. If not specified, the extension's default schema is used.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					validators.PostgresqlObjectName(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.StringAttribute{
				Description: "The version of the extension to install. If not specified, the default version is installed. Can be updated to upgrade/downgrade the extension.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"database": schema.StringAttribute{
				Description: "The database in which to install the extension. Defaults to the provider's configured database.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					validators.PostgresqlObjectName(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cascade": schema.BoolAttribute{
				Description: "If true, also installs any extensions that this extension depends on. Default is false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"drop_cascade": schema.BoolAttribute{
				Description: "If true, drops all objects that depend on this extension when the extension is destroyed. Default is false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *postgresqlExtensionResource) Configure(_ context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
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
	r.extensionRepo = pgclient.NewExtensionRepo()
}

func (r *postgresqlExtensionResource) Create(ctx context.Context, req resource.CreateRequest, res *resource.CreateResponse) {
	tflog.Debug(ctx, fmt.Sprintf("creating '%s' resource", r.resName))

	var model postgresqlExtensionModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	var conn pgclient.DBTX
	var err error

	if !model.Database.IsNull() && model.Database.ValueString() != "" {
		conn, err = r.pgClient.GetConnection(ctx, model.Database.ValueString())
	} else {
		conn, err = r.pgClient.GetConnection(ctx)
		// Set computed database value
		if err == nil {
			model.Database = types.StringValue(r.pgClient.GetInitConfig().Database)
		}
	}
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	params := pgclient.ExtensionCreateParams{
		Name:     model.Name.ValueString(),
		Schema:   model.Schema.ValueString(),
		Version:  model.Version.ValueString(),
		Cascade:  model.Cascade.ValueBool(),
		Database: model.Database.ValueString(),
	}

	err = r.extensionRepo.Create(ctx, conn, params)
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFCreateAction, "extension"), err.Error())
		return
	}

	extModel, err := r.extensionRepo.GetOne(ctx, conn, model.Name.ValueString())
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFReadAction, "extension"), err.Error())
		return
	}

	model.Id = types.StringValue(extModel.Name.String)
	model.Name = types.StringValue(extModel.Name.String)
	model.Schema = types.StringValue(extModel.Schema.String)
	model.Version = types.StringValue(extModel.Version.String)
	model.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	res.Diagnostics.Append(res.State.Set(ctx, &model)...)
}

func (r *postgresqlExtensionResource) Read(ctx context.Context, req resource.ReadRequest, res *resource.ReadResponse) {
	tflog.Debug(ctx, fmt.Sprintf("reading '%s' resource", r.resName))

	var model postgresqlExtensionModel
	res.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	var conn pgclient.DBTX
	var err error

	if !model.Database.IsNull() && model.Database.ValueString() != "" {
		conn, err = r.pgClient.GetConnection(ctx, model.Database.ValueString())
	} else {
		conn, err = r.pgClient.GetConnection(ctx)
	}
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	exists, err := r.extensionRepo.Exists(ctx, conn, model.Name.ValueString())
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFReadAction, "extension"), err.Error())
		return
	}

	if !exists {
		res.State.RemoveResource(ctx)
		return
	}

	extModel, err := r.extensionRepo.GetOne(ctx, conn, model.Name.ValueString())
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFReadAction, "extension"), err.Error())
		return
	}

	model.Name = types.StringValue(extModel.Name.String)
	model.Schema = types.StringValue(extModel.Schema.String)
	model.Version = types.StringValue(extModel.Version.String)

	res.Diagnostics.Append(res.State.Set(ctx, &model)...)
}

func (r *postgresqlExtensionResource) Update(ctx context.Context, req resource.UpdateRequest, res *resource.UpdateResponse) {
	tflog.Debug(ctx, fmt.Sprintf("updating '%s' resource", r.resName))

	var plan, state postgresqlExtensionModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if res.Diagnostics.HasError() {
		return
	}

	var conn pgclient.DBTX
	var err error

	if !plan.Database.IsNull() && plan.Database.ValueString() != "" {
		conn, err = r.pgClient.GetConnection(ctx, plan.Database.ValueString())
	} else {
		conn, err = r.pgClient.GetConnection(ctx)
	}
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	params := plan.buildPgExtensionUpdateParams(&state)

	err = r.extensionRepo.Update(ctx, conn, plan.Name.ValueString(), params)
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFUpdateAction, "extension"), err.Error())
		return
	}

	extModel, err := r.extensionRepo.GetOne(ctx, conn, plan.Name.ValueString())
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFReadAction, "extension"), err.Error())
		return
	}

	plan.Name = types.StringValue(extModel.Name.String)
	plan.Schema = types.StringValue(extModel.Schema.String)
	plan.Version = types.StringValue(extModel.Version.String)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	res.Diagnostics.Append(res.State.Set(ctx, &plan)...)
}

func (r *postgresqlExtensionResource) Delete(ctx context.Context, req resource.DeleteRequest, res *resource.DeleteResponse) {
	tflog.Debug(ctx, fmt.Sprintf("deleting '%s' resource", r.resName))

	var model postgresqlExtensionModel
	res.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	var conn pgclient.DBTX
	var err error

	if !model.Database.IsNull() && model.Database.ValueString() != "" {
		conn, err = r.pgClient.GetConnection(ctx, model.Database.ValueString())
	} else {
		conn, err = r.pgClient.GetConnection(ctx)
	}
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	err = r.extensionRepo.Drop(ctx, conn, model.Name.ValueString(), model.DropCascade.ValueBool())
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFDeleteAction, "extension"), err.Error())
		return
	}
}

func (r *postgresqlExtensionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, res *resource.ImportStateResponse) {
	// Supports two import formats:
	//   extension_name           — uses the provider's configured database
	//   database:extension_name  — explicitly targets a specific database
	parts := strings.SplitN(req.ID, ":", 2)
	var extensionName, database string
	if len(parts) == 2 {
		database = parts[0]
		extensionName = parts[1]
	} else {
		extensionName = parts[0]
	}

	res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root("name"), extensionName)...)
	res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root("id"), extensionName)...)
	if database != "" {
		res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root("database"), database)...)
	}
}
