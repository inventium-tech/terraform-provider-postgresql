package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"strings"
	"terraform-provider-postgresql/internal/helpers"
	"terraform-provider-postgresql/internal/pgclient"
	"terraform-provider-postgresql/internal/provider/validators"
)

var (
	_ resource.Resource                = &postgresqlUserFunctionResource{}
	_ resource.ResourceWithConfigure   = &postgresqlUserFunctionResource{}
	_ resource.ResourceWithImportState = &postgresqlUserFunctionResource{}
)

func NewPostgresqlUserFunctionResource() resource.Resource {
	return &postgresqlUserFunctionResource{
		resName: "postgresql_user_function",
	}
}

type postgresqlUserFunctionResource struct {
	resName          string
	pgClient         pgclient.PostgresqlClient
	userFunctionRepo pgclient.UserFunctionRepo
}

func (r *postgresqlUserFunctionResource) Metadata(_ context.Context, _ resource.MetadataRequest, res *resource.MetadataResponse) {
	res.TypeName = r.resName
}

func (r *postgresqlUserFunctionResource) Schema(_ context.Context, _ resource.SchemaRequest, res *resource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "Manages a Postgresql user-defined function in a specified database and schema. [Postgresql documentation](https://www.postgresql.org/docs/current/sql-createfunction.html)",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for the user function, in the format `<database>.<schema>.<function_name>(<arguments>)`",
			},
			"last_updated": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp of the resource's last modification",
			},
			"name": schema.StringAttribute{
				Description: "The name of the Postgresql function.",
				Required:    true,
				Validators: []validator.String{
					validators.PostgresqlObjectName(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"args": schema.ListNestedAttribute{
				Description: "The arguments of the Postgresql function.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "The name of the argument.",
						},
						"type": schema.StringAttribute{
							Required:    true,
							Description: "The data type of the argument, preferably in lowercase (e.g., 'integer', 'text').",
						},
						"mode": schema.StringAttribute{
							Optional:    true,
							Description: "The mode of the argument (IN, OUT, INOUT, VARIADIC). If not specified, Postgresql assumes 'IN' by default.",
							Validators: []validator.String{
								stringvalidator.OneOf("IN", "OUT", "INOUT", "VARIADIC"),
							},
						},
						"default": schema.StringAttribute{
							Optional:    true,
							Description: "The default value of the argument.",
						},
					},
				},
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
					listplanmodifier.RequiresReplace(),
				},
			},
			"returns": schema.StringAttribute{
				Description: "The return type of the Postgresql function. If not specified, the default is 'void'.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Default: stringdefault.StaticString("void"),
			},
			"body": schema.StringAttribute{
				Description: "The body of the Postgresql function.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"allow_replace": schema.BoolAttribute{
				Description: "If true, the generated SQL statements will contain 'OR REPLACE'. The default is true.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
					boolplanmodifier.RequiresReplace(),
				},
				Default: booldefault.StaticBool(true),
			},
			"language": schema.StringAttribute{
				Description: "The language of the Postgresql function, the value must be either `plpgsql` or `sql`. If not specified, the default is 'plpgsql'.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("plpgsql", "sql"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Default: stringdefault.StaticString("plpgsql"),
			},
			"comment": schema.StringAttribute{
				Optional:    true,
				Description: "Comment associated with the Postgresql Function.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"database": schema.StringAttribute{
				Description: "Name of the Postgresql database where the Function is located. If not specified, the provider's configured database will be used.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"schema": schema.StringAttribute{
				Description: "The name of the Postgresql schema to create the function in. The default is 'public'.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Default: stringdefault.StaticString("public"),
			},
			"owner": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The owner of the Postgresql function. If not specified, the owner will be the User used in the provider's configuration.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *postgresqlUserFunctionResource) Configure(ctx context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	tflog.Debug(ctx, fmt.Sprintf("configuring '%s' resource", r.resName))

	if req.ProviderData == nil {
		return
	}

	res.Diagnostics.Append(parsePgClientFromRequest(req, &r.pgClient))
	if res.Diagnostics.HasError() {
		return
	}

	r.userFunctionRepo = pgclient.NewUserFunctionRepo()
}

