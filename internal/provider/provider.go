package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"os"
	"strconv"
	"terraform-provider-postgresql/internal/client"
)

const (
	providerAttrHost        = "host"
	providerAttrPort        = "port"
	providerAttrUsername    = "username"
	providerAttrPassword    = "password"
	providerAttrDatabase    = "database"
	providerAttrSchema      = "scheme"
	providerAttrSSLMode     = "sslmode"
	providerAttrMaxOpenConn = "max_open_conn"
	providerAttrMaxIdleConn = "max_idle_conn"
)

// Ensure PostgresqlProvider satisfies various provider interfaces.
var _ provider.Provider = &PostgresqlProvider{}

// PostgresqlProvider defines the provider implementation.
type PostgresqlProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version  string
	pgClient client.PgClient //nolint:unused
}

// PostgresqlProviderConfig describes the provider data model.
type PostgresqlProviderConfig struct {
	Host        types.String `tfsdk:"host" validate:"required"`
	Port        types.Int64  `tfsdk:"port" validate:"required"`
	Username    types.String `tfsdk:"username" validate:"required"`
	Password    types.String `tfsdk:"password" validate:"required"`
	Database    types.String `tfsdk:"database" validate:"required"`
	Scheme      types.String `tfsdk:"scheme" validate:"required"`
	SSLMode     types.String `tfsdk:"sslmode" validate:"required"`
	MaxOpenConn types.Int64  `tfsdk:"max_open_conn" validate:"required"`
	MaxIdleConn types.Int64  `tfsdk:"max_idle_conn" validate:"required"`
}

func (p *PostgresqlProvider) Metadata(_ context.Context, _ provider.MetadataRequest, res *provider.MetadataResponse) {
	res.TypeName = "postgresql"
	res.Version = p.version
}

func (p *PostgresqlProvider) Schema(_ context.Context, _ provider.SchemaRequest, res *provider.SchemaResponse) {
	res.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			providerAttrHost: schema.StringAttribute{
				MarkdownDescription: "The hostname of the PostgreSQL server. Default is 5432)",
				Optional:            true,
			},
			providerAttrPort: schema.Int64Attribute{
				MarkdownDescription: "The port of the PostgreSQL server.",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			providerAttrUsername: schema.StringAttribute{
				MarkdownDescription: "The username to use when connecting to the PostgreSQL server.",
				Optional:            true,
			},
			providerAttrPassword: schema.StringAttribute{
				MarkdownDescription: "The password to use when connecting to the PostgreSQL server.",
				Optional:            true,
				Sensitive:           true,
			},
			providerAttrDatabase: schema.StringAttribute{
				MarkdownDescription: "The name of the PostgreSQL database to connect to.",
				Optional:            true,
			},
			providerAttrSchema: schema.StringAttribute{
				Optional: true,
				MarkdownDescription: `
The schema to use when connecting to the PostgreSQL database. The value must be one of the following:
	* 'postgres'
	* 'gcppostgres'	
	* 'awspostgres'	
				`,
				Validators: []validator.String{
					stringvalidator.OneOf("postgres", "gcppostgres", "awspostgres"),
				},
			},
			providerAttrSSLMode: schema.StringAttribute{
				MarkdownDescription: `
The SSL mode to use when connecting to the PostgreSQL server. The value must be one of the following:
	* 'disable' (No SSL)
	* 'require' (*default*. Always SSL, skip verification)
	* 'verify-ca' (Always SSL, verify that the server certificate is issued by a trusted CA)
	* 'verify-full' (Always SSL, *same as 'verify-ca', plus the server host name matches the one in the certificate)
				`,
				Optional: true,
				Validators: []validator.String{
					stringvalidator.OneOf("disable", "require", "verify-ca", "verify-full"),
				},
			},
			providerAttrMaxOpenConn: schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Maximum number of open connections to the database. Default is 0.",
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
			},
			providerAttrMaxIdleConn: schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Maximum number of idle connections to the database. Default is 5.",
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
			},
		},
		MarkdownDescription: mdDocProviderOverview,
	}
}

func (p *PostgresqlProvider) Configure(ctx context.Context, req provider.ConfigureRequest, res *provider.ConfigureResponse) {
	tflog.Trace(ctx, "Configuring 'postgresql' Provider")

	var providerConfig PostgresqlProviderConfig

	res.Diagnostics.Append(req.Config.Get(ctx, &providerConfig)...)
	if res.Diagnostics.HasError() {
		return
	}

	// Load provider configuration from current values or environment variables
	res.Diagnostics.Append(providerConfig.loadConfig()...)
	if res.Diagnostics.HasError() {
		return
	}

	validate := client.GetValidatorFromCtx(ctx)
	if err := validate.Struct(providerConfig); err != nil {
		res.Diagnostics.AddError("Invalid provider configuration", err.Error())
		return
	}

	ctx = tflog.SetField(ctx, providerAttrHost, providerConfig.Host.ValueString())
	ctx = tflog.SetField(ctx, providerAttrUsername, providerConfig.Username.ValueString())
	ctx = tflog.SetField(ctx, providerAttrDatabase, providerConfig.Database.ValueString())

	tflog.Trace(ctx, "Creating 'postgresql' client")
	config, diags := providerConfig.toPgConnectionOpts(ctx)
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	pgClient := client.NewPgClient(*config)
	res.DataSourceData = pgClient
	res.ResourceData = pgClient
	tflog.Trace(ctx, "Successfully configured 'postgresql' Provider with the respective client")
}

