package provider

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-postgresql/internal/pgclient"
	"terraform-provider-postgresql/internal/provider/validators"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &postgresqlGrantResource{}
	_ resource.ResourceWithConfigure   = &postgresqlGrantResource{}
	_ resource.ResourceWithImportState = &postgresqlGrantResource{}
)

// NewPostgresqlGrantResource creates a new postgresql_grant resource.
func NewPostgresqlGrantResource() resource.Resource {
	return &postgresqlGrantResource{
		resName: "postgresql_grant",
	}
}

type postgresqlGrantResource struct {
	resName   string
	pgClient  pgclient.PostgresqlClient
	grantRepo pgclient.GrantRepo
}

// Metadata sets the resource type name.
func (r *postgresqlGrantResource) Metadata(_ context.Context, _ resource.MetadataRequest, res *resource.MetadataResponse) {
	res.TypeName = r.resName
}

// Schema defines the resource schema.
func (r *postgresqlGrantResource) Schema(_ context.Context, _ resource.SchemaRequest, res *resource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "Manages PostgreSQL privileges using GRANT and REVOKE. " +
			"[PostgreSQL GRANT documentation](https://www.postgresql.org/docs/current/sql-grant.html)",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for the grant (computed)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"object_type": schema.StringAttribute{
				Description: "Type of database object. Supported values: `database`, `schema`, `table`, `sequence`.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"database", "schema", "table", "sequence",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"object_name": schema.StringAttribute{
				Description: "Name of the database object",
				Required:    true,
				Validators: []validator.String{
					validators.PostgresqlObjectName(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"schema": schema.StringAttribute{
				Description: "Schema containing the object (for schema-qualified objects). Optional, defaults to 'public' for applicable objects.",
				Optional:    true,
				Validators: []validator.String{
					validators.PostgresqlObjectName(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role": schema.StringAttribute{
				Description: "Name of the role to grant privileges to",
				Required:    true,
				Validators: []validator.String{
					validators.PostgresqlObjectName(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"privileges": schema.SetAttribute{
				Description: "Set of privileges to grant (e.g., SELECT, INSERT, UPDATE, DELETE, ALL)",
				Required:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
			},
			"with_grant_option": schema.BoolAttribute{
				Description: "If true, the grantee can grant the privilege to others. Default is false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Configure sets up the resource with the PostgreSQL client.
func (r *postgresqlGrantResource) Configure(_ context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
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

	r.pgClient = pgClient
	r.grantRepo = pgclient.NewGrantRepo()
}

// Create grants privileges to a role on a database object.
func (r *postgresqlGrantResource) Create(ctx context.Context, req resource.CreateRequest, res *resource.CreateResponse) {
	tflog.Debug(ctx, fmt.Sprintf("creating '%s' resource", r.resName))

	var model postgresqlGrantModel
	res.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	conn, err := r.pgClient.GetConnection(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	// Build grant parameters
	params, err := model.buildGrantParams(ctx)
	if err != nil {
		res.Diagnostics.AddError("Error building grant parameters", err.Error())
		return
	}

	// Execute GRANT
	if err := r.grantRepo.Grant(ctx, conn, params); err != nil {
		res.Diagnostics.AddError(
			fmt.Sprintf(msgErrorExecutingPgAction, TFCreateAction, "GRANT"),
			err.Error(),
		)
		return
	}

	// Generate ID
	model.ID = types.StringValue(model.generateID())

	tflog.Debug(ctx, fmt.Sprintf("successfully created '%s' resource with id: %s", r.resName, model.ID.ValueString()))

	res.Diagnostics.Append(res.State.Set(ctx, &model)...)
}

// Read retrieves the current state of the grant from PostgreSQL.
// Note: Reading grants is complex due to PostgreSQL's ACL system. This implementation
// makes a best-effort attempt but may not work for all object types.
func (r *postgresqlGrantResource) Read(ctx context.Context, req resource.ReadRequest, res *resource.ReadResponse) {
	tflog.Debug(ctx, fmt.Sprintf("reading '%s' resource", r.resName))

	var model postgresqlGrantModel
	res.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	conn, err := r.pgClient.GetConnection(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	// Get grant information
	params := model.buildGetGrantParams()
	grant, err := r.grantRepo.GetGrant(ctx, conn, params)
	if err != nil {
		res.Diagnostics.AddError(
			fmt.Sprintf(msgErrorExecutingPgAction, TFReadAction, "grant"),
			err.Error(),
		)
		return
	}

	if grant == nil {
		// Grant was revoked outside Terraform — remove from state to allow reconciliation.
		res.State.RemoveResource(ctx)
		return
	}

	// Update model from database
	if err := model.fromGrantModel(ctx, grant); err != nil {
		res.Diagnostics.AddError("Error updating model from grant", err.Error())
		return
	}

	// Regenerate the ID so that Import and Create paths produce the same value.
	model.ID = types.StringValue(model.generateID())

	tflog.Debug(ctx, fmt.Sprintf("successfully read '%s' resource with id: %s", r.resName, model.ID.ValueString()))

	res.Diagnostics.Append(res.State.Set(ctx, &model)...)
}

// Update is not supported - grants are immutable (ForceNew on all attributes).
func (r *postgresqlGrantResource) Update(ctx context.Context, req resource.UpdateRequest, res *resource.UpdateResponse) {
	// This should never be called due to ForceNew plan modifiers on all attributes
	res.Diagnostics.AddError(
		"Update Not Supported",
		"Grant resources cannot be updated. All changes require replacement (destroy and create).",
	)
}

// Delete revokes privileges from a role on a database object.
func (r *postgresqlGrantResource) Delete(ctx context.Context, req resource.DeleteRequest, res *resource.DeleteResponse) {
	tflog.Debug(ctx, fmt.Sprintf("deleting '%s' resource", r.resName))

	var model postgresqlGrantModel
	res.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if res.Diagnostics.HasError() {
		return
	}

	conn, err := r.pgClient.GetConnection(ctx)
	if err != nil {
		res.Diagnostics.AddError(msgErrGetPgConnection, err.Error())
		return
	}

	// Build revoke parameters
	params, err := model.buildRevokeParams(ctx)
	if err != nil {
		res.Diagnostics.AddError("Error building revoke parameters", err.Error())
		return
	}

	// Execute REVOKE
	if err := r.grantRepo.Revoke(ctx, conn, params); err != nil {
		res.Diagnostics.AddError(
			fmt.Sprintf(msgErrorExecutingPgAction, TFDeleteAction, "REVOKE"),
			err.Error(),
		)
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("successfully deleted '%s' resource with id: %s", r.resName, model.ID.ValueString()))
}

// ImportState imports an existing grant into Terraform state.
// Expected import ID format: object_type:schema:object_name:role
// For objects without a schema (e.g. database), use an empty segment: database::mydb:my_role
//
// After setting the identifying attributes, Terraform calls Read which queries
// the actual privileges from the database and populates the full state.
func (r *postgresqlGrantResource) ImportState(ctx context.Context, req resource.ImportStateRequest, res *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 4)
	if len(parts) != 4 {
		res.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf(
				"Expected format object_type:schema:object_name:role, got: %q. "+
					"For database-level objects use an empty schema segment, e.g. database::mydb:my_role",
				req.ID,
			),
		)
		return
	}

	objectType, schemaName, objectName, role := parts[0], parts[1], parts[2], parts[3]

	// Validate required segments.
	if objectType == "" || objectName == "" || role == "" {
		res.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf(
				"object_type, object_name, and role must be non-empty in import ID %q",
				req.ID,
			),
		)
		return
	}

	// Validate object_type is a supported value.
	supportedTypes := map[string]bool{"database": true, "schema": true, "table": true, "sequence": true}
	if !supportedTypes[objectType] {
		res.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf(
				"object_type %q is not supported; must be one of: database, schema, table, sequence",
				objectType,
			),
		)
		return
	}

	res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root("object_type"), objectType)...)
	res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root("object_name"), objectName)...)
	res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root("role"), role)...)
	if schemaName != "" {
		res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root("schema"), schemaName)...)
	}
	// Initialize required attributes so that Read can populate them from the database.
	// privileges must be set to a non-null value; Read will overwrite it with actual grants.
	emptyPrivileges, diags := types.SetValueFrom(ctx, types.StringType, []string{})
	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}
	res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root("privileges"), emptyPrivileges)...)
	res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root("with_grant_option"), false)...)
}
