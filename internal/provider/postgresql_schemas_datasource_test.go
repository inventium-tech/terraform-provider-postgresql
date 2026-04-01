package provider

import (
	"fmt"
	"testing"

	"terraform-provider-postgresql/internal/test"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPostgresqlSchemasDataSource(t *testing.T) {
	runOpts := test.PostgresContainerRunOptions{
		Database: "postgres",
		Username: "postgres",
	}
	test.LoadPostgresTestContainer(t, runOpts, true)

	dataSourceName := "data.postgresql_schemas.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "postgresql_schema" "test1" {
  name = "test_list_1"
}

resource "postgresql_schema" "test2" {
  name = "test_list_2"
}

data "postgresql_schemas" "test" {
  include_system_schemas = false
  like_any_patterns      = ["test_list_%%"]
  depends_on             = [postgresql_schema.test1, postgresql_schema.test2]
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
					resource.TestCheckResourceAttr(dataSourceName, "schemas.#", "2"),
				),
			},
		},
	})
}
