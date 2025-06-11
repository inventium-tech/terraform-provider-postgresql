package provider

import (
	"context"
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
	"terraform-provider-postgresql/internal/helpers"
	"terraform-provider-postgresql/internal/pgclient"
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
	version string
}

// postgresqlProviderConfig describes the provider data model.
type postgresqlProviderConfig struct {
	Host     types.String `tfsdk:"host" validate:"required"`
	Port     types.Int64  `tfsdk:"port" validate:"required"`
	Username types.String `tfsdk:"username" validate:"required"`
	Password types.String `tfsdk:"password" validate:"required"`
	Database types.String `tfsdk:"database" validate:"required"`
	Scheme   types.String `tfsdk:"scheme" validate:"required"`
	SSLMode  types.String `tfsdk:"sslmode" validate:"required"`
}

func NewPostgresqlProvider(version string) func() provider.Provider {
	return func() provider.Provider {
		return &PostgresqlProvider{
			version: version,
		}
	}
}

func (p *PostgresqlProvider) Metadata(_ context.Context, _ provider.MetadataRequest, res *provider.MetadataResponse) {
	res.TypeName = "postgresql"
	res.Version = p.version
}

func (p *PostgresqlProvider) Schema(_ context.Context, _ provider.SchemaRequest, res *provider.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "The PostgreSQL Provider is used to manage PostgreSQL resources such as roles, databases, and event triggers.",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description: "The hostname of the PostgreSQL server. May be set via the environment variable `POSTGRES_HOST`.",
				Optional:    true,
			},
			"port": schema.Int64Attribute{
				Description: "The port of the PostgreSQL server. May be set via the environment variable `POSTGRES_PORT`. (default: 5432)",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"username": schema.StringAttribute{
				Description: "The username to use when connecting to the PostgreSQL server. May be set via the environment variable `POSTGRES_USERNAME`.",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "The password to use when connecting to the PostgreSQL server. May be set via the environment variable `POSTGRES_PASSWORD`.",
				Optional:    true,
				Sensitive:   true,
			},
			"database": schema.StringAttribute{
				Description: "The name of the PostgreSQL database to connect to. May be set via the environment variable `POSTGRES_DATABASE`.",
				Optional:    true,
			},
			"scheme": schema.StringAttribute{
				Optional: true,
				Description: `
The schema to use when connecting to the PostgreSQL database. The value must be one of the following:
	* 'postgres' (default)
	* 'gcppostgres'	
	* 'awspostgres'
May be set via the environment variable 'POSTGRES_SCHEMA'. (default: 'postgres')
				`,
				Validators: []validator.String{
					stringvalidator.OneOf("postgres", "gcppostgres", "awspostgres"),
				},
			},
			"sslmode": schema.StringAttribute{
				Description: `
The SSL mode to use when connecting to the PostgreSQL server. The value must be one of the following:
	* 'disable' (No SSL)
	* 'require' (*default*. Always SSL, skip verification)
	* 'verify-ca' (Always SSL, verify that the server certificate is issued by a trusted CA)
	* 'verify-full' (Always SSL, *same as 'verify-ca', plus the server host name matches the one in the certificate).
May be set via the environment variable 'POSTGRES_SSLMODE'. (default: 'disable')
				`,
				Optional: true,
				Validators: []validator.String{
					stringvalidator.OneOf("disable", "require", "verify-ca", "verify-full"),
				},
			},
		},
	}
}

func (p *PostgresqlProvider) Configure(ctx context.Context, req provider.ConfigureRequest, res *provider.ConfigureResponse) {
	tflog.Trace(ctx, "Configuring 'postgresql' Provider")

	var providerConfig postgresqlProviderConfig

	res.Diagnostics.Append(req.Config.Get(ctx, &providerConfig)...)
	if res.Diagnostics.HasError() {
		return
	}

	// Load provider configuration from current values or environment variables
	res.Diagnostics.Append(providerConfig.loadConfig()...)
	if res.Diagnostics.HasError() {
		return
	}

	validate := helpers.GetSafeValidator()
	if err := validate.Struct(providerConfig); err != nil {
		res.Diagnostics.AddError("Invalid provider configuration", err.Error())
		return
	}

	ctx = tflog.SetField(ctx, providerAttrHost, providerConfig.Host.ValueString())
	ctx = tflog.SetField(ctx, providerAttrUsername, providerConfig.Username.ValueString())
	ctx = tflog.SetField(ctx, providerAttrDatabase, providerConfig.Database.ValueString())

	tflog.Trace(ctx, "Creating 'postgresql' client")
	pgConnConfig := &pgclient.ConnConfig{
		Host:     providerConfig.Host.ValueString(),
		Port:     int(providerConfig.Port.ValueInt64()),
		Username: providerConfig.Username.ValueString(),
		Password: providerConfig.Password.ValueString(),
		Database: providerConfig.Database.ValueString(),
		SSLMode:  providerConfig.SSLMode.ValueString(),
	}
	pgClient, err := pgclient.NewPostgresqlClient(ctx, pgConnConfig)
	if err != nil {
		res.Diagnostics.AddError("Failed to create Postgres client", err.Error())
		return
	}
	res.DataSourceData = pgClient
	res.ResourceData = pgClient
	tflog.Trace(ctx, "Successfully configured 'postgresql' Provider with the respective client")
}

func (p *PostgresqlProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewPostgresqlEventTriggerResource,
		NewPostgresqlRoleResource,
		NewPostgresqlUserFunctionResource,
	}
}

func (p *PostgresqlProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewPostgresqlEventTriggerDataSource,
	}
}

// loadConfig loads the provider configuration from the given context.
// Read and parse the expected environment variables to assign to the postgresqlProviderConfig, but only if the
// respective attribute is empty or not set.
//
// Returns:
// - None.
func (c *postgresqlProviderConfig) loadConfig() diag.Diagnostics {
	diags := diag.Diagnostics{}

	var envVarValue string
	var ok bool
	var err error

	if envVarValue, ok = os.LookupEnv("POSTGRES_HOST"); c.Host.IsNull() && ok {
		c.Host = types.StringValue(envVarValue)
	}

	portValue := 5432
	if envVarValue, ok = os.LookupEnv("POSTGRES_PORT"); c.Port.IsNull() && ok {
		portValue, err = strconv.Atoi(envVarValue)
		if err != nil {
			diags.AddAttributeError(path.Root(providerAttrPort), "failed to parse environment variable 'POSTGRES_PORT' to int", err.Error())
			return diags
		}
	}
	c.Port = types.Int64Value(int64(portValue))

	if envVarValue, ok = os.LookupEnv("POSTGRES_USER"); c.Username.IsNull() && ok {
		c.Username = types.StringValue(envVarValue)
	}

	if envVarValue, ok = os.LookupEnv("POSTGRES_PASSWORD"); c.Password.IsNull() && ok {
		c.Password = types.StringValue(envVarValue)
	}

	if envVarValue, ok = os.LookupEnv("POSTGRES_DATABASE"); c.Database.IsNull() && ok {
		c.Database = types.StringValue(envVarValue)
	}

	schemeValue := "postgres"
	if envVarValue, ok = os.LookupEnv("POSTGRES_SCHEME"); c.Scheme.IsNull() && ok {
		schemeValue = envVarValue
	}
	c.Scheme = types.StringValue(schemeValue)

	sslModeValue := "require"
	if envVarValue, ok = os.LookupEnv("POSTGRES_SSLMODE"); c.SSLMode.IsNull() && ok {
		sslModeValue = envVarValue
	}
	c.SSLMode = types.StringValue(sslModeValue)

	return diags
}
