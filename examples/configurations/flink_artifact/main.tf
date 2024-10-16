terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.6.0"
    }
  }
}

provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret
}


resource "confluent_environment" "staging" {
  display_name = "Staging"

  stream_governance {
    package = "ESSENTIALS"
  }
}

resource "confluent_flink_artifact" "main" {
  environment {
    id = confluent_environment.staging.id
  }
  class          = "io.confluent.example.SumScalarFunction"
  region         = "us-west-2"
  cloud          = "AWS"
  display_name   = "flink_sumscalar_artifact"
  content_format = "JAR"
  artifact_file  = var.artifact_file
}
