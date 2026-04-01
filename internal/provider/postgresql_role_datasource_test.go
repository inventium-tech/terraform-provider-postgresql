package provider

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"terraform-provider-postgresql/internal/helpers"
	"terraform-provider-postgresql/internal/test"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestMatchesLoginFilter(t *testing.T) {
	testCases := []struct {
		name    string
		filter  types.Bool
		actual  bool
		matches bool
	}{
		{
			name:    "null filter matches",
			filter:  types.BoolNull(),
			actual:  true,
			matches: true,
		},
		{
			name:    "match true",
			filter:  types.BoolValue(true),
			actual:  true,
			matches: true,
		},
		{
			name:    "match false",
			filter:  types.BoolValue(false),
			actual:  false,
			matches: true,
		},
		{
			name:    "mismatch",
			filter:  types.BoolValue(true),
			actual:  false,
			matches: false,
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if matchesLoginFilter(tt.filter, tt.actual) != tt.matches {
				t.Fatalf("expected match=%t for filter=%v actual=%t", tt.matches, tt.filter, tt.actual)
			}
		})
	}
}

func TestAccPostgresqlRoleDataSource(t *testing.T) {
	runOpts := test.PostgresContainerRunOptions{
		Database: "test_role_datasource_db",
		Username: "test_role_datasource_user",
	}
	test.LoadPostgresTestContainer(t, runOpts, true)

	loginRoleName := "ds_login_role"
	noLoginRoleName := "ds_group_role"
	loginDataSource := fmt.Sprintf("data.postgresql_role.%s", loginRoleName)
	noLoginDataSource := fmt.Sprintf("data.postgresql_role.%s", noLoginRoleName)

	const connectionLimit int32 = 5

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFormatRoleDataSource(t, loginRoleName, noLoginRoleName, connectionLimit),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(loginDataSource, "name", loginRoleName),
					resource.TestCheckResourceAttr(loginDataSource, "login", "true"),
					resource.TestCheckResourceAttr(loginDataSource, "create_db", "true"),
					resource.TestCheckResourceAttr(loginDataSource, "create_role", "true"),
					resource.TestCheckResourceAttr(loginDataSource, "inherit", "true"),
					resource.TestCheckResourceAttr(loginDataSource, "connection_limit", strconv.Itoa(int(connectionLimit))),
					resource.TestCheckResourceAttr(loginDataSource, "valid_until", "infinity"),
					resource.TestCheckResourceAttr(noLoginDataSource, "name", noLoginRoleName),
					resource.TestCheckResourceAttr(noLoginDataSource, "login", "false"),
					resource.TestCheckResourceAttr(noLoginDataSource, "connection_limit", "-1"),
				),
			},
			{
				Config:      testAccFormatRoleDataSourceMismatch(t, loginRoleName),
				ExpectError: regexp.MustCompile("Login filter mismatch"),
			},
		},
	})
}

func testAccFormatRoleDataSource(t *testing.T, loginRoleName, noLoginRoleName string, loginRoleConnLimit int32) string {
	t.Helper()

	result := []string{
		fmt.Sprintf(`resource "postgresql_role" "%s" {`, loginRoleName),
		fmt.Sprintf(`  name = "%s"`, loginRoleName),
		`  login = true`,
		`  create_db = true`,
		`  create_role = true`,
		fmt.Sprintf(`  connection_limit = %d`, loginRoleConnLimit),
		`  valid_until = "infinity"`,
		`}`,
		``,
		fmt.Sprintf(`resource "postgresql_role" "%s" {`, noLoginRoleName),
		fmt.Sprintf(`  name = "%s"`, noLoginRoleName),
		`  login = false`,
		`}`,
		``,
		fmt.Sprintf(`data "postgresql_role" "%s" {`, loginRoleName),
		fmt.Sprintf(`  name = postgresql_role.%s.name`, loginRoleName),
		`  login = true`,
		`}`,
		``,
		fmt.Sprintf(`data "postgresql_role" "%s" {`, noLoginRoleName),
		fmt.Sprintf(`  name = postgresql_role.%s.name`, noLoginRoleName),
		`  login = false`,
		`}`,
	}

	result = helpers.CleanUpSlice(result)
	return strings.Join(result, "\n")
}

func testAccFormatRoleDataSourceMismatch(t *testing.T, roleName string) string {
	t.Helper()

	result := []string{
		fmt.Sprintf(`resource "postgresql_role" "%s" {`, roleName),
		fmt.Sprintf(`  name = "%s"`, roleName),
		`  login = true`,
		`}`,
		``,
		fmt.Sprintf(`data "postgresql_role" "%s" {`, roleName),
		fmt.Sprintf(`  name = postgresql_role.%s.name`, roleName),
		`  login = false`,
		`}`,
	}

	result = helpers.CleanUpSlice(result)
	return strings.Join(result, "\n")
}
