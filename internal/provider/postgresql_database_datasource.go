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
	_ datasource.DataSource              = &postgresqlDatabaseDataSource{}
	_ datasource.DataSourceWithConfigure = &postgresqlDatabaseDataSource{}
)

func NewPostgresqlDatabaseDataSource() datasource.DataSource {
	return &postgresqlDatabaseDataSource{
		dsName: "postgresql_database",
	}
}

type postgresqlDatabaseDataSource struct {
	dsName       string
	pgClient     pgclient.PostgresqlClient
	databaseRepo pgclient.DatabaseRepo
}

type postgresqlDatabaseDataSourceModel struct {
	Id               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Owner            types.String `tfsdk:"owner"`
	Encoding         types.String `tfsdk:"encoding"`
	Collation        types.String `tfsdk:"collation"`
	Ctype            types.String `tfsdk:"ctype"`
	IsTemplate       types.Bool   `tfsdk:"is_template"`
	AllowConnections types.Bool   `tfsdk:"allow_connections"`
	ConnectionLimit  types.Int32  `tfsdk:"connection_limit"`
	Tablespace       types.String `tfsdk:"tablespace"`
	Comment          types.String `tfsdk:"comment"`
}

func (d *postgresqlDatabaseDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, res *datasource.MetadataResponse) {
	res.TypeName = d.dsName
}

func (d *postgresqlDatabaseDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "Retrieves information about a PostgreSQL database. [PostgreSQL documentation](https://www.postgresql.org/docs/current/manage-ag-overview.html)",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for the database",
			},
			"name": schema.StringAttribute{
				Description: "The name of the database to query.",
				Required:    true,
				Validators: []validator.String{
					validators.PostgresqlObjectName(),
				},
			},
			"owner": schema.StringAttribute{
				Description: "The role that owns the database.",
				Computed:    true,
			},
			"encoding": schema.StringAttribute{
				Description: "Character set encoding of the database.",
				Computed:    true,
			},
			"collation": schema.StringAttribute{
				Description: "Collation order (LC_COLLATE) of the database.",
				Computed:    true,
			},
			"ctype": schema.StringAttribute{
				Description: "Character classification (LC_CTYPE) of the database.",
				Computed:    true,
			},
			"is_template": schema.BoolAttribute{
				Description: "Whether this database is a template database.",
				Computed:    true,
			},
			"allow_connections": schema.BoolAttribute{
				Description: "Whether connections are allowed to this database.",
				Computed:    true,
			},
			"connection_limit": schema.Int32Attribute{
				Description: "Maximum concurrent connections to the database. -1 means no limit.",
				Computed:    true,
			},
			"tablespace": schema.StringAttribute{
				Description: "The tablespace for the database.",
				Computed:    true,
			},
			"comment": schema.StringAttribute{
				Description: "The comment for the database.",
				Computed:    true,
			},
		},
	}
}

func (d *postgresqlDatabaseDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
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
	d.databaseRepo = pgclient.NewDatabaseRepo()
}

func (d *postgresqlDatabaseDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Debug(ctx, fmt.Sprintf("reading '%s' data source", d.dsName))

	var model postgresqlDatabaseDataSourceModel
	res.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	conn, err := d.pgClient.GetConnection(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	dbModel, err := d.databaseRepo.GetOne(ctx, conn, model.Name.ValueString())
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFReadAction, PGDatabase), err.Error())
		return
	}

	model.Id = types.StringValue(dbModel.Name.String)
	model.Name = types.StringValue(dbModel.Name.String)
	model.Owner = types.StringValue(dbModel.Owner.String)
	model.Encoding = types.StringValue(dbModel.Encoding.String)
	model.Collation = types.StringValue(dbModel.Collation.String)
	model.Ctype = types.StringValue(dbModel.Ctype.String)
	model.IsTemplate = types.BoolValue(dbModel.IsTemplate.Bool)
	model.AllowConnections = types.BoolValue(dbModel.AllowConnections.Bool)
	model.ConnectionLimit = types.Int32Value(dbModel.ConnectionLimit.Int32)
	model.Tablespace = types.StringValue(dbModel.Tablespace.String)
	model.Comment = types.StringValue(dbModel.Comment.String)

	res.Diagnostics.Append(res.State.Set(ctx, &model)...)
}
