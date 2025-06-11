resource "postgresql_user_function" "greet_example" {
  name = "greet"
  args = [
    { name = "greeting", type = "text" },
    { name = "name", type = "text" }
  ]
  returns  = "text"
  language = "plpgsql"
  body     = <<-EOT
    BEGIN
      RETURN greeting || ' ' || name;
    END;
  EOT

  comment = "A simple greeting function"
  owner   = "jhon_doe"
}
