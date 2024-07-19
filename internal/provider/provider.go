package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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
var _ provider.ProviderWithFunctions = &PostgresqlProvider{}

// PostgresqlProvider defines the provider implementation.
type PostgresqlProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
	client  client.PGClient
}

// PostgresqlProviderConfig describes the provider data model.
type PostgresqlProviderConfig struct {
	Host        types.String `tfsdk:"host"`
	Port        types.Int64  `tfsdk:"port"`
	Username    types.String `tfsdk:"username"`
	Password    types.String `tfsdk:"password"`
	Database    types.String `tfsdk:"database"`
	Scheme      types.String `tfsdk:"scheme"`
	SSLMode     types.String `tfsdk:"sslmode"`
	MaxOpenConn types.Int64  `tfsdk:"max_open_conn"`
	MaxIdleConn types.Int64  `tfsdk:"max_idle_conn"`
}

type providerAttrValidation int

const (
	ProValUnknown providerAttrValidation = iota + 1
	ProValMissing
)

func (p *PostgresqlProvider) Metadata(ctx context.Context, req provider.MetadataRequest, res *provider.MetadataResponse) {
	res.TypeName = "postgresql"
	res.Version = p.version
}

func (p *PostgresqlProvider) Schema(ctx context.Context, req provider.SchemaRequest, res *provider.SchemaResponse) {
	res.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			providerAttrHost: schema.StringAttribute{
				Description:         "The hostname of the PostgreSQL server.",
				MarkdownDescription: "The hostname of the PostgreSQL server.",
				Optional:            true,
			},
			providerAttrPort: schema.Int64Attribute{
				Description:         "The port of the PostgreSQL server.",
				MarkdownDescription: "The port of the PostgreSQL server.",
				Optional:            true,
			},
			providerAttrUsername: schema.StringAttribute{
				Description:         "The username to use when connecting to the PostgreSQL server.",
				MarkdownDescription: "The username to use when connecting to the PostgreSQL server.",
				Optional:            true,
			},
			providerAttrPassword: schema.StringAttribute{
				Description:         "The password to use when connecting to the PostgreSQL server.",
				MarkdownDescription: "The password to use when connecting to the PostgreSQL server.",
				Optional:            true,
				Sensitive:           true,
			},
			providerAttrDatabase: schema.StringAttribute{
				Description:         "The name of the PostgreSQL database to connect to.",
				MarkdownDescription: "The name of the PostgreSQL database to connect to.",
				Optional:            true,
			},
			providerAttrSchema: schema.StringAttribute{
				Description: `The schema to use when connecting to the PostgreSQL database.
				The value must be one of the following:
				- 'postgres'
				- 'gcppostgres'
				- 'awspostgres'
				`,
				MarkdownDescription: `The schema to use when connecting to the PostgreSQL database.
				The value must be one of the following:
				- 'postgres'
				- 'gcppostgres'
				- 'awspostgres'	
				`,
				Optional: true,
			},
			providerAttrSSLMode: schema.StringAttribute{
				Description: `The SSL mode to use when connecting to the PostgreSQL server.
				The value must be one of the following:
				- 'disable' (No SSL)
				- 'require' (*default*. Always SSL, skip verification)
				- 'verify-ca' (Always SSL, verify that the server certificate is issued by a trusted CA)
				- 'verify-full' (Always SSL, *same as 'verify-ca', plus the server host name matches the one in the certificate)
				`,
				MarkdownDescription: `The SSL mode to use when connecting to the PostgreSQL server.
				The value must be one of the following:
				- 'disable' (No SSL)
				- 'require' (*default*. Always SSL, skip verification)
				- 'verify-ca' (Always SSL, verify that the server certificate is issued by a trusted CA)
				- 'verify-full' (Always SSL, *same as 'verify-ca', plus the server host name matches the one in the certificate)
				`,
				Optional: true,
			},
			providerAttrMaxOpenConn: schema.Int64Attribute{
				Optional:            true,
				Description:         "Maximum number of open connections to the database. Default is 0.",
				MarkdownDescription: "Maximum number of open connections to the database. Default is 0.",
			},
			providerAttrMaxIdleConn: schema.Int64Attribute{
				Optional:            true,
				Description:         "Maximum number of idle connections to the database. Default is 5.",
				MarkdownDescription: "Maximum number of idle connections to the database. Default is 5.",
			},
		},
	}
}

