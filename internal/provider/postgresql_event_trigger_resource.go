package provider

import (
	"context"
	"fmt"
	"strings"
	"terraform-provider-postgresql/internal/pgclient"
	"terraform-provider-postgresql/internal/provider/validators"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jackc/pgx/v5"
)

var (
	_ resource.Resource                = &resourceEventTrigger{}
	_ resource.ResourceWithConfigure   = &resourceEventTrigger{}
	_ resource.ResourceWithImportState = &resourceEventTrigger{}
)

func NewPostgresqlEventTriggerResource() resource.Resource {
	return &resourceEventTrigger{
		resName: "postgresql_event_trigger",
	}
}

type resourceEventTrigger struct {
	resName          string
	pgClient         pgclient.PostgresqlClient
	eventTriggerRepo pgclient.EventTriggerRepo
}

func (r *resourceEventTrigger) Metadata(_ context.Context, _ resource.MetadataRequest, res *resource.MetadataResponse) {
	res.TypeName = r.resName
}

func (r *resourceEventTrigger) Schema(_ context.Context, _ resource.SchemaRequest, res *resource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "Manages a PostgreSQL Event Trigger. Event triggers are used to execute functions in response to certain database events, such as DDL commands.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for the event trigger, in the format `<database_name>.<event_trigger_name>`",
			},
			"last_updated": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp of the resource's last modification",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the event trigger. Must be unique within the database.",
				Validators: []validator.String{
					validators.PostgresqlObjectName(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"database": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Name of the database where the Event Trigger is located. If not specified, the provider's configured database will be used.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"event": schema.StringAttribute{
				Required:    true,
				Description: "The type of event that the trigger responds to. Valid values are `ddl_command_start`, `ddl_command_end`, `sql_drop`, etc.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("ddl_command_start", "ddl_command_end", "sql_drop", "table_rewrite"),
				},
			},
			"tags": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "A set of tags associated with the event trigger. Tags can be used for categorization and filtering.",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
					setplanmodifier.RequiresReplace(),
				},
			},
			"exec_func": schema.StringAttribute{
				Required:    true,
				Description: "Name of the function to be executed when the event trigger is fired.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the event trigger is enabled. Defaults to `true`.",
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"owner": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The owner of the event trigger. If not specified, the owner will be the User used in the provider's configuration.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"comment": schema.StringAttribute{
				Optional:    true,
				Description: "Comment associated with the Postgresql Event Trigger.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *resourceEventTrigger) Configure(ctx context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	tflog.Debug(ctx, fmt.Sprintf("configuring '%s' resource", r.resName))

	if req.ProviderData == nil {
		return
	}

	res.Diagnostics.Append(parsePgClientFromRequest(req, &r.pgClient))
	if res.Diagnostics.HasError() {
		return
	}

	r.eventTriggerRepo = pgclient.NewEventTriggerRepo()
}

func (r *resourceEventTrigger) Create(ctx context.Context, req resource.CreateRequest, res *resource.CreateResponse) {
	tflog.Debug(ctx, fmt.Sprintf("creating '%s' resource", r.resName))

	var model resourceModelEventTrigger

	res.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	if model.Database.IsNull() || model.Database.ValueString() == "" {
		model.Database = types.StringValue(r.pgClient.GetInitConfig().Database)
	}

	if model.Owner.IsNull() || model.Owner.ValueString() == "" {
		model.Owner = types.StringValue(r.pgClient.GetInitConfig().Username)
	}

	var tags []string
	res.Diagnostics.Append(model.Tags.ElementsAs(ctx, &tags, false)...)
	if res.Diagnostics.HasError() {
		return
	}

	pool, err := r.pgClient.GetPool(ctx, model.Database.ValueString())
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

	params := pgclient.EventTriggerCreateParams{
		Name:     model.Name.ValueString(),
		Event:    model.Event.ValueString(),
		Tags:     tags,
		ExecFunc: model.ExecFunc.ValueString(),
		Enabled:  model.Enabled.ValueBool(),
		Database: model.Database.ValueString(),
		Owner:    model.Owner.ValueString(),
		Comment:  model.Comment.ValueString(),
	}

	err = r.eventTriggerRepo.Create(ctx, tx, params)
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFCreateAction, PGEventTrigger), err.Error())
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

