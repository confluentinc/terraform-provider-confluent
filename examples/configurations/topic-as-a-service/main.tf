module "topic_as_a_service" {
  source = "./topic_as_a_service_module"

  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret
  topic_name       = var.topic_name
}
