package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"terraform-provider-postgresql/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &DatabaseDataSource{}
	_ datasource.DataSourceWithConfigure = &DatabaseDataSource{}
)

type DatabaseDataSource struct {
	client client.PGClient
}

type DatabaseDataSourceModel struct {
	Name             types.String `tfsdk:"name"`
	Owner            types.String `tfsdk:"owner"`
	Comment          types.String `tfsdk:"comment"`
	Encoding         types.String `tfsdk:"encoding"`
	LcCollate        types.String `tfsdk:"lc_collate"`
	LcType           types.String `tfsdk:"lc_type"`
	ConnectionLimit  types.Int64  `tfsdk:"connection_limit"`
	AllowConnections types.Bool   `tfsdk:"allow_connections"`
}

func NewDatabaseDataSource() datasource.DataSource {
	return &DatabaseDataSource{}
}

func (dbDs *DatabaseDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
	tflog.Info(ctx, "Configuring 'database' datasource")

	pgClient, diags := standardDataSourceConfigure(ctx, req)

	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Configured Database Datasource")
	dbDs.client = pgClient
}

func (dbDs *DatabaseDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, res *datasource.MetadataResponse) {
	res.TypeName = req.ProviderTypeName + "_database"
}

func (dbDs *DatabaseDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, res *datasource.SchemaResponse) {
	res.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the database",
			},
			"comment": schema.StringAttribute{
				Computed: true,
			},
			"owner": schema.StringAttribute{
				Computed: true,
			},
			"encoding": schema.StringAttribute{
				Computed: true,
			},
			"lc_collate": schema.StringAttribute{
				Computed: true,
			},
			"lc_type": schema.StringAttribute{
				Computed: true,
			},
			"connection_limit": schema.Int64Attribute{
				Computed: true,
			},
			"allow_connections": schema.BoolAttribute{
				Computed: true,
			},
		},
	}
}

func (dbDs *DatabaseDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Info(ctx, "Executing Read Database Datasource")

	var state DatabaseDataSourceModel

	res.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if res.Diagnostics.HasError() {
		return
	}

	conn, err := dbDs.client.GetConnection(ctx)
	if err != nil {
		res.Diagnostics.AddError("Error establishing a PostgreSQL connection", err.Error())
		return
	}

	query := dbDs.client.GetDatabaseQuery(state.Name.ValueString())

	var (
		owner, comment, encoding, lcCollate, lcType string
		connectionLimit                             int64
		allowConnections                            bool
	)

	err = conn.QueryRowContext(ctx, query).Scan(&owner, &comment, &encoding, &lcCollate, &lcType, &connectionLimit, &allowConnections)
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf("Error reading database '%s'", state.Name), err.Error())
		return
	}

	state.Owner = types.StringValue(owner)
	state.Comment = types.StringValue(comment)
	state.Encoding = types.StringValue(encoding)
	state.LcCollate = types.StringValue(lcCollate)
	state.LcType = types.StringValue(lcType)
	state.ConnectionLimit = types.Int64Value(connectionLimit)
	state.AllowConnections = types.BoolValue(allowConnections)

	diags := res.State.Set(ctx, &state)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}
}
