---
page_title: "Confluent Provider 2.45.0: Upgrade Guide for OAuth Authentication Migration"
---
# Confluent Provider 2.45.0: Upgrade Guide for OAuth Authentication Migration

## Summary

This guide provides detailed instructions for migrating from API key/secret authentication (both Cloud-level and resource-specific) to OAuth-based authentication in the Confluent Terraform Provider.

**Note:** 
- OAuth authentication requires Confluent Terraform Provider version v2.37.0 or later, and v2.43.0 or later is recommended for best functionality and compatibility.
- Please read the [OAuth Authentication Guide](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs#oauth-credentials) for more fundamentals on OAuth authentication.

## Provider Configuration Migration Instructions

Before reading further, ensure that your current configuration using API key/secret credentials blocks
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

The next step is to manually update your Terraform configuration to do the following:
- Remove the `cloud_api_key` and `cloud_api_secret` attributes from the `provider "confluent"` block and add the `oauth` block with the necessary attributes.
- Remove all `credentials` blocks (including the nested `credentials` blocks).

### Terraform Configuration Before Migration

#### Option #1: Manage multiple Kafka/Schema Registry clusters in the same Terraform workspace

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

#### Option #2: Manage a single Kafka/Schema Registry cluster in the same Terraform workspace

```hcl
provider "confluent" {
  // Confluent Cloud API key/secret
  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret

  // Kafka cluster API key/secret
  kafka_id            = var.kafka_id                   # optionally use KAFKA_ID env var
  kafka_rest_endpoint = var.kafka_rest_endpoint        # optionally use KAFKA_REST_ENDPOINT env var
  kafka_api_key       = var.kafka_api_key              # optionally use KAFKA_API_KEY env var
  kafka_api_secret    = var.kafka_api_secret           # optionally use KAFKA_API_SECRET env var
  
  // Schema Registry cluster API key/secret
  schema_registry_id            = var.schema_registry_id            # optionally use SCHEMA_REGISTRY_ID env var
  schema_registry_rest_endpoint = var.schema_registry_rest_endpoint # optionally use SCHEMA_REGISTRY_REST_ENDPOINT env var
  schema_registry_api_key       = var.schema_registry_api_key       # optionally use SCHEMA_REGISTRY_API_KEY env var
  schema_registry_api_secret    = var.schema_registry_api_secret    # optionally use SCHEMA_REGISTRY_API_SECRET env var
}

#... other resources ...
resource "confluent_kafka_topic" "purchase" {
  topic_name    = "purchase"
  lifecycle {
    prevent_destroy = true
  }
}

resource "confluent_schema" "purchase" {
  subject_name = "${confluent_kafka_topic.purchase.topic_name}-value"
  format       = "PROTOBUF"
  schema       = file("./schemas/proto/purchase.proto")
  lifecycle {
    prevent_destroy = true
  }
}

resource "confluent_schema_exporter" "main" {
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

### Terraform Configuration After Migration

#### Option #1: Manage multiple Kafka/Schema Registry clusters in the same Terraform workspace

```hcl
provider "confluent" {
  // Replace all cloud and resource-specific API key/secret with this new oauth block
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

#### Option #2: Manage a single Kafka/Schema Registry cluster in the same Terraform workspace

```hcl
provider "confluent" {
  // Replace all cloud and resource-specific API key/secret with this new oauth block
  oauth {
    oauth_external_token_url = var.oauth_external_token_url
    oauth_external_client_id = var.oauth_external_client_id
    oauth_external_client_secret = var.oauth_external_client_secret
    oauth_external_token_scope = var.oauth_external_token_scope
    oauth_identity_pool_id = var.oauth_identity_pool_id
  }

  // Kafka cluster ID and REST endpoint
  kafka_id            = var.kafka_id                   # optionally use KAFKA_ID env var
  kafka_rest_endpoint = var.kafka_rest_endpoint        # optionally use KAFKA_REST_ENDPOINT env var
  
  // Schema Registry ID and REST endpoint
  schema_registry_id            = var.schema_registry_id            # optionally use SCHEMA_REGISTRY_ID env var
  schema_registry_rest_endpoint = var.schema_registry_rest_endpoint # optionally use SCHEMA_REGISTRY_REST_ENDPOINT env var
}
#... other resources ...
resource "confluent_kafka_topic" "purchase" {
  topic_name    = "purchase"
  lifecycle {
    prevent_destroy = true
  }
}

resource "confluent_schema" "purchase" {
  subject_name = "${confluent_kafka_topic.purchase.topic_name}-value"
  format       = "PROTOBUF"
  schema       = file("./schemas/proto/purchase.proto")
  lifecycle {
    prevent_destroy = true
  }
}

resource "confluent_schema_exporter" "main" {
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

**Note:** Once the migration is complete, you can optionally remove any `confluent_api_key` or `confluent_role_binding` resources from your configuration, if they are no longer needed elsewhere.

### Verify the Terraform Plan Output

After making the above changes to your Terraform configuration, run the `terraform plan` command again:

For **Option #1**, since the API key/secret credentials are now removed from the provider-level block, no Terraform configuration changes should be detected, and your output should resemble:

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

For **Option #2**, you should see output similar to the following, indicating that the resources are being updated in place to remove the `credentials` blocks:

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

**Note:** The plan output should display `0 to add` **and** `0 to destroy`, confirming that no resources will be recreated or deleted.

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

In the Terraform state file, the `credentials` block will appear as an empty array `[]`. This indicates that all resources/data-sources are now successfully authenticated via the provider’s OAuth configuration, rather than Cloud/resource-specific credentials.
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

For more details on using OAuth authentication in Confluent Cloud, please refer to the [OAuth Authentication Guide](https://docs.confluent.io/cloud/current/security/authenticate/workload-identities/identity-providers/oauth/overview.html).

If you run into any problems, [report an issue](https://github.com/confluentinc/terraform-provider-confluent/issues) to Confluent.
