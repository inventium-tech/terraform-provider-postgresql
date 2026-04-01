package provider

import (
	"context"
	"fmt"

	"terraform-provider-postgresql/internal/pgclient"
	"terraform-provider-postgresql/internal/provider/validators"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &postgresqlExtensionDataSource{}
	_ datasource.DataSourceWithConfigure = &postgresqlExtensionDataSource{}
)

func NewPostgresqlExtensionDataSource() datasource.DataSource {
	return &postgresqlExtensionDataSource{
		dsName: "postgresql_extension",
	}
}

type postgresqlExtensionDataSource struct {
	dsName        string
	pgClient      pgclient.PostgresqlClient
	extensionRepo pgclient.ExtensionRepo
}

type postgresqlExtensionDataSourceModel struct {
	Id       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Schema   types.String `tfsdk:"schema"`
	Version  types.String `tfsdk:"version"`
	Database types.String `tfsdk:"database"`
}

func (d *postgresqlExtensionDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, res *datasource.MetadataResponse) {
	res.TypeName = d.dsName
}

func (d *postgresqlExtensionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "Retrieves information about a PostgreSQL extension. [PostgreSQL documentation](https://www.postgresql.org/docs/current/sql-createextension.html)",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for the extension",
			},
			"name": schema.StringAttribute{
				Description: "The name of the extension to query.",
				Required:    true,
			},
			"schema": schema.StringAttribute{
				Description: "The schema in which the extension is installed.",
				Computed:    true,
			},
			"version": schema.StringAttribute{
				Description: "The version of the extension.",
				Computed:    true,
			},
			"database": schema.StringAttribute{
				Description: "The database to query for the extension. Defaults to the provider's configured database.",
				Optional:    true,
				Validators: []validator.String{
					validators.PostgresqlObjectName(),
				},
			},
		},
	}
}

func (d *postgresqlExtensionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
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

	d.pgClient = pgClient
	d.extensionRepo = pgclient.NewExtensionRepo()
}

func (d *postgresqlExtensionDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Debug(ctx, fmt.Sprintf("reading '%s' data source", d.dsName))

	var model postgresqlExtensionDataSourceModel
	res.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	var conn pgclient.DBTX
	var err error

	if !model.Database.IsNull() && model.Database.ValueString() != "" {
		conn, err = d.pgClient.GetConnection(ctx, model.Database.ValueString())
	} else {
		conn, err = d.pgClient.GetConnection(ctx)
	}
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	extModel, err := d.extensionRepo.GetOne(ctx, conn, model.Name.ValueString())
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFReadAction, "extension"), err.Error())
		return
	}

	model.Id = types.StringValue(extModel.Name.String)
	model.Name = types.StringValue(extModel.Name.String)
	model.Schema = types.StringValue(extModel.Schema.String)
	model.Version = types.StringValue(extModel.Version.String)

	res.Diagnostics.Append(res.State.Set(ctx, &model)...)
}
