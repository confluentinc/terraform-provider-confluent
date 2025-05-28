### Notes

1. Add credentials and other settings to `$HOME/.aws/config` for AWS TF Provider to work: https://registry.terraform.io/providers/hashicorp/aws/latest/docs#shared-configuration-and-credentials-files
2. See [Sample Project for Confluent Terraform Provider](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/sample-project) that provides step-by-step instructions of running this example.
3. See [Get Started with Tableflow in Confluent Cloud: Quick Start with Custom Storage](https://docs.confluent.io/cloud/current/topics/tableflow/get-started/quick-start-custom-storage-glue.html#cloud-tableflow-quick-start) for more details.
4. See [Quick Start for Confluent Cloud Provider Integration](https://docs.confluent.io/cloud/current/connectors/provider-integration/index.html) for more details.
5. See the following docs for more details:

    * [Hands On: Configure, Build and Register Protobuf and Avro Schemas](https://developer.confluent.io/learn-kafka/schema-registry/configure-schemas-hands-on/).

    * [Integrating Schema Registry with Client Applications](https://developer.confluent.io/learn-kafka/schema-registry/integrate-schema-registry-with-clients/).

    * [Managing multiple event types in a single topic with Schema Registry](https://www.confluent.io/events/kafka-summit-europe-2021/managing-multiple-event-types-in-a-single-topic-with-schema-registry/).

    * [Multiple Event Types in the Same Kafka Topic - Revisited](https://www.confluent.io/blog/multiple-event-types-in-the-same-kafka-topic/).
6. Make sure to use `confluent_catalog_integration` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_catalog_integration) if you want to integrate Tableflow with AWS Glue Catalog or Snowflake Open Catalog.
    - [Integrate Tableflow with the AWS Glue Catalog in Confluent Cloud](https://docs.confluent.io/cloud/current/topics/tableflow/how-to-guides/catalog-integration/integrate-with-aws-glue-catalog.html)
    - [Integrate Tableflow with Snowflake Open Catalog or Apache Polaris in Confluent Cloud](https://docs.confluent.io/cloud/current/topics/tableflow/how-to-guides/catalog-integration/integrate-with-snowflake-open-catalog-or-apache-polaris.html)
7. You may find the [datagen-connector-byob-aws-storage](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/tableflow/datagen-connector-byob-aws-storage) example useful as well. Instead of enabling Tableflow for an empty Kafka topic, the [datagen-connector-byob-aws-storage](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/tableflow/datagen-connector-byob-aws-storage) example also creates a Managed Datagen Source connector that produces a large number of messages to the `stock-trades` topic. As a result, users can actually query the data.
