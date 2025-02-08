variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID)."
  type        = string
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret."
  type        = string
  sensitive   = true
}

variable "mysqldb_password" {
  description = "MySQL DB Password."
  type        = string
  sensitive   = true
}

variable "mongodb_password" {
  description = "Mongo DB Password."
  type        = string
  sensitive   = true
}

variable "mongodb_topic_prefix" {
  description = "Mongo DB Kafka topic prefix."
  type        = string
  default     = "tf"
}

variable "mongodb_database" {
  description = "MongoDB Database Name."
  type        = string
  default     = "sample_mflix"
}

variable "mongodb_collection" {
  description = "MongoDB Collection Name."
  type        = string
  default     = "movies"
}

variable "mongodb_connection_host" {
  description = "MongoDB Connection Host."
  type        = string
  sensitive   = true
}

variable "mongodb_connection_user" {
  description = "Confluent Cloud API Secret."
  type        = string
  sensitive   = true
}

variable "mysqldb_user" {
  description = "MySQL DB User."
  type        = string
  default     = "admin"
  sensitive   = true
}

variable "mysqldb_host" {
  description = "MySQLDB Host"
  type        = string
  sensitive   = true
}
              
variable "mysqldb_port" {
  description = "Confluent Cloud API Secret."
  type        = string
  default     = "3306"
}

variable "mysqldb_topic_name" {
  description = "MySQL DB Kafka topic name."
  type        = string
  default     = "orders"
}

variable "mysqldb_name" {
  description = "MySQL DB Name."
  type        = string
  default     = "test"
}
