# AI Agent Instructions for terraform-provider-postgresql

📚 **Full Documentation**: [README.md](/README.md) | [ARCHITECTURE.md](/ARCHITECTURE.md) | [CONTRIBUTING.md](/CONTRIBUTING.md) | [Go Instructions](.github/instructions/go.instructions.md)

A modern Terraform Provider for PostgreSQL built with Terraform Plugin Framework.

## Principles

- **Be thorough**: Choose facts over opinions. When debugging, investigate root causes systematically.
- **Be pragmatic**: Favor elegant, maintainable solutions. Avoid unnecessary changes or verbose code.
- **Be expert-level**: Assume understanding of Go idioms and design patterns. Focus on 'why' not 'what'.
- **Be proactive**: Address edge cases, race conditions, and security implications without prompting.

## Project Structure

```
internal/
├── helpers/          # Generic utility functions
├── pgclient/         # PostgreSQL client (use this)
├── provider/         # Resources, data sources, validators
└── test/             # Test utilities
docs/                 # ⚠️ AUTO-GENERATED - DO NOT EDIT
templates/            # Doc generation templates
```

## Critical Rules

1. **NEVER edit `docs/` directly** - Auto-generated from `templates/`
2. **Use `internal/pgclient`** for database operations (do not use deprecated legacy client paths)
3. **Follow DDD/Clean Architecture** - See ARCHITECTURE.md
4. **Run tests before committing** - Both unit and acceptance

## Build, Lint, and Test Commands

```bash
# Build
go build -o terraform-provider-postgresql

# Format (always run before commit)
gofmt -w . && goimports -w .

# Lint
golangci-lint run

# All tests
go test ./... -v

# Single test (unit)
go test -v -run '^TestDatabaseRepo_Integration$' ./internal/pgclient

# Single test (acceptance)
TF_ACC=1 go test -v -run '^TestAccPostgresqlDatabaseResource$' ./internal/provider -timeout 120m

# With race detection
go test ./... -race -v

# Coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Skip integration tests (short mode)
go test ./... -short

# Generate docs
go generate ./...

# Tidy dependencies
go mod tidy

# MegaLinter (comprehensive)
make lint
```

**Acceptance Tests**: Require `TF_ACC=1` and Docker (uses testcontainers-go). Use `TF_LOG=DEBUG` for debugging.

## Code Style Guidelines

### Imports

Order: stdlib → third-party → internal

```go
import (
    "context"
    "fmt"
    
    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/jackc/pgx/v5"
    
    "terraform-provider-postgresql/internal/pgclient"
    "terraform-provider-postgresql/internal/provider/validators"
)
```

### Naming Conventions

- **Packages**: lowercase, single-word, no underscores (`pgclient`, not `pg_client`)
- **Functions/vars**: mixedCaps/MixedCaps (`connectionLimit`, not `connection_limit`)
- **Interfaces**: `-er` suffix (`DatabaseRepo`, `Reader`)
- **Test functions**: `TestAccPostgresql*` (acceptance) or `Test*_Integration` (unit integration)
- **Constants**: MixedCaps for exported, mixedCaps for unexported

### Error Handling

```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("invalid database name %q: %w", name, err)
}

// Check immediately after call
data, err := repo.GetOne(ctx, conn, name)
if err != nil {
    return fmt.Errorf("failed to retrieve database: %w", err)
}

// Keep error messages lowercase, no punctuation
```

### Types and Pointers

- **Pointers**: For optional fields in update params (`*string`, `*int32`, `*bool`)
- **Values**: For required fields and small structs
- **pgx types**: Use `pgtype.Text`, `pgtype.Int4`, `pgtype.Bool` for database models

## Architecture Patterns

See ARCHITECTURE.md: "Architectural Style"

## Testing Standards

- **Unit tests**: Same directory as code (`pg_repo_database_test.go`), use `testing.Short()` to skip integration
- **Acceptance tests**: `TestAcc*` prefix, requires Docker
- **Naming**: `TestAccPostgresqlDatabaseResource`, `TestAccPostgresqlDatabaseResource_ForceDrop`, `TestDatabaseRepo_Integration`
- **Assertions**: Use testify (`assert.NoError(t, err)`, `assert.Equal(t, expected, actual)`)

## Commit Message Format

See CONTRIBUTING.md: "Commit Message Format"

## Completion Checklist

See CONTRIBUTING.md: "Code Review Checklist"
