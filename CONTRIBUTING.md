# Contributing to Terraform Provider for PostgreSQL

Thank you for your interest in contributing to this project. We welcome improvements to provider behavior, tests,
architecture, and documentation.

## Table of Contents

- [Directory Structure](#directory-structure)
- [Development Environment Setup](#development-environment-setup)
- [Coding Standards](#coding-standards)
- [Testing](#testing)
- [Branching Conventions](#branching-conventions)
- [PR / MR Process](#pr--mr-process)
- [Bug Reporting](#bug-reporting)
- [Commit Message Format](#commit-message-format)
- [Proposing Design Changes](#proposing-design-changes)
- [Code of Conduct](#code-of-conduct)

## Directory Structure

See ARCHITECTURE.md: "Repository Structure"

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
- Handle errors explicitly; never ignore errors without a good reason
- Keep functions small and focused on a single responsibility

### Architecture Principles

See ARCHITECTURE.md: "Architectural Style"

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

Acceptance tests verify the provider works with a real PostgreSQL instance. They
use [terraform-plugin-testing](https://github.com/hashicorp/terraform-plugin-testing)
and [testcontainers-go](https://github.com/testcontainers/testcontainers-go).

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

## Branching Conventions

Create a branch using one of the standard prefixes:

| Prefix     | Purpose                                                  |
|------------|----------------------------------------------------------|
| `feature/` | Introduce a new feature                                  |
| `fix/`     | Fix or patch an existing bug                             |
| `docs/`    | Documentation-only changes                               |
| `perf/`    | Performance improvements                                 |
| `ci/`      | CI/CD workflow or automation changes                     |
| `chore/`   | Refactors, maintenance, or other non-user-facing changes |

Examples: `feature/role-inheritance`, `fix/event-trigger-validation`, `docs/quickstart`.

## PR / MR Process

1. Pull the latest changes from your **target branch** and branch from it before starting work.
2. Create a branch that follows [Branching Conventions](#branching-conventions).
3. Make your changes and add/adjust tests as needed.
4. If docs need updates, edit `templates/` and code comments first, then regenerate docs with `go generate ./...` (do
   not edit generated `docs/` directly).
5. Ensure each commit and the PR/MR title comply with [Commit Message Format](#commit-message-format).
6. Open the PR/MR with clear context, linked issue(s), and test evidence.
7. Assign a reviewer/maintainer.
8. Ensure the pipeline passes before marking the PR/MR ready for review or merge.
9. Address reviewer feedback promptly.

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

We follow
the [ESLint Conventions](https://github.com/conventional-changelog/conventional-changelog/tree/master/packages/conventional-changelog-eslint)
commit style. Commit history is consumed by [Semantic Release](https://semantic-release.gitbook.io/semantic-release/) to
drive automated versioning and release notes.

Every commit must use this structure:

```text
Tag: short description

Longer description here if necessary.

---
[OPTIONAL]
Closes #123
```

| Tag      | Description                         |
|----------|-------------------------------------|
| Breaking | Backwards-incompatible change       |
| Feature  | New functionality                   |
| Fix      | Bug fix                             |
| Docs     | Documentation-only change           |
| Chore    | Maintenance or non-user-facing work |
| Perf     | Performance improvement             |
| CI       | CI/CD pipeline or automation update |

Examples:

```text
Feature: add REPLICATION role attribute support

Implements REPLICATION handling for role resources and updates acceptance coverage.

---
Closes #123
```

```text
Fix: correct event trigger filter validation

Rejects only invalid filter combinations and allows valid multi-event configurations.
```

## Proposing Design Changes

For changes that affect architecture, resource/data source behavior, or provider contracts:

1. Open an issue describing the problem, motivation, and proposed approach.
2. Reference relevant architecture context (see ARCHITECTURE.md) and any considered alternatives.
3. Align with maintainers before implementing broad-impact changes.
4. Open a PR referencing the issue once the approach is agreed.

## Code of Conduct

Contributors are expected to collaborate respectfully and professionally. This project follows the
[Contributor Covenant](https://www.contributor-covenant.org/version/2/1/code_of_conduct/) as its baseline code of
conduct. If you experience unacceptable behavior, open a private report with maintainers through the repository
maintainership channel.

## Questions?

If you have questions about contributing, feel free to open an issue for discussion.

---

Thank you for contributing to Terraform Provider for PostgreSQL! 🎉
