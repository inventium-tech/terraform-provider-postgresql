data "postgresql_schemas" "all" {
  include_system_schemas = false
  like_any_patterns      = ["public", "app_%"]
}
