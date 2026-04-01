package provider

import (
	"context"
	"errors"
	"fmt"

	"terraform-provider-postgresql/internal/pgclient"
	"terraform-provider-postgresql/internal/provider/validators"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jackc/pgx/v5"
)

var (
	_ datasource.DataSource              = &postgresqlRoleDataSource{}
	_ datasource.DataSourceWithConfigure = &postgresqlRoleDataSource{}
)

func NewPostgresqlRoleDataSource() datasource.DataSource {
	return &postgresqlRoleDataSource{
		dsName: "postgresql_role",
	}
}

type postgresqlRoleDataSource struct {
	dsName   string
	pgClient pgclient.PostgresqlClient
	roleRepo pgclient.RoleRepo
}

func (d *postgresqlRoleDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, res *datasource.MetadataResponse) {
	res.TypeName = d.dsName
}

func (d *postgresqlRoleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "Retrieves information about a PostgreSQL role. [PostgreSQL documentation](https://www.postgresql.org/docs/current/sql-createrole.html)",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for the role.",
			},
			"name": schema.StringAttribute{
				Description: "The name of the role to query.",
				Required:    true,
				Validators: []validator.String{
					validators.PostgresqlObjectName(),
				},
			},
			"superuser": schema.BoolAttribute{
				Description: "Whether the role is a superuser.",
				Computed:    true,
			},
			"inherit": schema.BoolAttribute{
				Description: "Whether the role inherits privileges from roles it is a member of.",
				Computed:    true,
			},
			"create_role": schema.BoolAttribute{
				Description: "Whether the role can create new roles.",
				Computed:    true,
			},
			"create_db": schema.BoolAttribute{
				Description: "Whether the role can create new databases.",
				Computed:    true,
			},
			"login": schema.BoolAttribute{
				Description: "Whether the role can log in. When set, acts as a filter to distinguish login roles (users) from group roles.",
				Optional:    true,
				Computed:    true,
			},
			"replication": schema.BoolAttribute{
				Description: "Whether the role can initiate streaming replication or control backup mode.",
				Computed:    true,
			},
			"bypass_rls": schema.BoolAttribute{
				Description: "Whether the role bypasses all row-level security (RLS) policies.",
				Computed:    true,
			},
			"connection_limit": schema.Int32Attribute{
				Description: "Maximum concurrent connections for the role. -1 means no limit.",
				Computed:    true,
			},
			"valid_until": schema.StringAttribute{
				Description: "Timestamp after which the role's password is invalid. May be null for no expiration.",
				Computed:    true,
			},
			"comment": schema.StringAttribute{
				Description: "Comment associated with the role.",
				Computed:    true,
			},
			"role": schema.SetAttribute{
				Description: "Roles this role belongs to (membership without admin option).",
				ElementType: types.StringType,
				Computed:    true,
			},
			"admin": schema.SetAttribute{
				Description: "Roles this role can administer (membership with admin option).",
				ElementType: types.StringType,
				Computed:    true,
			},
		},
	}
}

func (d *postgresqlRoleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, res *datasource.ConfigureResponse) {
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
	d.roleRepo = pgclient.NewRoleRepo()
}

func (d *postgresqlRoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	tflog.Debug(ctx, fmt.Sprintf("reading '%s' data source", d.dsName))

	var model postgresqlRoleDataSourceModel
	res.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	conn, err := d.pgClient.GetConnection(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	roleModel, err := d.roleRepo.GetOne(ctx, conn, model.Name.ValueString())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			res.Diagnostics.AddAttributeError(
				path.Root("name"),
				fmt.Sprintf(msgErrorPgObjectNotFund, PGRole),
				fmt.Sprintf(msgErrorPgObjectNotFundDetail, PGRole, model.Name.ValueString()),
			)
			return
		}

		res.Diagnostics.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFReadAction, PGRole), err.Error())
		return
	}

	if !matchesLoginFilter(model.Login, roleModel.Login.Bool) {
		res.Diagnostics.AddAttributeError(
			path.Root("login"),
			"Login filter mismatch",
			fmt.Sprintf("Role %q has login=%t which does not match requested login=%t.", model.Name.ValueString(), roleModel.Login.Bool, model.Login.ValueBool()),
		)
		return
	}

	model.Id = types.StringValue(roleModel.Name.String)
	model.Name = types.StringValue(roleModel.Name.String)
	model.Superuser = types.BoolValue(roleModel.Superuser.Bool)
	model.Inherit = types.BoolValue(roleModel.Inherit.Bool)
	model.CreateRole = types.BoolValue(roleModel.CreateRole.Bool)
	model.CreateDB = types.BoolValue(roleModel.CreateDB.Bool)
	model.Login = types.BoolValue(roleModel.Login.Bool)
	model.Replication = types.BoolValue(roleModel.Replication.Bool)
	model.BypassRLS = types.BoolValue(roleModel.BypassRLS.Bool)
	if roleModel.Comment.Valid {
		model.Comment = types.StringValue(roleModel.Comment.String)
	} else {
		model.Comment = types.StringNull()
	}

	if roleModel.ConnectionLimit.Valid {
		model.ConnectionLimit = types.Int32Value(roleModel.ConnectionLimit.Int32)
	} else {
		model.ConnectionLimit = types.Int32Null()
	}

	if roleModel.ValidUntil.Valid {
		model.ValidUntil = types.StringValue(roleModel.ValidUntil.String)
	} else {
		model.ValidUntil = types.StringNull()
	}

	res.Diagnostics.Append(parseSliceIntoStringSet(ctx, &model.Role, roleModel.Roles)...)
	res.Diagnostics.Append(parseSliceIntoStringSet(ctx, &model.Admin, roleModel.AdminRoles)...)

	if res.Diagnostics.HasError() {
		return
	}

	res.Diagnostics.Append(res.State.Set(ctx, &model)...)
}

func matchesLoginFilter(filter types.Bool, actual bool) bool {
	if filter.IsNull() || filter.IsUnknown() {
		return true
	}

	return filter.ValueBool() == actual
}
