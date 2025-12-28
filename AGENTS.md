# AI Agent Instructions for terraform-provider-postgresql

📚 **Full Documentation**: [README.md](/README.md) | [CONTRIBUTING.md](/CONTRIBUTING.md)

A modern Terraform Provider for PostgreSQL built with Terraform Plugin Framework. This file contains AI-specific
operational instructions.

## Overall Principles

1. **Review the code**: Check for correctness, efficiency, and readability.
2. **Check for best practices**: Ensure that the code follows the best practices for the language and framework used.
   Use any tool available.
3. **Avoid unnecessary changes**: Only suggest changes that improve the code or fix issues. Avoid making changes that do
   not add value.
4. **Be concise**: Provide clear and concise feedback. Avoid long explanations unless necessary.
5. **Use examples**: If you suggest a change, provide an example of how to implement it.
6. **Be straightforward**: Focus on practical solutions that can be implemented easily. Avoid overly complex solutions
   unless necessary. Do not argue for the sake of arguing, but rather provide constructive feedback. Avoid any
   confirmation bias or unnecessary praise.
7. **Be thorough**: When either you or I suggest something is not working, investigate the issue thoroughly. Choose
   facts over opinions or feelings, don't be apologetic, and focus on finding the root cause of the issue and therefore
   the best solution.

## Expert-Level Support Guidelines

- Favor elegant, maintainable solutions over verbose code. Assume understanding of language idioms and design patterns.
- Highlight potential performance implications and optimization opportunities in suggested code.
- Frame solutions within broader architectural contexts and suggest design alternatives when appropriate.
- Focus comments on 'why' not 'what' - assume code readability through well-named functions and variables.
- Proactively address edge cases, race conditions, and security considerations without being prompted.
- When debugging, provide targeted diagnostic approaches rather than shotgun solutions.
- Suggest comprehensive testing strategies rather than just example tests, including considerations for mocking, test
  organization, and coverage.

## Project Structure

```
internal/
├── client/           # Deprecated - DO NOT USE for new code
├── helpers/          # Generic utility functions
├── pgclient/         # Current PostgreSQL client (use this)
│   └── pgcustomtypes/
├── provider/         # Provider, resources, data sources
│   └── validators/   # Custom Terraform validators
└── test/             # Test utilities
docs/                 # Auto-generated - DO NOT edit manually
templates/            # Doc generation templates
tools/                # Development tools
```

## Critical Rules

1. **Never edit files in `docs/` directly** - They are auto-generated from templates
2. **Use `internal/pgclient` for database operations** - `internal/client` is deprecated
3. **Follow DDD and Clean Architecture** - See copilot-instructions.md for patterns
4. **Always run tests before committing** - Both unit and acceptance tests

## Workflow Requirements

### Code Generation

Documentation is auto-generated. After modifying resources or data sources:

```bash
go generate ./...
```

This regenerates files in `docs/` from `templates/`
using [terraform-plugin-docs](https://github.com/hashicorp/terraform-plugin-docs).

### Testing Requirements

**Unit tests** - Run these for quick feedback:

```bash
go test ./... -v
```

**Acceptance tests** - Run these for full provider validation:

```bash
make testacc
# Or directly:
TF_ACC=1 go test ./... -v -timeout 120m
```

**Critical**: Acceptance tests use testcontainers-go to spin up PostgreSQL. They must:

- Properly clean up containers
- Not be flaky
- Cover only critical paths (not exhaustive)

### Architecture Enforcement

**Domain-Driven Design (DDD)**:

- Define bounded contexts in `pgclient`
- Use ubiquitous language matching PostgreSQL terminology
- Rich domain models with behavior, not just data

**Clean Architecture**:

- `pgclient` = domain layer (entities, repositories)
- `provider` = interface layer (Terraform resources/data sources)
- Dependencies point inward

**Repository Pattern**:

- Database operations go in `pg_repo_*.go` files
- Each PostgreSQL object type gets its own repository file

## Non-Interactive Commands

### Testing (CI/Agent Mode)

```bash
# Unit tests (fast)
go test ./... -v -count=1

# Acceptance tests (slow, requires Docker)
TF_ACC=1 go test ./internal/provider -v -timeout 120m

# With race detection
go test ./... -race -v

# Coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Code Quality

```bash
# Format code
gofmt -w .
goimports -w .

# Lint (if golangci-lint is configured)
golangci-lint run

# Generate documentation
go generate ./...

# Verify go.mod is tidy
go mod tidy
```

### Build

```bash
# Build provider binary
go build -o terraform-provider-postgresql

# Build with specific version
go build -ldflags="-X 'main.version=1.0.0'" -o terraform-provider-postgresql
```

## Environment Variables

### For Testing

- `TF_ACC=1` - Enable acceptance tests
- `TF_LOG=DEBUG` - Enable Terraform debug logging
- `TF_ACC_TERRAFORM_VERSION` - Specific Terraform version for tests

### For Provider

The provider reads connection details from Terraform configuration, not environment variables. See examples in
`examples/provider/provider.tf`.

## Static Analysis Tools

### SonarQube

- Quality gates enforce coverage and maintainability thresholds
- Use SonarLint in IDE before committing
- Security hotspots must be reviewed

### CodeCov

- Minimum 80% coverage for critical paths
- Branch coverage in addition to line coverage
- Coverage reports automatically posted to PRs

## Testing Libraries

- **Testify**: Use for assertions (`assert`, `require`, `mock`)
- **terraform-plugin-testing**: Acceptance test framework
- **testcontainers-go**: Spin up PostgreSQL for integration tests

## Files That Should Never Be Manually Edited

- `docs/**/*.md` - Auto-generated from templates
- `go.sum` - Managed by Go modules
- `.mega-linter.yml` - MegaLinter configuration (modify only if necessary)

## Completion Checklist

Before considering work complete:

- [ ] All unit tests pass (`go test ./... -v`)
- [ ] Acceptance tests pass if resources/data sources changed (`make testacc`)
- [ ] Code is formatted (`gofmt`, `goimports`)
- [ ] Documentation regenerated if provider interface changed (`go generate ./...`)
- [ ] No linting errors
- [ ] Commit messages follow Conventional Commits format
- [ ] `go.mod` is tidy (`go mod tidy`)

## Architecture Notes

### PostgreSQL Client (`pgclient`)

The current implementation uses `pgx/v5` for database connectivity. Key files:

- `pg_connection.go` - Connection management
- `pg_repo_*.go` - Repository implementations for each object type
- `pg_global_helpers.go` - Shared database helpers
- `pg_global_queries.go` - Common SQL queries
- `pg_global_types.go` - Shared type definitions

### Provider Implementation (`provider`)

Each Terraform resource/data source follows this pattern:

1. `postgresql_*_types.go` - Terraform schema models
2. `postgresql_*_resource.go` or `postgresql_*_datasource.go` - Resource/data source implementation
3. `postgresql_*_test.go` - Acceptance tests

### Validators

Custom validators in `provider/validators/` enforce PostgreSQL naming rules and other constraints.

## Language-Specific Instructions

See [`.github/instructions/go.instructions.md`](.github/instructions/go.instructions.md) for detailed Go coding
standards.
