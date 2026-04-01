package provider

import (
	"fmt"
	"terraform-provider-postgresql/internal/test"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPostgresqlGrantResource(t *testing.T) {
	runOpts := test.PostgresContainerRunOptions{
		Database: "test_grant_db",
		Username: "test_grant_user",
	}
	test.LoadPostgresTestContainer(t, runOpts, true)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing - schema grant
			{
				Config: testAccPostgresqlGrantResourceSchemaConfig("test_schema", "test_grant_role"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("postgresql_grant.test", "object_type", "schema"),
					resource.TestCheckResourceAttr("postgresql_grant.test", "object_name", "test_schema"),
					resource.TestCheckResourceAttr("postgresql_grant.test", "role", "test_grant_role"),
					resource.TestCheckResourceAttr("postgresql_grant.test", "with_grant_option", "false"),
					resource.TestCheckTypeSetElemAttr("postgresql_grant.test", "privileges.*", "USAGE"),
				),
			},
			// ImportState testing — import by object_type:schema:object_name:role
			{
				ResourceName:            "postgresql_grant.test",
				ImportState:             true,
				ImportStateId:           "schema::test_schema:test_grant_role",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"id"},
				Destroy:                 false,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccPostgresqlGrantResource_DatabasePrivileges(t *testing.T) {
	runOpts := test.PostgresContainerRunOptions{
		Database: "test_grant_db",
		Username: "test_grant_user",
	}
	test.LoadPostgresTestContainer(t, runOpts, true)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPostgresqlGrantResourceDatabaseConfig("test_grant_db", "test_db_role"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("postgresql_grant.db_test", "object_type", "database"),
					resource.TestCheckResourceAttr("postgresql_grant.db_test", "object_name", "test_grant_db"),
					resource.TestCheckResourceAttr("postgresql_grant.db_test", "role", "test_db_role"),
					resource.TestCheckTypeSetElemAttr("postgresql_grant.db_test", "privileges.*", "CONNECT"),
				),
			},
		},
	})
}

func testAccPostgresqlGrantResourceSchemaConfig(schemaName, roleName string) string {
	return fmt.Sprintf(`
resource "postgresql_schema" "test" {
  name = "%s"
}

resource "postgresql_role" "test" {
  name = "%s"
  login = false
}

resource "postgresql_grant" "test" {
  object_type = "schema"
  object_name = postgresql_schema.test.name
  role        = postgresql_role.test.name
  privileges  = ["USAGE"]
}
`, schemaName, roleName)
}

func testAccPostgresqlGrantResourceDatabaseConfig(dbName, roleName string) string {
	return fmt.Sprintf(`
resource "postgresql_role" "db_test" {
  name  = "%s"
  login = false
}

resource "postgresql_grant" "db_test" {
  object_type = "database"
  object_name = "%s"
  role        = postgresql_role.db_test.name
  privileges  = ["CONNECT"]
}
`, roleName, dbName)
}
