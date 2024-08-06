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

func TestAccEventTriggerDataSource(t *testing.T) {
	runOpts := test.PostgresContainerRunOptions{
		Database: "test_event_trigger_datasource_db",
		Username: "test_event_trigger_datasource_user",
	}
	pgContainer := test.LoadPostgresTestContainer(t, runOpts, true)
	connString := test.GetPostgresConnectionString(t, pgContainer)
	ctx := context.TODO()

	db, err := postgres.Open(ctx, connString)
	assert.NoError(t, err)
	defer db.Close()

	mockUserFunctionCreateParams := client.UserFunctionCreateParams{
		Name:    "test_event_trigger_datasource_func",
		Args:    nil,
		Returns: "event_trigger",
		Lang:    "plpgsql",
		Body:    "BEGIN RAISE NOTICE 'DDL command executed'; END;",
		Replace: true,
	}
	mockEventTriggerCreateParams := client.EventTriggerCreateParams{
		Name:     "test_event_trigger_datasource",
		Event:    "ddl_command_start",
		ExecFunc: mockUserFunctionCreateParams.Name,
		Enabled:  true,
		Tags:     []string{"CREATE TABLE"},
		Comment:  "test comment",
	}
	mockResourceId := "test_event_trigger"
	mockResourceName := fmt.Sprintf("data.postgresql_event_trigger.%s", mockResourceId)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck: func() {
			userFunctionRepo := client.NewUserFunctionRepository(db)
			eventTriggerRepo := client.NewEventTriggerRepository(db)

			assert.NoError(t, userFunctionRepo.Create(ctx, mockUserFunctionCreateParams))
			assert.NoError(t, eventTriggerRepo.Create(ctx, mockEventTriggerCreateParams))
		},
		Steps: []resource.TestStep{
			{
				Config: testAccEventTriggerToTFDataSource(t, mockResourceId, mockEventTriggerCreateParams.Name, runOpts.Database),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mockResourceName, "name", mockEventTriggerCreateParams.Name),
					resource.TestCheckResourceAttr(mockResourceName, "database", runOpts.Database),
					resource.TestCheckResourceAttr(mockResourceName, "event", mockEventTriggerCreateParams.Event),
					resource.TestCheckResourceAttr(mockResourceName, "exec_func", mockUserFunctionCreateParams.Name),
					resource.TestCheckResourceAttr(mockResourceName, "enabled", strconv.FormatBool(mockEventTriggerCreateParams.Enabled)),
					resource.TestCheckResourceAttr(mockResourceName, "tags.#", strconv.Itoa(len(mockEventTriggerCreateParams.Tags))),
					resource.TestCheckResourceAttr(mockResourceName, "tags.0", mockEventTriggerCreateParams.Tags[0]),
					resource.TestCheckResourceAttr(mockResourceName, "owner", runOpts.Username),
					resource.TestCheckResourceAttr(mockResourceName, "comment", mockEventTriggerCreateParams.Comment),
				)},
		},
	})
}

func testAccEventTriggerToTFDataSource(t *testing.T, resName, etName, etDatabase string) string {
	t.Helper()
	return fmt.Sprintf(`data "postgresql_event_trigger" "%s" {
  			name     = "%s"
			database = "%s"
		}`, resName, etName, etDatabase)
}
