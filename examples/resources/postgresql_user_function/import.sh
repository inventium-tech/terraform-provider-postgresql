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
