terraform {
  required_providers {
    postgresql = {
      source = "registry.terraform.io/inventium-tech/postgresql"
    }
  }
}

provider "postgresql" {}

data "postgresql_database" "example" {}
