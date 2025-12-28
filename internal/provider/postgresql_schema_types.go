package provider

import (
	"terraform-provider-postgresql/internal/pgclient"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

type postgresqlSchemaModel struct {
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Owner       types.String `tfsdk:"owner"`
	IfNotExists types.Bool   `tfsdk:"if_not_exists"`
	DropCascade types.Bool   `tfsdk:"drop_cascade"`
	Policy      types.String `tfsdk:"policy"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

func (m *postgresqlSchemaModel) buildPgSchemaUpdateParams(stateModel *postgresqlSchemaModel) pgclient.SchemaUpdateParams {
	var result pgclient.SchemaUpdateParams

	if !m.Owner.Equal(stateModel.Owner) {
		result.Owner = m.Owner.ValueStringPointer()
	}

	return result
}
