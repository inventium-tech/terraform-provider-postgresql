# Example: PostgreSQL Event Trigger Data Source
#
# This example demonstrates querying existing PostgreSQL event triggers
# to inspect their configuration and use the information in other resources.

# Query a specific event trigger by name
data "postgresql_event_trigger" "ddl_monitor" {
  name     = "track_table_creation"
  database = "postgres"
}

# Use the data source output in locals for conditional logic
locals {
  event_trigger_enabled = data.postgresql_event_trigger.ddl_monitor.enabled

  # Extract information about the event trigger
  trigger_info = {
    name      = data.postgresql_event_trigger.ddl_monitor.name
    event     = data.postgresql_event_trigger.ddl_monitor.event
    exec_func = data.postgresql_event_trigger.ddl_monitor.exec_func
    owner     = data.postgresql_event_trigger.ddl_monitor.owner
    tags      = data.postgresql_event_trigger.ddl_monitor.tags
  }
}

# Example: Use event trigger data for monitoring/alerting configuration
output "event_trigger_status" {
  description = "Status information about the queried event trigger"
  value = {
    name             = data.postgresql_event_trigger.ddl_monitor.name
    database         = data.postgresql_event_trigger.ddl_monitor.database
    event            = data.postgresql_event_trigger.ddl_monitor.event
    exec_function    = data.postgresql_event_trigger.ddl_monitor.exec_func
    enabled          = data.postgresql_event_trigger.ddl_monitor.enabled
    owner            = data.postgresql_event_trigger.ddl_monitor.owner
    comment          = data.postgresql_event_trigger.ddl_monitor.comment
    filtered_by_tags = length(data.postgresql_event_trigger.ddl_monitor.tags) > 0
  }
}

# Example: Conditional resource creation based on event trigger state
# Only create a new trigger if the existing one is disabled
resource "postgresql_event_trigger" "backup_trigger" {
  count = local.event_trigger_enabled ? 0 : 1

  name      = "backup_ddl_monitor"
  database  = "postgres"
  event     = data.postgresql_event_trigger.ddl_monitor.event
  exec_func = data.postgresql_event_trigger.ddl_monitor.exec_func
  tags      = data.postgresql_event_trigger.ddl_monitor.tags
  enabled   = true
  comment   = "Backup event trigger created because primary is disabled"
  owner     = "postgres"
}

# Example: Create monitoring alert configuration based on trigger settings
output "monitoring_config" {
  description = "Monitoring configuration derived from event trigger"
  value = {
    alert_name         = "EventTrigger-${data.postgresql_event_trigger.ddl_monitor.name}"
    monitor_database   = data.postgresql_event_trigger.ddl_monitor.database
    expected_enabled   = true
    current_enabled    = data.postgresql_event_trigger.ddl_monitor.enabled
    requires_attention = !data.postgresql_event_trigger.ddl_monitor.enabled
  }
}