func (r *postgresqlUserFunctionResource) Create(ctx context.Context, req resource.CreateRequest, res *resource.CreateResponse) {
	tflog.Debug(ctx, fmt.Sprintf("creating '%s' resource", r.resName))

	var model postgresqlUserFunctionModel

	res.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	if model.Database.IsNull() || model.Database.ValueString() == "" {
		model.Database = types.StringValue(r.pgClient.GetInitConfig().Database)
	}

	if model.Schema.IsNull() || model.Schema.ValueString() == "" {
		model.Schema = types.StringValue("public")
	}

	if model.Owner.IsNull() || model.Owner.ValueString() == "" {
		model.Owner = types.StringValue(r.pgClient.GetInitConfig().Username)
	}

	pgArgs := helpers.SliceMap(model.Args, func(arg postgresqlUserFunctionArgType) pgclient.PgFunctionArgType {
		return pgclient.PgFunctionArgType{
			Name:    arg.Name.ValueString(),
			Type:    arg.Type.ValueString(),
			Mode:    arg.Mode.ValueString(),
			Default: arg.Default.ValueString(),
		}
	})

	conn, err := r.pgClient.GetConnection(ctx, model.Database.ValueString())
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

	params := pgclient.UserFunctionCreateParams{
		Name:         model.Name.ValueString(),
		Args:         pgArgs,
		Returns:      model.Returns.ValueString(),
		Language:     model.Language.ValueString(),
		Body:         model.Body.ValueString(),
		AllowReplace: model.AllowReplace.ValueBool(),
		Schema:       model.Schema.ValueString(),
		Owner:        model.Owner.ValueString(),
		Comment:      model.Comment.ValueString(),
	}

	err = r.userFunctionRepo.Create(ctx, conn, params)
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFCreateAction, PGUserFunction), err.Error())
		return
	}

	err = tx.Commit(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrCommitPgTransaction, err.Error())
		return
	}

	model.SetId()
	setLastUpdatedFieldValue(&model.LastUpdated)

	res.Diagnostics.Append(res.State.Set(ctx, &model)...)
}

func (r *postgresqlUserFunctionResource) Read(ctx context.Context, req resource.ReadRequest, res *resource.ReadResponse) {
	tflog.Debug(ctx, fmt.Sprintf("reading '%s' resource", r.resName))

	var model postgresqlUserFunctionModel

	// retrieve values from the state
	res.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	if model.Id.IsUnknown() || model.Id.IsNull() {
		res.Diagnostics.AddAttributeError(path.Root("id"), msgErrorMissingResId, fmt.Sprintf(msgInvalidFieldPatternDetail, "id", "<database_name>.<schema>.<user_function_name>"))
		return
	}

	dbName, funcId, foundSep := strings.Cut(model.Id.ValueString(), ".")
	if !foundSep {
		res.Diagnostics.AddAttributeError(path.Root("id"), fmt.Sprintf(msgInvalidFieldPattern, "id"), fmt.Sprintf(msgInvalidFieldPatternDetail, "id", "<database_name>.<schema>.<user_function_name>"))
		return
	}

	conn, err := r.pgClient.GetConnection(ctx, dbName)
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	pgUserFunction, err := r.userFunctionRepo.GetOne(ctx, conn, funcId)
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFReadAction, PGUserFunction), err.Error())
		return
	}

	if pgUserFunction == nil {
		res.Diagnostics.AddAttributeError(
			path.Root("id"),
			fmt.Sprintf(msgErrorPgObjectNotFund, PGUserFunction),
			fmt.Sprintf(msgErrorPgObjectNotFundDetail, PGUserFunction, model.Id.ValueString()),
		)
		return
	}

	// manual mapping from pgUserFunction to model
	model.Name = types.StringValue(pgUserFunction.Name.String)
	model.Returns = types.StringValue(pgUserFunction.Returns.String)
	model.Body = types.StringValue(pgUserFunction.Body.String)
	model.Language = types.StringValue(pgUserFunction.Language.String)
	model.AllowReplace = types.BoolValue(pgUserFunction.AllowReplace.Bool)
	model.Comment = types.StringValue(pgUserFunction.Comment.String)
	model.Database = types.StringValue(pgUserFunction.Database.String)
	model.Schema = types.StringValue(pgUserFunction.Schema.String)
	model.Owner = types.StringValue(pgUserFunction.Owner.String)

	// transform PostgreSQL function arguments values to Terraform types
	model.Args = helpers.SliceMap(pgUserFunction.Args.Items(), func(arg pgclient.PgFunctionArgType) postgresqlUserFunctionArgType {
		var result postgresqlUserFunctionArgType

		if arg.Name != "" {
			result.Name = types.StringValue(arg.Name)
		}

		if arg.Type != "" {
			result.Type = types.StringValue(arg.Type)
		}

		if arg.Mode != "" {
			result.Mode = types.StringValue(arg.Mode)
		}

		if arg.Default != "" {
			result.Default = types.StringValue(arg.Default)
		}

		return result
	})

	model.SetId()
	res.Diagnostics.Append(res.State.Set(ctx, &model)...)
}

