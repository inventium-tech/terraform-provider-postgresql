package provider

import (
	"context"
	"fmt"
	"time"

	"terraform-provider-postgresql/internal/pgclient"
	"terraform-provider-postgresql/internal/provider/validators"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &postgresqlSchemaResource{}
	_ resource.ResourceWithConfigure   = &postgresqlSchemaResource{}
	_ resource.ResourceWithImportState = &postgresqlSchemaResource{}
)

func NewPostgresqlSchemaResource() resource.Resource {
	return &postgresqlSchemaResource{
		resName: "postgresql_schema",
	}
}

type postgresqlSchemaResource struct {
	resName    string
	pgClient   pgclient.PostgresqlClient
	schemaRepo pgclient.SchemaRepo
}

func (r *postgresqlSchemaResource) Metadata(_ context.Context, _ resource.MetadataRequest, res *resource.MetadataResponse) {
	res.TypeName = r.resName
}

func (r *postgresqlSchemaResource) Schema(_ context.Context, _ resource.SchemaRequest, res *resource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "Creates a PostgreSQL schema. [PostgreSQL documentation](https://www.postgresql.org/docs/current/sql-createschema.html)",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for the schema",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp of the resource's last modification",
			},
			"name": schema.StringAttribute{
				Description: "The name of the schema.",
				Required:    true,
				Validators: []validator.String{
					validators.PostgresqlObjectName(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"owner": schema.StringAttribute{
				Description: "The role that owns the schema. Defaults to the user executing the command.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					validators.PostgresqlObjectName(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"if_not_exists": schema.BoolAttribute{
				Description: "If true, do not throw an error if the schema already exists. Default is false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"drop_cascade": schema.BoolAttribute{
				Description: "If true, automatically drop objects contained in the schema when deleting. Default is false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"policy": schema.StringAttribute{
				Description: "Policy for handling name collisions. Valid values: 'error_on_collision' (default), 'skip', 'replace_on_collision'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("error_on_collision"),
				Validators: []validator.String{
					stringvalidator.OneOf("error_on_collision", "skip", "replace_on_collision"),
				},
			},
		},
	}
}

func (r *postgresqlSchemaResource) Configure(_ context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
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
	r.schemaRepo = pgclient.NewSchemaRepo()
}

func (r *postgresqlSchemaResource) Create(ctx context.Context, req resource.CreateRequest, res *resource.CreateResponse) {
	tflog.Debug(ctx, fmt.Sprintf("creating '%s' resource", r.resName))

	var model postgresqlSchemaModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	conn, err := r.pgClient.GetConnection(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	// Handle policy
	switch policy := model.Policy.ValueString(); policy {
	case "skip":
		exists, err := r.schemaRepo.Exists(ctx, conn, model.Name.ValueString())
		if err != nil {
			res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFReadAction, PGSchema), err.Error())
			return
		}
		if exists {
			// Schema exists, skip creation but still read and return state
			schemaModel, err := r.schemaRepo.GetOne(ctx, conn, model.Name.ValueString())
			if err != nil {
				res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFReadAction, PGSchema), err.Error())
				return
			}
			model.Id = types.StringValue(schemaModel.Name.String)
			model.Name = types.StringValue(schemaModel.Name.String)
			model.Owner = types.StringValue(schemaModel.Owner.String)
			model.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))
			res.Diagnostics.Append(res.State.Set(ctx, &model)...)
			return
		}
	case "replace_on_collision":
		exists, err := r.schemaRepo.Exists(ctx, conn, model.Name.ValueString())
		if err != nil {
			res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFReadAction, PGSchema), err.Error())
			return
		}
		if exists {
			// Drop and recreate
			if err := r.schemaRepo.Drop(ctx, conn, model.Name.ValueString(), model.DropCascade.ValueBool()); err != nil {
				res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFDeleteAction, PGSchema), err.Error())
				return
			}
		}
	}

	params := pgclient.SchemaCreateParams{
		Name:        model.Name.ValueString(),
		Owner:       model.Owner.ValueString(),
		IfNotExists: model.IfNotExists.ValueBool(),
		Policy:      model.Policy.ValueStringPointer(),
	}

	err = r.schemaRepo.Create(ctx, conn, params)
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFCreateAction, PGSchema), err.Error())
		return
	}

	schemaModel, err := r.schemaRepo.GetOne(ctx, conn, model.Name.ValueString())
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFReadAction, PGSchema), err.Error())
		return
	}

	model.Id = types.StringValue(schemaModel.Name.String)
	model.Name = types.StringValue(schemaModel.Name.String)
	model.Owner = types.StringValue(schemaModel.Owner.String)
	model.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	res.Diagnostics.Append(res.State.Set(ctx, &model)...)
}

func (r *postgresqlSchemaResource) Read(ctx context.Context, req resource.ReadRequest, res *resource.ReadResponse) {
	tflog.Debug(ctx, fmt.Sprintf("reading '%s' resource", r.resName))

	var model postgresqlSchemaModel
	res.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	conn, err := r.pgClient.GetConnection(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	exists, err := r.schemaRepo.Exists(ctx, conn, model.Name.ValueString())
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFReadAction, PGSchema), err.Error())
		return
	}

	if !exists {
		res.State.RemoveResource(ctx)
		return
	}

	schemaModel, err := r.schemaRepo.GetOne(ctx, conn, model.Name.ValueString())
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFReadAction, PGSchema), err.Error())
		return
	}

	model.Name = types.StringValue(schemaModel.Name.String)
	model.Owner = types.StringValue(schemaModel.Owner.String)

	res.Diagnostics.Append(res.State.Set(ctx, &model)...)
}

func (r *postgresqlSchemaResource) Update(ctx context.Context, req resource.UpdateRequest, res *resource.UpdateResponse) {
	tflog.Debug(ctx, fmt.Sprintf("updating '%s' resource", r.resName))

	var model, stateModel postgresqlSchemaModel
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

	params := model.buildPgSchemaUpdateParams(&stateModel)
	err = r.schemaRepo.Update(ctx, conn, model.Name.ValueString(), params)
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFUpdateAction, PGSchema), err.Error())
		return
	}

	schemaModel, err := r.schemaRepo.GetOne(ctx, conn, model.Name.ValueString())
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFReadAction, PGSchema), err.Error())
		return
	}

	model.Owner = types.StringValue(schemaModel.Owner.String)
	model.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	res.Diagnostics.Append(res.State.Set(ctx, &model)...)
}

func (r *postgresqlSchemaResource) Delete(ctx context.Context, req resource.DeleteRequest, res *resource.DeleteResponse) {
	tflog.Debug(ctx, fmt.Sprintf("deleting '%s' resource", r.resName))

	var model postgresqlSchemaModel
	res.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	conn, err := r.pgClient.GetConnection(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	err = r.schemaRepo.Drop(ctx, conn, model.Name.ValueString(), model.DropCascade.ValueBool())
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFDeleteAction, PGSchema), err.Error())
		return
	}
}

func (r *postgresqlSchemaResource) ImportState(ctx context.Context, req resource.ImportStateRequest, res *resource.ImportStateResponse) {
	// Import by name
	res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
	res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
