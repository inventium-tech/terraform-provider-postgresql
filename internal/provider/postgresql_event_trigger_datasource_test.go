package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"strconv"
	"strings"
	"terraform-provider-postgresql/internal/helpers"
	"terraform-provider-postgresql/internal/test"
	"testing"
)

func TestAccPostgresqlEventTriggerDataSource(t *testing.T) {
	runOpts := test.PostgresContainerRunOptions{
		Database: "test_event_trigger_datasource_db",
		Username: "test_event_trigger_datasource_user",
	}

	pgContainer := test.LoadPostgresTestContainer(t, runOpts, true)
	connString := test.GetPostgresConnectionString(t, pgContainer)

	ctx := t.Context()
	conn, err := pgx.Connect(ctx, connString)
	assert.NoError(t, err)

	defer func() { assert.NoError(t, conn.Close(ctx)) }()

	mockEventTriggerExecFuncName := "test_event_trigger_func"
	mockEventTriggerName := "test_event_trigger"
	mockEventTriggerComment := "test event trigger datasource comment"
	mockDataSourceId := "test_event_trigger_datasource"
	mockDataSourceName := fmt.Sprintf("data.postgresql_event_trigger.%s", mockDataSourceId)

	// create a function that will be used by the event trigger
	createFuncSQL := `
	CREATE OR REPLACE FUNCTION %s() RETURNS event_trigger AS $$
	BEGIN
		RAISE NOTICE 'Event trigger function executed';
	END;
	$$ LANGUAGE plpgsql;
	`
	_, err = conn.Exec(ctx, fmt.Sprintf(createFuncSQL, mockEventTriggerExecFuncName))
	assert.NoError(t, err)

	// create the event trigger in the database
	createEventTriggerSQL := `
	CREATE EVENT TRIGGER %s
	ON ddl_command_end
	WHEN TAG IN ('CREATE TABLE')
	EXECUTE PROCEDURE %s();
	`
	_, err = conn.Exec(ctx, fmt.Sprintf(createEventTriggerSQL, mockEventTriggerName, mockEventTriggerExecFuncName))
	assert.NoError(t, err)

	// Add a comment to the event trigger
	_, err = conn.Exec(ctx, fmt.Sprintf("COMMENT ON EVENT TRIGGER %s IS '%s';", mockEventTriggerName, mockEventTriggerComment))
	assert.NoError(t, err)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFormatEventTriggerDataSource(t, mockDataSourceId, mockEventTriggerName, runOpts.Database),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the datasource reads the event trigger correctly
					resource.TestCheckResourceAttr(mockDataSourceName, "name", mockEventTriggerName),
					resource.TestCheckResourceAttr(mockDataSourceName, "database", runOpts.Database),
					resource.TestCheckResourceAttr(mockDataSourceName, "event", "ddl_command_end"),
					resource.TestCheckResourceAttr(mockDataSourceName, "exec_func", mockEventTriggerExecFuncName),
					resource.TestCheckResourceAttr(mockDataSourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(mockDataSourceName, "tags.#", strconv.Itoa(1)),
					resource.TestCheckResourceAttr(mockDataSourceName, "tags.0", "CREATE TABLE"),
					resource.TestCheckResourceAttr(mockDataSourceName, "owner", runOpts.Username),
					resource.TestCheckResourceAttr(mockDataSourceName, "comment", mockEventTriggerComment),
				),
			},
		},
	})
}

func testAccFormatEventTriggerDataSource(t *testing.T, resName, etName, etDatabase string) string {
	t.Helper()

	result := make([]string, 0)
	result = append(result,
		fmt.Sprintf(`data "postgresql_event_trigger" "%s" {`, resName),
		fmt.Sprintf(`  name     = "%s"`, etName),
		fmt.Sprintf(`  database = "%s"`, etDatabase),
		"}",
	)

	result = helpers.CleanUpSlice(result)
	return strings.Join(result, "\n")
}
