package provider

import (
	"context"
	"fmt"
	"terraform-provider-postgresql/internal/pgclient"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// postgresqlGrantModel represents the Terraform resource model for postgresql_grant.
type postgresqlGrantModel struct {
	ID              types.String `tfsdk:"id"`
	ObjectType      types.String `tfsdk:"object_type"`
	ObjectName      types.String `tfsdk:"object_name"`
	Schema          types.String `tfsdk:"schema"`
	Role            types.String `tfsdk:"role"`
	Privileges      types.Set    `tfsdk:"privileges"`
	WithGrantOption types.Bool   `tfsdk:"with_grant_option"`
}

// buildGrantParams converts the Terraform model to repository parameters for granting.
func (m *postgresqlGrantModel) buildGrantParams(ctx context.Context) (pgclient.GrantParams, error) {
	var privileges []string
	diags := m.Privileges.ElementsAs(ctx, &privileges, false)
	if diags.HasError() {
		return pgclient.GrantParams{}, fmt.Errorf("failed to convert privileges: %s", diags[0].Summary())
	}

	return pgclient.GrantParams{
		ObjectType:      m.ObjectType.ValueString(),
		ObjectName:      m.ObjectName.ValueString(),
		Schema:          m.Schema.ValueString(),
		Role:            m.Role.ValueString(),
		Privileges:      privileges,
		WithGrantOption: m.WithGrantOption.ValueBool(),
	}, nil
}

// buildRevokeParams converts the Terraform model to repository parameters for revoking.
func (m *postgresqlGrantModel) buildRevokeParams(ctx context.Context) (pgclient.RevokeParams, error) {
	var privileges []string
	diags := m.Privileges.ElementsAs(ctx, &privileges, false)
	if diags.HasError() {
		return pgclient.RevokeParams{}, fmt.Errorf("failed to convert privileges: %s", diags[0].Summary())
	}

	return pgclient.RevokeParams{
		ObjectType: m.ObjectType.ValueString(),
		ObjectName: m.ObjectName.ValueString(),
		Schema:     m.Schema.ValueString(),
		Role:       m.Role.ValueString(),
		Privileges: privileges,
		Cascade:    false,
	}, nil
}

// buildGetGrantParams converts the Terraform model to repository parameters for reading.
func (m *postgresqlGrantModel) buildGetGrantParams() pgclient.GetGrantParams {
	return pgclient.GetGrantParams{
		ObjectType: m.ObjectType.ValueString(),
		ObjectName: m.ObjectName.ValueString(),
		Schema:     m.Schema.ValueString(),
		Role:       m.Role.ValueString(),
	}
}

// fromGrantModel updates the Terraform model from repository grant model.
func (m *postgresqlGrantModel) fromGrantModel(ctx context.Context, grant *pgclient.GrantModel) error {
	// Convert privileges slice to types.Set
	privElements := make([]attr.Value, len(grant.Privileges))
	for i, priv := range grant.Privileges {
		privElements[i] = types.StringValue(priv)
	}

	privSet, diags := types.SetValue(types.StringType, privElements)
	if diags.HasError() {
		return fmt.Errorf("failed to convert privileges to set: %s", diags[0].Summary())
	}

	m.ObjectType = types.StringValue(grant.ObjectType)
	m.ObjectName = types.StringValue(grant.ObjectName)
	if grant.Schema != "" {
		m.Schema = types.StringValue(grant.Schema)
	} else {
		m.Schema = types.StringNull()
	}
	m.Role = types.StringValue(grant.Role)
	m.Privileges = privSet
	m.WithGrantOption = types.BoolValue(grant.WithGrantOption)

	return nil
}

// generateID creates a unique identifier for the grant resource.
func (m *postgresqlGrantModel) generateID() string {
	schema := m.Schema.ValueString()
	if schema == "" {
		schema = "public"
	}
	return m.ObjectType.ValueString() + ":" + schema + ":" + m.ObjectName.ValueString() + ":" + m.Role.ValueString()
}
