---
page_title: "Confluent Provider 0.12.0: Upgrade Guide"
---
# Confluent Provider 0.12.0: Upgrade Guide

This guide is intended to help with the upgrading process and focuses only on the changes necessary to upgrade to
version `0.12.0` of Confluent Provider from either 
* version `0.11.0` of Confluent Provider or
* version `0.10.0` of Confluent Provider.

-> **Note:** If you're upgrading from a version that's earlier than `0.10.0`, upgrade to
version `0.10.0` before starting this guide. For more information, see
[Confluent Provider 0.7.0: Upgrade Guide](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/upgrade-guide-0.7.0).

!> **Warning:** Don't forget to create backups of the `terraform.tfstate` state file and your TF configuration (for
example, `main.tf`) before upgrading.

## Upgrade Guide: Upgrading from version `0.11.0` of Confluent Provider

### Provider Version Configuration

-> **Note:** This guide uses [basic-kafka-acls](https://github.com/confluentinc/terraform-provider-confluent/tree/v0.10.0/examples/configurations/basic-kafka-acls) Terraform configuration as an example of a Terraform configuration that has a Kafka cluster and multiple ACLs.

Before upgrading to version `0.12.0`, ensure that your environment successfully
runs [`terraform plan`](https://www.terraform.io/docs/commands/plan.html)
without unexpected changes. Run the following command:

```bash
terraform plan
```

Your output should resemble:

```
confluent_service_account_v2.test-sa: Refreshing state... [id=sa-xyz123]
confluent_environment_v2.test-env: Refreshing state... [id=env-dge456]
confluent_kafka_cluster_v2.test-basic-cluster: Refreshing state... [id=lkc-abc123]
confluent_kafka_acl_v3.describe-test-basic-cluster: Refreshing state... [id=lkc-abc123/CLUSTER#kafka-cluster#LITERAL#User:12345#*#DESCRIBE#ALLOW]
confluent_kafka_topic_v3.orders: Refreshing state... [id=lkc-abc123/orders]
confluent_kafka_acl_v3.describe-orders: Refreshing state... [id=lkc-n2kvd/TOPIC#orders#LITERAL#User:12345#*#DESCRIBE#ALLOW]
...
No changes. Infrastructure is up-to-date.
```

The next step is to set the latest `0.12.0` version in a `required_providers` block of your Terraform configuration.

#### Before

```hcl
terraform {
  required_providers {
    # ...
    confluent = {
      source  = "confluentinc/confluent"
      version = "0.11.0"
    }
  }
}

provider "confluent" {
```

#### After

```hcl
terraform {
  required_providers {
    # ...
    confluent = {
      source  = "confluentinc/confluent"
      version = "0.12.0"
    }
  }
}

provider "confluent" {
```

### Upgrade Terraform Configuration

#### Changes to all resources and data sources

-> **Note:** The `0.12.0` release reverts resource versioning changes introduced in `0.11.0`. User feedback on versioned resources made it clear that the pain of manually updating the TF state file outweighs the potential benefits of deprecation flexibility that versioned resources could have provided. In order to avoid forcing users to edit their TF state files (either manually or by running commands like `terraform state mv`) in the future, TF [state migrations](https://www.terraform.io/plugin/sdkv2/resources/state-migration) will be handled within the Confluent Terraform Provider whenever possible.

All resources and data sources have been renamed not to include corresponding API versions. For
example, `confluent_environment_v2` resource was renamed to `confluent_environment`).

Run the following commands to update your TF configuration file.

```bash
# Replaces confluent_environment with confluent_environment_v2 (and others in a similar fashion) in main.tf
sed -i '' 's/confluent_api_key_v2/confluent_api_key/g' main.tf
sed -i '' 's/confluent_connector_v1/confluent_connector/g' main.tf
sed -i '' 's/confluent_environment_v2/confluent_environment/g' main.tf
sed -i '' 's/confluent_kafka_acl_v3/confluent_kafka_acl/g' main.tf
sed -i '' 's/confluent_kafka_cluster_v2/confluent_kafka_cluster/g' main.tf
sed -i '' 's/confluent_kafka_topic_v3/confluent_kafka_topic/g' main.tf
sed -i '' 's/confluent_network_v1/confluent_network/g' main.tf
sed -i '' 's/confluent_organization_v2/confluent_organization/g' main.tf
sed -i '' 's/confluent_peering_v1/confluent_peering/g' main.tf
sed -i '' 's/confluent_private_link_access_v1/confluent_private_link_access/g' main.tf
sed -i '' 's/confluent_role_binding_v2/confluent_role_binding/g' main.tf
sed -i '' 's/confluent_service_account_v2/confluent_service_account/g' main.tf
sed -i '' 's/confluent_user_v2/confluent_user/g' main.tf
```

### Upgrade State File

Similarly, you need to rename all resources and data sources in your TF state file. You can do it by running the following commands:

```bash
# Replaces confluent_environment_v2 with confluent_environment (and others in a similar fashion) in terraform.tfstate
sed -i '' 's/confluent_api_key_v2/confluent_api_key/g' terraform.tfstate
sed -i '' 's/confluent_connector_v1/confluent_connector/g' terraform.tfstate
sed -i '' 's/confluent_environment_v2/confluent_environment/g' terraform.tfstate
sed -i '' 's/confluent_kafka_acl_v3/confluent_kafka_acl/g' terraform.tfstate
sed -i '' 's/confluent_kafka_cluster_v2/confluent_kafka_cluster/g' terraform.tfstate
sed -i '' 's/confluent_kafka_topic_v3/confluent_kafka_topic/g' terraform.tfstate
sed -i '' 's/confluent_network_v1/confluent_network/g' terraform.tfstate
sed -i '' 's/confluent_peering_v1/confluent_peering/g' terraform.tfstate
sed -i '' 's/confluent_private_link_access_v1/confluent_private_link_access/g' terraform.tfstate
sed -i '' 's/confluent_role_binding_v2/confluent_role_binding/g' terraform.tfstate
sed -i '' 's/confluent_service_account_v2/confluent_service_account/g' terraform.tfstate
sed -i '' 's/confluent_organization_v2/confluent_organization/g' terraform.tfstate
sed -i '' 's/confluent_user_v2/confluent_user/g' terraform.tfstate
```

### Sanity Check

Check that the upgrade was successful by ensuring that your environment successfully
runs [`terraform plan`](https://www.terraform.io/docs/commands/plan.html)
without unexpected changes. Run the following command:

```bash
terraform plan
```

Your output should resemble:

```
confluent_service_account.test-sa: Refreshing state... [id=sa-xyz123]
confluent_environment.test-env: Refreshing state... [id=env-dge456]
confluent_kafka_cluster.test-basic-cluster: Refreshing state... [id=lkc-abc123]
confluent_kafka_acl.describe-test-basic-cluster: Refreshing state... [id=lkc-abc123/CLUSTER#kafka-cluster#LITERAL#User:sa-xyz123#*#DESCRIBE#ALLOW]
confluent_kafka_topic.orders: Refreshing state... [id=lkc-abc123/orders]
confluent_kafka_acl.describe-orders: Refreshing state... [id=lkc-abc123/TOPIC#orders#LITERAL#User:sa-xyz123#*#DESCRIBE#ALLOW]
...
No changes. Infrastructure is up-to-date.
```

If you run into any problems,
please [report an issue](https://github.com/confluentinc/terraform-provider-confluent/issues).

## Upgrade Guide: Upgrading from version `0.10.0` of Confluent Provider

### Provider Version Configuration

-> **Note:** This guide uses [basic-kafka-acls](https://github.com/confluentinc/terraform-provider-confluent/tree/v0.10.0/examples/configurations/basic-kafka-acls) Terraform configuration as an example of a Terraform configuration that has a Kafka cluster and multiple ACLs.

Before upgrading to version `0.12.0`, ensure that your environment successfully
runs [`terraform plan`](https://www.terraform.io/docs/commands/plan.html)
without unexpected changes. Run the following command:

```bash
terraform plan
```

Your output should resemble:

```
confluentcloud_service_account.test-sa: Refreshing state... [id=sa-xyz123]
confluentcloud_environment.test-env: Refreshing state... [id=env-dge456]
confluentcloud_kafka_cluster.test-basic-cluster: Refreshing state... [id=lkc-abc123]
confluentcloud_kafka_acl.describe-test-basic-cluster: Refreshing state... [id=lkc-abc123/CLUSTER#kafka-cluster#LITERAL#User:12345#*#DESCRIBE#ALLOW]
confluentcloud_kafka_topic.orders: Refreshing state... [id=lkc-abc123/orders]
confluentcloud_kafka_acl.describe-orders: Refreshing state... [id=lkc-n2kvd/TOPIC#orders#LITERAL#User:12345#*#DESCRIBE#ALLOW]
...
No changes. Infrastructure is up-to-date.
```

The next step is to set the latest `0.12.0` version in a `required_providers` block of your Terraform configuration.

#### Before

```hcl
terraform {
  required_providers {
    # ...
    confluent = {
      source  = "confluentinc/confluent"
      version = "0.10.0"
    }
  }
}

provider "confluent" {
```

#### After

```hcl
terraform {
  required_providers {
    # ...
    confluent = {
      source  = "confluentinc/confluent"
      version = "0.12.0"
    }
  }
}

provider "confluent" {
```

### Upgrade Terraform Configuration

* The `api_key`, `api_secret` attributes were renamed to `cloud_api_key`, `cloud_api_secret`, respectively. Update your TF
  configuration accordingly.

  #### Before
    ```hcl
    provider "confluent" {
      api_key    = var.confluent_cloud_api_key    # optionally use CONFLUENT_CLOUD_API_KEY env var
      api_secret = var.confluent_cloud_api_secret # optionally use CONFLUENT_CLOUD_API_SECRET env var
    }
    ```

  #### After
    ```hcl
    provider "confluent" {
      cloud_api_key    = var.confluent_cloud_api_key    # optionally use CONFLUENT_CLOUD_API_KEY env var
      cloud_api_secret = var.confluent_cloud_api_secret # optionally use CONFLUENT_CLOUD_API_SECRET env var
    }
    ```

-> **Note:** If you don't use `api_key` and `api_secret` attributes and use environment variables instead (i.e.,
your `provider` block is empty: `provider "confluent" {}`) then no changes are necessary.

#### Changes to `confluentcloud_kafka_acl` resource

* The `http_endpoint` attribute was renamed to `rest_endpoint`. Update your TF configuration accordingly.

  ##### Before
    ```hcl
    resource "confluentcloud_kafka_acl" "describe-orders" {
      http_endpoint = "lkc-abc123"
    }
    ```

  ##### After
    ```hcl
    resource "confluentcloud_kafka_acl" "describe-orders" {
      rest_endpoint = "lkc-abc123"
    }
    ```

#### Changes to `confluentcloud_kafka_topic` resource

* The `http_endpoint` attribute was renamed to `rest_endpoint`. Update your TF configuration accordingly.

  ##### Before
    ```hcl
    resource "confluentcloud_kafka_topic" "orders" {
      http_endpoint = "lkc-abc123"
    }
    ```

  ##### After
    ```hcl
    resource "confluentcloud_kafka_topic" "orders" {
      rest_endpoint = "lkc-abc123"
    }
    ```

#### Changes to `confluentcloud_kafka_cluster` resource

* The `http_endpoint` attribute was renamed to `rest_endpoint`. No changes in your TF configuration are necessary, since the
  attribute is [computed](https://www.terraform.io/plugin/sdkv2/schemas/schema-behaviors#computed).

You might find it helpful to run

```bash
# Replaces http_endpoint with rest_endpoint in main.tf
sed -i '' 's/http_endpoint/rest_endpoint/g' main.tf
```

instead of updating your TF configuration file (for example, called `main.tf`) manually.

### Upgrade State File

The `0.12.0` release automatically renames the `http_endpoint` attribute to `rest_endpoint` in your TF state file if you run the following commands:

```bash
terraform plan
```

Your output should resemble:
```
...
confluent_service_account.test-sa: Refreshing state... [id=sa-xyz123]
confluent_environment.test-env: Refreshing state... [id=env-dge456]
confluent_kafka_cluster.test-basic-cluster: Refreshing state... [id=lkc-abc123]
confluent_kafka_acl.describe-test-basic-cluster: Refreshing state... [id=lkc-abc123/CLUSTER#kafka-cluster#LITERAL#User:sa-xyz123#*#DESCRIBE#ALLOW]
confluent_kafka_topic.orders: Refreshing state... [id=lkc-abc123/orders]
confluent_kafka_acl.describe-orders: Refreshing state... [id=lkc-abc123/TOPIC#orders#LITERAL#User:sa-xyz123#*#DESCRIBE#ALLOW]

No changes. Infrastructure is up-to-date.

This means that Terraform did not detect any differences between your
configuration and real physical resources that exist. As a result, no
actions need to be performed.
```

and

```bash
terraform apply
```

Your output should resemble:
```
...
confluent_service_account.test-sa: Refreshing state... [id=sa-xyz123]
confluent_environment.test-env: Refreshing state... [id=env-dge456]
confluent_kafka_cluster.test-basic-cluster: Refreshing state... [id=lkc-abc123]
confluent_kafka_acl.describe-test-basic-cluster: Refreshing state... [id=lkc-abc123/CLUSTER#kafka-cluster#LITERAL#User:sa-xyz123#*#DESCRIBE#ALLOW]
confluent_kafka_topic.orders: Refreshing state... [id=lkc-abc123/orders]
confluent_kafka_acl.describe-orders: Refreshing state... [id=lkc-abc123/TOPIC#orders#LITERAL#User:sa-xyz123#*#DESCRIBE#ALLOW]

Apply complete! Resources: 0 added, 0 changed, 0 destroyed.
```

-> **Note:** Even though the previous commands didn't show any changes, it's required to run these commands, because they rename the `http_endpoint` attribute to `rest_endpoint` in your TF state file through `StateUpgrader` [function](https://www.terraform.io/plugin/sdkv2/resources/state-migration#terraform-v0-12-sdk-state-migrations).

### Sanity Check

Check that the upgrade was successful by ensuring that your environment successfully
runs [`terraform plan`](https://www.terraform.io/docs/commands/plan.html)
without unexpected changes. Run the following command:

```bash
terraform plan
```

Your output should resemble:

```
confluent_service_account.test-sa: Refreshing state... [id=sa-xyz123]
confluent_environment.test-env: Refreshing state... [id=env-dge456]
confluent_kafka_cluster.test-basic-cluster: Refreshing state... [id=lkc-abc123]
confluent_kafka_acl.describe-test-basic-cluster: Refreshing state... [id=lkc-abc123/CLUSTER#kafka-cluster#LITERAL#User:sa-xyz123#*#DESCRIBE#ALLOW]
confluent_kafka_topic.orders: Refreshing state... [id=lkc-abc123/orders]
confluent_kafka_acl.describe-orders: Refreshing state... [id=lkc-abc123/TOPIC#orders#LITERAL#User:sa-xyz123#*#DESCRIBE#ALLOW]
...
No changes. Infrastructure is up-to-date.
```

Check that the renaming was successful by running the following command:
```bash
grep "http_endpoint" main.tf

# Run this command only if you store TF state file locally
grep "http_endpoint" terraform.tfstate
```

Both commands should output 0 matches.

If you run into any problems,
please [report an issue](https://github.com/confluentinc/terraform-provider-confluent/issues).