func (p *PostgresqlProvider) Configure(ctx context.Context, req provider.ConfigureRequest, res *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring 'postgresql' Provider")

	var providerConfig PostgresqlProviderConfig

	res.Diagnostics.Append(req.Config.Get(ctx, &providerConfig)...)
	if res.Diagnostics.HasError() {
		return
	}

	// Validate unknown attrs from the provider configuration
	res.Diagnostics.Append(providerConfig.validateFields(ProValUnknown)...)
	if res.Diagnostics.HasError() {
		return
	}

	// Load provider configuration from current values or environment variables
	res.Diagnostics.Append(providerConfig.loadConfig()...)
	if res.Diagnostics.HasError() {
		return
	}

	// Validate missing attrs from the provider configuration
	res.Diagnostics.Append(providerConfig.validateFields(ProValMissing)...)
	if res.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, providerAttrHost, providerConfig.Host.ValueString())
	ctx = tflog.SetField(ctx, providerAttrUsername, providerConfig.Username.ValueString())
	ctx = tflog.SetField(ctx, providerAttrPassword, providerConfig.Password.ValueString())
	ctx = tflog.SetField(ctx, providerAttrDatabase, providerConfig.Database.ValueString())
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, providerAttrPassword)

	tflog.Debug(ctx, "Creating 'postgresql' client")
	config, diags := providerConfig.transposeToPGClientConfig()
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	pgClient := client.NewPGClient(config)
	res.DataSourceData = pgClient
	res.ResourceData = pgClient
	ctx = context.WithValue(ctx, "pgclient", pgClient)
}

func (p *PostgresqlProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewEventTriggerResource,
	}
}

func (p *PostgresqlProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDatabaseDataSource,
		NewEventTriggerDataSource,
	}
}

