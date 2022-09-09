### Notes

1. See [Sample Project for Confluent Terraform Provider](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/sample-project) that provides step-by-step instructions of running this example.

2. This example assumes that Terraform is run from a host in the private network, where it will have connectivity to the [Kafka REST API](https://docs.confluent.io/cloud/current/api.html#tag/Topic-(v3)) in other words, to the [REST endpoint](https://docs.confluent.io/cloud/current/clusters/broker-config.html#access-cluster-settings-in-the-ccloud-console) on the provisioned Kafka cluster. If it is not, you must make these changes:

    * Update the `confluent_api_key` resources by setting their `disable_wait_for_ready` flag to `true`. Otherwise, Terraform will attempt to validate API key creation by listing topics, which will fail without access to the Kafka REST API. Otherwise, you might see errors like:

        ```
        Error: error waiting for Kafka API Key "[REDACTED]" to sync: error listing Kafka Topics using Kafka API Key "[REDACTED]": Get "[https://[REDACTED]/kafka/v3/clusters/[REDACTED]/topics](https://[REDACTED]/kafka/v3/clusters/[REDACTED]/topics)": GET [https://[REDACTED]/kafka/v3/clusters/[REDACTED]/topics](https://[REDACTED]/kafka/v3/clusters/[REDACTED]/topics) giving up after 5 attempt(s): Get "[https://[REDACTED]/kafka/v3/clusters/[REDACTED]/topics](https://[REDACTED]/kafka/v3/clusters/[REDACTED/topics)": dial tcp [REDACTED]:443: i/o timeout
        ```

    * Remove the `confluent_kafka_topic` resource. These resources are provisioned using the Kafka REST API, which is only accessible from the private network.

3. One common deployment workflow for environments with private networking is as follows:

    * A initial (centrally-run) Terraform deployment provisions infrastructure: network, Kafka cluster, and other resources on cloud provider of your choice to setup private network connectivity (like DNS records)

    * A secondary Terraform deployment (run from within the private network) provisions data-plane resources (Kafka Topics and ACLs)

    * Note that RBAC role bindings can be provisioned in either the first or second step, as they are provisioned through the [Confluent Cloud API](https://docs.confluent.io/cloud/current/api.html), not the [Kafka REST API](https://docs.confluent.io/cloud/current/api.html#tag/Topic-(v3))


4. See [AWS PrivateLink](https://docs.confluent.io/cloud/current/networking/private-links/aws-privatelink.html) for more details.
