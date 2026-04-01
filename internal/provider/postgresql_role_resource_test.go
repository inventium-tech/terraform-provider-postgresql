package provider

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"terraform-provider-postgresql/internal/helpers"
	"terraform-provider-postgresql/internal/test"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccPostgresqlRoleResource(t *testing.T) {
	runOpts := test.PostgresContainerRunOptions{
		Database: "test_role_resource_db",
		Username: "test_role_resource_user",
	}
	test.LoadPostgresTestContainer(t, runOpts, true)

	supportInRoleName := "support_in_role"
	supportRoleName := "support_role"
	supportAdminRoleName := "support_admin_role"

	mockRolePasswd := types.StringValue("test_password")
	mockRoleModel := postgresqlRoleModel{
		Name:            types.StringValue("test_role"),
		PasswordVersion: types.Int32Value(1),
		Superuser:       types.BoolValue(false),
		Inherit:         types.BoolValue(true),
		CreateRole:      types.BoolValue(false),
		CreateDB:        types.BoolValue(false),
		Login:           types.BoolValue(true),
		Replication:     types.BoolValue(false),
		BypassRLS:       types.BoolValue(false),
		ConnectionLimit: types.Int32Value(-1),
		InRole:          types.SetValueMust(types.StringType, []attr.Value{types.StringValue(supportInRoleName)}),
		Role:            types.SetValueMust(types.StringType, []attr.Value{types.StringValue(supportRoleName)}),
		Admin:           types.SetValueMust(types.StringType, []attr.Value{types.StringValue(supportAdminRoleName)}),
		Comment:         types.StringValue("test role"),
	}

	mockUpdatedPasswordVersion := types.Int32Value(2)
	mockUpdatedComment := types.StringValue("updated test role")
	mockUpdatedLogin := types.BoolValue(false)

	mockRoleName := "test_role"
	mockResourceName := fmt.Sprintf("postgresql_role.%s", mockRoleName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create and Read testing with default values
				Config: testAccFormatRoleResource(t, mockRoleName, mockRoleModel, mockRolePasswd),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(mockResourceName, plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mockResourceName, "name", mockRoleModel.Name.ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "superuser", strconv.FormatBool(mockRoleModel.Superuser.ValueBool())),
					resource.TestCheckResourceAttr(mockResourceName, "inherit", strconv.FormatBool(mockRoleModel.Inherit.ValueBool())),
					resource.TestCheckResourceAttr(mockResourceName, "create_role", strconv.FormatBool(mockRoleModel.CreateRole.ValueBool())),
					resource.TestCheckResourceAttr(mockResourceName, "create_db", strconv.FormatBool(mockRoleModel.CreateDB.ValueBool())),
					resource.TestCheckResourceAttr(mockResourceName, "login", strconv.FormatBool(mockRoleModel.Login.ValueBool())),
					resource.TestCheckResourceAttr(mockResourceName, "replication", strconv.FormatBool(mockRoleModel.Replication.ValueBool())),
					resource.TestCheckResourceAttr(mockResourceName, "bypass_rls", strconv.FormatBool(mockRoleModel.BypassRLS.ValueBool())),
					resource.TestCheckResourceAttr(mockResourceName, "connection_limit", strconv.FormatInt(int64(mockRoleModel.ConnectionLimit.ValueInt32()), 10)),
					resource.TestCheckResourceAttr(mockResourceName, "in_role.#", "1"),
					resource.TestCheckResourceAttr(mockResourceName, "role.#", "1"),
					resource.TestCheckResourceAttr(mockResourceName, "admin.#", "1"),
					resource.TestCheckTypeSetElemAttr(mockResourceName, "in_role.*", supportInRoleName),
					resource.TestCheckTypeSetElemAttr(mockResourceName, "role.*", supportRoleName),
					resource.TestCheckTypeSetElemAttr(mockResourceName, "admin.*", supportAdminRoleName),
					resource.TestCheckResourceAttr(mockResourceName, "comment", mockRoleModel.Comment.ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "password_wo_version", strconv.FormatInt(int64(mockRoleModel.PasswordVersion.ValueInt32()), 10)),
					// Password is sensitive and write-only, so we don't check it
				),
			},
			{
				// ImportState testing
				// Note: in_role is a create-time only attribute, so during import all memberships go into role
				ResourceName:            mockResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"last_updated", "password_wo", "password_wo_version", "in_role", "role"},
				Destroy:                 false,
			},
			{
				// Update testing - Change password and comment
				Config: func() string {
					mockRoleModel.PasswordVersion = mockUpdatedPasswordVersion
					mockRoleModel.Comment = mockUpdatedComment
					return testAccFormatRoleResource(t, mockRoleName, mockRoleModel, types.StringValue("updated_password"))
				}(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(mockResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mockResourceName, "name", mockRoleModel.Name.ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "comment", mockUpdatedComment.ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "password_wo_version", strconv.FormatInt(int64(mockUpdatedPasswordVersion.ValueInt32()), 10)),
					// Password is sensitive and write-only, so we don't check it
				),
			},
			{
				// Update testing - Change login attribute
				Config: func() string {
					mockRoleModel.Login = mockUpdatedLogin
					return testAccFormatRoleResource(t, mockRoleName, mockRoleModel, mockRolePasswd)
				}(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(mockResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mockResourceName, "name", mockRoleModel.Name.ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "login", strconv.FormatBool(mockUpdatedLogin.ValueBool())),
					resource.TestCheckResourceAttr(mockResourceName, "in_role.#", "1"),
					resource.TestCheckResourceAttr(mockResourceName, "role.#", "1"),
					resource.TestCheckResourceAttr(mockResourceName, "admin.#", "1"),
				),
			},
		},
	})
}