func (p *PostgresqlProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewEventTriggerResource,
	}
}

func (p *PostgresqlProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewEventTriggerDataSource,
	}
}

func NewProvider(version string) func() provider.Provider {
	return func() provider.Provider {
		return &PostgresqlProvider{
			version: version,
		}
	}
}

// loadConfig loads the provider configuration from the given context.
// Read and parse the expected environment variables to assign to the PostgresqlProviderConfig, but only if the
// respective attribute is empty or not set.
//
// Returns:
// - None.
func (c *PostgresqlProviderConfig) loadConfig() diag.Diagnostics {
	diags := diag.Diagnostics{}

	if c.Host.IsNull() {
		c.Host = types.StringValue(os.Getenv("POSTGRES_HOST"))
	}
	if c.Port.IsNull() {
		portValue := types.Int64Value(int64(5432))
		if value := os.Getenv("POSTGRES_PORT"); value != "" {
			port, err := strconv.Atoi(value)
			if err != nil {
				diags.AddAttributeError(path.Root(providerAttrPort), "Failed to parse environment variable 'POSTGRES_PORT' to int", err.Error())
			}
			portValue = types.Int64Value(int64(port))
		}
		c.Port = portValue
	}
	if c.Username.IsNull() {
		c.Username = types.StringValue(os.Getenv("POSTGRES_USER"))
	}
	if c.Password.IsNull() {
		c.Password = types.StringValue(os.Getenv("POSTGRES_PASSWORD"))
	}
	if c.Database.IsNull() {
		c.Database = types.StringValue(os.Getenv("POSTGRES_DATABASE"))
	}
	if c.Scheme.IsNull() {
		schemeValue := types.StringValue("postgres")
		if value := os.Getenv("POSTGRES_SCHEME"); value != "" {
			schemeValue = types.StringValue(value)
		}
		c.Scheme = schemeValue
	}
	if c.SSLMode.IsNull() {
		sslModeValue := types.StringValue("require") // default value
		if value := os.Getenv("POSTGRES_SSLMODE"); value != "" {
			sslModeValue = types.StringValue(value)
		}
		c.SSLMode = sslModeValue
	}

	if c.MaxOpenConn.IsNull() {
		defaultMaxOpenConn := types.Int64Value(int64(0))
		if value := os.Getenv("POSTGRES_MAX_OPEN_CONN"); value != "" {
			tflog.Info(context.Background(), fmt.Sprintf("\n\n\nPOSTGRES_MAX_OPEN_CONN: %s\n\n\n", value))
			maxOpenConn, err := strconv.Atoi(value)
			if err != nil {
				diags.AddAttributeError(path.Root(providerAttrMaxOpenConn), "Failed to parse environment variable 'POSTGRES_MAX_OPEN_CONN' to int", err.Error())
				return diags
			}
			defaultMaxOpenConn = types.Int64Value(int64(maxOpenConn))
		}
		c.MaxOpenConn = defaultMaxOpenConn
	}

	if c.MaxIdleConn.IsNull() {
		defaultMaxIdleConn := types.Int64Value(int64(5))
		if value := os.Getenv("POSTGRES_MAX_IDLE_CONN"); value != "" {
			maxIdleConn, err := strconv.Atoi(value)
			if err != nil {
				diags.AddAttributeError(path.Root(providerAttrMaxIdleConn), "Failed to parse environment variable 'POSTGRES_MAX_IDLE_CONN' to int", err.Error())
				return diags
			}
			defaultMaxIdleConn = types.Int64Value(int64(maxIdleConn))
		}
		c.MaxIdleConn = defaultMaxIdleConn
	}

	return diags
}

func (c *PostgresqlProviderConfig) toPgConnectionOpts(ctx context.Context) (*client.PgConnectionOpts, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	opts, err := client.NewPgConnectionOpts(
		ctx,
		client.WithHost(c.Host.ValueString()),
		client.WithPort(int(c.Port.ValueInt64())),
		client.WithUsername(c.Username.ValueString()),
		client.WithPassword(c.Password.ValueString()),
		client.WithDatabase(c.Database.ValueString()),
		client.WithScheme(c.Scheme.ValueString()),
		client.WithSSLMode(c.SSLMode.ValueString()),
		client.WithMaxOpenConn(int(c.MaxOpenConn.ValueInt64())),
		client.WithMaxIdleConn(int(c.MaxIdleConn.ValueInt64())),
	)
	if err != nil {
		tflog.Debug(ctx, fmt.Sprintf("\n\n\nFailed to create Postgres client connection options. Error: %v\n\n\n", err))
		diags.AddError("Failed to create Postgres client connection options", err.Error())
	}

	return opts, diags
}
