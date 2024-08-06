data "postgresql_event_trigger" "test" {
  name     = "test_trigger"
  database = "postgres"
}
