### Notes

1. This example is a copy of [`byob-aws-storage`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/tableflow/confluent-managed-storage), but instead of enabling Tableflow for an empty Kafka topic, this example also creates a Managed Datagen Source connector, so a user can immediately see populated data in their S3 bucket.
2. Add credentials and other settings to `$HOME/.aws/config` for AWS TF Provider to work: https://registry.terraform.io/providers/hashicorp/aws/latest/docs#shared-configuration-and-credentials-files
3. See [Sample Project for Confluent Terraform Provider](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/sample-project) that provides step-by-step instructions of running this example.
4. See [Get Started with Tableflow in Confluent Cloud: Quick Start with Custom Storage](https://docs.confluent.io/cloud/current/topics/tableflow/get-started/quick-start-custom-storage-glue.html#cloud-tableflow-quick-start) for more details.
5. See [Quick Start for Confluent Cloud Provider Integration](https://docs.confluent.io/cloud/current/connectors/provider-integration/index.html) for more details.
6. See the following docs for more details:

   * [Hands On: Configure, Build and Register Protobuf and Avro Schemas](https://developer.confluent.io/learn-kafka/schema-registry/configure-schemas-hands-on/).

   * [Integrating Schema Registry with Client Applications](https://developer.confluent.io/learn-kafka/schema-registry/integrate-schema-registry-with-clients/).

   * [Managing multiple event types in a single topic with Schema Registry](https://www.confluent.io/events/kafka-summit-europe-2021/managing-multiple-event-types-in-a-single-topic-with-schema-registry/).

   * [Multiple Event Types in the Same Kafka Topic - Revisited](https://www.confluent.io/blog/multiple-event-types-in-the-same-kafka-topic/).
