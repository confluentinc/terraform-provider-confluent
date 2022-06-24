---
page_title: "Confluent Provider 0.11.0: Upgrade Guide"
---
# Confluent Provider 0.11.0: Upgrade Guide

This guide is intended to help with the upgrading process and focuses only on the changes necessary to upgrade to
version `0.11.0` of Confluent Provider from version `0.10.0` of Confluent Provider.

-> **Note:** If you're upgrading from a version that's earlier than `0.10.0`, upgrade to
version `0.10.0` before starting this guide. For more information, see
[Confluent Provider 0.7.0: Upgrade Guide](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/upgrade-guide-0.7.0).

!> **Warning:** Don't forget to create backups of the `terraform.tfstate` state file and your TF configuration (for
example, `main.tf`) before upgrading.

## Upgrade Notes

- [Provider Version Configuration](#provider-version-configuration)
- [Upgrade Terraform Configuration](#upgrade-terraform-configuration)
- [Upgrade State File Using sed Command](#upgrade-state-file-using-sed-command)
- [Upgrade State File Using import Command](#upgrade-state-file-using-import-command)

## Provider Version Configuration

-> **Note:** This guide uses [basic-kafka-acls](https://github.com/confluentinc/terraform-provider-confluent/tree/v0.10.0/examples/configurations/basic-kafka-acls) Terraform configuration as an example of a Terraform configuration that has a Kafka cluster and multiple ACLs.

Before upgrading to version `0.11.0`, ensure that your environment successfully
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

The next step is to set the latest `0.11.0` version in a `required_providers` block of your Terraform configuration.

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
      version = "0.11.0"
    }
  }
}

provider "confluent" {
```

## Upgrade Terraform Configuration

* `api_key`, `api_secret` attributes were renamed to `cloud_api_key`, `cloud_api_secret`, respectively. Update your TF
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

### Changes to `confluentcloud_kafka_acl` resource

* `http_endpoint` attribute was renamed to `rest_endpoint`. Update your TF configuration accordingly.

  #### Before
    ```hcl
    resource "confluentcloud_kafka_acl" "describe-orders" {
      http_endpoint = "lkc-abc123"
    }
    ```

  #### After
    ```hcl
    resource "confluentcloud_kafka_acl" "describe-orders" {
      rest_endpoint = "lkc-abc123"
    }
    ```

### Changes to `confluentcloud_kafka_topic` resource

* `http_endpoint` attribute was renamed to `rest_endpoint`. Update your TF configuration accordingly.

  #### Before
    ```hcl
    resource "confluentcloud_kafka_topic" "orders" {
      http_endpoint = "lkc-abc123"
    }
    ```

  #### After
    ```hcl
    resource "confluentcloud_kafka_topic" "orders" {
      rest_endpoint = "lkc-abc123"
    }
    ```

### Changes to `confluentcloud_kafka_cluster` resource

* `http_endpoint` attribute was renamed to `rest_endpoint`. No changes in your TF configuration are necessary since the
  attribute is [computed](https://www.terraform.io/plugin/sdkv2/schemas/schema-behaviors#computed).

You might find it helpful to run

```bash
# Replaces http_endpoint with rest_endpoint in main.tf
sed -i '' 's/http_endpoint/rest_endpoint/g' main.tf
```

instead of updating your TF configuration file (for example, called `main.tf`) manually.

### Changes to all resources and data sources

All resources and data sources have been renamed to include corresponding API versions. For
example, `confluent_environment` resource was renamed to `confluent_environment_v2` (since the
corresponding [API version](https://docs.confluent.io/cloud/current/api.html#operation/getOrgV2Environment) is `org/v2`)
.

### Resources

| Old Name                                               | New Name                                           |
|--------------------------------------------------------|----------------------------------------------------|
| confluent_api_key                                      | confluent_api_key_v2                               |
| confluent_connector                                    | confluent_connector_v1                             |
| confluent_environment                                  | confluent_environment_v2                           |
| confluent_kafka_acl                                    | confluent_kafka_acl_v3                             |
| confluent_kafka_cluster                                | confluent_kafka_cluster_v2                         |
| confluent_kafka_topic                                  | confluent_kafka_topic_v3                           |
| confluent_network                                      | confluent_network_v1                               |
| confluent_peering                                      | confluent_peering_v1                               |
| confluent_private_link_access                          | confluent_private_link_access_v1                   |
| confluent_role_binding                                 | confluent_role_binding_v2                          |
| confluent_service_account                              | confluent_service_account_v2                       |

### Data Sources

| Old Name                      | New Name                         |
|-------------------------------|----------------------------------|
| confluent_environment         | confluent_environment_v2         |
| confluent_kafka_cluster       | confluent_kafka_cluster_v2       |
| confluent_kafka_topic         | confluent_kafka_topic_v3         |
| confluent_network             | confluent_network_v1             |
| confluent_organization        | confluent_organization_v2        |
| confluent_peering             | confluent_peering_v1             |
| confluent_private_link_access | confluent_private_link_access_v1 |
| confluent_role_binding        | confluent_role_binding_v2        |
| confluent_service_account     | confluent_service_account_v2     |
| confluent_user                | confluent_user_v2                |

Therefore, run the following commands to update your TF configuration file.

```bash
# Replaces confluent_environment with confluent_environment_v2 (and others in a similar fashion) in main.tf
sed -i '' 's/confluent_api_key/confluent_api_key_v2/g' main.tf
sed -i '' 's/confluent_connector/confluent_connector_v1/g' main.tf
sed -i '' 's/confluent_environment/confluent_environment_v2/g' main.tf
sed -i '' 's/confluent_kafka_acl/confluent_kafka_acl_v3/g' main.tf
sed -i '' 's/confluent_kafka_cluster/confluent_kafka_cluster_v2/g' main.tf
sed -i '' 's/confluent_kafka_topic/confluent_kafka_topic_v3/g' main.tf
sed -i '' 's/confluent_network/confluent_network_v1/g' main.tf
sed -i '' 's/confluent_organization/confluent_organization_v2/g' main.tf
sed -i '' 's/confluent_peering/confluent_peering_v1/g' main.tf
sed -i '' 's/confluent_private_link_access/confluent_private_link_access_v1/g' main.tf
sed -i '' 's/confluent_role_binding/confluent_role_binding_v2/g' main.tf
sed -i '' 's/confluent_service_account/confluent_service_account_v2/g' main.tf
sed -i '' 's/confluent_user/confluent_user_v2/g' main.tf
```

## Upgrade State File Using sed Command

-> **Note:** If you have a remote backend configured, skip this section and continue reading at [Upgrade State File Using import Command](#upgrade-state-file-using-import-command).

Similarly, you need to rename

* `http_endpoint` attribute to `rest_endpoint`
* all resources and data sources

in your TF state file. You can do it by running the following commands:

```bash
# Replaces http_endpoint with rest_endpoint in main.tf
sed -i '' 's/http_endpoint/rest_endpoint/g' terraform.tfstate

# Replaces confluent_environment with confluent_environment_v2 (and others in a similar fashion) in terraform.tfstate
sed -i '' 's/confluent_api_key/confluent_api_key_v2/g' terraform.tfstate
sed -i '' 's/confluent_connector/confluent_connector_v1/g' terraform.tfstate
sed -i '' 's/confluent_environment/confluent_environment_v2/g' terraform.tfstate
sed -i '' 's/confluent_kafka_acl/confluent_kafka_acl_v3/g' terraform.tfstate
sed -i '' 's/confluent_kafka_cluster/confluent_kafka_cluster_v2/g' terraform.tfstate
sed -i '' 's/confluent_kafka_topic/confluent_kafka_topic_v3/g' terraform.tfstate
sed -i '' 's/confluent_network/confluent_network_v1/g' terraform.tfstate
sed -i '' 's/confluent_peering/confluent_peering_v1/g' terraform.tfstate
sed -i '' 's/confluent_private_link_access/confluent_private_link_access_v1/g' terraform.tfstate
sed -i '' 's/confluent_role_binding/confluent_role_binding_v2/g' terraform.tfstate
sed -i '' 's/confluent_service_account/confluent_service_account_v2/g' terraform.tfstate
sed -i '' 's/confluent_organization/confluent_organization_v2/g' terraform.tfstate
sed -i '' 's/confluent_user/confluent_user_v2/g' terraform.tfstate
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
confluent_service_account_v2.test-sa: Refreshing state... [id=sa-xyz123]
confluent_environment_v2.test-env: Refreshing state... [id=env-dge456]
confluent_kafka_cluster_v2.test-basic-cluster: Refreshing state... [id=lkc-abc123]
confluent_kafka_acl_v3.describe-test-basic-cluster: Refreshing state... [id=lkc-abc123/CLUSTER#kafka-cluster#LITERAL#User:sa-xyz123#*#DESCRIBE#ALLOW]
confluent_kafka_topic_v3.orders: Refreshing state... [id=lkc-abc123/orders]
confluent_kafka_acl_v3.describe-orders: Refreshing state... [id=lkc-abc123/TOPIC#orders#LITERAL#User:sa-xyz123#*#DESCRIBE#ALLOW]
...
No changes. Infrastructure is up-to-date.
```

## Upgrade State File Using import Command

Another way to upgrade the Terraform state file is to reimport resources manually, which is especially useful if the remote backend is configured.

Run the `terraform state list` command to display the full list of managed resources to import:

```bash
$ terraform state list
confluent_service_account.test-sa
confluent_environment.test-env
...
```

To migrate from the old resource to the new resource, remove the old resource from the state, and then use Terraform's `import` [command](https://www.terraform.io/cli/import#import) command to migrate to the new resource.

To import a resource in Terraform, you need its Resource ID, which you can get by running the following command.
```bash
$ echo confluent_service_account.test-sa.id | terraform console 
"sa-xyz123"
```

Now you can remove the existing resource:

```bash
$ terraform state rm confluent_service_account.test-sa
Removed confluent_service_account.test-sa
Successfully removed 1 resource instance(s).
```

With the old resource removed from Terraform's state file, run the following command to import it into the Terraform state file as the new resource.

```bash
$ terraform import confluent_service_account_v2.test-sa "sa-xyz123"
confluent_service_account_v2.test-sa: Importing from ID "sa-xyz123"...
confluent_service_account_v2.test-sa: Import prepared!
  Prepared confluent_service_account_v2 for import
confluent_service_account_v2.test-sa: Refreshing state... [id=sa-xyz123]

Import successful!

The resources that were imported are shown above. These resources are now in
your Terraform state and will henceforth be managed by Terraform.
```

-> **Note:** Some resources might require you to set environment variables before running `terraform import`, which is why it might be useful to check out _Import_ [section](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_acl_v3#import) on a corresponding doc page.

Once this has been done for all the managed resources, the `terraform plan` command should show no changes:

```bash
terraform plan                                                       
...
confluent_service_account_v2.test-sa: Refreshing state... [id=sa-xyz123]

No changes. Infrastructure is up-to-date.

This means that Terraform did not detect any differences between your
configuration and real physical resources that exist. As a result, no
actions need to be performed.
```

If you run into any problems,
please [report an issue](https://github.com/confluentinc/terraform-provider-confluent/issues).