func testAccFormatRoleResource(t *testing.T, resName string, m postgresqlRoleModel, passwd types.String) string {
	t.Helper()

	result := make([]string, 0)

	supportingRoles := collectSupportRoles(t, []types.Set{m.InRole, m.Role, m.Admin})

	for _, supportingRole := range supportingRoles {
		result = append(result,
			fmt.Sprintf(`resource "postgresql_role" "%s" {`, supportingRole),
			fmt.Sprintf(`name = "%s"`, supportingRole),
			"login = false",
			"}",
			"",
		)
	}

	result = append(result,
		fmt.Sprintf(`resource "postgresql_role" "%s" {`, resName),
		test.FormatTerraformAttribute(t, m.Name, "name"),
	)

	// Only include 'password' if it's not null
	if !passwd.IsNull() {
		result = append(result, test.FormatTerraformAttribute(t, passwd, "password_wo"))
	}

	// Only include password_version if it's not null
	if !m.PasswordVersion.IsNull() {
		result = append(result, test.FormatTerraformAttribute(t, m.PasswordVersion, "password_wo_version"))
	}

	result = append(result,
		test.FormatTerraformAttribute(t, m.Superuser, "superuser"),
		test.FormatTerraformAttribute(t, m.Inherit, "inherit"),
		test.FormatTerraformAttribute(t, m.CreateRole, "create_role"),
		test.FormatTerraformAttribute(t, m.CreateDB, "create_db"),
		test.FormatTerraformAttribute(t, m.Login, "login"),
		test.FormatTerraformAttribute(t, m.Replication, "replication"),
		test.FormatTerraformAttribute(t, m.BypassRLS, "bypass_rls"),
		test.FormatTerraformAttribute(t, m.ConnectionLimit, "connection_limit"),
		test.FormatTerraformAttribute(t, m.InRole, "in_role"),
		test.FormatTerraformAttribute(t, m.Role, "role"),
		test.FormatTerraformAttribute(t, m.Admin, "admin"),
	)

	// Only include valid_until if it's not null
	if !m.ValidUntil.IsNull() {
		result = append(result, test.FormatTerraformAttribute(t, m.ValidUntil, "valid_until"))
	}

	// Only include a comment if it's not null
	if !m.Comment.IsNull() {
		result = append(result, test.FormatTerraformAttribute(t, m.Comment, "comment"))
	}

	if len(supportingRoles) > 0 {
		deps := make([]string, 0, len(supportingRoles))
		for _, supportingRole := range supportingRoles {
			deps = append(deps, fmt.Sprintf("postgresql_role.%s", supportingRole))
		}

		result = append(result, fmt.Sprintf("depends_on = [%s]", strings.Join(deps, ", ")))
	}

	result = append(result, "}")

	result = helpers.CleanUpSlice(result)
	return strings.Join(result, "\n")
}

func collectSupportRoles(t *testing.T, sets []types.Set) []string {
	t.Helper()

	rolesMap := make(map[string]struct{})

	for _, setValue := range sets {
		if setValue.IsNull() || setValue.IsUnknown() {
			continue
		}

		for _, element := range setValue.Elements() {
			strValue, ok := element.(types.String)
			if !ok {
				t.Fatalf("expected types.String in membership set, got %T", element)
			}

			rolesMap[strValue.ValueString()] = struct{}{}
		}
	}

	result := make([]string, 0, len(rolesMap))
	for roleName := range rolesMap {
		result = append(result, roleName)
	}

	slices.Sort(result)

	return result
}
