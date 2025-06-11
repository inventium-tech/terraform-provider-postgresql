# Postgresql User Function can be imported by specifying the id with the format <database>.<schema>.<function_name>(<arguments>)
terraform import postgresql_user_function.greet_example "demos.public.greet_example(greeting text, name text)"