func (r *postgresqlUserFunctionResource) Update(ctx context.Context, req resource.UpdateRequest, res *resource.UpdateResponse) {
	tflog.Debug(ctx, fmt.Sprintf("updating '%s' resource", r.resName))

	var planModel postgresqlUserFunctionModel
	var stateModel postgresqlUserFunctionModel

	res.Diagnostics.Append(req.Plan.Get(ctx, &planModel)...)
	res.Diagnostics.Append(req.State.Get(ctx, &stateModel)...)

	if res.Diagnostics.HasError() {
		return
	}

	conn, err := r.pgClient.GetConnection(ctx, stateModel.Database.ValueString())
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

	var updateParams pgclient.UserFunctionUpdateParams

	if planModel.Name != stateModel.Name {
		updateParams.Name = planModel.Name.ValueStringPointer()
	}
	if planModel.Owner != stateModel.Owner {
		updateParams.Owner = planModel.Owner.ValueStringPointer()
	}
	if planModel.Comment != stateModel.Comment {
		updateParams.Comment = planModel.Comment.ValueStringPointer()
	}

	err = r.userFunctionRepo.Update(ctx, tx, stateModel.Id.ValueString(), updateParams)
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFUpdateAction, PGUserFunction), err.Error())
		return
	}

	err = tx.Commit(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrCommitPgTransaction, err.Error())
		return
	}

	planModel.SetId()
	setLastUpdatedFieldValue(&planModel.LastUpdated)
	res.Diagnostics.Append(res.State.Set(ctx, &planModel)...)
}

func (r *postgresqlUserFunctionResource) Delete(ctx context.Context, req resource.DeleteRequest, res *resource.DeleteResponse) {
	tflog.Debug(ctx, fmt.Sprintf("deleting '%s' resource", r.resName))

	var model postgresqlUserFunctionModel

	// retrieve values from the state
	res.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	if model.Id.IsUnknown() || model.Id.IsNull() {
		res.Diagnostics.AddError(msgErrorMissingResId, fmt.Sprintf(msgRequiredField, "Id", r.resName))
		return
	}

	dbName, fullFnName, foundSep := strings.Cut(model.Id.ValueString(), ".")
	if !foundSep {
		res.Diagnostics.AddAttributeError(path.Root("Id"), fmt.Sprintf(msgInvalidFieldPattern, "Id"), "The Id must be in the format '<database_name>.<schema>.<user_function_name>'")
	}

	conn, err := r.pgClient.GetConnection(ctx, dbName)
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	err = r.userFunctionRepo.Drop(ctx, conn, fullFnName)
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFDeleteAction, PGUserFunction), err.Error())
		return
	}
}

func (r *postgresqlUserFunctionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, res *resource.ImportStateResponse) {
	tflog.Debug(ctx, fmt.Sprintf("importing '%s' resource", r.resName))
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, res)
}