func (r *resourceEventTrigger) Read(ctx context.Context, req resource.ReadRequest, res *resource.ReadResponse) {
	tflog.Debug(ctx, fmt.Sprintf("reading '%s' resource", r.resName))

	var model resourceModelEventTrigger

	// retrieve values from the state
	res.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	if model.Id.IsUnknown() || model.Id.IsNull() {
		res.Diagnostics.AddAttributeError(path.Root("id"), msgErrorMissingResId, fmt.Sprintf(msgInvalidFieldPatternDetail, "id", "<database_name>.<event_trigger_name>"))
		return
	}

	dbName, eventName, foundSep := strings.Cut(model.Id.ValueString(), ".")
	if !foundSep {
		res.Diagnostics.AddAttributeError(path.Root("id"), fmt.Sprintf(msgInvalidFieldPattern, "id"), fmt.Sprintf(msgInvalidFieldPatternDetail, "id", "<database_name>.<event_trigger_name>"))
		return
	}

	poolConn, err := r.pgClient.AcquireConn(ctx, dbName)
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}
	defer poolConn.Release()

	diags := readEventTriggerFromDB(ctx, poolConn.Conn(), r.eventTriggerRepo, eventName, &model)
	if diags.HasError() {
		res.Diagnostics.Append(diags...)
		return
	}

	model.SetId()
	res.Diagnostics.Append(res.State.Set(ctx, &model)...)
}

func (r *resourceEventTrigger) Update(ctx context.Context, req resource.UpdateRequest, res *resource.UpdateResponse) {
	tflog.Debug(ctx, fmt.Sprintf("updating '%s' resource", r.resName))

	var planModel resourceModelEventTrigger
	var stateModel resourceModelEventTrigger

	res.Diagnostics.Append(req.Plan.Get(ctx, &planModel)...)
	res.Diagnostics.Append(req.State.Get(ctx, &stateModel)...)

	if res.Diagnostics.HasError() {
		return
	}

	var err error
	var tx pgx.Tx

	pool, err := r.pgClient.GetPool(ctx, stateModel.Database.ValueString())
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}
	if tx, err = pool.Begin(ctx); err != nil {
		res.Diagnostics.AddError(msgErrStartPgTransaction, err.Error())
		return
	}
	defer pgclient.DeferredRollback(ctx, tx)

	repo := pgclient.NewEventTriggerRepo()
	var updateParams pgclient.EventTriggerUpdateParams

	if planModel.Name != stateModel.Name {
		updateParams.Name = planModel.Name.ValueStringPointer()
	}
	if planModel.Enabled != stateModel.Enabled {
		updateParams.Enabled = planModel.Enabled.ValueBoolPointer()
	}
	if planModel.Owner != stateModel.Owner {
		updateParams.Owner = planModel.Owner.ValueStringPointer()
	}
	if planModel.Comment != stateModel.Comment {
		updateParams.Comment = planModel.Comment.ValueStringPointer()
	}

	err = repo.Update(ctx, tx, stateModel.Name.ValueString(), updateParams)
	if err != nil {
		res.Diagnostics.AddError("Error updating PostgreSQL Event Trigger", err.Error())
		return
	}

	if err = tx.Commit(ctx); err != nil {
		res.Diagnostics.AddError(msgErrCommitPgTransaction, err.Error())
		return
	}

	planModel.SetId()
	setLastUpdatedFieldValue(&planModel.LastUpdated)
	res.Diagnostics.Append(res.State.Set(ctx, &planModel)...)
}

func (r *resourceEventTrigger) Delete(ctx context.Context, req resource.DeleteRequest, res *resource.DeleteResponse) {
	tflog.Debug(ctx, fmt.Sprintf("deleting '%s' resource", r.resName))

	var model resourceModelEventTrigger
	res.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	if model.Id.IsUnknown() || model.Id.IsNull() {
		res.Diagnostics.AddAttributeError(path.Root("id"), msgErrorMissingResId, fmt.Sprintf(msgInvalidFieldPatternDetail, "id", "<database_name>.<event_trigger_name>"))
		return
	}

	dbName, eventName, foundSep := strings.Cut(model.Id.ValueString(), ".")
	if !foundSep {
		res.Diagnostics.AddAttributeError(path.Root("id"), fmt.Sprintf(msgInvalidFieldPattern, "id"), fmt.Sprintf(msgInvalidFieldPatternDetail, "id", "<database_name>.<event_trigger_name>"))
		return
	}

	conn, err := r.pgClient.GetConnection(ctx, dbName)
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}
	eventTriggerRepo := pgclient.NewEventTriggerRepo()
	err = eventTriggerRepo.Drop(ctx, conn, eventName)
	if err != nil {
		res.Diagnostics.AddError("Error deleting PostgreSQL Event Trigger", err.Error())
		return
	}
}

func (r *resourceEventTrigger) ImportState(ctx context.Context, req resource.ImportStateRequest, res *resource.ImportStateResponse) {
	tflog.Debug(ctx, fmt.Sprintf("importing '%s' resource", r.resName))
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, res)
}
