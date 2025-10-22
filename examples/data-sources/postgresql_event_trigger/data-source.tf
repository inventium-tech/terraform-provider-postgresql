data "postgresql_event_trigger" "example" {
  name     = "test_event_trigger"
  database = "postgres"
}
