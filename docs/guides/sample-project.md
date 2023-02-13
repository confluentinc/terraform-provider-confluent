---
page_title: "Sample Project"
---

# Sample Project for Confluent Terraform Provider

## Summary

[![asciicast](https://asciinema.org/a/559625.svg)](https://asciinema.org/a/559625)

Use the Confluent Terraform provider to enable the lifecycle management of Confluent Cloud resources:
   * [Environments](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_environment)
   * [Kafka Clusters](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_cluster)
   * [Topics](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_topic)
   * [Connectors](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_connector)
   * [Networks](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_network)
   * [Private Link Accesses](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_private_link_access)
   * [Peerings](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_peering)
   * [Transit Gateway Attachments](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_transit_gateway_attachment)
   * [Service Accounts](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_service_account)
   * [Cloud API Keys and Kafka API Keys](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_api_key)
   * [Access Control Lists (ACLs)](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_acl)
   * [Role Bindings](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_role_binding)
   * [Cluster Links](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_cluster_link)
   * [Identity Providers](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_identity_provider)
   * [Identity Pool](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_identity_pool)
   * [Kafka Client Quotas](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_client_quota)
   * [Kafka Cluster Configs](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_cluster_config)
   * [Kafka Mirror Topics](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_mirror_topic)
   * [Schema Registry Clusters](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema_registry_cluster)
   * [ksqlDB Clusters](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_ksql_cluster)
   * [Schemas](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema)

In this guide, you will:

1. [Create a Cloud API Key](#create-a-cloud-api-key)
   * Create a [service account](https://docs.confluent.io/cloud/current/access-management/identity/service-accounts.html) called `tf_runner`in Confluent Cloud
   * Assign the `OrganizationAdmin` [role](https://docs.confluent.io/cloud/current/access-management/access-control/cloud-rbac.html#organizationadmin) to the `tf_runner` service account
   * Create a [Cloud API Key](https://docs.confluent.io/cloud/current/access-management/authenticate/api-keys/api-keys.html#cloud-cloud-api-keys) for the `tf_runner` service account
2. [Create Resources on Confluent Cloud via Terraform](#create-resources-on-confluent-cloud-via-terraform)
   * Select the appropriate Terraform configuration from a list of example configurations that describe the following infrastructure setup:
      * An [environment](https://docs.confluent.io/cloud/current/access-management/hierarchy/cloud-environments.html) named `Staging` that contains a [Kafka cluster](https://docs.confluent.io/cloud/current/clusters/index.html) called `inventory` with a [Kafka topic](https://docs.confluent.io/cloud/current/client-apps/topics/manage.html) named `orders`
      * 3 service accounts: `app_manager`, `app_producer`, and `app_consumer` with an associated [Kafka API Key](https://docs.confluent.io/cloud/current/access-management/authenticate/api-keys/api-keys.html#resource-specific-api-keys) each
      * Appropriate permissions for service accounts granted either using [Role-based Access Control (RBAC)](https://docs.confluent.io/cloud/current/access-management/access-control/cloud-rbac.html) or [ACLs](https://docs.confluent.io/cloud/current/access-management/access-control/acl.html):
         * the `app_manager` service account's Kafka API Key is used for creating the `orders` topic on the `inventory` Kafka cluster and creating ACLs if needed.
         * the `app_producer` service account's Kafka API Key is used for _producing_ messages to the `orders` topic
         * the `app_consumer` service account's Kafka API Key is used for _consuming_ messages from the `orders` topic
         
         -> **Note:** API Keys inherit the permissions granted to the owner.

   * Initialize and apply the selected Terraform configuration
3. [[Optional] Run a quick test](#optional-run-a-quick-test)
   * Use the [Confluent CLI v2](https://docs.confluent.io/confluent-cli/current/migrate.html#directly-install-confluent-cli-v2-x) to 
      * Produce messages to the `orders` topic using the `app_producer` service account's Kafka API Key
      * Consume messages from the `orders` topic using the `app_consumer` service account's Kafka API Key
4. [[Optional] Destroy created resources on Confluent Cloud](#optional-teardown-confluent-cloud-resources)

## Prerequisites

1.  A Confluent Cloud account. If you do not have a Confluent Cloud account, [create one now](https://www.confluent.io/confluent-cloud/tryfree/). 
2.  Terraform (0.14+) installed:
    * Install Terraform version manager [tfutils/tfenv](https://github.com/tfutils/tfenv)
    * Alternatively, install the [Terraform CLI](https://learn.hashicorp.com/tutorials/terraform/install-cli#install-terraform)
    * To ensure you're using the acceptable version of Terraform you may run the following command:

        ```bash
        terraform version
        ```
    
        Your output should resemble:
        ```bash
        Terraform v0.14.0 # any version >= v0.14.0 is OK
        ...
        ```

## Create a Cloud API Key

1. Open the [Confluent Cloud Console](https://confluent.cloud/settings/api-keys/create) and click **Granular access** tab, and then click **Next**.
2. Click **Create a new one to create** tab. Enter the new service account name (`tf_runner`), then click **Next**.
3. The Cloud API key and secret are generated for the `tf_runner` service account. Save your Cloud API key and secret in a secure location. You will need this API key and secret **to use the Confluent Terraform Provider**.
4. [Assign](https://confluent.cloud/settings/org/assignments) the `OrganizationAdmin` role to the `tf_runner` service account by following [this guide](https://docs.confluent.io/cloud/current/access-management/access-control/cloud-rbac.html#add-a-role-binding-for-a-user-or-service-account).

![Assigning the OrganizationAdmin role to tf_runner service account](https://github.com/confluentinc/terraform-provider-confluent/raw/master/docs/images/OrganizationAdmin.png)

## Create Resources on Confluent Cloud via Terraform

1. Clone the [repository](https://github.com/confluentinc/terraform-provider-confluent) containing the example configurations:

    ```bash
    git clone https://github.com/confluentinc/terraform-provider-confluent.git
    ```

2. Change into `configurations` subdirectory:

    ```bash
    cd terraform-provider-confluent/examples/configurations
    ```

3. The `configurations` directory has a subdirectory for each of the following configurations:
    * [`basic-kafka-acls`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/basic-kafka-acls): _Basic_ Kafka cluster with authorization using ACLs
    * [`basic-kafka-acls-with-alias`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/basic-kafka-acls-with-alias): _Basic_ Kafka cluster with authorization using ACLs
    * [`standard-kafka-acls`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/standard-kafka-acls): _Standard_ Kafka cluster with authorization using ACLs
    * [`standard-kafka-rbac`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/standard-kafka-rbac): _Standard_ Kafka cluster with authorization using RBAC
    * [`dedicated-public-kafka-acls`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-public-kafka-acls): _Dedicated_ Kafka cluster that is accessible over the public internet with authorization using ACLs
    * [`dedicated-public-kafka-rbac`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-public-kafka-rbac): _Dedicated_ Kafka cluster that is accessible over the public internet with authorization using RBAC
    * [`dedicated-privatelink-aws-kafka-acls`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-privatelink-aws-kafka-acls): _Dedicated_ Kafka cluster on AWS that is accessible via PrivateLink connections with authorization using ACLs
    * [`dedicated-privatelink-aws-kafka-rbac`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-privatelink-aws-kafka-rbac): _Dedicated_ Kafka cluster on AWS that is accessible via PrivateLink connections with authorization using RBAC
    * [`dedicated-privatelink-azure-kafka-rbac`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-privatelink-azure-kafka-rbac): _Dedicated_ Kafka cluster on Azure that is accessible via PrivateLink connections with authorization using RBAC
    * [`dedicated-privatelink-azure-kafka-acls`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-privatelink-azure-kafka-acls): _Dedicated_ Kafka cluster on Azure that is accessible via PrivateLink connections with authorization using ACLs
    * [`dedicated-private-service-connect-gcp-kafka-acls`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-private-service-connect-gcp-kafka-acls): _Dedicated_ Kafka cluster on GCP that is accessible via Private Service Connect connections with authorization using ACLs
    * [`dedicated-private-service-connect-gcp-kafka-rbac`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-private-service-connect-gcp-kafka-rbac): _Dedicated_ Kafka cluster on GCP that is accessible via Private Service Connect connections with authorization using RBAC
    * [`dedicated-vnet-peering-azure-kafka-acls`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-vnet-peering-azure-kafka-acls): _Dedicated_ Kafka cluster on Azure that is accessible via VPC Peering connections with authorization using ACLs
    * [`dedicated-vnet-peering-azure-kafka-rbac`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-vnet-peering-azure-kafka-rbac): _Dedicated_ Kafka cluster on Azure that is accessible via VPC Peering connections with authorization using RBAC
    * [`dedicated-vpc-peering-aws-kafka-acls`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-vpc-peering-aws-kafka-acls): _Dedicated_ Kafka cluster on AWS that is accessible via VPC Peering connections with authorization using ACLs
    * [`dedicated-vpc-peering-aws-kafka-rbac`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-vpc-peering-aws-kafka-rbac): _Dedicated_ Kafka cluster on AWS that is accessible via VPC Peering connections with authorization using RBAC
    * [`dedicated-vpc-peering-gcp-kafka-acls`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-vpc-peering-gcp-kafka-acls): _Dedicated_ Kafka cluster on GCP that is accessible via VPC Peering connections with authorization using ACLs
    * [`dedicated-vpc-peering-gcp-kafka-rbac`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-vpc-peering-gcp-kafka-rbac): _Dedicated_ Kafka cluster on GCP that is accessible via VPC Peering connections with authorization using RBAC
    * [`dedicated-transit-gateway-attachment-aws-kafka-acls`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-transit-gateway-attachment-aws-kafka-acls): _Dedicated_ Kafka cluster on AWS that is accessible via Transit Gateway Endpoint with authorization using ACLs
    * [`dedicated-transit-gateway-attachment-aws-kafka-rbac`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-transit-gateway-attachment-aws-kafka-rbac): _Dedicated_ Kafka cluster on AWS that is accessible via Transit Gateway Endpoint with authorization using RBAC

    -> **Note:** _Basic_ Kafka cluster with authorization using RBAC configuration is not supported, because both `DeveloperRead` and `DeveloperWrite` roles are not available for _Basic_ Kafka clusters.
    
    -> **Note:** When considering whether to use RBAC or ACLs for access control, it is suggested you use RBAC as the default because of its ease of use and manageability at scale, but for edge cases where you need to have more granular access control, or wish to explicitly deny access, ACLs may make more sense. For example, you could use RBAC to allow access for a group of users, but an ACL to deny access for a particular member of that group.

    -> **Note:** When using a private networking option, you must execute `terraform` on a system with connectivity to the Kafka REST API. Check the [Kafka REST API docs](https://docs.confluent.io/cloud/current/api.html#tag/Topic-(v3)) to learn more about it.

    -> **Note:** If you're interested in a more granular setup with TF configuration split between a Kafka Ops team and a Product team, see [kafka-ops-env-admin-product-team](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/kafka-ops-env-admin-product-team) and [kafka-ops-kafka-admin-product-team](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/kafka-ops-kafka-admin-product-team).

5. Select the target configuration and change into its directory:
    ```bash
    # Using the example configuration #1 as an example 
    cd basic-kafka-acls
    ```

6. Download and install the providers defined in the configuration:
    ```bash
    terraform init
    ```

7. Use the saved Cloud API Key of the `tf_runner` service account to set values to the `confluent_cloud_api_key` and `confluent_cloud_api_secret` input variables [using environment variables](https://www.terraform.io/language/values/variables#environment-variables):
    ```bash
    export TF_VAR_confluent_cloud_api_key="<cloud_api_key>"
    export TF_VAR_confluent_cloud_api_secret="<cloud_api_secret>"
    ```

8. Ensure the configuration is syntactically valid and internally consistent:
    ```bash
    terraform validate
    ```
   
9. Apply the configuration:
    ```bash
    terraform apply
    ```

   !> **Warning:** Before running `terraform apply`, please take a look at the corresponding [README file](https://github.com/confluentinc/terraform-provider-confluent/blob/master/examples/configurations/basic-kafka-acls/README.md) for other instructions.

10. You have now created infrastructure using Terraform! Visit the [Confluent Cloud Console](https://confluent.cloud/environments) or use the [Confluent CLI v2](https://docs.confluent.io/confluent-cli/current/migrate.html#directly-install-confluent-cli-v2-x) to see the resources you provisioned.

## [Optional] Run a Quick Test

1.  Ensure you're using the acceptable version of the [Confluent CLI v2](https://docs.confluent.io/confluent-cli/current/migrate.html#directly-install-confluent-cli-v2-x) by running the following command:

    ```bash
    confluent version
    ```
    
    Your output should resemble:
    ```bash
    ...
    Version:     v2.5.1 # any version >= v2.0 is OK
    ...
    ```
2.  Run the following command to print out generated Confluent CLI v2 commands with the correct resource IDs injected:  

    ```bash
    # Alternatively, you could also run terraform output -json resource-ids
    terraform output resource-ids
    ```

    Your output should resemble:
    
    ```bash
    # 1. Log in to Confluent Cloud
    $ confluent login
    
    # 2. Produce key-value records to topic '<TOPIC_NAME>' by using <APP-PRODUCER'S NAME>'s Kafka API Key
    # Enter a few records and then press 'Ctrl-C' when you're done.
    # Sample records:
    # {"number":1,"date":18500,"shipping_address":"899 W Evelyn Ave, Mountain View, CA 94041, USA","cost":15.00}
    # {"number":2,"date":18501,"shipping_address":"1 Bedford St, London WC2E 9HG, United Kingdom","cost":5.00}
    # {"number":3,"date":18502,"shipping_address":"3307 Northland Dr Suite 400, Austin, TX 78731, USA","cost":10.00} 
    $ confluent kafka topic produce <TOPIC_NAME> --environment <ENVIRONMENT_ID> --cluster <CLUSTER_ID> --api-key "<APP-PRODUCER'S KAFKA API KEY>" --api-secret "<APP-PRODUCER'S KAFKA API SECRET>"
    
    # 3. Consume records from topic '<TOPIC_NAME>' by using <APP-CONSUMER'S NAME>'s Kafka API Key
    $ confluent kafka topic consume <TOPIC_NAME> --from-beginning --environment <ENVIRONMENT_ID> --cluster <CLUSTER_ID> --api-key "<APP-CONSUMER'S KAFKA API KEY>" --api-secret "<APP-CONSUMER'S KAFKA API SECRET>"
    # When you are done, press 'Ctrl-C'.
    ```

3. Execute printed out commands.

   -> **Note:** Add the `--from-beginning` flag to enable printing all messages from the beginning of the topic.

## [Optional] Teardown Confluent Cloud resources
Run the following command to destroy all the resources you created:

```bash
terraform destroy
```

This command destroys all the resources specified in your Terraform state. `terraform destroy` doesn't destroy resources running elsewhere that aren't managed by the current Terraform project.

Now you've created and destroyed an entire Confluent Cloud deployment!

Visit the [Confluent Cloud Console](https://confluent.cloud/environments) to verify the resources have been destroyed to avoid unexpected charges.

If you're interested in additional Confluent Cloud infrastructure configurations view our [repository](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations) for more end-to-end examples.
