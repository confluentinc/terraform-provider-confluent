---
page_title: "Resource Importer"
---

# Resource Importer for Confluent Terraform Provider

[![General Availability](https://img.shields.io/badge/Lifecycle%20Stage-General%20Availability-%2345c6e8)](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy)

## Summary

[![asciicast](https://asciinema.org/a/574292.svg)](https://asciinema.org/a/574292)

-> **Note:** Running _Resource Importer for Confluent Terraform Provider_ is a read-only operation. It will not edit Confluent Cloud infrastructure.
For additional safety, the _Resource Importer for Confluent Terraform Provider_ adds `lifecycle { prevent_destroy = true }` for every imported instance to prevent accidental instance deletion. This setting rejects plans that would destroy or recreate the instance, such as attempting to change uneditable attributes. For more information, see the [Terraform docs](https://www.terraform.io/language/meta-arguments/lifecycle#prevent_destroy).

-> **Note:** _Resource Importer for Confluent Terraform Provider_ is particularly useful when importing a lot of existing Confluent Cloud resources to Terraform when running `terraform import` for every resource is tedious.
For example, it's useful if you want to migrate to TF Provider for Confluent from other TF Providers for managing Confluent Cloud resources.
It's also super convenient if you have been using the Confluent Cloud Console or Confluent CLI to manage your Confluent Cloud resources but now need to import 1000s of ACLs and 100s of Service Accounts into Terraform.

_Resource Importer for Confluent Terraform Provider_ enables importing your existing Confluent Cloud resources to Terraform Configuration (`main.tf`) and Terraform State (`terraform.tfstate`) files to a local directory named `imported_confluent_infrastructure`.

These are the importable resources:
   * [Service Accounts](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_service_account)
   * [Environments](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_environment)
   * [Connectors](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_connector)
   * [Kafka Clusters](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_cluster)
   * [Access Control Lists (ACLs)](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_acl)
   * [Topics](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_topic)
   * [Schemas](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema)

-> **Note:** [File an issue](https://github.com/confluentinc/terraform-provider-confluent/issues) to request a support for other resources.

In this guide, you will:

1. [Create a Cloud API Key](#create-a-cloud-api-key)
   * Create a [service account](https://docs.confluent.io/cloud/current/access-management/identity/service-accounts.html) named `tf_runner` in Confluent Cloud
   * Assign the `OrganizationAdmin` [role](https://docs.confluent.io/cloud/current/access-management/access-control/cloud-rbac.html#organizationadmin) to the `tf_runner` service account
   * Create a [Cloud API Key](https://docs.confluent.io/cloud/current/access-management/authenticate/api-keys/api-keys.html#cloud-cloud-api-keys) for the `tf_runner` service account
2. [Import Confluent Cloud Resources via Terraform](#import-confluent-cloud-resources-via-terraform)
   * Select the appropriate Terraform configuration from a list of example configurations
   * Initialize and apply the selected Terraform configuration

## Prerequisites

1.  A Confluent Cloud account. If you don't have a Confluent Cloud account, [create one now](https://www.confluent.io/confluent-cloud/tryfree/). 
2.  Terraform (0.14+) installed:
    * Install Terraform version manager [tfutils/tfenv](https://github.com/tfutils/tfenv)
    * Alternatively, install the [Terraform CLI](https://learn.hashicorp.com/tutorials/terraform/install-cli#install-terraform)
    * Run the following command to ensure that you're using a compatible version of Terraform.

        ```bash
        terraform version
        ```
    
        Your output should resemble:
        ```bash
        Terraform v0.14.0 # any version >= v0.14.0 is OK
        ...
        ```

## Create a Cloud API Key

1. Open the [Confluent Cloud Console](https://confluent.cloud/settings/api-keys/create) and click the **Granular access** tab, and then click **Next**.
2. Click **Create a new one to create** tab. Enter the new service account name (`tf_runner`), then click **Next**.
3. The Cloud API key and secret are generated for the `tf_runner` service account. Save your Cloud API key and secret in a secure location. You will need this API key and secret **to use the Confluent Terraform Provider**.
4. [Assign](https://confluent.cloud/settings/org/assignments) the `OrganizationAdmin` role to the `tf_runner` service account by following [this guide](https://docs.confluent.io/cloud/current/access-management/access-control/cloud-rbac.html#add-a-role-binding-for-a-user-or-service-account).

![Assigning the OrganizationAdmin role to tf_runner service account](https://github.com/confluentinc/terraform-provider-confluent/raw/master/docs/images/OrganizationAdmin.png)

## Import Confluent Cloud Resources via Terraform

1. Clone the [repository](https://github.com/confluentinc/terraform-provider-confluent) containing the example configurations:

    ```bash
    git clone https://github.com/confluentinc/terraform-provider-confluent.git
    ```

2. Navigate into the `configurations` subdirectory:

    ```bash
    cd terraform-provider-confluent/examples/configurations
    ```

3. The `configurations` directory has a subdirectory for each of the following configurations:
    * [`cloud-importer`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/cloud-importer): Import _Cloud_ resources (for example, Service Accounts, Environments)
    * [`kafka-importer`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/kafka-importer): Import _Kafka_ resources (for example, ACLs, Topics)
    * [`schema-registry-importer`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/schema-registry-importer): Import _Schema Registry_ resources (for example, Schemas)

4. Select the target configuration and navigate into its directory:
    ```bash
    # Using the example configuration #1 as an example 
    cd cloud-importer
    ```

5. Download and install the providers defined in the configuration:
    ```bash
    terraform init
    ```

6. Use the saved Cloud API Key of the `tf_runner` service account to set values to the `confluent_cloud_api_key` and `confluent_cloud_api_secret` input variables [using environment variables](https://www.terraform.io/language/values/variables#environment-variables):
    ```bash
    export TF_VAR_confluent_cloud_api_key="<cloud_api_key>"
    export TF_VAR_confluent_cloud_api_secret="<cloud_api_secret>"
    ```

7. Ensure the configuration is syntactically valid and internally consistent:
    ```bash
    terraform validate
    ```
   
8. Apply the configuration:
    ```bash
    terraform apply
    ```

    -> **Note:**  If the import process is taking longer than expected, you can improve the speed by increasing the parallelism flag. For example, you can set it to 100 like this: `terraform apply -parallelism=100`. Increasing parallelism can help speed up the import process, especially when dealing with a large number of resources.

9. You have now imported Confluent Cloud infrastructure using Terraform under `cloud-importer/imported_confluent_infrastructure`! The `terraform apply` command created a new folder called `cloud-importer/imported_confluent_infrastructure` that contains 2 files: Terraform Configuration (`main.tf`) and Terraform State (`terraform.tfstate`) files.

10. Navigate into its directory:
    ```bash
    # Using the example configuration #1 as an example 
    cd imported_confluent_infrastructure
    ```

11. Download and install the providers defined in the configuration:
    ```bash
    terraform init
    ```
    Your output should resemble:
    ```bash
    ...
    Terraform has been successfully initialized!

    You may now begin working with Terraform. Try running "terraform plan" to see
    any changes that are required for your infrastructure. All Terraform commands
    should now work.
    
    If you ever set or change modules or backend configuration for Terraform,
    rerun this command to reinitialize your working directory. If you forget, other
    commands will detect it and remind you to do so if necessary.
    ```

12. Refresh the configuration:
    ```bash
    terraform refresh
    ```
    Your output should resemble:
    ```bash
    ...
    confluent_service_account.test_account: Refreshing state... [id=sa-oz5q19]
    ...
    ```

13. Ensure that import is successful by running `terraform plan`:
    ```bash
    terraform plan
    ```
    Your output should resemble:
    ```bash
    ...
    confluent_service_account.test_account: Refreshing state... [id=sa-oz5q19]

    No changes. Your infrastructure matches the configuration.

    Terraform has compared your real infrastructure against your configuration and found no differences, so no changes are needed.
    ```

    -> **Note:**  If the import process is taking longer than expected, you can improve the speed by increasing the parallelism flag. For example, you can set it to 100 like this: `terraform plan -parallelism=100`. Increasing parallelism can help speed up the import process, especially when dealing with a large number of resources.

You've successfully imported your Confluent Cloud infrastructure to Terraform. Terraform also compared your real infrastructure against your imported configuration and found no differences, so no changes are needed.
