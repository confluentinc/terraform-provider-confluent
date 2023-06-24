variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID)"
  type        = string
  default = "YW4YATHNHKWLFAV4"
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret"
  type        = string
  sensitive   = true
  default = "No0cNdfhjDyX2lBE2apmyuR04NGHqmCcW2C5HemlGgu2/Ls+VpqBACmX5n+ai5W2"
}
