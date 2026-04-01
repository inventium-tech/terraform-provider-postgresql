# Architecture

## System Overview

`terraform-provider-postgresql` is a Terraform provider implemented in Go with the
[Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework).
It maps Terraform resource/data source operations to PostgreSQL DDL and catalog queries.

At runtime, Terraform Core communicates with the provider plugin process, which delegates
database operations to repository-style components in `internal/pgclient`.

## Architectural Style

The project uses:

- **Terraform Plugin Framework** for provider/resource/data source lifecycle handling.
- **Domain-Driven Design (DDD)** for PostgreSQL bounded contexts (role, schema, database,
  extension, event trigger, grants, user functions).
- **Clean Architecture direction of dependencies**: `internal/provider` depends on
  `internal/pgclient`, while database/domain logic does not depend on Terraform framework types.

## Component Breakdown and Responsibility Boundaries

### Provider interface layer (`internal/provider`)

- Declares provider schema and environment fallback behavior.
- Implements Terraform resources and data sources.
- Performs Terraform-level validation and diagnostics.
- Translates Terraform plans/state into calls to `pgclient` repositories.

### Domain and data access layer (`internal/pgclient`)

- Owns PostgreSQL connection configuration and pooled connection lifecycle.
- Encapsulates SQL statements and repository operations per PostgreSQL object type.
- Provides typed models and operation boundaries used by provider implementations.

### Shared utility and test support (`internal/helpers`, `internal/test`)

- `internal/helpers`: reusable helpers (validation, pointer helpers, etc.).
- `internal/test`: integration/acceptance test utilities, including Postgres testcontainers.

## Data and Control Flow

1. Terraform Core calls provider/resource/data source entry points.
2. `internal/provider` reads config/state/plan and validates inputs.
3. Provider code acquires a configured `pgclient.PostgresqlClient`.
4. Resource/data source handlers call repository methods in `internal/pgclient`.
5. `pgclient` executes SQL via `pgx`/`pgxpool` and returns typed results/errors.
6. Provider maps results back to Terraform state/diagnostics.

## External Dependencies

- **Terraform provider runtime**: `terraform-plugin-framework`,
  `terraform-plugin-go`, `terraform-plugin-log`.
- **PostgreSQL access**: `github.com/jackc/pgx/v5` (+ `pgxpool`).
- **Testing**: `terraform-plugin-testing`, `testcontainers-go`, `testify`.
- **Docs generation**: `terraform-plugin-docs` via `go generate`.

## Repository Structure

```text
terraform-provider-postgresql/
├── AGENTS.md
├── ARCHITECTURE.md
├── CONTRIBUTING.md
├── README.md
├── docs/                 # generated docs (do not edit directly)
├── examples/             # Terraform usage examples
├── internal/
│   ├── helpers/          # shared helpers
│   ├── pgclient/         # PostgreSQL client + repositories
│   ├── provider/         # provider/resources/data sources
│   └── test/             # test utilities
├── templates/            # doc generation templates
└── main.go               # provider entry point
```

## Key Design Constraints and Decisions

- `docs/` is generated from templates and provider schemas; edit `templates/`/code instead.
- Provider database operations must use `internal/pgclient` (not deprecated legacy paths).
- Acceptance tests rely on Docker/testcontainers and are expected for resource/data source changes.
- Connection handling is pooled and keyed by database target to support multi-database operations.

## Upstream, Downstream, and Integration Notes

- **Upstream**:
  - Terraform Core plugin protocol/runtime.
  - PostgreSQL server behavior, permissions, and SQL semantics.
- **Downstream**:
  - Terraform configurations consuming provider resources/data sources.
  - Generated documentation consumed by Terraform Registry users.
- **Integration points**:
  - Local dev overrides via `~/.terraformrc`.
  - CI/automation through Go test and lint workflows.
  - Acceptance test environment via Docker containers.

## Version Compatibility

- **Go toolchain**: module targets Go `1.24.0` (`go.mod`), with `toolchain go1.24.6`.
- **Terraform Core**: docs currently state Terraform `1.0+`.
- **PostgreSQL**: docs currently state PostgreSQL `12+` (recommended `14+`).
- **TODO**: Confirm and document the authoritative, tested compatibility matrix per release.
