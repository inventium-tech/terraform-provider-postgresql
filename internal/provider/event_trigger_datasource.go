package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"terraform-provider-postgresql/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &eventTriggerDataSource{}
	_ datasource.DataSourceWithConfigure = &eventTriggerDataSource{}

	eventTriggerEventOptions = []string{
		"ddl_command_start",
		"ddl_command_end",
		"sql_drop",
		"table_rewrite",
	}
)

type eventTriggerDataSource struct {
	client client.PgClient
}

type eventTriggerDataSourceModel struct {
	Name     types.String `tfsdk:"name"`
	Event    types.String `tfsdk:"event"`
	Tags     types.Set    `tfsdk:"tags"`
	ExecFunc types.String `tfsdk:"exec_func"`
	Enabled  types.Bool   `tfsdk:"enabled"`
	Database types.String `tfsdk:"database"`
	Owner    types.String `tfsdk:"owner"`
	Comment  types.String `tfsdk:"comment"`
}

func NewEventTriggerDataSource() datasource.DataSource {
	return &eventTriggerDataSource{}
}

func (d *eventTriggerDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
	tflog.Trace(ctx, "Configuring 'event_trigger' datasource")

	pgClient, diags := parsePgClientFromRequest(ctx, req)

	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "Configured 'event_trigger' datasource")
	d.client = pgClient
}

func (d *eventTriggerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, res *datasource.MetadataResponse) {
	res.TypeName = req.ProviderTypeName + "_event_trigger"
}

func (d *eventTriggerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
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
		MarkdownDescription: mdDocResourceEventTrigger,
	}
}

func (d *eventTriggerDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Trace(ctx, "Reading 'event_trigger' datasource")

	var model eventTriggerDataSourceModel

	res.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	if model.Database.IsNull() {
		model.Database = types.StringValue(d.client.GetInitConfig().Database)
	}

	res.Diagnostics.Append(readEventTrigger(ctx, d.client, model.Database.ValueString(), model.Name.ValueString(), &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	res.Diagnostics.Append(res.State.Set(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}
	tflog.Trace(ctx, "Read 'event_trigger' datasource")
}
