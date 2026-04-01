data "postgresql_role" "app_user" {
  name  = "app_user"
  login = true
}
