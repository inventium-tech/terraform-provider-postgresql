package provider

import (
	"fmt"
	"testing"

	"terraform-provider-postgresql/internal/test"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccPostgresqlSchemaResource(t *testing.T) {
	runOpts := test.PostgresContainerRunOptions{
		Database: "postgres",
		Username: "postgres",
	}
	test.LoadPostgresTestContainer(t, runOpts, true)

	mockSchemaModel := postgresqlSchemaModel{
		Name:        types.StringValue("test_schema"),
		Owner:       types.StringValue("postgres"),
		IfNotExists: types.BoolValue(false),
		DropCascade: types.BoolValue(false),
		Policy:      types.StringValue("error_on_collision"),
	}

	mockSchemaName := "test_schema"
	mockResourceName := fmt.Sprintf("postgresql_schema.%s", mockSchemaName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create and Read testing
				Config: testAccFormatSchemaResource(t, mockSchemaName, mockSchemaModel),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(mockResourceName, plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mockResourceName, "name", mockSchemaModel.Name.ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "owner", mockSchemaModel.Owner.ValueString()),
				),
			},
			{
				// ImportState testing
				ResourceName:            mockResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"last_updated", "if_not_exists", "drop_cascade", "policy", "database"},
			},
		},
	})
}

func TestAccPostgresqlSchemaDataSource(t *testing.T) {
	runOpts := test.PostgresContainerRunOptions{
		Database: "postgres",
		Username: "postgres",
	}
	test.LoadPostgresTestContainer(t, runOpts, true)

	schemaName := "test_ds_schema"
	dataSourceName := fmt.Sprintf("data.postgresql_schema.%s", schemaName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "postgresql_schema" "%s" {
  name = "%s"
  owner = "postgres"
}

data "postgresql_schema" "%s" {
  name = postgresql_schema.%s.name
}
`, schemaName, schemaName, schemaName, schemaName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "name", schemaName),
					resource.TestCheckResourceAttr(dataSourceName, "owner", "postgres"),
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
				),
			},
		},
	})
}

func testAccFormatSchemaResource(t *testing.T, name string, model postgresqlSchemaModel) string {
	t.Helper()

	config := fmt.Sprintf(`
resource "postgresql_schema" "%s" {
  name         = "%s"
  owner        = "%s"
  if_not_exists = %t
  drop_cascade = %t
  policy       = "%s"
}
`, name,
		model.Name.ValueString(),
		model.Owner.ValueString(),
		model.IfNotExists.ValueBool(),
		model.DropCascade.ValueBool(),
		model.Policy.ValueString(),
	)

	return config
}
