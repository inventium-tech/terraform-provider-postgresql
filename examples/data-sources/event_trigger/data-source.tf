terraform {
  required_providers {
    postgresql = {
      source = "registry.terraform.io/inventium-tech/postgresql"
    }
  }
}

provider "postgresql" {}

data "postgresql_event_trigger" "test" {
  name     = "test_trigger"
  database = "postgres"
}

output "event_trigger_data" {
  value = data.postgresql_event_trigger.test
}
