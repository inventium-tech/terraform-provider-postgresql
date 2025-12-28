# Terraform Provider PostgreSQL Examples

This directory contains example Terraform configurations demonstrating how to use the PostgreSQL provider resources and data sources.

## Directory Structure

```
examples/
├── provider/          # Provider configuration examples
├── resources/         # Resource usage examples
└── data-sources/      # Data source usage examples
```

## Quick Start

### 1. Configure the Provider

See [provider/provider.tf](provider/provider.tf) for provider configuration examples.

```terraform
terraform {
  required_providers {
    postgresql = {
      source  = "inventium-tech/postgresql"
      version = "~> 1.0"
    }
  }
}

provider "postgresql" {
  host     = "localhost"
  port     = 5432
  username = "postgres"
  password = var.postgres_password  # Use variables for sensitive data
  database = "postgres"
  sslmode  = "require"
}
```

### 2. Use Resources

Browse the [resources/](resources/) directory for complete examples of each resource type:

- **[postgresql_event_trigger](resources/postgresql_event_trigger/)** - Manage PostgreSQL event triggers
- **[postgresql_role](resources/postgresql_role/)** - Manage PostgreSQL roles and users
- **[postgresql_user_function](resources/postgresql_user_function/)** - Manage user-defined functions

### 3. Query Data Sources

Browse the [data-sources/](data-sources/) directory for data source examples:

- **[postgresql_event_trigger](data-sources/postgresql_event_trigger/)** - Query existing event triggers

## Security Best Practices

### Never Hardcode Sensitive Data

❌ **Bad:**

```terraform
provider "postgresql" {
  password = "my-secret-password"
}
```

✅ **Good:**

```terraform
variable "postgres_password" {
  description = "PostgreSQL password"
  type        = string
  sensitive   = true
}

provider "postgresql" {
  password = var.postgres_password
}
```

### Use Ephemeral Resources for Passwords

When creating roles, use Terraform's ephemeral resources to generate secure passwords:

```terraform
ephemeral "random_password" "db_password" {
  length  = 64
  lower   = true
  upper   = true
  numeric = true
  special = false
}

resource "postgresql_role" "user" {
  name            = "app_user"
  password_wo     = random_password.db_password.result
  password_wo_version = 1
}
```

### Configure SSL

Always use SSL in production environments:

```terraform
provider "postgresql" {
  sslmode = "verify-full"  # Most secure option
}
```

## Running Examples

Each example directory contains:

- `resource.tf` or `data-source.tf` - The example configuration
- `import.sh` (for resources) - How to import existing resources

To test an example:

```bash
cd examples/resources/postgresql_role
terraform init
terraform plan
terraform apply
```

## Importing Existing Resources

Each resource example includes an `import.sh` script showing the import format.

Example for importing a role:

```bash
terraform import postgresql_role.john_doe "john_doe"
```

See individual resource directories for specific import syntax.

## Testing

These examples are also used in the provider's acceptance tests to ensure they work correctly.

## Contributing

When adding new resources or data sources:

1. Create a new directory under `resources/` or `data-sources/`
2. Add a complete example configuration
3. Include an `import.sh` script for resources
4. Test the example with `terraform plan` and `terraform apply`
5. Add comments explaining parameters and best practices

## Additional Resources

- [Provider Documentation](https://registry.terraform.io/providers/inventium-tech/postgresql/latest/docs)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Terraform Documentation](https://www.terraform.io/docs/)
