<p align="center">
  <img src="./assets/provider_logo.svg" width="200" alt="logo"/>
</p>

---

![Golang](https://img.shields.io/badge/-Golang-black?style=for-the-badge&logoColor=white&logo=go&color=00ADD8)
![Postgres](https://img.shields.io/badge/-PostgreSQL-black?style=for-the-badge&logoColor=white&logo=postgresql&color=4169E1)
![Terraform](https://img.shields.io/badge/-Terraform-black?style=for-the-badge&logoColor=white&logo=terraform&color=844FBA)

[![🛠️ Build Workflow](https://github.com/inventium-tech/terraform-provider-postgresql/actions/workflows/build.yml/badge.svg)](https://github.com/inventium-tech/terraform-provider-postgresql/actions/workflows/build.yml)
[![🔎 MegaLinter](https://github.com/inventium-tech/terraform-provider-postgresql/actions/workflows/mega-linter.yml/badge.svg)](https://github.com/inventium-tech/terraform-provider-postgresql/actions/workflows/mega-linter.yml)
[![❇️ CodeQL](https://github.com/inventium-tech/terraform-provider-postgresql/actions/workflows/codeql.yml/badge.svg)](https://github.com/inventium-tech/terraform-provider-postgresql/actions/workflows/codeql.yml)

![GitHub language count](https://img.shields.io/github/languages/count/inventium-tech/terraform-provider-postgresql)
![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/inventium-tech/terraform-provider-postgresql/go.yml?branch=main&logo=githubactions&logoColor=white&logoSize=5)
![GitHub License](https://img.shields.io/github/license/inventium-tech/terraform-provider-postgresql)

<h2>📋 Table of Contents</h2>

<!-- TOC -->
* [🐘 Terraform Provider for PostgreSQL](#-terraform-provider-for-postgresql)
  * [❗ READ BEFORE USE](#-read-before-use)
  * [✨ Features](#-features)
  * [📦 Installation](#-installation)
    * [Terraform Registry (Coming Soon)](#terraform-registry-coming-soon)
    * [Local Development](#local-development)
  * [🚀 Quick Start](#-quick-start)
  * [📚 Usage Examples](#-usage-examples)
    * [Managing Roles](#managing-roles)
    * [Managing User Functions](#managing-user-functions)
    * [Managing Event Triggers](#managing-event-triggers)
  * [🏁 Roadmap](#-roadmap)
    * [Core Database Objects (Epic 001)](#core-database-objects-epic-001)
    * [Access Control (Epic 002)](#access-control-epic-002)
    * [Extensions and Advanced Features (Epic 003)](#extensions-and-advanced-features-epic-003)
    * [Replication and High Availability (Epic 004)](#replication-and-high-availability-epic-004)
    * [Data Discovery (Epic 005)](#data-discovery-epic-005)
    * [Already Implemented](#already-implemented)
    * [Documentation and Quality (Epic 006)](#documentation-and-quality-epic-006)
  * [🤝 Contributing](#-contributing)
  * [📄 License](#-license)
<!-- TOC -->

# 🐘 Terraform Provider for PostgreSQL

A modern and efficient Terraform provider for PostgreSQL, built using the latest best practices with
the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework). This provider enables you
to manage PostgreSQL resources (roles, functions, event triggers, databases, and schemas) as Infrastructure as Code.

## ❗ READ BEFORE USE

* This provider is still in development and has limited support for PostgreSQL resources.
* Check the [🏁 Roadmap](#-roadmap) for the list of supported resources.
* Not recommended for production use until v1.0.0 release.

## ✨ Features

* **Modern Architecture**: Built with Terraform Plugin Framework (not the legacy SDK)
* **Type Safety**: Strong typing and validation for PostgreSQL resources
* **Clean Code**: Follows Domain-Driven Design and Clean Architecture principles
* **Well Tested**: Comprehensive unit and acceptance tests using testcontainers
* **Active Development**: Regular updates and new features

## 📦 Installation

### Terraform Registry (Coming Soon)

Once published to the Terraform Registry, add to your Terraform configuration:

```hcl
terraform {
    required_providers {
        postgresql = {
            source  = "inventium-tech/postgresql"
            version = "~> 0.1"
        }
    }
}
```

### Local Development

For local development or testing:

1. Clone the repository:

   ```bash
   git clone https://github.com/inventium-tech/terraform-provider-postgresql.git
   cd terraform-provider-postgresql
   ```

2. Build the provider:

   ```bash
   go build -o terraform-provider-postgresql
   ```

3. Configure Terraform to use the local build by creating `~/.terraformrc`:

   ```hcl
   provider_installation {
     dev_overrides {
       "inventium-tech/postgresql" = "/path/to/terraform-provider-postgresql"
     }
     direct {}
   }
   ```

## 🚀 Quick Start

Configure the provider with your PostgreSQL connection:

```hcl
provider "postgresql" {
    host     = "localhost"
    port     = 5432
    database = "postgres"
    username = "postgres"
    password = "your-password"
    sslmode  = "disable"  # Use "require" for production
}
```

Create a PostgreSQL role:

```hcl
resource "postgresql_role" "app_user" {
    name     = "app_user"
    login    = true
    password = "secure-password"
}
```

## 📚 Usage Examples

### Managing Roles

```hcl
# Create a role with specific privileges
resource "postgresql_role" "readonly" {
    name             = "readonly_user"
    login            = true
    password         = "password123"
    connection_limit = 10
    valid_until = "2025-12-31 23:59:59"

    # Role attributes
    superuser       = false
    create_database = false
    create_role     = false
    inherit         = true
    replication     = false
    bypass_rls      = false
}
```

### Managing User Functions

```hcl
# Create a custom PostgreSQL function
resource "postgresql_user_function" "calculate_total" {
    name     = "calculate_total"
    schema   = "public"
    language = "plpgsql"

    arguments = [
        {
            name = "qty"
            type = "integer"
        },
        {
            name = "price"
            type = "numeric"
        }
    ]

    return_type = "numeric"

    body = <<-SQL
    BEGIN
      RETURN qty * price;
    END;
  SQL

    volatility = "IMMUTABLE"
}
```

### Managing Event Triggers

```hcl
# Create an event trigger
resource "postgresql_event_trigger" "ddl_audit" {
    name  = "audit_ddl_commands"
    event = "ddl_command_end"

    function_name = "audit_ddl_function"

    tags = ["DROP TABLE", "ALTER TABLE"]

    enabled = true
}

# Query event triggers
data "postgresql_event_trigger" "existing" {
    name = "audit_ddl_commands"
}
```

For more examples, see the [`examples/`](examples/) directory.

## 🏁 Roadmap

Here you can find a status of the resources that are supported by the provider. Implementation is organized by category
in separate Epics under `.specs/` directory.

_status:_

* ✅ Supported
* 🔜 Coming Soon
* 📋 Planned

### Core Database Objects ([Epic 001](.specs/epic-001.spec.md))

| Name           | Resource | Data Source | Write-Only Attr | Epic Task |
|----------------|:--------:|:-----------:|:---------------:|-----------|
| Database       |    🔜    |     🔜      |                 | 001-1     |
| Schema         |    🔜    |     🔜      |                 | 001-2     |
| Schemas (List) |    -     |     📋      |                 | 001-3     |

### Access Control ([Epic 002](.specs/epic-002.spec.md))

| Name                | Resource | Data Source | Write-Only Attr | Epic Task |
|---------------------|:--------:|:-----------:|:---------------:|-----------|
| Role                |    ✅     |     🔜      |        ✅        | 002-1,2   |
| Grant (Privileges)  |    📋    |      -      |                 | 002-3     |
| Default Privileges  |    📋    |      -      |                 | 002-4     |
| Grant Role (Member) |    📋    |      -      |                 | 002-5     |

### Extensions and Advanced Features ([Epic 003](.specs/epic-003.spec.md))

| Name               | Resource | Data Source | Epic Task |
|--------------------|:--------:|:-----------:|-----------|
| Extension          |    📋    |     📋      | 003-1     |
| Server (FDW)       |    📋    |      -      | 003-2     |
| User Mapping (FDW) |    📋    |      -      | 003-3     |
| Security Label     |    📋    |      -      | 003-4     |

### Replication and High Availability ([Epic 004](.specs/epic-004.spec.md))

| Name                       | Resource | Data Source | Epic Task |
|----------------------------|:--------:|:-----------:|-----------|
| Replication Slot (Logical) |    📋    |      -      | 004-1     |
| Physical Replication Slot  |    📋    |      -      | 004-2     |
| Publication                |    📋    |      -      | 004-3     |
| Subscription               |    📋    |      -      | 004-4     |

### Data Discovery ([Epic 005](.specs/epic-005.spec.md))

| Name                | Resource | Data Source | Epic Task |
|---------------------|:--------:|:-----------:|-----------|
| Function            |    -     |     🔜      | 005-1     |
| Tables (List)       |    -     |     📋      | 005-2     |
| Sequences (List)    |    -     |     📋      | 005-3     |
| Columns (Metadata)  |    -     |     📋      | 005-4     |
| Indexes (Metadata)  |    -     |     📋      | 005-5     |
| Constraints (Meta)  |    -     |     📋      | 005-6     |
| Triggers (Metadata) |    -     |     📋      | 005-7     |

### Already Implemented

| Name          | Resource | Data Source | Notes                     |
|---------------|:--------:|:-----------:|---------------------------|
| Event Trigger |    ✅     |      ✅      | Already complete          |
| User Function |    ✅     |      -      | Data source in Epic 005-1 |

### Documentation and Quality ([Epic 006](.specs/epic-006.spec.md))

See [Epic 006](.specs/epic-006.spec.md) for documentation, examples, performance testing, and migration guide tasks.

## 🤝 Contributing

Contributions are welcome! Please read our [Contributing Guidelines](CONTRIBUTING.md) for details on:

* Development environment setup
* Coding standards and best practices
* Testing requirements
* Pull request process
* Commit message format

## 📄 License

This project is licensed under the [Mozilla Public License 2.0](LICENSE).

---

<a href="https://www.buymeacoffee.com/refucktor" target="_blank">
  <img src="https://cdn.buymeacoffee.com/buttons/v2/default-red.png" alt="Buy Me A Coffee"
    style="height: 60px !important;width: 217px !important;">
</a>
