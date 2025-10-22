package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"strconv"
	"strings"
	"terraform-provider-postgresql/internal/helpers"
	"terraform-provider-postgresql/internal/test"
	"testing"
)

func TestAccPostgresqlEventTriggerResource(t *testing.T) {
	runOpts := test.PostgresContainerRunOptions{
		Database: "test_event_trigger_resource_db",
		Username: "test_event_trigger_resource_user",
	}

	pgContainer := test.LoadPostgresTestContainer(t, runOpts, true)
	connString := test.GetPostgresConnectionString(t, pgContainer)

	ctx := t.Context()
	conn, err := pgx.Connect(ctx, connString)
	assert.NoError(t, err)

	defer func() { assert.NoError(t, conn.Close(ctx)) }()

	// Create a function that will be used by the event trigger
	_, err = conn.Exec(ctx, `
		CREATE OR REPLACE FUNCTION test_event_trigger_func() RETURNS event_trigger AS $$
		BEGIN
			RAISE NOTICE 'Event trigger function executed';
		END;
		$$ LANGUAGE plpgsql;
	`)
	assert.NoError(t, err)

	mockTagsSet := []attr.Value{
		types.StringValue("ALTER TABLE"),
		types.StringValue("CREATE TABLE"),
		types.StringValue("DROP TABLE"),
	}

	mockEventTriggerModel := resourceModelEventTrigger{
		Name:     types.StringValue("test_event_trigger"),
		Event:    types.StringValue("ddl_command_start"),
		Tags:     types.SetValueMust(types.StringType, mockTagsSet),
		ExecFunc: types.StringValue("test_event_trigger_func"),
		Enabled:  types.BoolValue(true),
		Comment:  types.StringValue("test event trigger comment"),
	}

	mockUpdatedName := "test_event_trigger_updated"
	mockUpdatedComment := "updated test event trigger comment"
	mockUpdatedEvent := "ddl_command_end"
	mockUpdatedTagsSet := []attr.Value{
		types.StringValue("CREATE INDEX"),
		types.StringValue("DROP INDEX"),
	}

	mockEventTriggerName := "test_event_trigger"
	mockResourceName := fmt.Sprintf("postgresql_event_trigger.%s", mockEventTriggerName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create and Read testing
				Config: testAccFormatEventTriggerResource(t, mockEventTriggerName, mockEventTriggerModel),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(mockResourceName, plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mockResourceName, "name", mockEventTriggerModel.Name.ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "event", mockEventTriggerModel.Event.ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "exec_func", mockEventTriggerModel.ExecFunc.ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "tags.#", strconv.Itoa(len(mockTagsSet))),
					resource.TestCheckResourceAttr(mockResourceName, "tags.0", mockTagsSet[0].(types.String).ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "tags.1", mockTagsSet[1].(types.String).ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "tags.2", mockTagsSet[2].(types.String).ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "enabled", strconv.FormatBool(mockEventTriggerModel.Enabled.ValueBool())),
					resource.TestCheckResourceAttr(mockResourceName, "database", runOpts.Database),
					resource.TestCheckResourceAttr(mockResourceName, "owner", runOpts.Username),
					resource.TestCheckResourceAttr(mockResourceName, "comment", mockEventTriggerModel.Comment.ValueString()),
				),
			},
			{
				// ImportState testing
				ResourceName:            mockResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateId:           fmt.Sprintf("%s.%s", runOpts.Database, mockEventTriggerName),
				ImportStateVerifyIgnore: []string{"last_updated"},
				Destroy:                 false,
			},
			{
				// Update testing - Properties WITHOUT Resource replacement (name, enabled, comment)
				Config: func() string {
					mockEventTriggerModel.Name = types.StringValue(mockUpdatedName)
					mockEventTriggerModel.Enabled = types.BoolValue(false)
					mockEventTriggerModel.Comment = types.StringValue(mockUpdatedComment)
					return testAccFormatEventTriggerResource(t, mockEventTriggerName, mockEventTriggerModel)
				}(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(mockResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mockResourceName, "name", mockUpdatedName),
					resource.TestCheckResourceAttr(mockResourceName, "enabled", strconv.FormatBool(false)),
					resource.TestCheckResourceAttr(mockResourceName, "comment", mockUpdatedComment),
				),
			},
			{
				// Update testing - Properties WITH Resource replacement (event, exec_func, tags)
				Config: func() string {
					mockEventTriggerModel.Event = types.StringValue(mockUpdatedEvent)
					mockEventTriggerModel.Tags = types.SetValueMust(types.StringType, mockUpdatedTagsSet)
					return testAccFormatEventTriggerResource(t, mockEventTriggerName, mockEventTriggerModel)
				}(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(mockResourceName, plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mockResourceName, "name", mockUpdatedName),
					resource.TestCheckResourceAttr(mockResourceName, "event", "ddl_command_end"),
					resource.TestCheckResourceAttr(mockResourceName, "tags.#", strconv.Itoa(len(mockUpdatedTagsSet))),
					resource.TestCheckResourceAttr(mockResourceName, "tags.0", mockUpdatedTagsSet[0].(types.String).ValueString()),
					resource.TestCheckResourceAttr(mockResourceName, "tags.1", mockUpdatedTagsSet[1].(types.String).ValueString()),
				),
			},
		},
	})
}

func testAccFormatEventTriggerResource(t *testing.T, resName string, m resourceModelEventTrigger) string {
	t.Helper()

	result := make([]string, 0)
	result = append(result,
		fmt.Sprintf(`resource "postgresql_event_trigger" "%s" {`, resName),
		test.FormatTerraformAttribute(t, m.Name, "name"),
		test.FormatTerraformAttribute(t, m.Event, "event"),
		test.FormatTerraformAttribute(t, m.Tags, "tags"),
		test.FormatTerraformAttribute(t, m.ExecFunc, "exec_func"),
		test.FormatTerraformAttribute(t, m.Enabled, "enabled"),
		test.FormatTerraformAttribute(t, m.Database, "database"),
		test.FormatTerraformAttribute(t, m.Owner, "owner"),
		test.FormatTerraformAttribute(t, m.Comment, "comment"),
		"}",
	)

	result = helpers.CleanUpSlice(result)
	return strings.Join(result, "\n")
}
