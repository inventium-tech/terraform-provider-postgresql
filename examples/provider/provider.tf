terraform {
  required_providers {
    postgresql = {
      source = "registry.terraform.io/inventium-tech/postgresql"
    }
  }
}

provider "postgresql" {
  host     = "localhost"
  port     = 5432
  username = "postgres"
  password = "password"
}
