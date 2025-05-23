---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "confluent_tag_binding Data Source - terraform-provider-confluent"
subcategory: ""
description: |-
   
---

# confluent_tag_binding Data Source

[![General Availability](https://img.shields.io/badge/Lifecycle%20Stage-General%20Availability-%2345c6e8)](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy)

`confluent_tag_binding` describes a Tag Binding data source.

## Example Usage

### Option #1: Manage multiple Schema Registry clusters in the same Terraform workspace

```terraform
provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key    # optionally use CONFLUENT_CLOUD_API_KEY env var
  cloud_api_secret = var.confluent_cloud_api_secret # optionally use CONFLUENT_CLOUD_API_SECRET env var
}

data "confluent_tag_binding" "main" {
  schema_registry_cluster {
    id = data.confluent_schema_registry_cluster.essentials.id
  }
  rest_endpoint = data.confluent_schema_registry_cluster.essentials.rest_endpoint
  credentials {
    key    = "<Schema Registry API Key for data.confluent_schema_registry_cluster.essentials>"
    secret = "<Schema Registry API Secret for data.confluent_schema_registry_cluster.essentials>"
  }
  
  tag_name = "PII"
  entity_name = "lsrc-8wrx70:.:100001"
  entity_type = "sr_schema"
}
```

### Option #2: Manage a single Schema Registry cluster in the same Terraform workspace

```terraform
provider "confluent" {
  schema_registry_id            = var.schema_registry_id            # optionally use SCHEMA_REGISTRY_ID env var
  catalog_rest_endpoint         = var.catalog_rest_endpoint         # optionally use CATALOG_REST_ENDPOINT env var
  schema_registry_api_key       = var.schema_registry_api_key       # optionally use SCHEMA_REGISTRY_API_KEY env var
  schema_registry_api_secret    = var.schema_registry_api_secret    # optionally use SCHEMA_REGISTRY_API_SECRET env var
}

data "confluent_tag_binding" "main" {
  tag_name = "PII"
  entity_name = "lsrc-8wrx70:.:100001"
  entity_type = "sr_schema"
}
```
-> **Note:** We also support `schema_registry_rest_endpoint` instead of `catalog_rest_endpoint` for the time being.

<!-- schema generated by tfplugindocs -->
## Argument Reference

The following arguments are supported:

- `schema_registry_cluster` - (Optional Configuration Block) supports the following:
    - `id` - (Required String) The ID of the Schema Registry cluster, for example, `lsrc-abc123`.
- `rest_endpoint` - (Optional String) The REST endpoint of the Schema Registry cluster, for example, `https://psrc-00000.us-central1.gcp.confluent.cloud:443`).
- `credentials` (Optional Configuration Block) supports the following:
    - `key` - (Required String) The Schema Registry API Key.
    - `secret` - (Required String, Sensitive) The Schema Registry API Secret.
- `tag_name` - (Required String) The name of the tag to be applied, for example, `PII`. The name must not be empty and consist of a letter followed by a sequence of letter, number, space, or _ characters.
- `entity_name` - (Required String) The qualified name of the entity, for example, `${data.confluent_schema_registry_cluster.essentials.id}:.:${confluent_schema.purchase.schema_identifier}`, `${data.confluent_schema_registry_cluster.essentials.id}:${confluent_kafka_cluster.basic.id}:${confluent_kafka_topic.purchase.topic_name}`. Refer to the [Examples of qualified names](https://docs.confluent.io/cloud/current/stream-governance/stream-catalog-rest-apis.html#examples-of-qualified-names) to see the full list of supported values for the `entity_name` attribute.
- `entity_type` - (Required String) The entity type, for example, `sr_schema`.

-> **Note:** A Schema Registry API key consists of a key and a secret. Schema Registry API keys are required to interact with Schema Registry clusters in Confluent Cloud. Each Schema Registry API key is valid for one specific Schema Registry cluster.

## Attributes Reference

In addition to the preceding arguments, the following attributes are exported:

- `id` - (Required String) The ID of the Tag Binding, in the format `<Schema Registry Cluster Id>/<Tag Name>/<Entity Name>/<Entity Type>`, for example, `lsrc-8wrx70/PII/lsrc-8wrx70:.:100001/sr_schema`.
