package provider

import (
	"fmt"
	"testing"

	"terraform-provider-postgresql/internal/test"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPostgresqlExtensionDataSource(t *testing.T) {
	runOpts := test.PostgresContainerRunOptions{
		Database: "postgres",
		Username: "postgres",
	}
	test.LoadPostgresTestContainer(t, runOpts, true)

	extName := "uuid_ossp"
	extRealName := "uuid-ossp"
	dataSourceName := fmt.Sprintf("data.postgresql_extension.%s", extName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// First create an extension resource to query
				Config: fmt.Sprintf(`
resource "postgresql_extension" "%s" {
  name = "%s"
}

data "postgresql_extension" "%s" {
  name = postgresql_extension.%s.name
}
`, extName, extRealName, extName, extName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "name", extRealName),
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
					resource.TestCheckResourceAttrSet(dataSourceName, "schema"),
					resource.TestCheckResourceAttrSet(dataSourceName, "version"),
				),
			},
		},
	})
}
