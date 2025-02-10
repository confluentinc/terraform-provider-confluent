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

variable "mysqldb_user" {
  description = "MySQL DB User."
  type        = string
  default     = "admin"
}

variable "mysqldb_host" {
  description = "MySQLDB Host"
  type        = string
  default     = "dev-testing-temp.abcdefghijk.us-west-7.rds.amazonaws.com"
}

variable "mysqldb_port" {
  description = "MySQL DB Port."
  type        = string
  default     = "3306"
}

variable "mysqldb_name" {
  description = "MySQL DB Name."
  type        = string
  default     = "test_database"
}
