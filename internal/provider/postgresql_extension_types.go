package provider

import (
	"terraform-provider-postgresql/internal/pgclient"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

type postgresqlExtensionModel struct {
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Schema      types.String `tfsdk:"schema"`
	Version     types.String `tfsdk:"version"`
	Database    types.String `tfsdk:"database"`
	Cascade     types.Bool   `tfsdk:"cascade"`
	DropCascade types.Bool   `tfsdk:"drop_cascade"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

func (m *postgresqlExtensionModel) buildPgExtensionUpdateParams(stateModel *postgresqlExtensionModel) pgclient.ExtensionUpdateParams {
	var result pgclient.ExtensionUpdateParams

	if !m.Version.Equal(stateModel.Version) {
		result.Version = m.Version.ValueStringPointer()
	}

	if !m.Schema.Equal(stateModel.Schema) {
		result.Schema = m.Schema.ValueStringPointer()
	}

	return result
}
