package provider

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/lib/pq"
	"terraform-provider-postgresql/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &EventTriggerDataSource{}
	_ datasource.DataSourceWithConfigure = &EventTriggerDataSource{}
)

type EventTriggerDataSource struct {
	client client.PGClient
}

func NewEventTriggerDataSource() datasource.DataSource {
	return &EventTriggerDataSource{}
}

func (d *EventTriggerDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
	tflog.Info(ctx, "Configuring 'event_trigger' datasource")

	pgClient, diags := standardDataSourceConfigure(ctx, req)

	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Configured 'event_trigger' datasource")
	d.client = pgClient
}

func (d *EventTriggerDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, res *datasource.MetadataResponse) {
	res.TypeName = req.ProviderTypeName + "_event_trigger"
}

func (d *EventTriggerDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, res *datasource.SchemaResponse) {
	res.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the event trigger",
			},
			"database": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the database where the event trigger is located",
			},
			"event": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The event that triggers the event trigger",
			},
			"tags": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of command tags that the event trigger will respond to",
			},
			"exec_func": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The Function that will be executed when the event trigger fires",
			},
			"enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the event trigger is enabled",
			},
			"owner": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The owner of the event trigger",
			},
			"comment": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Comment associated with the event trigger",
			},
		},
	}
}

func (d *EventTriggerDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Info(ctx, "Reading 'event_trigger' datasource")

	var model eventTriggerDSModel

	res.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	if model.Database.IsNull() {
		model.Database = types.StringValue(d.client.GetConfig().Database)
	}

	conn, err := d.client.GetConnection(ctx, model.Database.ValueString())
	if err != nil {
		res.Diagnostics.AddError("Error establishing a PostgreSQL connection", err.Error())
		return
	}

	query := d.client.GetEventTriggerQuery(model.Name.ValueString())

	var owner, comment, event, execFunc, evtEnabled sql.NullString
	var tags pq.StringArray

	err = conn.QueryRowContext(ctx, query).Scan(&owner, &comment, &event, &tags, &evtEnabled, &execFunc)
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf("Error reading event_trigger: '%s'", model.Name.ValueString()), err.Error())
		return
	}

	parsedTypes, diags := types.SetValueFrom(ctx, types.StringType, tags)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	model.Owner = types.StringValue(owner.String)
	model.Comment = types.StringValue(comment.String)
	model.Event = types.StringValue(event.String)
	model.ExecFunc = types.StringValue(execFunc.String)
	model.Tags = parsedTypes
	// evtenabled: Controls in which session_replication_role modes the event trigger fires.
	// O = trigger fires in “origin” and “local” modes
	// D = trigger is disabled
	// R = trigger fires in “replica” mode
	// A = trigger fires always.
	model.Enabled = types.BoolValue(evtEnabled.String != "D")

	diags = res.State.Set(ctx, &model)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}
}
