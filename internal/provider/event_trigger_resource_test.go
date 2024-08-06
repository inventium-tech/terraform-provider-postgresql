package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
	"gocloud.dev/postgres"
	"strconv"
	"terraform-provider-postgresql/internal/client"
	"terraform-provider-postgresql/internal/test"
	"testing"
)

func TestAccEventTriggerResource(t *testing.T) {
	runOpts := test.PostgresContainerRunOptions{
		Database: "test_event_trigger_resource_db",
		Username: "test_event_trigger_resource_user",
	}
	pgContainer := test.LoadPostgresTestContainer(t, runOpts, true)
	connString := test.GetPostgresConnectionString(t, pgContainer)
	ctx := context.TODO()

	db, err := postgres.Open(ctx, connString)
	assert.NoError(t, err)
	defer db.Close()

	mockUserFunctionCreateParams := client.UserFunctionCreateParams{
		Name:    "test_event_trigger_resource_func",
		Args:    nil,
		Returns: "event_trigger",
		Lang:    "plpgsql",
		Body:    "BEGIN RAISE NOTICE 'DDL command executed'; END;",
		Replace: true,
	}
	mockEventTriggerModel := client.EventTriggerModel{
		Name:     "test_event_trigger_resource",
		Event:    "ddl_command_start",
		Tags:     []string{"CREATE TABLE"},
		ExecFunc: mockUserFunctionCreateParams.Name,
		Enabled:  true,
		Database: runOpts.Database,
		Owner:    runOpts.Username,
		Comment:  "test comment",
	}
	mockResourceId := "test_event_trigger"
	mockResourceName := fmt.Sprintf("postgresql_event_trigger.%s", mockResourceId)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck: func() {
			userFunctionRepo := client.NewUserFunctionRepository(db)
			assert.NoError(t, userFunctionRepo.Create(ctx, mockUserFunctionCreateParams))
		},
		Steps: []resource.TestStep{
			{
				// Create and Read testing
				Config: testAccEventTriggerToTFResource(t, mockResourceId, mockEventTriggerModel),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mockResourceName, "name", mockEventTriggerModel.Name),
					resource.TestCheckResourceAttr(mockResourceName, "database", mockEventTriggerModel.Database),
					resource.TestCheckResourceAttr(mockResourceName, "event", mockEventTriggerModel.Event),
					resource.TestCheckResourceAttr(mockResourceName, "exec_func", mockEventTriggerModel.ExecFunc),
					resource.TestCheckResourceAttr(mockResourceName, "enabled", strconv.FormatBool(mockEventTriggerModel.Enabled)),
					resource.TestCheckResourceAttr(mockResourceName, "tags.#", strconv.Itoa(len(mockEventTriggerModel.Tags))),
					resource.TestCheckResourceAttr(mockResourceName, "tags.0", mockEventTriggerModel.Tags[0]),
					resource.TestCheckResourceAttr(mockResourceName, "owner", mockEventTriggerModel.Owner),
					resource.TestCheckResourceAttr(mockResourceName, "comment", mockEventTriggerModel.Comment),
				),
			},
			{
				// ImportState testing
				ResourceName:            mockResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
			{
				// Update testing - Properties without re-creating the resource
				PreConfig: func() {
					mockEventTriggerModel.Name = "test_event_trigger_resource_modified"
					mockEventTriggerModel.Enabled = false
					mockEventTriggerModel.Comment = "test comment modified"
				},
				Config: testAccEventTriggerToTFResource(t, mockResourceId, mockEventTriggerModel),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mockResourceName, "name", mockEventTriggerModel.Name),
					resource.TestCheckResourceAttr(mockResourceName, "enabled", strconv.FormatBool(mockEventTriggerModel.Enabled)),
					resource.TestCheckResourceAttr(mockResourceName, "comment", mockEventTriggerModel.Comment),
				),
			},
			{
				// Update testing - Properties WITH re-creating the resource
				PreConfig: func() {
					mockEventTriggerModel.Event = "ddl_command_end"
				},
				Config: testAccEventTriggerToTFResource(t, mockResourceId, mockEventTriggerModel),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mockResourceName, "name", mockEventTriggerModel.Name),
					resource.TestCheckResourceAttr(mockResourceName, "event", mockEventTriggerModel.Event),
				),
			},
			{
				// Delete testing
				Config:  testAccEventTriggerToTFResource(t, mockResourceId, mockEventTriggerModel),
				Destroy: true,
			},
		},
	})
}

func testAccEventTriggerToTFResource(t *testing.T, resId string, pgModel client.EventTriggerModel) string {
	t.Helper()
	return fmt.Sprintf(`resource "postgresql_event_trigger" "%s" {
			name          = "%s"
			event         = "%s"		
			tags          =  %s
			exec_func     = "%s"
			database      = "%s"
			enabled       =  %v
			comment       = "%s"
		}`, resId, pgModel.Name, pgModel.Event, sliceToTerraformSetString(pgModel.Tags), pgModel.ExecFunc, pgModel.Database, pgModel.Enabled, pgModel.Comment)
}
