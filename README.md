# Terraform Provider for PostgreSQL

Terraform provider for managing PostgreSQL objects with the Terraform Plugin Framework.

## What it does

- Manages PostgreSQL resources such as roles, databases, schemas, extensions, grants, user functions, and event triggers.
- Exposes data sources to read database, schema(s), role, extension, and event trigger metadata.
- Supports local development/testing workflows for provider contributors.

## Quick start (local development)

1. Build the provider:

   ```bash
   go build -o terraform-provider-postgresql
   ```

2. Point Terraform to the local binary via `~/.terraformrc`:

   ```hcl
   provider_installation {
     dev_overrides {
       "inventium-tech/postgresql" = "/path/to/terraform-provider-postgresql"
     }
     direct {}
   }
   ```

3. Configure the provider in Terraform:

   ```hcl
   terraform {
     required_providers {
       postgresql = {
         source = "inventium-tech/postgresql"
       }
     }
   }

   provider "postgresql" {
     host     = "localhost"
     port     = 5432
     username = "postgres"
     password = var.postgres_password
     database = "postgres"
     sslmode  = "require"
   }
   ```

4. Initialize Terraform in your test configuration directory:

   ```bash
   terraform init
   ```

Expected result: `terraform init` completes successfully and uses the local `inventium-tech/postgresql` provider override.

## Status and support

This provider is under active development. The current codebase registers these resource types:
`postgresql_database`, `postgresql_event_trigger`, `postgresql_extension`, `postgresql_grant`,
`postgresql_role`, `postgresql_schema`, `postgresql_user_function`; and these data sources:
`postgresql_database`, `postgresql_event_trigger`, `postgresql_extension`, `postgresql_role`,
`postgresql_schema`, `postgresql_schemas`.

## Project documentation

- [README.md](README.md)
- [ARCHITECTURE.md](ARCHITECTURE.md)
- [CONTRIBUTING.md](CONTRIBUTING.md)
- [AGENTS.md](AGENTS.md)
- [LICENSE](LICENSE)
- [Generated provider docs index](docs/index.md)
- [Examples index](examples/README.md)
