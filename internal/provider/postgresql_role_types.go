package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"terraform-provider-postgresql/internal/pgclient"
)

type postgresqlRoleModel struct {
	Id              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Password        types.String `tfsdk:"password_wo"`
	PasswordVersion types.Int32  `tfsdk:"password_wo_version"`
	Superuser       types.Bool   `tfsdk:"superuser"`
	Inherit         types.Bool   `tfsdk:"inherit"`
	CreateRole      types.Bool   `tfsdk:"create_role"`
	CreateDB        types.Bool   `tfsdk:"create_db"`
	Login           types.Bool   `tfsdk:"login"`
	Replication     types.Bool   `tfsdk:"replication"`
	BypassRLS       types.Bool   `tfsdk:"bypass_rls"`
	ConnectionLimit types.Int32  `tfsdk:"connection_limit"`
	ValidUntil      types.String `tfsdk:"valid_until"`
	Comment         types.String `tfsdk:"comment"`
	LastUpdated     types.String `tfsdk:"last_updated"`
}

func (r *postgresqlRoleModel) buildPgRoleUpdateParams(stateModel *postgresqlRoleModel) pgclient.RoleUpdateParams {
	var result pgclient.RoleUpdateParams

	if !r.Name.Equal(stateModel.Name) {
		result.Name = r.Name.ValueStringPointer()
	}

	if !r.Superuser.Equal(stateModel.Superuser) {
		result.Superuser = r.Superuser.ValueBoolPointer()
	}

	if !r.Inherit.Equal(stateModel.Inherit) {
		result.Inherit = r.Inherit.ValueBoolPointer()
	}

	if !r.CreateRole.Equal(stateModel.CreateRole) {
		result.CreateRole = r.CreateRole.ValueBoolPointer()
	}

	if !r.CreateDB.Equal(stateModel.CreateDB) {
		result.CreateDB = r.CreateDB.ValueBoolPointer()
	}

	if !r.Login.Equal(stateModel.Login) {
		result.Login = r.Login.ValueBoolPointer()
	}
	if !r.Replication.Equal(stateModel.Replication) {
		result.Replication = r.Replication.ValueBoolPointer()
	}

	if !r.BypassRLS.Equal(stateModel.BypassRLS) {
		result.BypassRLS = r.BypassRLS.ValueBoolPointer()
	}

	if !r.ConnectionLimit.Equal(stateModel.ConnectionLimit) {
		result.ConnectionLimit = r.ConnectionLimit.ValueInt32Pointer()
	}

	if !r.ValidUntil.Equal(stateModel.ValidUntil) {
		result.ValidUntil = r.ValidUntil.ValueStringPointer()
	}

	if !r.Comment.Equal(stateModel.Comment) {
		result.Comment = r.Comment.ValueStringPointer()
	}

	return result
}
