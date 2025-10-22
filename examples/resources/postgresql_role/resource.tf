ephemeral "random_password" "db_password" {
  length  = 64
  lower   = true
  upper   = true
  numeric = true
  special = false
}

resource "postgresql_role" "john_doe" {
  name        = "john_doe"
  login       = true
  superuser   = false
  inherit     = true
  createdb    = false
  createrole  = false
  replication = false

  password_wo         = random_password.db_password.result
  password_wo_version = 1

  comment = "A role for John Doe"
}
