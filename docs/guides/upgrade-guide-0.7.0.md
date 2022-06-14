---
page_title: "Confluent Provider 0.7.0: Upgrade Guide"
---
# Confluent Provider 0.7.0: Upgrade Guide

This guide is intended to help with the upgrading process and focuses only on the changes necessary to upgrade to version `0.7.0` of Confluent Provider from version `0.5.0` of Confluent Cloud Provider.

-> **Note:** If you're currently using one of the earlier versions older than `0.5.0` please complete [Confluent Cloud Provider 0.5.0: Upgrade Guide](https://registry.terraform.io/providers/confluentinc/confluentcloud/latest/docs/guides/upgrade-guide-0.5.0) before starting this one.

!> **Warning:** Don't forget to create a backup of the `terraform.tfstate` state file before upgrading.

## Upgrade Notes

- [Provider Version Configuration](#provider-version-configuration)
- [Upgrade Terraform Configuration](#upgrade-terraform-configuration)
- [Upgrade State File](#upgrade-state-file)

## Provider Version Configuration

-> **Note:** This guide uses the Terraform configuration from the [Sample Project](https://registry.terraform.io/providers/confluentinc/confluentcloud/latest/docs/guides/sample-project) as an example of a Terraform configuration that has a cluster and 2 ACLs.

Before upgrading to version `0.7.0`, ensure that your environment
successfully runs [`terraform plan`](https://www.terraform.io/docs/commands/plan.html)
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

The next step is to replace Confluent Cloud Provider (`confluentinc/confluentcloud`) with Confluent Provider (`confluentinc/confluent`) and to set the latest `0.7.0` version in a `required_providers` block of your Terraform configuration.

#### Before
```hcl
terraform {
  required_providers {
   # ...
   confluentcloud = {
      source  = "confluentinc/confluentcloud"
      version = "0.5.0"
    }
  }
}

provider "confluentcloud" {
```

#### After
```hcl
terraform {
  required_providers {
   # ...
   confluent = {
      source  = "confluentinc/confluent"
      version = "0.7.0"
    }
  }
}

provider "confluent" {
```

## Upgrade Terraform Configuration

### Changes to `confluentcloud_kafka_acl` resource
* `host` attribute is now required. Update your configuration to set it to `"*"`.

  #### Before
    ```hcl
    resource "confluentcloud_kafka_acl" "describe-orders" {
      # ...
    }
    ```

  #### After
    ```hcl
    resource "confluentcloud_kafka_acl" "describe-orders" {
      # ...
      host = "*"
    }
    ```

* `kafka_cluster` attribute type was changed from _string_ to a _configuration block_. Update your configuration accordingly.

  #### Before
    ```hcl
    resource "confluentcloud_kafka_acl" "describe-orders" {
      kafka_cluster = "lkc-abc123"
    }
    ```

  #### After
    ```hcl
    resource "confluentcloud_kafka_acl" "describe-orders" {
      kafka_cluster {
        id = "lkc-abc123"
      }
    }
    ```

### Changes to `confluentcloud_kafka_topic` resource
* `kafka_cluster` attribute type was changed from _string_ to a _configuration block_. Update your configuration accordingly.

  #### Before
    ```hcl
    resource "confluentcloud_kafka_topic" "orders" {
      kafka_cluster = "lkc-abc123"
    }
    ```

  #### After
    ```hcl
    resource "confluentcloud_kafka_topic" "orders" {
      kafka_cluster {
        id = "lkc-abc123"
      }
    }
    ```

### Changes to all resources and data sources
All resources and data sources have been renamed in the new [Confluent Terraform Provider](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs). The prefix has changed from `confluentcloud` to `confluent`. For example, `confluentcloud_environment` resource was updated to `confluent_environment`. Therefore, run the following commands to update your TF configuration file, for example, called `main.tf`.
```bash
# In-place rename resources confluentcloud_environment -> confluent_environment etc in main.tf
sed -i '' 's/confluentcloud_environment/confluent_environment/' main.tf
sed -i '' 's/confluentcloud_kafka_acl/confluent_kafka_acl/' main.tf
sed -i '' 's/confluentcloud_kafka_cluster/confluent_kafka_cluster/' main.tf
sed -i '' 's/confluentcloud_kafka_topic/confluent_kafka_topic/' main.tf
sed -i '' 's/confluentcloud_role_binding/confluent_role_binding/' main.tf
sed -i '' 's/confluentcloud_service_account/confluent_service_account/' main.tf
```

Check that the replacement was successful by running the following command:
```bash
grep "confluentcloud" main.tf
```

The command should output 0 matches.

## Upgrade State File
Similarly, you need to rename all resources and data sources in your TF state file, for example, called `terraform.tfstate`. You can do it by running the following commands:
```bash
# Alternatively you could run
# terraform state replace-provider "registry.terraform.io/confluentinc/confluentcloud" "registry.terraform.io/confluentinc/confluent"
# Replaces confluentinc/confluentcloud with confluentinc/confluent TF Provider in terraform.tfstate
terraform state replace-provider confluentinc/confluentcloud confluentinc/confluent

# In-place rename resources confluentcloud_environment -> confluent_environment etc in confluent.state
sed -i '' 's/confluentcloud_environment/confluent_environment/' terraform.tfstate
sed -i '' 's/confluentcloud_kafka_acl/confluent_kafka_acl/' terraform.tfstate
sed -i '' 's/confluentcloud_kafka_cluster/confluent_kafka_cluster/' terraform.tfstate
sed -i '' 's/confluentcloud_kafka_topic/confluent_kafka_topic/' terraform.tfstate
sed -i '' 's/confluentcloud_role_binding/confluent_role_binding/' terraform.tfstate
sed -i '' 's/confluentcloud_service_account/confluent_service_account/' terraform.tfstate

# Find, download, and install new Confluent Provider (confluentinc/confluent) locally
terraform init
```

Check that the replacement was successful by running the following command:
```bash
grep "confluentcloud" terraform.tfstate
```

The command should output 0 matches.

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
confluent_kafka_cluster.test-basic-cluster: Refreshing state... [id=lkc-abc123]
confluent_kafka_acl.describe-test-basic-cluster: Refreshing state... [id=lkc-abc123/CLUSTER#kafka-cluster#LITERAL#User:sa-xyz123#*#DESCRIBE#ALLOW]
confluent_kafka_topic.orders: Refreshing state... [id=lkc-abc123/orders]
confluent_kafka_acl.describe-orders: Refreshing state... [id=lkc-abc123/TOPIC#orders#LITERAL#User:sa-xyz123#*#DESCRIBE#ALLOW]
...
No changes. Infrastructure is up-to-date.
```

If you run into any problems, please [report an issue](https://github.com/confluentinc/terraform-provider-confluent/issues).
