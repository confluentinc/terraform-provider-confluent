terraform {
  required_providers {
    confluent = {
     # source  = "confluentinc/confluent"
     # version = "2.2.0"
      source = "registry.terraform.io/confluentinc/confluent"

    }
  }
}

provider "confluent" {
#   cloud_api_key    = var.confluent_cloud_api_key
#   cloud_api_secret = var.confluent_cloud_api_secret
  cloud_api_key    = "25QVCWFOVGMBXWWO"
  cloud_api_secret = "qz1fhHmQ4qcha+8cPCfzrtuE+dzGoz9qIOXX3Rw5h+b/jELIHKLiFYAQZTTC12Co"
  endpoint = "https://api.devel.cpdev.cloud"

    organization_id       = "eddb9896-1b19-4085-86c5-65af828ccdbf"            # optionally use CONFLUENT_ORGANIZATION_ID env var
    environment_id        = "env-devc1yv79z"            # optionally use CONFLUENT_ENVIRONMENT_ID env var
    flink_compute_pool_id = "lfcp-devcg03vjr"     # optionally use FLINK_COMPUTE_POOL_ID env var
    flink_rest_endpoint   = "https://flink.us-west-2.aws.devel.cpdev.cloud"    # optionally use FLINK_REST_ENDPOINT env var
    flink_api_key         = "FV4U2GWLV6OTKRTG"              # optionally use FLINK_API_KEY env var
    flink_api_secret      = "fERix6CtHPzexABP4GggyNYialoUJXxnOzV0ypthHDQV+g9cpIhJHHca8huCVfrA"          # optionally use FLINK_API_SECRET env var
    flink_principal_id    = "u-devc730n3j"        # optionally use FLINK_PRINCIPAL_ID env var
}

# resource "confluent_environment" "staging" {
#   display_name = "Staging2"
#
#   stream_governance {
#     package = "ESSENTIALS"
#   }
# }

resource "confluent_flink_artifact" "main" {
  environment {
    id = "env-devc1yv79z"
  }
  class = "class1"
  region = "us-west-2"
  cloud = "AWS"
  display_name = "flink_artifact_main_26"
  content_format = "JAR"
  artifact_file = "/Users/tusharmalik/git/go/src/github.com/confluentinc/airlock-terraform-provider-confluent/examples/configurations/flink_artifact-new/target/udf_example-1.0.jar"
}

data "confluent_flink_artifact" "data-readfromid" {
environment {
    id = "env-devc1yv79z"
  }
  id = resource.confluent_flink_artifact.main.id
  cloud = "AWS"
  region = "us-west-2"
}

locals {
    plugin_id = confluent_flink_artifact.main.id
    version_id = confluent_flink_artifact.main.versions[0].version
    }

resource "confluent_flink_statement" "create-function" {
  statement     = "CREATE FUNCTION is_smaller  AS 'io.confluent.flink.table.modules.remoteudf.TShirtSizingIsSmaller' USING JAR 'confluent-artifact://${local.plugin_id}/${local.version_id}';"
#   statement     = "show user functions;"
  properties = {
      "sql.current-catalog"  = "Staging2"
      "sql.current-database" = "cluster_0"
    }
}

# resource "confluent_flink_statement" "select-current-timestamp" {
#   statement     = "SELECT CURRENT_TIMESTAMP;"
# }
