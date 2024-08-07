package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	"strings"
	"terraform-provider-postgresql/internal/client"
	"time"
)

type eventTriggerResource struct {
	client client.PgClient
}

type eventTriggerResourceModel struct {
	Id          types.String `tfsdk:"id"`
	LastUpdated types.String `tfsdk:"last_updated"`
	Name        types.String `tfsdk:"name"`
	Event       types.String `tfsdk:"event"`
	Tags        types.Set    `tfsdk:"tags"`
	ExecFunc    types.String `tfsdk:"exec_func"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	Database    types.String `tfsdk:"database"`
	Owner       types.String `tfsdk:"owner"`
	Comment     types.String `tfsdk:"comment"`
}

var (
	_ resource.Resource                = &eventTriggerResource{}
	_ resource.ResourceWithConfigure   = &eventTriggerResource{}
	_ resource.ResourceWithImportState = &eventTriggerResource{}
)

func NewEventTriggerResource() resource.Resource {
	return &eventTriggerResource{}
}

func (r *eventTriggerResource) Configure(ctx context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	tflog.Trace(ctx, "Configuring 'event_trigger' resource")

	pgClient, diags := parsePgClientFromRequest(ctx, req)

	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	r.client = pgClient
}

func (r *eventTriggerResource) Metadata(_ context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	res.TypeName = req.ProviderTypeName + "_event_trigger"
}

func (r *eventTriggerResource) Schema(_ context.Context, _ resource.SchemaRequest, res *resource.SchemaResponse) {
	res.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier for the event trigger, in the format `database_name.event_trigger_name`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The timestamp of the last modification of the event trigger",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the event trigger",
			},
			"comment": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Comment associated with the event trigger",
			},
			"database": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Name of the database where the event trigger is located. If not provided, the database from the provider configuration will be used.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"event": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The event that will trigger the event trigger",
				Validators: []validator.String{
					stringvalidator.OneOf(eventTriggerEventOptions...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"tags": schema.SetAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of command tags that the event trigger will respond to",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
			},
			"exec_func": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The function that will be executed when the event trigger fires",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether the event trigger is enabled",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"owner": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The owner of the event trigger",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		MarkdownDescription: mdDocResourceEventTrigger,
	}
}

func (r *eventTriggerResource) Create(ctx context.Context, req resource.CreateRequest, res *resource.CreateResponse) {
	tflog.Trace(ctx, "Creating 'event_trigger' resource")

	var model eventTriggerResourceModel

	res.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	if model.Database.IsNull() {
		model.Database = types.StringValue(r.client.GetInitConfig().Database)
	}

	conn, err := r.client.GetConnection(ctx, model.Database.ValueString())
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	createParams := client.EventTriggerCreateParams{
		Name:     model.Name.ValueString(),
		Event:    model.Event.ValueString(),
		ExecFunc: model.ExecFunc.ValueString(),
		Enabled:  model.Enabled.ValueBool(),
		Tags:     mapSetValueToSlice[string](model.Tags),
		Comment:  model.Comment.ValueString(),
	}
	err = conn.EventTriggerRepository().Create(ctx, createParams)
	if err != nil {
		res.Diagnostics.AddError("Error creating event_trigger", err.Error())
		return
	}

	model.SetId()
	model.SetLastUpdated()

	// execute a Read operation to populate computed values
	res.Diagnostics.Append(readEventTrigger(ctx, r.client, model.Database.ValueString(), model.Name.ValueString(), &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	res.Diagnostics.Append(res.State.Set(ctx, model)...)
	if res.Diagnostics.HasError() {
		return
	}
	tflog.Trace(ctx, "Created 'event_trigger' resource")
}

func (r *eventTriggerResource) Read(ctx context.Context, req resource.ReadRequest, res *resource.ReadResponse) {
	tflog.Trace(ctx, "Reading 'event_trigger' resource")

	var model eventTriggerResourceModel

	// retrieve values from plan
	res.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	if model.Id.IsUnknown() || model.Id.IsNull() {
		res.Diagnostics.AddError("Missing Identifier for the event trigger", "Id is required for reading event trigger")
		return
	}

	idParts := strings.Split(model.Id.ValueString(), ".")
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		res.Diagnostics.AddError("Invalid Identifier for the event trigger", "Id should be in the format 'database_name.event_trigger_name'")
		return
	}

	targetDb := idParts[0]
	targetName := idParts[1]

	res.Diagnostics.Append(readEventTrigger(ctx, r.client, targetDb, targetName, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	if model.Id.IsNull() {
		model.SetId()
	}

	res.Diagnostics.Append(res.State.Set(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}
	tflog.Trace(ctx, "Read 'event_trigger' resource")
}

func (r *eventTriggerResource) Update(ctx context.Context, req resource.UpdateRequest, res *resource.UpdateResponse) {
	tflog.Trace(ctx, "Updating 'event_trigger' resource")

	var stateModel eventTriggerResourceModel
	var planModel eventTriggerResourceModel

	res.Diagnostics.Append(req.State.Get(ctx, &stateModel)...)
	res.Diagnostics.Append(req.Plan.Get(ctx, &planModel)...)

	conn, err := r.client.GetConnection(ctx, stateModel.Database.ValueString())
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	updateParams := client.EventTriggerUpdateParams{
		Name:    stateModel.Name.ValueString(),
		NewName: planModel.Name.ValueStringPointer(),
		Enabled: planModel.Enabled.ValueBoolPointer(),
		Owner:   planModel.Owner.ValueStringPointer(),
		Comment: planModel.Comment.ValueStringPointer(),
	}

	pgModel, err := conn.EventTriggerRepository().Update(ctx, updateParams)
	if err != nil {
		res.Diagnostics.AddError("Error updating event_trigger", err.Error())
		return
	}

	err = mapPgModelToTerraformModel(pgModel, &planModel, make(map[string]any))
	if err != nil {
		res.Diagnostics.AddError(msgErrMapPgModel, err.Error())
		return
	}

	planModel.SetLastUpdated()

	res.Diagnostics.Append(res.State.Set(ctx, &planModel)...)
	if res.Diagnostics.HasError() {
		return
	}
	tflog.Trace(ctx, "Updated 'event_trigger' resource")
}

func (r *eventTriggerResource) Delete(ctx context.Context, req resource.DeleteRequest, res *resource.DeleteResponse) {
	tflog.Trace(ctx, "Deleting 'event_trigger' resource")

	var model eventTriggerResourceModel

	res.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	conn, err := r.client.GetConnection(ctx, model.Database.ValueString())
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}
	err = conn.EventTriggerRepository().Drop(ctx, model.Name.ValueString())
	if err != nil {
		res.Diagnostics.AddError("Error deleting event_trigger", err.Error())
		return
	}
	tflog.Trace(ctx, "Deleted 'event_trigger' resource")
}

func (r *eventTriggerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, res *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, res)
}

func readEventTrigger(ctx context.Context, pgClient client.PgClient, db, name string, target interface{}) diag.Diagnostics {
	diags := diag.Diagnostics{}

	conn, err := pgClient.GetConnection(ctx, db)
	if err != nil {
		diags.AddError(msgErrGetPgConnection, err.Error())
		return diags
	}

	pgModel, err := conn.EventTriggerRepository().Get(ctx, name)
	if err != nil {
		diags.AddError(fmt.Sprintf("Error reading event_trigger: '%s'", name), err.Error())
		return diags
	}

	_, diagErr := types.SetValueFrom(ctx, types.StringType, pgModel.Tags)
	if diagErr.HasError() {
		diags.Append(diagErr...)
		return diags
	}

	err = mapPgModelToTerraformModel(pgModel, target, make(map[string]any))
	if err != nil {
		diags.AddError("Error mapping pg model to terraform model", err.Error())
		return diags
	}

	return diags
}

func (rm *eventTriggerResourceModel) SetId() {
	rm.Id = types.StringValue(fmt.Sprintf("%s.%s", rm.Database.ValueString(), rm.Name.ValueString()))
}

func (rm *eventTriggerResourceModel) SetLastUpdated() {
	rm.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
}
