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
	_ datasource.DataSource              = &postgresqlSchemaDataSource{}
	_ datasource.DataSourceWithConfigure = &postgresqlSchemaDataSource{}
)

func NewPostgresqlSchemaDataSource() datasource.DataSource {
	return &postgresqlSchemaDataSource{
		dsName: "postgresql_schema",
	}
}

type postgresqlSchemaDataSource struct {
	dsName     string
	pgClient   pgclient.PostgresqlClient
	schemaRepo pgclient.SchemaRepo
}

type postgresqlSchemaDataSourceModel struct {
	Id    types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	Owner types.String `tfsdk:"owner"`
}

func (d *postgresqlSchemaDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, res *datasource.MetadataResponse) {
	res.TypeName = d.dsName
}

func (d *postgresqlSchemaDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "Retrieves information about a PostgreSQL schema. [PostgreSQL documentation](https://www.postgresql.org/docs/current/ddl-schemas.html)",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for the schema",
			},
			"name": schema.StringAttribute{
				Description: "The name of the schema to query.",
				Required:    true,
				Validators: []validator.String{
					validators.PostgresqlObjectName(),
				},
			},
			"owner": schema.StringAttribute{
				Description: "The role that owns the schema.",
				Computed:    true,
			},
		},
	}
}

func (d *postgresqlSchemaDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
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
	d.schemaRepo = pgclient.NewSchemaRepo()
}

func (d *postgresqlSchemaDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Debug(ctx, fmt.Sprintf("reading '%s' data source", d.dsName))

	var model postgresqlSchemaDataSourceModel
	res.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	conn, err := d.pgClient.GetConnection(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	schemaModel, err := d.schemaRepo.GetOne(ctx, conn, model.Name.ValueString())
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFReadAction, PGSchema), err.Error())
		return
	}

	model.Id = types.StringValue(schemaModel.Name.String)
	model.Name = types.StringValue(schemaModel.Name.String)
	model.Owner = types.StringValue(schemaModel.Owner.String)

	res.Diagnostics.Append(res.State.Set(ctx, &model)...)
}
