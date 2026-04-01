package provider

import (
	"fmt"
	"testing"

	"terraform-provider-postgresql/internal/test"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccPostgresqlExtensionResource(t *testing.T) {
	runOpts := test.PostgresContainerRunOptions{
		Database: "postgres",
		Username: "postgres",
	}
	test.LoadPostgresTestContainer(t, runOpts, true)

	mockExtModel := postgresqlExtensionModel{
		Name:        types.StringValue("uuid-ossp"),
		Schema:      types.StringValue("public"),
		Cascade:     types.BoolValue(false),
		DropCascade: types.BoolValue(false),
	}

	mockExtName := "uuid_ossp"
	mockResourceName := fmt.Sprintf("postgresql_extension.%s", mockExtName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create and Read testing
				Config: testAccFormatExtensionResource(t, mockExtName, mockExtModel),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(mockResourceName, plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mockResourceName, "name", mockExtModel.Name.ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "schema", mockExtModel.Schema.ValueString()),
					resource.TestCheckResourceAttrSet(mockResourceName, "version"),
				),
			},
			{
				// ImportState testing — import by extension name only
				ResourceName:            mockResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"last_updated", "cascade", "drop_cascade", "database"},
				Destroy:                 false,
			},
		},
	})
}

func TestAccPostgresqlExtensionResource_ImportWithDatabase(t *testing.T) {
	runOpts := test.PostgresContainerRunOptions{
		Database: "postgres",
		Username: "postgres",
	}
	test.LoadPostgresTestContainer(t, runOpts, true)

	mockExtModel := postgresqlExtensionModel{
		Name:        types.StringValue("hstore"),
		Schema:      types.StringValue("public"),
		Database:    types.StringValue("postgres"),
		Cascade:     types.BoolValue(false),
		DropCascade: types.BoolValue(false),
	}

	mockExtName := "hstore"
	mockResourceName := fmt.Sprintf("postgresql_extension.%s", mockExtName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFormatExtensionResource(t, mockExtName, mockExtModel),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mockResourceName, "name", "hstore"),
					resource.TestCheckResourceAttr(mockResourceName, "database", "postgres"),
				),
			},
			{
				// ImportState with database:extension_name format
				ResourceName:            mockResourceName,
				ImportState:             true,
				ImportStateId:           "postgres:hstore",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"last_updated", "cascade", "drop_cascade"},
				Destroy:                 false,
			},
		},
	})
}

func TestAccPostgresqlExtensionResource_WithVersion(t *testing.T) {
	runOpts := test.PostgresContainerRunOptions{
		Database: "postgres",
		Username: "postgres",
	}
	test.LoadPostgresTestContainer(t, runOpts, true)

	mockExtModel := postgresqlExtensionModel{
		Name:        types.StringValue("pg_trgm"),
		Schema:      types.StringValue("public"),
		Cascade:     types.BoolValue(false),
		DropCascade: types.BoolValue(false),
	}

	mockExtName := "pg_trgm"
	mockResourceName := fmt.Sprintf("postgresql_extension.%s", mockExtName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create and Read testing
				Config: testAccFormatExtensionResource(t, mockExtName, mockExtModel),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(mockResourceName, plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mockResourceName, "name", mockExtModel.Name.ValueString()),
					resource.TestCheckResourceAttrSet(mockResourceName, "version"),
				),
			},
		},
	})
}

func TestAccPostgresqlExtensionResource_DropCascade(t *testing.T) {
	runOpts := test.PostgresContainerRunOptions{
		Database: "postgres",
		Username: "postgres",
	}
	test.LoadPostgresTestContainer(t, runOpts, true)

	mockExtModel := postgresqlExtensionModel{
		Name:        types.StringValue("citext"),
		Schema:      types.StringValue("public"),
		Cascade:     types.BoolValue(false),
		DropCascade: types.BoolValue(true),
	}

	mockExtName := "citext"
	mockResourceName := fmt.Sprintf("postgresql_extension.%s", mockExtName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create and Read testing with drop_cascade
				Config: testAccFormatExtensionResource(t, mockExtName, mockExtModel),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(mockResourceName, plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mockResourceName, "name", mockExtModel.Name.ValueString()),
				),
			},
		},
	})
}

func testAccFormatExtensionResource(t *testing.T, name string, model postgresqlExtensionModel) string {
	t.Helper()

	config := fmt.Sprintf(`
resource "postgresql_extension" "%s" {
  name        = "%s"`, name, model.Name.ValueString())

	if !model.Schema.IsNull() && model.Schema.ValueString() != "" {
		config += fmt.Sprintf(`
  schema      = "%s"`, model.Schema.ValueString())
	}

	if !model.Version.IsNull() && model.Version.ValueString() != "" {
		config += fmt.Sprintf(`
  version     = "%s"`, model.Version.ValueString())
	}

	if !model.Database.IsNull() && model.Database.ValueString() != "" {
		config += fmt.Sprintf(`
  database    = "%s"`, model.Database.ValueString())
	}

	config += fmt.Sprintf(`
  cascade     = %t
  drop_cascade = %t
}
`, model.Cascade.ValueBool(),
		model.DropCascade.ValueBool(),
	)

	return config
}
