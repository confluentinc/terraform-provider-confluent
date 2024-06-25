---
page_title: "Confluent Provider 2.0.0: Upgrade Guide"
---
# Confluent Provider 2.0.0: Upgrade Guide

!> **Warning:** Version `2.0.0` of Confluent Provider hasn't been released yet and this guide describes how to resolve `Warning: Deprecated Resource` for deprecated `confluent_schema_registry_cluster` [resource](https://registry.terraform.io/providers/confluentinc/confluent/1.65.0/docs/resources/confluent_schema_registry_cluster) and 
 deprecated `confluent_schema_registry_region` [data source](https://registry.terraform.io/providers/confluentinc/confluent/1.65.0/docs/data-sources/confluent_schema_registry_region) as the warning message references this guide.

## Provider Version Configuration

-> **Note:** This guide uses the [basic-kafka-acls](https://github.com/confluentinc/terraform-provider-confluent/tree/v1.65.0/examples/configurations/basic-kafka-acls) Terraform configuration as an example of a Terraform configuration that has a Kafka cluster and a Schema Registry cluster.

Before reading further, ensure that your environment
successfully runs [`terraform plan`](https://www.terraform.io/docs/commands/plan.html)
without unexpected changes. Run the following command:
```bash
terraform plan
```
Your output should resemble:
```
confluent_service_account.test-sa: Refreshing state... [id=sa-xyz123]
confluent_environment.test-env: Refreshing state... [id=env-dge456]
confluent_kafka_cluster.basic: Refreshing state... [id=lkc-vrp3op]
data.confluent_schema_registry_region.essentials: Refreshing state... [id=sgreg-4]
...

No changes. Your infrastructure matches the configuration.

Terraform has compared your real infrastructure against your configuration and found no differences, so no changes are needed.
╷
│ Warning: Deprecated Resource
│ 
│   with data.confluent_schema_registry_region.essentials,
│   on main.tf line 20, in data "confluent_schema_registry_region" "essentials":
│   20: data "confluent_schema_registry_region" "essentials" {
│ 
│ The schema_registry_region data source has been deprecated and will be removed in the next major release (2.0.0). 
│ Refer to the Upgrade Guide at https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/version-2-upgrade for more details.
│ 
│ (and 2 more similar warnings elsewhere)
```

## Upgrade Terraform Configuration

### Changes to `confluent_schema_registry_cluster` resource

Deprecated `confluent_schema_registry_cluster`
[resource](https://registry.terraform.io/providers/confluentinc/confluent/1.65.0/docs/resources/confluent_schema_registry_cluster) will be removed in version `2.0.0`.

Use the `confluent_schema_registry_cluster` data source instead to avoid `Warning: Deprecated Resource` messages.

!> **Warning:** Ensure that you **do not** delete / destroy the Schema Registry cluster from Confluent Cloud (`confluent_schema_registry_cluster` resource) when going through this guide, as opposed to only removing it from the TF state.

The next step is to upgrade your TF configuration:

#### Before
    ```hcl
    resource "confluent_schema_registry_cluster" "essentials" {
      # ...
      environment {
        id = confluent_environment.staging.id
      }
    }
    ```

#### After
    ```hcl
    data "confluent_schema_registry_cluster" "essentials" {
      environment {
        id = confluent_environment.staging.id
      }
    }

    # Also make sure to replace all resource references confluent_schema_registry_cluster.essentials with
    # data.confluent_schema_registry_cluster.essentials
    ```

Next, remove the `confluent_schema_registry_cluster` resource from TF state (again, just from TF state and not from Confluent Cloud).

```bash
$ terraform state list | grep confluent_schema_registry_cluster 
$ terraform state rm confluent_schema_registry_cluster.essentials
```

Your output should resemble:
```
$ terraform state list | grep confluent_schema_registry_cluster 
confluent_schema_registry_cluster.essentials
$ terraform state rm confluent_schema_registry_cluster.essentials
Removed confluent_schema_registry_cluster.essentials
Successfully removed 1 resource instance(s).
```

-> **Note:** After running these commands your Schema Registry cluster still exists on Confluent Cloud, it was removed just from TF state.

### Changes to `confluent_schema_registry_region` data source

Deprecated `confluent_schema_registry_region`
[data source](https://registry.terraform.io/providers/confluentinc/confluent/1.65.0/docs/data-sources/confluent_schema_registry_region) will be removed in version `2.0.0`.


Remove the `confluent_schema_registry_cluster` data source only from TF configuration (as data sources are not stored in the TF state) instead 
to avoid `Warning: Deprecated Resource` messages.

To remove `confluent_schema_registry_cluster` data source from TF configuration, you can just remove its definition:

#### Before
    ```hcl
    data "confluent_stream_governance_region" "essentials" {
      # ...
    }
    ```

#### After
    ```hcl
    # empty
    ```

##### Sanity Check

Check that the upgrade was successful by ensuring that your environment
successfully runs [`terraform plan`](https://www.terraform.io/docs/commands/plan.html)
without unexpected changes. Run the following command:
```bash
terraform plan
```
Your output should resemble:
```
confluent_service_account.test-sa: Refreshing state... [id=sa-xyz123]
confluent_environment.test-env: Refreshing state... [id=env-dge456]
confluent_kafka_cluster.basic: Refreshing state... [id=lkc-vrp3op]
confluent_schema_registry_cluster.essentials: Refreshing state... [id=lsrc-abc123]
confluent_kafka_acl.describe-test-basic-cluster: Refreshing state... [id=lkc-abc123/CLUSTER#kafka-cluster#LITERAL#User:sa-xyz123#*#DESCRIBE#ALLOW]
confluent_kafka_topic.orders: Refreshing state... [id=lkc-abc123/orders]
confluent_kafka_acl.describe-orders: Refreshing state... [id=lkc-abc123/TOPIC#orders#LITERAL#User:sa-xyz123#*#DESCRIBE#ALLOW]
...
No changes. Infrastructure is up-to-date.
```

without any `Warning: Deprecated Resource` messages.

!> **Warning:** Ensure that you **do not** delete / destroy the Schema Registry cluster from Confluent Cloud (`confluent_schema_registry_cluster` resource) when going through this guide, as opposed to only removing it from the TF state.

### Changes to `confluent_kafka_cluster` resource and `confluent_kafka_cluster` data source

The deprecated `dedicated[0].encryption_key` attribute will be removed since it no 
longer exists in the [Confluent Cloud API](https://docs.confluent.io/cloud/current/api.html#tag/Clusters-(cmkv2)). You should use `byok_key[0].id` attribute instead.

If you run into any problems, [report an issue](https://github.com/confluentinc/terraform-provider-confluent/issues) to Confluent.
