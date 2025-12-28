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
	_ datasource.DataSource              = &postgresqlSchemasDataSource{}
	_ datasource.DataSourceWithConfigure = &postgresqlSchemasDataSource{}
)

func NewPostgresqlSchemasDataSource() datasource.DataSource {
	return &postgresqlSchemasDataSource{
		dsName: "postgresql_schemas",
	}
}

type postgresqlSchemasDataSource struct {
	dsName     string
	pgClient   pgclient.PostgresqlClient
	schemaRepo pgclient.SchemaRepo
}

type postgresqlSchemasDataSourceModel struct {
	Id                   types.String         `tfsdk:"id"`
	IncludeSystemSchemas types.Bool           `tfsdk:"include_system_schemas"`
	LikeAnyPatterns      []types.String       `tfsdk:"like_any_patterns"`
	LikeAllPatterns      []types.String       `tfsdk:"like_all_patterns"`
	NotLikeAllPatterns   []types.String       `tfsdk:"not_like_all_patterns"`
	RegexPattern         types.String         `tfsdk:"regex_pattern"`
	Schemas              []postgresqlSchemaDS `tfsdk:"schemas"`
}

type postgresqlSchemaDS struct {
	Name  types.String `tfsdk:"name"`
	Owner types.String `tfsdk:"owner"`
}

func (d *postgresqlSchemasDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, res *datasource.MetadataResponse) {
	res.TypeName = d.dsName
}

func (d *postgresqlSchemasDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "Lists PostgreSQL schemas with optional filtering. [PostgreSQL documentation](https://www.postgresql.org/docs/current/ddl-schemas.html)",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for this data source",
			},
			"include_system_schemas": schema.BoolAttribute{
				Description: "Include system schemas (pg_*, information_schema). Default is false.",
				Optional:    true,
			},
			"like_any_patterns": schema.ListAttribute{
				Description: "List of LIKE patterns. Schema name must match at least one pattern.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"like_all_patterns": schema.ListAttribute{
				Description: "List of LIKE patterns. Schema name must match all patterns.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"not_like_all_patterns": schema.ListAttribute{
				Description: "List of LIKE patterns. Schema name must not match any pattern.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"regex_pattern": schema.StringAttribute{
				Description: "PostgreSQL regex pattern to filter schema names.",
				Optional:    true,
			},
			"schemas": schema.ListNestedAttribute{
				Description: "List of schemas matching the filter criteria.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The name of the schema.",
							Computed:    true,
						},
						"owner": schema.StringAttribute{
							Description: "The owner of the schema.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *postgresqlSchemasDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
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

func (d *postgresqlSchemasDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Debug(ctx, fmt.Sprintf("reading '%s' data source", d.dsName))

	var model postgresqlSchemasDataSourceModel
	res.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	conn, err := d.pgClient.GetConnection(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	// Build filter params
	params := pgclient.SchemaListParams{
		IncludeSystemSchemas: model.IncludeSystemSchemas.ValueBool(),
		RegexPattern:         model.RegexPattern.ValueString(),
	}

	if len(model.LikeAnyPatterns) > 0 {
		params.LikeAnyPatterns = make([]string, len(model.LikeAnyPatterns))
		for i, p := range model.LikeAnyPatterns {
			params.LikeAnyPatterns[i] = p.ValueString()
		}
	}

	if len(model.LikeAllPatterns) > 0 {
		params.LikeAllPatterns = make([]string, len(model.LikeAllPatterns))
		for i, p := range model.LikeAllPatterns {
			params.LikeAllPatterns[i] = p.ValueString()
		}
	}

	if len(model.NotLikeAllPatterns) > 0 {
		params.NotLikeAllPatterns = make([]string, len(model.NotLikeAllPatterns))
		for i, p := range model.NotLikeAllPatterns {
			params.NotLikeAllPatterns[i] = p.ValueString()
		}
	}

	schemas, err := d.schemaRepo.List(ctx, conn, params)
	if err != nil {
		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFReadAction, PGSchema), err.Error())
		return
	}

	// Convert to model
	model.Schemas = make([]postgresqlSchemaDS, len(schemas))
	for i, s := range schemas {
		model.Schemas[i] = postgresqlSchemaDS{
			Name:  types.StringValue(s.Name.String),
			Owner: types.StringValue(s.Owner.String),
		}
	}

	model.Id = types.StringValue("schemas")

	res.Diagnostics.Append(res.State.Set(ctx, &model)...)
}
