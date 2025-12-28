package provider

import (
	"fmt"
	"strconv"
	"testing"

	"terraform-provider-postgresql/internal/test"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccPostgresqlDatabaseResource(t *testing.T) {
	runOpts := test.PostgresContainerRunOptions{
		Database: "postgres",
		Username: "postgres",
	}
	test.LoadPostgresTestContainer(t, runOpts, true)

	mockDBModel := postgresqlDatabaseModel{
		Name:             types.StringValue("test_db"),
		Owner:            types.StringValue("postgres"),
		Encoding:         types.StringValue("UTF8"),
		Template:         types.StringValue("template1"),
		ConnectionLimit:  types.Int32Value(50),
		AllowConnections: types.BoolValue(true),
		IsTemplate:       types.BoolValue(false),
		Comment:          types.StringValue("test database"),
		ForceDrop:        types.BoolValue(false),
	}

	mockUpdatedConnectionLimit := types.Int32Value(100)
	mockUpdatedComment := types.StringValue("updated test database")

	mockDBName := "test_db"
	mockResourceName := fmt.Sprintf("postgresql_database.%s", mockDBName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create and Read testing
				Config: testAccFormatDatabaseResource(t, mockDBName, mockDBModel),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(mockResourceName, plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mockResourceName, "name", mockDBModel.Name.ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "owner", mockDBModel.Owner.ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "encoding", mockDBModel.Encoding.ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "template", mockDBModel.Template.ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "connection_limit", strconv.FormatInt(int64(mockDBModel.ConnectionLimit.ValueInt32()), 10)),
					resource.TestCheckResourceAttr(mockResourceName, "allow_connections", strconv.FormatBool(mockDBModel.AllowConnections.ValueBool())),
					resource.TestCheckResourceAttr(mockResourceName, "is_template", strconv.FormatBool(mockDBModel.IsTemplate.ValueBool())),
					resource.TestCheckResourceAttr(mockResourceName, "comment", mockDBModel.Comment.ValueString()),
				),
			},
			{
				// ImportState testing
				ResourceName:            mockResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"last_updated", "template", "force_drop"},
				Destroy:                 false,
			},
			{
				// Update testing - Change connection limit and comment
				Config: func() string {
					mockDBModel.ConnectionLimit = mockUpdatedConnectionLimit
					mockDBModel.Comment = mockUpdatedComment
					return testAccFormatDatabaseResource(t, mockDBName, mockDBModel)
				}(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(mockResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mockResourceName, "name", mockDBModel.Name.ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "connection_limit", strconv.FormatInt(int64(mockUpdatedConnectionLimit.ValueInt32()), 10)),
					resource.TestCheckResourceAttr(mockResourceName, "comment", mockUpdatedComment.ValueString()),
				),
			},
		},
	})
}

func TestAccPostgresqlDatabaseResource_ForceDrop(t *testing.T) {
	runOpts := test.PostgresContainerRunOptions{
		Database: "postgres",
		Username: "postgres",
	}
	test.LoadPostgresTestContainer(t, runOpts, true)

	mockDBModel := postgresqlDatabaseModel{
		Name:             types.StringValue("test_db_force"),
		Owner:            types.StringValue("postgres"),
		Template:         types.StringValue("template1"),
		ConnectionLimit:  types.Int32Value(-1),
		AllowConnections: types.BoolValue(true),
		IsTemplate:       types.BoolValue(false),
		Comment:          types.StringValue(""),
		ForceDrop:        types.BoolValue(true),
	}

	mockDBName := "test_db_force"
	mockResourceName := fmt.Sprintf("postgresql_database.%s", mockDBName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create and Read testing with force_drop
				Config: testAccFormatDatabaseResource(t, mockDBName, mockDBModel),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(mockResourceName, plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mockResourceName, "name", mockDBModel.Name.ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "force_drop", strconv.FormatBool(mockDBModel.ForceDrop.ValueBool())),
				),
			},
		},
	})
}

func testAccFormatDatabaseResource(t *testing.T, name string, model postgresqlDatabaseModel) string {
	t.Helper()

	config := fmt.Sprintf(`
resource "postgresql_database" "%s" {
  name              = "%s"
  owner             = "%s"`, name, model.Name.ValueString(), model.Owner.ValueString())

	if !model.Encoding.IsNull() && model.Encoding.ValueString() != "" {
		config += fmt.Sprintf(`
  encoding          = "%s"`, model.Encoding.ValueString())
	}

	config += fmt.Sprintf(`
  template          = "%s"
  connection_limit  = %d
  allow_connections = %t
  is_template       = %t
  comment           = "%s"
  force_drop        = %t
}
`, model.Template.ValueString(),
		model.ConnectionLimit.ValueInt32(),
		model.AllowConnections.ValueBool(),
		model.IsTemplate.ValueBool(),
		model.Comment.ValueString(),
		model.ForceDrop.ValueBool(),
	)

	return config
}
