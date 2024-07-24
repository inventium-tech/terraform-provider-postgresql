package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"strconv"
	"testing"
)

func TestAccEventTriggerResource(t *testing.T) {
	var (
		mockEvtTrigName         = "testing"
		mockEvtTrigEvent        = "ddl_command_start"
		mockEvtTrigTags         = []string{"CREATE TABLE"}
		mockEvtTrigExecFunc     = "func_testing_event_trigger"
		mockEvtTrigDatabase     = testPGDefaultDb
		mockEvtTrigEnabled      = true
		mockEvtTrigComment      = "test comment"
		mockEvtTrigResourceName = fmt.Sprintf("postgresql_event_trigger.%s", mockEvtTrigName)
		mockFnBody              = `
			BEGIN
				RAISE NOTICE 'DDL command executed';
			END`
	)

	pgClient := loadPostgresTestContainer(t, postgresTestContainerConfig{
		image: "postgres:16-alpine",
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck: func() {
			createTestFunction(t, pgClient, mockEvtTrigDatabase, mockEvtTrigExecFunc, mockFnBody, "event_trigger", "plpgsql")
		},
		Steps: []resource.TestStep{
			{
				// Create and Read testing
				Config: testAccEventTriggerTFResource(mockEvtTrigName, mockEvtTrigName, mockEvtTrigEvent, mockEvtTrigExecFunc, mockEvtTrigDatabase, mockEvtTrigComment, mockEvtTrigTags, mockEvtTrigEnabled),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mockEvtTrigResourceName, "name", mockEvtTrigName),
					resource.TestCheckResourceAttr(mockEvtTrigResourceName, "database", mockEvtTrigDatabase),
					resource.TestCheckResourceAttr(mockEvtTrigResourceName, "event", mockEvtTrigEvent),
					resource.TestCheckResourceAttr(mockEvtTrigResourceName, "exec_func", mockEvtTrigExecFunc),
					resource.TestCheckResourceAttr(mockEvtTrigResourceName, "enabled", strconv.FormatBool(mockEvtTrigEnabled)),
					resource.TestCheckResourceAttr(mockEvtTrigResourceName, "tags.#", strconv.Itoa(len(mockEvtTrigTags))),
					resource.TestCheckResourceAttr(mockEvtTrigResourceName, "tags.0", mockEvtTrigTags[0]),
					resource.TestCheckResourceAttr(mockEvtTrigResourceName, "owner", pgClient.GetConfig().Username),
				),
			},
			{
				// ImportState testing
				ResourceName:            mockEvtTrigResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
			{
				// Update testing
				Config: testAccEventTriggerTFResource(mockEvtTrigName, "modified", mockEvtTrigEvent, mockEvtTrigExecFunc, mockEvtTrigDatabase, mockEvtTrigComment, mockEvtTrigTags, mockEvtTrigEnabled),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mockEvtTrigResourceName, "name", "modified"),
					resource.TestCheckResourceAttr(mockEvtTrigResourceName, "database", mockEvtTrigDatabase),
					resource.TestCheckResourceAttr(mockEvtTrigResourceName, "event", mockEvtTrigEvent),
					resource.TestCheckResourceAttr(mockEvtTrigResourceName, "exec_func", mockEvtTrigExecFunc),
					resource.TestCheckResourceAttr(mockEvtTrigResourceName, "enabled", strconv.FormatBool(mockEvtTrigEnabled)),
					resource.TestCheckResourceAttr(mockEvtTrigResourceName, "tags.#", strconv.Itoa(len(mockEvtTrigTags))),
					resource.TestCheckResourceAttr(mockEvtTrigResourceName, "tags.0", mockEvtTrigTags[0]),
					resource.TestCheckResourceAttr(mockEvtTrigResourceName, "owner", pgClient.GetConfig().Username),
				),
			},
			{
				// Delete testing
				Config:  testAccEventTriggerTFResource(mockEvtTrigName, "modified", mockEvtTrigEvent, mockEvtTrigExecFunc, mockEvtTrigDatabase, mockEvtTrigComment, mockEvtTrigTags, mockEvtTrigEnabled),
				Destroy: true,
			},
		},
	})
}

func testAccEventTriggerTFResource(resName, etName, etEvent, etExecFn, etDatabase, etComment string, etTags []string, etEnabled bool) string {
	return fmt.Sprintf(
		`resource "postgresql_event_trigger" "%s" {
			name          = "%s"
			event         = "%s"		
			tags          =  %s
			exec_func     = "%s"
			database      = "%s"
			enabled       =  %v
			comment       = "%s"
		}`,
		resName,
		etName,
		etEvent,
		sliceToStringSet(etTags),
		etExecFn,
		etDatabase,
		etEnabled,
		etComment,
	)
}
