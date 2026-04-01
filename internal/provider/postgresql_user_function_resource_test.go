package provider

import (
	"fmt"
	"strconv"
	"strings"
	"terraform-provider-postgresql/internal/helpers"
	"terraform-provider-postgresql/internal/test"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccPostgresqlUserFunctionResource(t *testing.T) {
	runOpts := test.PostgresContainerRunOptions{
		Database: "test_user_function_resource_db",
		Username: "test_user_function_resource_user",
	}
	test.LoadPostgresTestContainer(t, runOpts, true)

	mockUserFuncModel := postgresqlUserFunctionModel{
		Name: types.StringValue("test_user_function"),
		Args: []postgresqlUserFunctionArgType{
			{
				Name: types.StringValue("arg1"),
				Type: types.StringValue("integer"),
			},
			{
				Name: types.StringValue("arg2"),
				Type: types.StringValue("text"),
			},
		},
		Returns:  types.StringValue("integer"),
		Body:     types.StringValue("BEGIN RETURN arg1 + length(arg2); END;"),
		Language: types.StringValue("plpgsql"),
		Comment:  types.StringValue("test user function"),
	}

	mockUpdatedName := types.StringValue("test_user_function_updated")
	mockUpdatedBody := types.StringValue("BEGIN RETURN 10 + arg1 + length(arg2); END;")
	mockUpdatedComment := types.StringValue("updated test user function")

	mockFunctionName := "test_user_function"
	mockResourceName := fmt.Sprintf("postgresql_user_function.%s", mockFunctionName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create and Read testing with default values
				Config: testAccFormatUserFunctionResource(t, mockFunctionName, mockUserFuncModel),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(mockResourceName, plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mockResourceName, "name", mockUserFuncModel.Name.ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "args.#", strconv.Itoa(len(mockUserFuncModel.Args))),
					resource.TestCheckResourceAttr(mockResourceName, "args.0.name", mockUserFuncModel.Args[0].Name.ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "args.0.type", mockUserFuncModel.Args[0].Type.ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "args.1.name", mockUserFuncModel.Args[1].Name.ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "args.1.type", mockUserFuncModel.Args[1].Type.ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "returns", mockUserFuncModel.Returns.ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "language", mockUserFuncModel.Language.ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "body", mockUserFuncModel.Body.ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "allow_replace", strconv.FormatBool(true)),
					resource.TestCheckResourceAttr(mockResourceName, "database", runOpts.Database),
					resource.TestCheckResourceAttr(mockResourceName, "schema", "public"),
					resource.TestCheckResourceAttr(mockResourceName, "owner", runOpts.Username),
					resource.TestCheckResourceAttr(mockResourceName, "comment", mockUserFuncModel.Comment.ValueString()),
				),
			},
			{
				// ImportState testing
				ResourceName:            mockResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateId:           fmt.Sprintf("%s.%s.%s(%s, %s)", runOpts.Database, "public", mockFunctionName, mockUserFuncModel.Args[0].String(), mockUserFuncModel.Args[1].String()),
				ImportStateVerifyIgnore: []string{"last_updated"},
				Destroy:                 false,
			},
			{
				// Update testing - Properties WITHOUT Resource replacement (name, owner, comment)
				Config: func() string {
					mockUserFuncModel.Name = mockUpdatedName
					mockUserFuncModel.Comment = mockUpdatedComment
					return testAccFormatUserFunctionResource(t, mockFunctionName, mockUserFuncModel)
				}(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(mockResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mockResourceName, "name", mockUpdatedName.ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "comment", mockUpdatedComment.ValueString()),
				),
			},
			{
				// Update testing - Properties WITH Resource replacement (name, args, returns, language, etc..)
				Config: func() string {
					mockUserFuncModel.Body = mockUpdatedBody
					return testAccFormatUserFunctionResource(t, mockFunctionName, mockUserFuncModel)
				}(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(mockResourceName, plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mockResourceName, "name", mockUpdatedName.ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "body", mockUpdatedBody.ValueString()),
				),
			},
		},
	})
}

func formatArgs(t *testing.T, args []postgresqlUserFunctionArgType, fieldName string) string {
	t.Helper()

	if len(args) == 0 {
		return ""
	}

	argStrings := make([]string, 0, len(args))
	for _, arg := range args {
		argAttrs := make([]string, 0, 4)
		argAttrs = append(argAttrs, fmt.Sprintf(`name = "%s"`, arg.Name.ValueString()))
		argAttrs = append(argAttrs, fmt.Sprintf(`type = "%s"`, arg.Type.ValueString()))

		if !arg.Mode.IsNull() {
			argAttrs = append(argAttrs, fmt.Sprintf(`mode = "%s"`, arg.Mode.ValueString()))
		}

		if !arg.Default.IsNull() {
			argAttrs = append(argAttrs, fmt.Sprintf(`default = "%s"`, arg.Default.ValueString()))
		}

		argStrings = append(argStrings, fmt.Sprintf("    {\n      %s\n    }", strings.Join(argAttrs, "\n      ")))
	}

	return fmt.Sprintf(`%s = [
%s
  ]`, fieldName, strings.Join(argStrings, ",\n"))
}

func testAccFormatUserFunctionResource(t *testing.T, resName string, m postgresqlUserFunctionModel) string {
	t.Helper()

	result := make([]string, 0)
	result = append(result,
		fmt.Sprintf(`resource "postgresql_user_function" "%s" {`, resName),
		test.FormatTerraformAttribute(t, m.Name, "name"),
		test.FormatTerraformAttribute(t, m.Body, "body"),
		formatArgs(t, m.Args, "args"),
		test.FormatTerraformAttribute(t, m.Returns, "returns"),
		test.FormatTerraformAttribute(t, m.Language, "language"),
		test.FormatTerraformAttribute(t, m.AllowReplace, "allow_replace"),
		test.FormatTerraformAttribute(t, m.Comment, "comment"),
		test.FormatTerraformAttribute(t, m.Database, "database"),
		test.FormatTerraformAttribute(t, m.Schema, "schema"),
		test.FormatTerraformAttribute(t, m.Owner, "owner"),
		"}",
	)

	result = helpers.CleanUpSlice(result)
	return strings.Join(result, "\n")
}
