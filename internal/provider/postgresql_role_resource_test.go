package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"strconv"
	"strings"
	"terraform-provider-postgresql/internal/helpers"
	"terraform-provider-postgresql/internal/test"
	"testing"
)

func TestAccPostgresqlRoleResource(t *testing.T) {
	runOpts := test.PostgresContainerRunOptions{
		Database: "test_role_resource_db",
		Username: "test_role_resource_user",
	}
	test.LoadPostgresTestContainer(t, runOpts, true)

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
					resource.TestCheckResourceAttr(mockResourceName, "comment", mockRoleModel.Comment.ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "password_wo_version", strconv.FormatInt(int64(mockRoleModel.PasswordVersion.ValueInt32()), 10)),
					// Password is sensitive and write-only, so we don't check it
				),
			},
			{
				// ImportState testing
				ResourceName:            mockResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"last_updated", "password_wo", "password_wo_version"},
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
				),
			},
		},
	})
}

func testAccFormatRoleResource(t *testing.T, resName string, m postgresqlRoleModel, passwd types.String) string {
	t.Helper()

	result := make([]string, 0)
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
	)

	// Only include valid_until if it's not null
	if !m.ValidUntil.IsNull() {
		result = append(result, test.FormatTerraformAttribute(t, m.ValidUntil, "valid_until"))
	}

	// Only include a comment if it's not null
	if !m.Comment.IsNull() {
		result = append(result, test.FormatTerraformAttribute(t, m.Comment, "comment"))
	}

	result = append(result, "}")

	result = helpers.CleanUpSlice(result)
	return strings.Join(result, "\n")
}
