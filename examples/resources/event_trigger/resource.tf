terraform {
  required_providers {
    postgresql = {
      source = "registry.terraform.io/inventium-tech/postgresql"
    }
  }
}

provider "postgresql" {}

import {
  id = "postgres.test_trigger_one"
  to = postgresql_event_trigger.test
}

resource "postgresql_event_trigger" "test" {
  name      = "test_trigger_change"
  database  = "postgres"
  event     = "ddl_command_end"
  tags      = ["CREATE TABLE"]
  exec_func = "alter_object_owner"
  enabled   = true
  comment   = "Test event trigger"
  owner     = "postgres"
}

output "event_trigger_data" {
  value = postgresql_event_trigger.test
}
