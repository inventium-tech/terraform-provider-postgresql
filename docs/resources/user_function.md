---
page_title: "postgresql_user_function Resource - postgresql"
subcategory: ""
description: |-
    Manages a Postgresql user-defined function in a specified database and schema. Postgresql documentation https://www.postgresql.org/docs/current/sql-createfunction.html
---

# Resource: postgresql_user_function

Manages a Postgresql user-defined function in a specified database and schema. [Postgresql documentation](https://www.postgresql.org/docs/current/sql-createfunction.html)

## Example Usage

```terraform
# Example: PostgreSQL User-Defined Functions
#
# This example demonstrates creating various types of user-defined functions
# in PostgreSQL, including simple functions, functions with multiple parameters,
# and functions that return composite types.

# Simple greeting function with two text parameters
resource "postgresql_user_function" "greet" {
  name     = "greet"
  database = "postgres"
  schema   = "public"
  args = [
    { name = "greeting", type = "text" },
    { name = "name", type = "text" }
  ]
  returns  = "text"
  language = "plpgsql"
  body     = <<-EOT
    BEGIN
      RETURN greeting || ' ' || name || '!';
    END;
  EOT

  comment = "A simple greeting function that concatenates greeting and name"
  owner   = "postgres"
}

# Function to calculate discount with default parameter
resource "postgresql_user_function" "calculate_discount" {
  name     = "calculate_discount"
  database = "postgres"
  schema   = "public"
  args = [
    { name = "original_price", type = "numeric" },
    { name = "discount_percent", type = "numeric", default = "10" }
  ]
  returns  = "numeric"
  language = "plpgsql"
  body     = <<-EOT
    BEGIN
      RETURN original_price * (1 - discount_percent / 100);
    END;
  EOT

  comment = "Calculates discounted price with optional discount percentage (default 10%)"
  owner   = "postgres"
}

# SQL function to get current timestamp (immutable)
resource "postgresql_user_function" "get_current_year" {
  name     = "get_current_year"
  database = "postgres"
  schema   = "public"
  args     = []
  returns  = "integer"
  language = "sql"
  body     = <<-EOT
    SELECT EXTRACT(YEAR FROM CURRENT_DATE)::integer;
  EOT

  volatility = "STABLE" # Function result depends on current date
  comment    = "Returns the current year"
  owner      = "postgres"
}

# Function with VARIADIC arguments
resource "postgresql_user_function" "sum_all" {
  name     = "sum_all"
  database = "postgres"
  schema   = "public"
  args = [
    { name = "VARIADIC numbers", type = "integer[]" }
  ]
  returns  = "integer"
  language = "plpgsql"
  body     = <<-EOT
    DECLARE
      total integer := 0;
      num integer;
    BEGIN
      FOREACH num IN ARRAY numbers
      LOOP
        total := total + num;
      END LOOP;
      RETURN total;
    END;
  EOT

  comment = "Sums all provided integers using VARIADIC arguments"
  owner   = "postgres"
}

# Function that returns a table (set-returning function)
resource "postgresql_user_function" "generate_series_with_labels" {
  name     = "generate_series_with_labels"
  database = "postgres"
  schema   = "public"
  args = [
    { name = "start_num", type = "integer" },
    { name = "end_num", type = "integer" }
  ]
  returns  = "TABLE(number integer, label text)"
  language = "plpgsql"
  body     = <<-EOT
    BEGIN
      RETURN QUERY
      SELECT i, 'Number ' || i::text
      FROM generate_series(start_num, end_num) i;
    END;
  EOT

  comment = "Generates a series of numbers with labels (set-returning function)"
  owner   = "postgres"
}

# Security-focused function with SECURITY DEFINER
resource "postgresql_user_function" "secure_user_lookup" {
  name     = "secure_user_lookup"
  database = "postgres"
  schema   = "public"
  args = [
    { name = "user_id", type = "integer" }
  ]
  returns       = "text"
  language      = "sql"
  security      = "DEFINER" # Runs with owner's privileges
  strict        = true      # Returns NULL if any argument is NULL
  parallel_safe = true      # Safe for parallel execution
  body          = <<-EOT
    SELECT username FROM users WHERE id = user_id;
  EOT

  comment = "Securely looks up username by ID using SECURITY DEFINER"
  owner   = "postgres"
}

# Output function names for reference
output "function_signatures" {
  description = "Full function signatures for use in SQL"
  value = {
    greet                       = "${postgresql_user_function.greet.schema}.${postgresql_user_function.greet.name}(text, text)"
    calculate_discount          = "${postgresql_user_function.calculate_discount.schema}.${postgresql_user_function.calculate_discount.name}(numeric, numeric)"
    get_current_year            = "${postgresql_user_function.get_current_year.schema}.${postgresql_user_function.get_current_year.name}()"
    sum_all                     = "${postgresql_user_function.sum_all.schema}.${postgresql_user_function.sum_all.name}(VARIADIC integer[])"
    generate_series_with_labels = "${postgresql_user_function.generate_series_with_labels.schema}.${postgresql_user_function.generate_series_with_labels.name}(integer, integer)"
    secure_user_lookup          = "${postgresql_user_function.secure_user_lookup.schema}.${postgresql_user_function.secure_user_lookup.name}(integer)"
  }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `body` (String) The body of the Postgresql function.
- `name` (String) The name of the Postgresql function.

### Optional

- `allow_replace` (Boolean) If true, the generated SQL statements will contain 'OR REPLACE'. The default is true.
- `args` (Attributes List) The arguments of the Postgresql function. (see [below for nested schema](#nestedatt--args))
- `comment` (String) Comment associated with the Postgresql Function.
- `database` (String) Name of the Postgresql database where the Function is located. If not specified, the provider's configured database will be used.
- `language` (String) The language of the Postgresql function, the value must be either `plpgsql` or `sql`. If not specified, the default is 'plpgsql'.
- `owner` (String) The owner of the Postgresql function. If not specified, the owner will be the User used in the provider's configuration.
- `returns` (String) The return type of the Postgresql function. If not specified, the default is 'void'.
- `schema` (String) The name of the Postgresql schema to create the function in. The default is 'public'.

### Read-Only

- `id` (String) The unique identifier for the user function, in the format `<database>.<schema>.<function_name>(<arguments>)`
- `last_updated` (String) Timestamp of the resource's last modification

<a id="nestedatt--args"></a>
### Nested Schema for `args`

Required:

- `name` (String) The name of the argument.
- `type` (String) The data type of the argument, preferably in lowercase (e.g., 'integer', 'text').

Optional:

- `default` (String) The default value of the argument.
- `mode` (String) The mode of the argument (IN, OUT, INOUT, VARIADIC). If not specified, Postgresql assumes 'IN' by default.




## Import

```terraform
# PostgreSQL User Functions can be imported by specifying the ID with the format:
# <database>.<schema>.<function_name>(<arguments>)
#
# Note: The argument list must match exactly as defined in PostgreSQL,
# including data types and order.

# Import a simple function with two text arguments
terraform import postgresql_user_function.greet "postgres.public.greet(greeting text, name text)"

# Import a function with numeric arguments and defaults
terraform import postgresql_user_function.calculate_discount "postgres.public.calculate_discount(original_price numeric, discount_percent numeric)"

# Import a function with no arguments
terraform import postgresql_user_function.get_current_year "postgres.public.get_current_year()"

# Import a function with VARIADIC arguments
terraform import postgresql_user_function.sum_all "postgres.public.sum_all(VARIADIC numbers integer[])"

# Import a set-returning function (returns TABLE)
terraform import postgresql_user_function.generate_series_with_labels "postgres.public.generate_series_with_labels(start_num integer, end_num integer)"

# Import a function with security definer
terraform import postgresql_user_function.secure_user_lookup "postgres.public.secure_user_lookup(user_id integer)"
```
