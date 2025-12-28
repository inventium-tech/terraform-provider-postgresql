# Example: PostgreSQL Roles
#
# This example demonstrates creating various types of PostgreSQL roles
# with different permissions and security best practices.

# Generate a secure random password using ephemeral resource
ephemeral "random_password" "app_user_password" {
  length  = 64
  lower   = true
  upper   = true
  numeric = true
  special = false # Avoid special characters for compatibility
}

# Application user role with login capability
resource "postgresql_role" "app_user" {
  name        = "app_user"
  login       = true
  superuser   = false
  inherit     = true
  createdb    = false
  createrole  = false
  replication = false

  # Use ephemeral password - never hardcoded
  password_wo         = random_password.app_user_password.result
  password_wo_version = 1

  comment = "Application database user"
}

# Read-only role for reporting
resource "postgresql_role" "readonly_user" {
  name        = "readonly_user"
  login       = true
  superuser   = false
  inherit     = true
  createdb    = false
  createrole  = false
  replication = false

  password_wo         = random_password.readonly_password.result
  password_wo_version = 1

  comment = "Read-only user for reporting and analytics"
}

ephemeral "random_password" "readonly_password" {
  length  = 64
  lower   = true
  upper   = true
  numeric = true
  special = false
}

# Group role (no login) for permission management
resource "postgresql_role" "developers" {
  name        = "developers"
  login       = false # Group role
  superuser   = false
  inherit     = true
  createdb    = true # Developers can create databases
  createrole  = false
  replication = false

  comment = "Developer group role"
}

# Replication user
resource "postgresql_role" "replication_user" {
  name        = "replication_user"
  login       = true
  superuser   = false
  inherit     = true
  createdb    = false
  createrole  = false
  replication = true # Can perform replication

  password_wo         = random_password.replication_password.result
  password_wo_version = 1

  comment = "Replication user for standby servers"
}

ephemeral "random_password" "replication_password" {
  length  = 64
  lower   = true
  upper   = true
  numeric = true
  special = false
}

# Output role names (not passwords!)
output "role_names" {
  value = {
    app_user         = postgresql_role.app_user.name
    readonly_user    = postgresql_role.readonly_user.name
    developers       = postgresql_role.developers.name
    replication_user = postgresql_role.replication_user.name
  }
  description = "Created role names"
}
