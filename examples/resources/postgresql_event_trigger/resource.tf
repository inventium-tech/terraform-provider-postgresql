# Example: PostgreSQL Event Trigger
#
# This example creates an event trigger that fires on DDL commands.
# Event triggers are database-wide triggers that capture DDL events.

# Prerequisites: The exec_func must already exist
# This example assumes a function named 'alter_object_owner' exists

# Basic event trigger for CREATE TABLE statements
resource "postgresql_event_trigger" "track_table_creation" {
  name      = "track_table_creation"
  database  = "postgres"
  event     = "ddl_command_end"
  tags      = ["CREATE TABLE"]
  exec_func = "alter_object_owner"
  enabled   = true
  comment   = "Tracks when new tables are created"
  owner     = "postgres"
}

# Event trigger for multiple DDL commands
resource "postgresql_event_trigger" "track_schema_changes" {
  name      = "track_schema_changes"
  database  = "postgres"
  event     = "ddl_command_end"
  tags      = ["CREATE TABLE", "ALTER TABLE", "DROP TABLE"]
  exec_func = "log_schema_changes"
  enabled   = true
  comment   = "Logs all table-related schema changes"
  owner     = "postgres"
}

# Event trigger for table rewrites (disabled by default)
resource "postgresql_event_trigger" "prevent_table_rewrite" {
  name      = "prevent_table_rewrite"
  database  = "postgres"
  event     = "table_rewrite"
  tags      = [] # table_rewrite event doesn't use tags
  exec_func = "check_rewrite_safety"
  enabled   = false
  comment   = "Prevents unsafe table rewrites when enabled"
  owner     = "postgres"
}

# Output the event trigger names for reference
output "event_trigger_names" {
  value = [
    postgresql_event_trigger.track_table_creation.name,
    postgresql_event_trigger.track_schema_changes.name,
    postgresql_event_trigger.prevent_table_rewrite.name,
  ]
  description = "Names of created event triggers"
}
