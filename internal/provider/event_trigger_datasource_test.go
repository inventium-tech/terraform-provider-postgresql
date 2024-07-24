package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"strconv"
	"testing"
)

func TestAccEventTriggerDataSource(t *testing.T) {
	var (
		mockEvtTrigName         = "testing"
		mockEvtTrigEvent        = "ddl_command_start"
		mockEvtTrigTags         = []string{"CREATE TABLE"}
		mockEvtTrigExecFunc     = "func_testing_event_trigger"
		mockEvtTrigDatabase     = testPGDefaultDb
		mockEvtTrigEnabled      = true
		mockEvtTrigComment      = "test comment"
		mockEvtTrigOwner        = "tester"
		mockEvtTrigResourceName = fmt.Sprintf("data.postgresql_event_trigger.%s", mockEvtTrigName)
		mockFnBody              = `
			BEGIN
				RAISE NOTICE 'DDL command executed';
			END`
	)

	pgClient := loadPostgresTestContainer(t, postgresTestContainerConfig{
		image:    "postgres:16-alpine",
		username: mockEvtTrigOwner,
	})

	mockModel := eventTriggerResModel{
		Name:     types.StringValue(mockEvtTrigName),
		Event:    types.StringValue(mockEvtTrigEvent),
		Tags:     types.SetValueMust(types.StringType, []attr.Value{types.StringValue("CREATE TABLE")}),
		ExecFunc: types.StringValue(mockEvtTrigExecFunc),
		Enabled:  types.BoolValue(true),
		Database: types.StringValue(testPGDefaultDb),
		Comment:  types.StringValue(mockEvtTrigComment),
		Owner:    types.StringValue(mockEvtTrigOwner),
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck: func() {
			createTestFunction(t, pgClient, mockEvtTrigDatabase, mockEvtTrigExecFunc, mockFnBody, "event_trigger", "plpgsql")
			createTestEventTrigger(t, pgClient, mockModel)

		},
		Steps: []resource.TestStep{
			{
				Config: testAccEventTriggerTFDataSource(mockEvtTrigName, mockEvtTrigDatabase),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mockEvtTrigResourceName, "name", mockEvtTrigName),
					resource.TestCheckResourceAttr(mockEvtTrigResourceName, "database", mockEvtTrigDatabase),
					resource.TestCheckResourceAttr(mockEvtTrigResourceName, "event", mockEvtTrigEvent),
					resource.TestCheckResourceAttr(mockEvtTrigResourceName, "exec_func", mockEvtTrigExecFunc),
					resource.TestCheckResourceAttr(mockEvtTrigResourceName, "enabled", strconv.FormatBool(mockEvtTrigEnabled)),
					resource.TestCheckResourceAttr(mockEvtTrigResourceName, "tags.#", strconv.Itoa(len(mockEvtTrigTags))),
					resource.TestCheckResourceAttr(mockEvtTrigResourceName, "tags.0", mockEvtTrigTags[0]),
					resource.TestCheckResourceAttr(mockEvtTrigResourceName, "owner", mockEvtTrigOwner),
					resource.TestCheckResourceAttr(mockEvtTrigResourceName, "comment", mockEvtTrigComment),
				),
			},
		},
	})
}

func testAccEventTriggerTFDataSource(etName, etDatabase string) string {
	return fmt.Sprintf(
		`data "postgresql_event_trigger" "%s" {
  			name     = "%s"
			database = "%s"
		}`,
		etName,
		etName,
		etDatabase,
	)
}
