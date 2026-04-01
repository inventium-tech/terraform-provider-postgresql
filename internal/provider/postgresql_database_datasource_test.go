package provider

import (
	"fmt"
	"testing"

	"terraform-provider-postgresql/internal/test"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPostgresqlDatabaseDataSource(t *testing.T) {
	runOpts := test.PostgresContainerRunOptions{
		Database: "postgres",
		Username: "postgres",
	}
	test.LoadPostgresTestContainer(t, runOpts, true)

	dbName := "test_ds_db"
	dataSourceName := fmt.Sprintf("data.postgresql_database.%s", dbName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// First create a database resource to query
				Config: fmt.Sprintf(`
resource "postgresql_database" "%s" {
  name = "%s"
  owner = "postgres"
}

data "postgresql_database" "%s" {
  name = postgresql_database.%s.name
}
`, dbName, dbName, dbName, dbName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "name", dbName),
					resource.TestCheckResourceAttr(dataSourceName, "owner", "postgres"),
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
					resource.TestCheckResourceAttrSet(dataSourceName, "encoding"),
					resource.TestCheckResourceAttrSet(dataSourceName, "collation"),
					resource.TestCheckResourceAttrSet(dataSourceName, "ctype"),
				),
			},
		},
	})
}
