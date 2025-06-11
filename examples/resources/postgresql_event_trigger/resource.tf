resource "postgresql_event_trigger" "test" {
  name      = "test_trigger_one"
  database  = "postgres"
  event     = "ddl_command_end"
  tags      = ["CREATE TABLE"]
  exec_func = "alter_object_owner"
  enabled   = true
  comment   = "Test event trigger"
  owner     = "postgres"
}