func (p *PostgresqlProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func NewProvider(version string) func() provider.Provider {
	return func() provider.Provider {
		return &PostgresqlProvider{
			version: version,
		}
	}
}

func (pav providerAttrValidation) String() string {
	switch pav {
	case ProValMissing:
		return "Missing or Empty"
	case ProValUnknown:
		return "Unknown"
	default:
		return "Unknown"
	}
}

// validateFields checks for unknown attributes in the PostgresqlProviderConfig and reports them.
// It iterates over the model's attributes, identifying any that are marked as unknown. For each unknown attribute,
// it generates a diagnostic error.
//
// Returns:
// - diag.Diagnostics: Contains diagnostic information for each unknown attribute found.
func (ppm *PostgresqlProviderConfig) validateFields(valType providerAttrValidation) diag.Diagnostics {
	tmplSummary := fmt.Sprintf("%s Provider attribute '%s'", valType.String(), "%s")
	tmplDetail := fmt.Sprintf("The Provider cannot Create a PostgreSQL Client as the attribute '%s' is currently %s.\n"+
		"Either set the value statically in the configuration, or use the '%s' environment variable.", "%s", valType.String(), "%s")

	diags := diag.Diagnostics{}

	if (valType == ProValUnknown && ppm.Host.IsUnknown()) || (valType == ProValMissing && ppm.Host.IsNull()) {
		diags.AddAttributeError(
			path.Root(providerAttrHost),
			fmt.Sprintf(tmplSummary, providerAttrHost),
			fmt.Sprintf(tmplDetail, providerAttrHost, "POSTGRES_HOST"),
		)
	}
	if (valType == ProValUnknown && ppm.Port.IsUnknown()) || (valType == ProValMissing && ppm.Port.IsNull()) {
		diags.AddAttributeError(
			path.Root(providerAttrHost),
			fmt.Sprintf(tmplSummary, providerAttrHost),
			fmt.Sprintf(tmplDetail, providerAttrHost, "POSTGRES_PORT"),
		)
	}
	if (valType == ProValUnknown && ppm.Username.IsUnknown()) || (valType == ProValMissing && ppm.Username.IsNull()) {
		diags.AddAttributeError(
			path.Root(providerAttrHost),
			fmt.Sprintf(tmplSummary, providerAttrHost),
			fmt.Sprintf(tmplDetail, providerAttrHost, "POSTGRES_USER"),
		)
	}
	if (valType == ProValUnknown && ppm.Password.IsUnknown()) || (valType == ProValMissing && ppm.Password.IsNull()) {
		diags.AddAttributeError(
			path.Root(providerAttrHost),
			fmt.Sprintf(tmplSummary, providerAttrHost),
			fmt.Sprintf(tmplDetail, providerAttrHost, "POSTGRES_PASSWORD"),
		)
	}
	if (valType == ProValUnknown && ppm.Database.IsUnknown()) || (valType == ProValMissing && ppm.Database.IsNull()) {
		diags.AddAttributeError(
			path.Root(providerAttrHost),
			fmt.Sprintf(tmplSummary, providerAttrHost),
			fmt.Sprintf(tmplDetail, providerAttrHost, "POSTGRES_DATABASE"),
		)
	}
	if (valType == ProValUnknown && ppm.Scheme.IsUnknown()) || (valType == ProValMissing && ppm.Scheme.IsNull()) {
		diags.AddAttributeError(
			path.Root(providerAttrHost),
			fmt.Sprintf(tmplSummary, providerAttrHost),
			fmt.Sprintf(tmplDetail, providerAttrHost, "POSTGRES_SCHEMA"),
		)
	}
	if (valType == ProValUnknown && ppm.SSLMode.IsUnknown()) || (valType == ProValMissing && ppm.SSLMode.IsNull()) {
		diags.AddAttributeError(
			path.Root(providerAttrHost),
			fmt.Sprintf(tmplSummary, providerAttrHost),
			fmt.Sprintf(tmplDetail, providerAttrHost, "POSTGRES_SSLMODE"),
		)
	}

	return diags
}

// loadConfig loads the provider configuration from the given context.
// Read and parse the expected environment variables to assign to the PostgresqlProviderConfig, but only if the
// respective attribute is empty or not set.
//
// Returns:
// - None
func (ppm *PostgresqlProviderConfig) loadConfig() diag.Diagnostics {
	diags := diag.Diagnostics{}

	if ppm.Host.IsNull() {
		ppm.Host = types.StringValue(os.Getenv("POSTGRES_HOST"))
	}
	if ppm.Port.IsNull() {
		if portRaw := os.Getenv("POSTGRES_PORT"); portRaw != "" {
			port, err := strconv.ParseInt(portRaw, 10, 64)
			if err != nil {
				diags.AddAttributeError(
					path.Root(providerAttrPort),
					"Failed to parse environment variable 'POSTGRES_PORT' to int",
					err.Error(),
				)
			}
			ppm.Port = types.Int64Value(port)
		}

	}
	if ppm.Username.IsNull() {
		ppm.Username = types.StringValue(os.Getenv("POSTGRES_USER"))
	}
	if ppm.Password.IsNull() {
		ppm.Password = types.StringValue(os.Getenv("POSTGRES_PASSWORD"))
	}
	if ppm.Database.IsNull() {
		ppm.Database = types.StringValue(os.Getenv("POSTGRES_DATABASE"))
	}
	if ppm.Scheme.IsNull() {
		ppm.Scheme = types.StringValue(os.Getenv("POSTGRES_SCHEME"))
	}
	if ppm.SSLMode.IsNull() {
		ppm.SSLMode = types.StringValue(os.Getenv("POSTGRES_SSLMODE"))
	}

	return diags
}

// transposeToPGClientConfig transforms the PostgresqlProviderConfig into a PGClientConfig.
// This method is responsible for converting the configuration stored in the PostgresqlProviderConfig
// structure into a format that is compatible with the PGClientConfig structure used by the PostgreSQL client.
// It extracts and converts each relevant field from the provider configuration, applying necessary transformations
// (e.g., type conversions) to match the expected types of the PGClientConfig fields.
//
// During this process, it also checks for errors that might occur during the conversion of port numbers from int64 to int,
// or any other potential issues that could arise from invalid configuration values. If an error is encountered,
// it is added to the diagnostics collection, which is then returned alongside the PGClientConfig.
//
// Returns:
// - client.PGClientConfig: The transformed PostgreSQL client configuration.
// - diag.Diagnostics: A collection of diagnostic messages that may include errors encountered during the transformation process.
func (ppm *PostgresqlProviderConfig) transposeToPGClientConfig() (client.PGClientConfig, diag.Diagnostics) {

	diags := diag.Diagnostics{}
	config, err := client.NewPGClientConfig(
		client.WithPGClientConfigHost(ppm.Host.ValueString()),
		client.WithPort(strconv.FormatInt(ppm.Port.ValueInt64(), 10)),
		client.WithUsername(ppm.Username.ValueString()),
		client.WithPassword(ppm.Password.ValueString()),
		client.WithDatabase(ppm.Database.ValueString()),
		client.WithScheme(ppm.Scheme.ValueString()),
		client.WithSSLMode(ppm.SSLMode.ValueString()),
		client.WithMaxOpenConn(int(ppm.MaxOpenConn.ValueInt64())),
		client.WithMaxIdleConn(int(ppm.MaxIdleConn.ValueInt64())),
	)
	if err != nil {
		diags.AddError("Failed to use Provider config to Create a PGClientConfig", err.Error())
	}

	return config, nil
}
