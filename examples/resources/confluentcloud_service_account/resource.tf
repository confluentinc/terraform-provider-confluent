resource "confluent_service_account" "example-sa" {
  display_name = "orders-app-sa"
  description  = "Service Account for orders app"
}
