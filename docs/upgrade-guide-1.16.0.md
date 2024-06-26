---
page_title: "Confluent Provider 1.16.0: Upgrade Guide"
---
# Confluent Provider 1.16.0: Upgrade Guide

This guide is intended to help with the upgrading process and focuses only on the changes necessary to upgrade to version `1.16.0` of Confluent Provider from version `1.15.0` of Confluent Cloud Provider.

-> **Note:** If you're upgrading from a version that's earlier than `1.15.0`, upgrade to
version `1.15.0` before starting this guide.

!> **Warning:** Don't forget to create a backup of the `terraform.tfstate` state file before upgrading.

## Upgrade Notes

- [Provider Version Configuration](#provider-version-configuration)
- [Upgrade Terraform Configuration](#upgrade-terraform-configuration)

## Provider Version Configuration

-> **Note:** This guide uses [basic-kafka-acls](https://github.com/confluentinc/terraform-provider-confluent/tree/v1.15.0/examples/configurations/basic-kafka-acls) Terraform configuration as an example of a Terraform configuration that has a Kafka cluster and a Schema Registry cluster.

Before upgrading to version `1.16.0`, ensure that your environment
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
confluent_stream_governance_cluster.essentials: Refreshing state... [id=lsrc-abc123]
confluent_kafka_acl.describe-test-basic-cluster: Refreshing state... [id=lkc-abc123/CLUSTER#kafka-cluster#LITERAL#User:12345#*#DESCRIBE#ALLOW]
confluent_kafka_topic.orders: Refreshing state... [id=lkc-abc123/orders]
confluent_kafka_acl.describe-orders: Refreshing state... [id=lkc-n2kvd/TOPIC#orders#LITERAL#User:12345#*#DESCRIBE#ALLOW]
...
No changes. Infrastructure is up-to-date.
```

The next step is to set the latest `1.16.0` version in a `required_providers` block of your Terraform configuration.

#### Before

```hcl
terraform {
  required_providers {
    # ...
    confluent = {
      source  = "confluentinc/confluent"
      version = "1.15.0"
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
      version = "1.16.0"
    }
  }
}

provider "confluent" {
```

Run
```bash
terraform init -upgrade
```

to upgrade to `1.16.0` version of TF Provider for Confluent.

Your output should resemble:
```
...
Initializing provider plugins...
- Finding confluentinc/confluent versions matching "1.16.0"...
- Installing confluentinc/confluent v1.16.0...
...
Terraform has been successfully initialized!

You may now begin working with Terraform. Try running "terraform plan" to see
any changes that are required for your infrastructure. All Terraform commands
should now work.

If you ever set or change modules or backend configuration for Terraform,
rerun this command to reinitialize your working directory. If you forget, other
commands will detect it and remind you to do so if necessary.
```

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
confluent_stream_governance_cluster.essentials: Refreshing state... [id=lsrc-abc123]
...

No changes. Your infrastructure matches the configuration.

Terraform has compared your real infrastructure against your configuration and found no differences, so no changes are needed.
╷
│ Warning: Deprecated Resource
│ 
│   with data.confluent_stream_governance_region.essentials,
│   on main.tf line 21, in data "confluent_stream_governance_region" "essentials":
│   21: data "confluent_stream_governance_region" "essentials" {
│ 
│ confluent_stream_governance_region data source is deprecated and will be removed in the next version. Use confluent_schema_registry_region instead.
│ 
│ (and 3 more similar warnings elsewhere)
```

## Upgrade Terraform Configuration

### Changes to `confluent_stream_governance` resource

#### Before
    ```hcl
    resource "confluent_stream_governance_cluster" "essentials" {
      # ...
    }
    ```

#### After
    ```hcl
    resource "confluent_stream_governance_cluster" "essentials" {
      # ...
    }
    
    # Copy definition and rename resource name from
    # confluent_stream_governance_cluster to
    # confluent_schema_registry_cluster
    resource "confluent_schema_registry_cluster" "essentials" {
      # ...
    }
    ```

The next step is to import a Schema Registry cluster.

You must have a Schema Registry Cluster ID.

The easiest way to find a Schema Registry Cluster ID is to examine the output from the `terraform plan` command that you ran earlier:
```
...
confluent_stream_governance_cluster.essentials: Refreshing state... [id=lsrc-abc123]
...
...
No changes. Infrastructure is up-to-date.
```

Save your SR Cluster ID: `lsrc-abc123`. You will need it when you import a Schema Registry cluster.

You can also use the [Confluent CLI](https://docs.confluent.io/confluent-cli/current/migrate.html#directly-install-confluent-cli-v2-x) to get the Schema Registry Cluster ID:

```bash
# Run confluent environment list to list environments if necessary
confluent environment use env-aap2wg # replace with your environment ID
confluent schema-registry cluster describe
```

Your output should resemble:
```
+-------------------------+--------------------------------------------------+
| Name                    | Stream Governance Package                        |
| Cluster ID              | lsrc-abc123                                      |
...
```

Also, you can create a new one-off TF workspace and use a data source `confluent_stream_governance_cluster` to find a Schema Registry cluster ID:
```hcl
data "confluent_stream_governance_cluster" "example_using_name" {
  display_name = "Stream Governance Package"
  environment {
    id = "env-aap2wg" # replace with your environment ID
  }
}

output "example_using_name" {
  value = data.confluent_stream_governance_cluster.example_using_name
}
```

The next step is to import a Schema Registry cluster. 
```bash
$ export CONFLUENT_CLOUD_API_KEY="<cloud_api_key>"
$ export CONFLUENT_CLOUD_API_SECRET="<cloud_api_secret>"
$ terraform import confluent_schema_registry_cluster.essentials env-aap2wg/lsrc-abc123
```

Your output should resemble:
```
confluent_schema_registry_cluster.essentials: Importing from ID "env-aap2wg/lsrc-abc123"...
confluent_schema_registry_cluster.essentials: Import prepared!
  Prepared confluent_schema_registry_cluster for import
confluent_schema_registry_cluster.essentials: Refreshing state... [id=lsrc-abc123]

Import successful!

The resources that were imported are shown above. These resources are now in
your Terraform state and will henceforth be managed by Terraform.
```

The last step is to remove `confluent_stream_governance_cluster.essentials` from both TF configuration and TF state.

To remove `confluent_stream_governance_cluster.essentials` from TF state, run the following command:
```bash
terraform state rm confluent_stream_governance_cluster.essentials
```

Your output should resemble:
```
Removed confluent_stream_governance_cluster.essentials
Successfully removed 1 resource instance(s).
```

To remove `confluent_stream_governance_cluster.essentials` from TF configuration, you can just remove its definition:

#### Before
    ```hcl
    resource "confluent_stream_governance_cluster" "essentials" {
      # ...
    }
    ```

#### After
    ```hcl
    # empty
    ```

### Changes to `confluent_stream_governance_cluster` data source

#### Before
    ```hcl
    data "confluent_stream_governance_cluster" "essentials" {
      # ...
    }
    ```

#### After
    ```hcl
    data "confluent_schema_registry_cluster" "essentials" {
      # ...
    }
    ```

### Changes to `confluent_stream_governance_region` data source

  #### Before
    ```hcl
    data "confluent_stream_governance_region" "essentials" {
      # ...
    }
    ```

  #### After
    ```hcl
    data "confluent_schema_registry_region" "essentials" {
      # ...
    }
    ```

Check that the replacement was successful by running the following command:
```bash
grep "_stream_governance_" main.tf
```

The command should output 0 matches.

If you see matches, make sure you replaced all references:

* `confluent_stream_governance_region` -> `confluent_schema_registry_region`
* `confluent_stream_governance_cluster` -> `confluent_schema_registry_cluster`

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

If you run into any problems, [report an issue](https://github.com/confluentinc/terraform-provider-confluent/issues) to Confluent.
