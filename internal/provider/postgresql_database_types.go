package provider

import (
	"terraform-provider-postgresql/internal/pgclient"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

type postgresqlDatabaseModel struct {
	Id               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Owner            types.String `tfsdk:"owner"`
	Encoding         types.String `tfsdk:"encoding"`
	Collation        types.String `tfsdk:"collation"`
	Ctype            types.String `tfsdk:"ctype"`
	Template         types.String `tfsdk:"template"`
	ConnectionLimit  types.Int32  `tfsdk:"connection_limit"`
	AllowConnections types.Bool   `tfsdk:"allow_connections"`
	IsTemplate       types.Bool   `tfsdk:"is_template"`
	Tablespace       types.String `tfsdk:"tablespace"`
	Comment          types.String `tfsdk:"comment"`
	ForceDrop        types.Bool   `tfsdk:"force_drop"`
	LastUpdated      types.String `tfsdk:"last_updated"`
}

func (m *postgresqlDatabaseModel) buildPgDatabaseUpdateParams(stateModel *postgresqlDatabaseModel) pgclient.DatabaseUpdateParams {
	var result pgclient.DatabaseUpdateParams

	if !m.Owner.Equal(stateModel.Owner) {
		result.Owner = m.Owner.ValueStringPointer()
	}

	if !m.ConnectionLimit.Equal(stateModel.ConnectionLimit) {
		result.ConnectionLimit = m.ConnectionLimit.ValueInt32Pointer()
	}

	if !m.AllowConnections.Equal(stateModel.AllowConnections) {
		result.AllowConnections = m.AllowConnections.ValueBoolPointer()
	}

	if !m.IsTemplate.Equal(stateModel.IsTemplate) {
		result.IsTemplate = m.IsTemplate.ValueBoolPointer()
	}

	if !m.Tablespace.Equal(stateModel.Tablespace) {
		result.Tablespace = m.Tablespace.ValueStringPointer()
	}

	if !m.Comment.Equal(stateModel.Comment) {
		result.Comment = m.Comment.ValueStringPointer()
	}

	return result
}
