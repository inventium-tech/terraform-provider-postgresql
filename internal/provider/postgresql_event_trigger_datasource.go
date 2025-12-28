package provider

import (
	"context"
	"fmt"
	"terraform-provider-postgresql/internal/pgclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &datasourceEventTrigger{}
	_ datasource.DataSourceWithConfigure = &datasourceEventTrigger{}
)

func NewPostgresqlEventTriggerDataSource() datasource.DataSource {
	return &datasourceEventTrigger{
		dsName: "postgresql_event_trigger",
	}
}

type datasourceEventTrigger struct {
	dsName           string
	pgClient         pgclient.PostgresqlClient
	eventTriggerRepo pgclient.EventTriggerRepo
}

func (d *datasourceEventTrigger) Metadata(ctx context.Context, _ datasource.MetadataRequest, res *datasource.MetadataResponse) {
	res.TypeName = d.dsName
}

func (d *datasourceEventTrigger) Configure(ctx context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
	tflog.Debug(ctx, fmt.Sprintf("configuring '%s' data-source", d.dsName))

	if req.ProviderData == nil {
		return
	}

	res.Diagnostics.Append(parsePgClientFromRequest(req, &d.pgClient))
	if res.Diagnostics.HasError() {
		return
	}

	d.eventTriggerRepo = pgclient.NewEventTriggerRepo()
}

func (d *datasourceEventTrigger) Schema(_ context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "A data source used to retrieve information about PostgreSQL event triggers.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Name of the event trigger.",
				Required:    true,
			},
			"database": schema.StringAttribute{
				Description: "Name of the database where the event trigger is defined.",
				Required:    true,
			},
			"event": schema.StringAttribute{
				Description: "The type of event that the trigger responds to. Valid values are `ddl_command_start`, `ddl_command_end`, `sql_drop`, etc.",
				Computed:    true,
			},
			"tags": schema.SetAttribute{
				Description: "A set of tags associated with the event trigger. Tags can be used for categorization and filtering.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"exec_func": schema.StringAttribute{
				Description: "Name of the function to be executed when the event trigger is fired.",
				Computed:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the event trigger is enabled. Defaults to `true`.",
				Computed:    true,
			},
			"owner": schema.StringAttribute{
				Description: "The owner of the event trigger. If not specified, the owner will be the User used in the provider's configuration.",
				Computed:    true,
			},
			"comment": schema.StringAttribute{
				Description: "Comment associated with the Postgresql Event Trigger.",
				Computed:    true,
			},
		},
	}
}

func (d *datasourceEventTrigger) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Debug(ctx, fmt.Sprintf("reading '%s' data-source", d.dsName))

	var model datasourceModelEventTrigger

	// retrieve values from the state
	res.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	poolConn, err := d.pgClient.AcquireConn(ctx, model.Database.ValueString())
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}
	defer poolConn.Release()

	diags := readEventTriggerFromDB(ctx, poolConn.Conn(), d.eventTriggerRepo, model.Name.ValueString(), &model)
	if diags.HasError() {
		res.Diagnostics.Append(diags...)
		return
	}

	res.Diagnostics.Append(res.State.Set(ctx, &model)...)
}
