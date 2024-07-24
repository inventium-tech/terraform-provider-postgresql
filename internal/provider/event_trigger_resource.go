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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"terraform-provider-postgresql/internal/client"
)

type eventTriggerResource struct {
	client client.PGClient
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
	tflog.Info(ctx, "Configuring 'event_trigger' resource")

	pgClient, diags := standardDataSourceConfigure(ctx, req)

	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	r.client = pgClient
	tflog.Info(ctx, "Configured 'event_trigger' datasource")
}

func (r *eventTriggerResource) Metadata(ctx context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	res.TypeName = req.ProviderTypeName + "_event_trigger"
}

func (r *eventTriggerResource) Schema(ctx context.Context, req resource.SchemaRequest, res *resource.SchemaResponse) {
	res.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier for the event trigger, in the format `database_name.event_trigger_name`",
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
				MarkdownDescription: "Name of the database where the event trigger is located. If not provided, the database from the provider configuration will be used.",
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
		MarkdownDescription: "The `event_trigger` resource allows you to Create and manage event triggers in a PostgreSQL database.",
	}
}

func (r *eventTriggerResource) Create(ctx context.Context, req resource.CreateRequest, res *resource.CreateResponse) {
	tflog.Info(ctx, "Creating 'event_trigger' resource")

	var model eventTriggerResModel

	// retrieve values from plan
	res.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	if model.Database.IsNull() {
		model.Database = types.StringValue(r.client.GetConfig().Database)
	}

	res.Diagnostics.Append(model.Create(ctx, r.client)...)
	if res.Diagnostics.HasError() {
		return
	}

	model.SetId()
	model.SetLastUpdated()

	// execute a Read operation to populate computed values
	res.Diagnostics.Append(model.Read(ctx, r.client)...)
	if res.Diagnostics.HasError() {
		return
	}

	res.Diagnostics.Append(res.State.Set(ctx, model)...)
	if res.Diagnostics.HasError() {
		return
	}
}

func (r *eventTriggerResource) Read(ctx context.Context, req resource.ReadRequest, res *resource.ReadResponse) {
	tflog.Info(ctx, "Reading 'event_trigger' resource")

	var model eventTriggerResModel

	// retrieve values from plan
	res.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, fmt.Sprintf("\n\n\nMODEL: %v\n\n\n", model))

	res.Diagnostics.Append(model.Read(ctx, r.client)...)
	if res.Diagnostics.HasError() {
		return
	}

	if model.LastUpdated.IsNull() {
		model.SetLastUpdated()
	}

	res.Diagnostics.Append(res.State.Set(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}
}

func (r *eventTriggerResource) Update(ctx context.Context, req resource.UpdateRequest, res *resource.UpdateResponse) {
	tflog.Info(ctx, "Updating 'event_trigger' resource")

	var currentModel eventTriggerResModel
	var newModel eventTriggerResModel

	// retrieve values from plan
	res.Diagnostics.Append(req.State.Get(ctx, &currentModel)...)
	res.Diagnostics.Append(req.Plan.Get(ctx, &newModel)...)
	if res.Diagnostics.HasError() {
		return
	}

	res.Diagnostics.Append(currentModel.Update(ctx, r.client, &newModel)...)
	if res.Diagnostics.HasError() {
		return
	}

	newModel.SetId()
	newModel.SetLastUpdated()

	res.Diagnostics.Append(res.State.Set(ctx, newModel)...)
	if res.Diagnostics.HasError() {
		return
	}
}

func (r *eventTriggerResource) Delete(ctx context.Context, req resource.DeleteRequest, res *resource.DeleteResponse) {
	tflog.Info(ctx, "Deleting 'event_trigger' resource")

	var model eventTriggerResModel

	// retrieve values from plan
	res.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	res.Diagnostics.Append(model.Delete(ctx, r.client)...)
	if res.Diagnostics.HasError() {
		return
	}
}

func (r *eventTriggerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, res *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, res)
}
