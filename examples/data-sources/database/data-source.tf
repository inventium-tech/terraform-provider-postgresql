terraform {
  required_providers {
    postgresql = {
      source = "registry.terraform.io/inventium-tech/postgresql"
    }
  }
}

provider "postgresql" {}

data "postgresql_database" "test" {
  name = "tf_provider"
}

output "database_data" {
  value = data.postgresql_database.test
}
