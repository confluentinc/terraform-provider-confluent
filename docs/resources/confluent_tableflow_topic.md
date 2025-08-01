---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "confluent_tableflow_topic Resource - terraform-provider-confluent"
subcategory: ""
description: |-
  
---

# confluent_tableflow_topic Resource

[![General Availability](https://img.shields.io/badge/Lifecycle%20Stage-General%20Availability-%2345c6e8)](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy)

-> **Note:** It is recommended to set `lifecycle { prevent_destroy = true }` on production instances to prevent accidental tableflow topic deletion. This setting rejects plans that would destroy or recreate the tableflow topic, such as attempting to change uneditable attributes. Read more about it in the [Terraform docs](https://www.terraform.io/language/meta-arguments/lifecycle#prevent_destroy).

-> **Note:** Make sure to use `confluent_catalog_integration` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_catalog_integration) if you want to integrate Tableflow with AWS Glue Catalog or Snowflake Open Catalog.

## Example Usage

### Option #1: Manage multiple Tableflow Topics in the same Terraform workspace

```terraform
# https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/tableflow/confluent-managed-storage
provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key    # optionally use CONFLUENT_CLOUD_API_KEY env var
  cloud_api_secret = var.confluent_cloud_api_secret # optionally use CONFLUENT_CLOUD_API_SECRET env var
}

resource "confluent_tableflow_topic" "example" {
  environment {
    id = data.confluent_environment.staging.id
  }
  kafka_cluster {
    id = data.confluent_kafka_cluster.staging.id
  }
  display_name = data.confluent_kafka_topic.orders.topic_name
  table_formats = ["ICEBERG", "DELTA"]
  managed_storage {}
  credentials {
    key    = confluent_api_key.env-admin-tableflow-api-key.id
    secret = confluent_api_key.env-admin-tableflow-api-key.secret
  }

  lifecycle {
    prevent_destroy = true
  }
}
```

### Option #2: Manage a single Tableflow Topic in the same Terraform workspace

```terraform
# https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/tableflow/byob-aws-storage
provider "confluent" {
  cloud_api_key        = var.confluent_cloud_api_key    # optionally use CONFLUENT_CLOUD_API_KEY env var
  cloud_api_secret     = var.confluent_cloud_api_secret # optionally use CONFLUENT_CLOUD_API_SECRET env var
  tableflpw_api_key    = var.tableflow_api_key          # optionally use TABLEFLOW_API_KEY env var
  tableflow_api_secret = var.tableflow_api_secret       # optionally use TABLEFLOW_API_SECRET env var
}

resource "confluent_tableflow_topic" "example" {
  environment {
    id = data.confluent_environment.staging.id
  }
  kafka_cluster {
    id = data.confluent_kafka_cluster.staging.id
  }
  display_name = data.confluent_kafka_topic.orders.topic_name
  byob_aws {
    # S3 bucket must be in the same region as the Kafka cluster
    bucket_name = "bucket_1"
    provider_integration_id = data.confluent_provider_integration.main.id
  }

  lifecycle {
    prevent_destroy = true
  }
}
```

<!-- schema generated by tfplugindocs -->
## Argument Reference

The following arguments are supported:

- `environment` (Required Configuration Block) supports the following:
    - `id` - (Required String) The ID of the Environment, for example, `env-abc123`. 
- `kafka_cluster` (Required Configuration Block) supports the following:
    - `id` - (Required String) The ID of the Kafka cluster, for example, `lkc-abc123`.
- `display_name` - (Required String) The name of the Kafka topic for which Tableflow is enabled.
- `retention_ms` - (Optional String) The max age of snapshots (Iceberg) or versions (Delta) (snapshot/version expiration) to keep on the table in milliseconds for the Tableflow enabled topic.
- `table_formats` - (Optional List) The supported table formats for the Tableflow-enabled topic. Accepted values are `DELTA`, `ICEBERG`.
- `record_failure_strategy` - (Optional String) The strategy to handle record failures in the Tableflow enabled topic during materialization. Accepted values are `SKIP`, `SUSPEND`. For `SKIP`, we skip the bad records and move to the next record. For `SUSPEND`, we suspend the materialization of the topic.
- `byob_aws` (Optional Configuration Block) supports the following (See [Quick Start with Custom Storage](https://docs.confluent.io/cloud/current/topics/tableflow/get-started/quick-start-custom-storage-glue.html#cloud-tableflow-quick-start) for more details):
    - `bucket_name` - (Required String) The bucket name.
    - `provider_integration_id` - (Required String) The provider integration id.
- `managed_storage` (Optional Configuration Block) The configuration of the Confluent managed storage. See [Quick Start with Managed Storage](https://docs.confluent.io/cloud/current/topics/tableflow/get-started/quick-start-managed-storage.html#cloud-tableflow-quick-start-managed-storage) for more details.
- `credentials` (Optional Configuration Block) supports the following:
    - `key` - (Required String) The Tableflow API Key.
    - `secret` - (Required String, Sensitive) The Tableflow API Secret.

-> **Note:** A Tableflow API key consists of a key and a secret. Tableflow API keys are required to interact with Tableflow Topics in Confluent Cloud.

-> **Note:** Use Option #2 to simplify the key rotation process. When using Option #1, to rotate a Tableflow API key, create a new Tableflow API key, update the `credentials` block in all configuration files to use the new Tableflow API key, run `terraform apply -target="confluent_tableflow_topic.example"`, and remove the old Tableflow API key. Alternatively, in case the old Tableflow API Key was deleted already, you might need to run `terraform plan -refresh=false -target="confluent_tableflow_topic.example" -out=rotate-tableflow-api-key` and `terraform apply rotate-tableflow-api-key` instead.

!> **Warning:** Use Option #2 to avoid exposing sensitive `credentials` value in a state file. When using Option #1, Terraform doesn't encrypt the sensitive `credentials` value of the `confluent_tableflow_topic` resource, so you must keep your state file secure to avoid exposing it. Refer to the [Terraform documentation](https://www.terraform.io/docs/language/state/sensitive-data.html) to learn more about securing your state file.

## Attributes Reference

In addition to the preceding arguments, the following attributes are exported:

- `enable_compaction` - (Optional Boolean) This flag determines whether to enable compaction for the Tableflow enabled topic.
- `enable_partitioning` - (Optional Boolean) This flag determines whether to enable partitioning for the Tableflow enabled topic.
- `suspended` - (Optional Boolean) Indicates whether the Tableflow should be suspended.
- `table_path` - (Optional String) The current storage path where the data and metadata is stored for this table.
- `byob_aws` (Optional Configuration Block) supports the following:
    - `bucket_region` - (Required String) The bucket region.

## Import

You can import a Tableflow Topic by using the Tableflow Topic name, Environment ID, and Kafka Cluster ID, in the format `<Environment ID>/<Kafka Cluster ID>/<Tableflow Topic name>`, for example:

```shell
# Option #1: Manage multiple Tableflow Topics in the same Terraform workspace
$ export IMPORT_TABLEFLOW_API_KEY="<tableflow_api_key>"
$ export IMPORT_TABLEFLOW_API_SECRET="<tableflow_api_secret>"
$ terraform import confluent_tableflow_topic.example env-abc123/lkc-abc123/orders

# Option #2: Manage a single Tableflow Topic in the same Terraform workspace
$ terraform import confluent_tableflow_topic.example env-abc123/lkc-abc123/orders
```

!> **Warning:** Do not forget to delete terminal command history afterwards for security purposes.

## Getting Started
The following end-to-end examples might help to get started with `confluent_tableflow_topic` resource:
* [confluent-managed-storage](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/tableflow/confluent-managed-storage): Tableflow topic with Confluent-managed storage.
* [byob-aws-storage](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/tableflow/confluent-managed-storage): Tableflow topic with custom (BYOB AWS) storage.
* [datagen-connector-byob-aws-storage](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/tableflow/datagen-connector-byob-aws-storage): Datagen Source connector with a Tableflow topic with custom (BYOB AWS) storage.
* [datagen-connector-confluent-managed-storage](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/tableflow/datagen-connector-confluent-managed-storage): Datagen Source connector with a Tableflow topic with Confluent-managed storage.
