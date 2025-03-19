### Notes

1. See [Sample Project for Confluent Terraform Provider](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/sample-project) that provides step-by-step instructions of running this example.
2. See [Get Started with Tableflow in Confluent Cloud: Quick Start with Managed Storage](https://docs.confluent.io/cloud/current/topics/tableflow/get-started/quick-start-managed-storage.html#cloud-tableflow-quick-start-managed-storage) for more details.
3. See the following docs for more details:

    * [Hands On: Configure, Build and Register Protobuf and Avro Schemas](https://developer.confluent.io/learn-kafka/schema-registry/configure-schemas-hands-on/).

    * [Integrating Schema Registry with Client Applications](https://developer.confluent.io/learn-kafka/schema-registry/integrate-schema-registry-with-clients/).

    * [Managing multiple event types in a single topic with Schema Registry](https://www.confluent.io/events/kafka-summit-europe-2021/managing-multiple-event-types-in-a-single-topic-with-schema-registry/).

    * [Multiple Event Types in the Same Kafka Topic - Revisited](https://www.confluent.io/blog/multiple-event-types-in-the-same-kafka-topic/).


4. Make sure to use `confluent_catalog_integration` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_catalog_integration) if you want to integrate Tableflow with AWS Glue Catalog or Snowflake Open Catalog.
    - [Integrate Tableflow with the AWS Glue Catalog in Confluent Cloud](https://docs.confluent.io/cloud/current/topics/tableflow/how-to-guides/catalog-integration/integrate-with-aws-glue-catalog.html)
    - [Integrate Tableflow with Snowflake Open Catalog or Apache Polaris in Confluent Cloud](https://docs.confluent.io/cloud/current/topics/tableflow/how-to-guides/catalog-integration/integrate-with-snowflake-open-catalog-or-apache-polaris.html)
5. You may find the [`datagen-connector-confluent-managed-storage`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/tableflow/datagen-connector-confluent-managed-storage) example useful as well. Instead of enabling Tableflow for an empty Kafka topic, the [`datagen-connector-confluent-managed-storage`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/tableflow/datagen-connector-confluent-managed-storage) example also creates a Managed Datagen Source connector that produces a large number of messages to the `stock-trades` topic. As a result, users can actually query the data.
