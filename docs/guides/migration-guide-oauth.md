---
page_title: "Confluent Provider OAuth Authentication: Migration Guide"
---
# Confluent Provider OAuth Authentication: Migration Guide

## Summary

This guide is intended to help with the migration process and focuses only on the changes necessary to migrate from API key/secret to OAuth authentication in Confluent Provider.

**Note:** It's recommended to use the Terraform provider Confluent version `v2.43.0` onwards for the most OAuth support.

## Provider Configuration Migration Instructions

**Note:** Please read the [OAuth Authentication Guide](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs#oauth-credentials) for more fundamentals on OAuth authentication.

Before reading further, ensure that your current environment with API key/secret credentials blocks
successfully runs [`terraform plan`](https://www.terraform.io/docs/commands/plan.html)
without unexpected changes. Run the following command:
```bash
terraform plan
```
Your output should resemble:
```bash
confluent_service_account.test-sa: Refreshing state... [id=sa-abc123]
confluent_environment.test-env: Refreshing state... [id=env-abc123]
confluent_kafka_cluster.source: Refreshing state... [id=lkc-abc123]
data.confluent_schema_registry_region.essentials: Refreshing state... [id=sgreg-4]
...

No changes. Your infrastructure matches the configuration.

Terraform has compared your real infrastructure against your configuration and found no differences, so no changes are needed.
```

The next step is to manually upgrade your TF configuration to:
- Remove the `cloud_api_key` and `cloud_api_secret` attributes from the `provider "confluent"` block and add the `oauth` block with the necessary attributes.
- Remove all `credentials` blocks (including the nested `credentials` blocks) from resources that required one before, now the `credentials` block becomes optional.

### Terraform Configuration Before

```hcl
provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret
}
#... other resources ...
resource "confluent_kafka_topic" "purchase" {
  kafka_cluster {
    id = confluent_kafka_cluster.source.id
  }
  topic_name    = "purchase"
  rest_endpoint = confluent_kafka_cluster.source.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager-kafka-api-key.id
    secret = confluent_api_key.app-manager-kafka-api-key.secret
  }
}

resource "confluent_schema" "purchase" {
  schema_registry_cluster {
    id = data.confluent_schema_registry_cluster.source.id
  }
  rest_endpoint = data.confluent_schema_registry_cluster.source.rest_endpoint
  # https://developer.confluent.io/learn-kafka/schema-registry/schema-subjects/#topicnamestrategy
  subject_name = "${confluent_kafka_topic.purchase.topic_name}-value"
  format       = "PROTOBUF"
  schema       = file("./schemas/proto/purchase.proto")
  credentials {
    key    = confluent_api_key.env-manager-schema-registry-api-key.id
    secret = confluent_api_key.env-manager-schema-registry-api-key.secret
  }
}

resource "confluent_schema_exporter" "main" {
  schema_registry_cluster {
    id = data.confluent_schema_registry_cluster.source.id
  }
  rest_endpoint = data.confluent_schema_registry_cluster.source.rest_endpoint
  credentials {
    key    = confluent_api_key.env-manager-schema-registry-api-key.id
    secret = confluent_api_key.env-manager-schema-registry-api-key.secret
  }

  name         = "my_exporter"
  context      = "schema_context"
  context_type = "CUSTOM"
  subjects     = [confluent_schema.purchase.subject_name]

  destination_schema_registry_cluster {
    rest_endpoint = data.confluent_schema_registry_cluster.destination.rest_endpoint
    credentials {
      key    = confluent_api_key.destination-schema-registry-api-key.id
      secret = confluent_api_key.destination-schema-registry-api-key.secret
    }
  }

  reset_on_update = true
}
```

### Terraform Configuration After

```hcl
provider "confluent" {
  oauth {
    oauth_external_token_url = var.oauth_external_token_url
    oauth_external_client_id = var.oauth_external_client_id
    oauth_external_client_secret = var.oauth_external_client_secret
    oauth_external_token_scope = var.oauth_external_token_scope
    oauth_identity_pool_id = var.oauth_identity_pool_id
  }
}
#... other resources ...
resource "confluent_kafka_topic" "purchase" {
  kafka_cluster {
    id = confluent_kafka_cluster.source.id
  }
  topic_name    = "purchase"
  rest_endpoint = confluent_kafka_cluster.source.rest_endpoint

  depends_on = [
    confluent_kafka_cluster.source
  ]
}

resource "confluent_schema" "purchase" {
  schema_registry_cluster {
    id = data.confluent_schema_registry_cluster.source.id
  }
  rest_endpoint = data.confluent_schema_registry_cluster.source.rest_endpoint
  # https://developer.confluent.io/learn-kafka/schema-registry/schema-subjects/#topicnamestrategy
  subject_name = "${confluent_kafka_topic.purchase.topic_name}-value"
  format       = "PROTOBUF"
  schema       = file("./schemas/proto/purchase.proto")
}

resource "confluent_schema_exporter" "main" {
  schema_registry_cluster {
    id = data.confluent_schema_registry_cluster.source.id
  }
  rest_endpoint = data.confluent_schema_registry_cluster.source.rest_endpoint
  name         = "my_exporter"
  context      = "schema_context"
  context_type = "CUSTOM"
  subjects     = [confluent_schema.purchase.subject_name]

  destination_schema_registry_cluster {
    id = data.confluent_schema_registry_cluster.destination.id
    rest_endpoint = data.confluent_schema_registry_cluster.destination.rest_endpoint
  }

  reset_on_update = true
}
```

**Note:** Once the migration is complete, you can remove any now-unused `confluent_api_key` and `confluent_role_binding` resources from your configuration.

### Expected Terraform Plan Output

After making the above changes to your TF configuration, run the `terraform plan` command again, you should see output similar to the following, indicating that the resources are being updated in place to remove the `credentials` blocks:

```bash
Terraform will perform the following actions:
  # confluent_kafka_topic.purchase will be updated in-place
  ~ resource "confluent_kafka_topic" "purchase" {
        ...
        # (4 unchanged attributes hidden)
      - credentials {
          - key    = (sensitive value) -> null
          - secret = (sensitive value) -> null
        }
        # (1 unchanged block hidden)
    }
  # confluent_schema.purchase will be updated in-place
  ~ resource "confluent_schema" "purchase" {
        ...
        # (9 unchanged attributes hidden)
      - credentials {
          - key    = (sensitive value) -> null
          - secret = (sensitive value) -> null
        }
        # (1 unchanged block hidden)
    }
  # confluent_schema_exporter.main will be updated in-place
  ~ resource "confluent_schema_exporter" "main" {
        ...
        # (8 unchanged attributes hidden)
      - credentials {
          - key    = (sensitive value) -> null
          - secret = (sensitive value) -> null
        }
        # (2 unchanged blocks hidden)
    }
Plan: 0 to add, 3 to change, 0 to destroy.
```

## Sanity Check

Check that the upgrade was successful by ensuring that your environment
successfully runs [`terraform plan`](https://www.terraform.io/docs/commands/plan.html)
without unexpected changes. Run the following command:
```bash
terraform plan
```
Your output should resemble:
```bash
confluent_environment.destination: Refreshing state... [id=env-def456]
confluent_environment.source: Refreshing state... [id=env-abc123]
confluent_kafka_cluster.source: Refreshing state... [id=lkc-abc123]
confluent_kafka_cluster.destination: Refreshing state... [id=lkc-def456]
data.confluent_schema_registry_cluster.source: Reading...
confluent_kafka_topic.purchase_source: Refreshing state... [id=lkc-abc123/purchase]
data.confluent_schema_registry_cluster.destination: Reading...
data.confluent_schema_registry_cluster.destination: Read complete after 1s [id=lsrc-def456]
data.confluent_schema_registry_cluster.source: Read complete after 1s [id=lsrc-abc123]
confluent_schema.purchase: Refreshing state... [id=lsrc-abc123/purchase-value/latest]
confluent_schema_exporter.main: Refreshing state... [id=lsrc-abc123/my_exporter]
No changes. Your infrastructure matches the configuration.
Terraform has compared your real infrastructure against your configuration and found no differences, so no
changes are needed.
```

In the state file, the `credentials` block now becomes an empty array, indicating a successful migration.
```
{
  "mode": "managed",
  "type": "confluent_kafka_topic",
  "name": "purchase",
  "provider": "provider[\"terraform.confluent.io/confluentinc/confluent\"]",
  "instances": [
    {
      "credentials": [],
      "id": "lkc-abc123/purchase",
      # ...additional attributes here...
    }
  ]
}
```

If you run into any problems, [report an issue](https://github.com/confluentinc/terraform-provider-confluent/issues) to Confluent.
