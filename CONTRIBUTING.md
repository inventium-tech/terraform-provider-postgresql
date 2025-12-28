# Contributing to Terraform Provider for PostgreSQL

Thank you for your interest in contributing to this project! We welcome contributions that improve the provider, fix bugs, add features, or enhance documentation.

## Table of Contents

- [Directory Structure](#directory-structure)
- [Development Environment Setup](#development-environment-setup)
- [Coding Standards](#coding-standards)
- [Testing](#testing)
- [Contribution Process](#contribution-process)
- [Bug Reporting](#bug-reporting)
- [Commit Message Format](#commit-message-format)

## Directory Structure

The project follows a standard Go project layout:

```text
terraform-provider-postgresql/
├── .github/               # GitHub workflows and instructions
│   ├── instructions/      # Language-specific coding instructions
│   └── workflows/         # CI/CD workflows
├── assets/                # Project assets (logos, images)
├── docs/                  # Terraform provider documentation
│   ├── data-sources/      # Data source documentation
│   └── resources/         # Resource documentation
├── examples/              # Example Terraform configurations
├── internal/              # Internal packages (not importable)
│   ├── client/           # Deprecated PostgreSQL client (legacy)
│   ├── helpers/          # Generic helper functions
│   ├── pgclient/         # Current PostgreSQL client implementation
│   ├── provider/         # Provider implementation (resources, data sources)
│   │   └── validators/   # Custom validators
│   └── test/             # Test helpers and utilities
├── templates/             # Markdown templates for doc generation
└── tools/                 # Development tools and utilities
```

## Development Environment Setup

### Prerequisites

- **Go**: Version 1.24.0 or higher
- **Terraform**: Version 1.0 or higher (for testing)
- **Docker**: For running PostgreSQL test containers
- **Make**: For running build tasks

### Setup Steps

1. **Clone the repository:**

   ```bash
   git clone https://github.com/inventium-tech/terraform-provider-postgresql.git
   cd terraform-provider-postgresql
   ```

2. **Install dependencies:**

   ```bash
   go mod download
   ```

3. **Verify installation:**

   ```bash
   go build
   ```

4. **Run tests:**

   ```bash
   go test ./... -v
   ```

### Local Provider Testing

To test the provider locally with Terraform:

1. Build the provider:

   ```bash
   go build -o terraform-provider-postgresql
   ```

2. Create a `.terraformrc` file in your home directory with:

   ```hcl
   provider_installation {
     dev_overrides {
       "inventium-tech/postgresql" = "/path/to/terraform-provider-postgresql"
     }
     direct {}
   }
   ```

3. Run Terraform commands in your test directory.

## Coding Standards

This project follows idiomatic Go practices and Terraform Plugin Framework conventions.

### Go Code Guidelines

- Follow the instructions in [`.github/instructions/go.instructions.md`](.github/instructions/go.instructions.md)
- Use `gofmt` and `goimports` for code formatting
- Write clear, self-documenting code with meaningful variable names
- Add comments for complex logic, focusing on "why" not "what"
- Handle errors explicitly; never ignore errors without good reason
- Keep functions small and focused on a single responsibility

### Architecture Principles

- **Domain-Driven Design (DDD)**: Create rich domain models for PostgreSQL objects
- **Clean Architecture**: Separate concerns into distinct layers (entities, use cases, interfaces)
- **Repository Pattern**: Database operations are encapsulated in the `pgclient` package

### Code Review Checklist

Before submitting a pull request, ensure:

- [ ] Code follows Go and Terraform best practices
- [ ] All tests pass locally
- [ ] New features include tests
- [ ] Documentation is updated
- [ ] Commit messages follow the format below
- [ ] No linting errors

## Testing

### Unit Tests

Run unit tests with:

```bash
go test ./... -v
```

### Acceptance Tests

Acceptance tests verify the provider works with a real PostgreSQL instance. They use [terraform-plugin-testing](https://github.com/hashicorp/terraform-plugin-testing) and [testcontainers-go](https://github.com/testcontainers/testcontainers-go).

Run acceptance tests with:

```bash
make testacc
```

Or directly:

```bash
TF_ACC=1 go test ./... -v -timeout 120m
```

### Testing Guidelines

- Use [Testify](https://github.com/stretchr/testify) for assertions
- Use testcontainers for integration tests requiring PostgreSQL
- Ensure proper setup and teardown to avoid flaky tests
- Cover critical paths in acceptance tests
- Mock external dependencies where appropriate

### Test Coverage

The project uses CodeCov for tracking test coverage. Aim for:

- Minimum 80% coverage for critical code paths
- 100% coverage for domain logic in `pgclient`
- Reasonable coverage for provider resources

## Contribution Process

### Reporting Issues

1. Check existing issues to avoid duplicates
2. Use issue templates when available
3. Provide clear reproduction steps
4. Include relevant logs and error messages
5. Specify your environment (Go version, Terraform version, PostgreSQL version)

### Submitting Pull Requests

1. **Fork the repository** and create a feature branch:

   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following the coding standards

3. **Add tests** for new functionality

4. **Update documentation** if needed:
   - Update relevant files in `docs/`
   - Update examples in `examples/`
   - Regenerate docs with: `go generate ./...`

5. **Commit your changes** following the commit message format

6. **Push to your fork** and create a pull request

7. **Respond to feedback** from maintainers during code review

### Pull Request Guidelines

- Keep PRs focused on a single feature or fix
- Write clear PR descriptions explaining the change
- Link related issues in the PR description
- Ensure CI checks pass
- Be responsive to review comments

## Bug Reporting

When reporting bugs, please include:

- **Description**: Clear description of the issue
- **Steps to Reproduce**: Step-by-step instructions
- **Expected Behavior**: What you expected to happen
- **Actual Behavior**: What actually happened
- **Environment**:
  - Go version (`go version`)
  - Terraform version (`terraform version`)
  - PostgreSQL version
  - Operating system
- **Logs**: Relevant error messages or logs
- **Configuration**: Minimal Terraform configuration reproducing the issue

## Commit Message Format

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```text
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

### Examples

```text
feat(role): add support for REPLICATION attribute

Implement support for the REPLICATION role attribute, allowing
creation of roles with replication privileges.

Closes #123
```

```text
fix(event_trigger): correct filter validation logic

Fixed validation logic that incorrectly rejected valid filter
configurations with multiple event types.
```

## Questions?

If you have questions about contributing, feel free to open an issue for discussion.

---

Thank you for contributing to Terraform Provider for PostgreSQL! 🎉
