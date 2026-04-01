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
