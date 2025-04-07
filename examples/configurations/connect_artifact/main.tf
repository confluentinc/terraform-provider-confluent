terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.24.0"
    }
  }
}

provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret
  environment_id   = var.environment_id        # optionally use CONFLUENT_ENVIRONMENT_ID env var
}

resource "confluent_connect_artifact" "main" {
  environment {
    id = var.environment_id
  }
  region         = "us-west-2"
  cloud          = "AWS"
  display_name   = "example-connect-artifact"
  content_format = "JAR"
  description    = "Example Connect Artifact for demonstration purposes"
  artifact_file  = var.artifact_file
}