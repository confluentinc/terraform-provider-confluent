### Notes

1. This example mirrors [Get Started with Tableflow in Confluent Cloud: Quick Start with Custom Storage](https://docs.confluent.io/cloud/current/topics/tableflow/get-started/quick-start-custom-storage-glue.html).
2. This example is a copy of the [`byob-aws-storage`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/tableflow/byob-aws-storage) example. However, instead of enabling Tableflow for an empty Kafka topic, this example also creates a Managed Datagen Source connector that produces a large number of messages to the `stock-trades` topic. As a result, users can actually query the data.
3. Add credentials and other settings to `$HOME/.aws/config` for AWS TF Provider to work: https://registry.terraform.io/providers/hashicorp/aws/latest/docs#shared-configuration-and-credentials-files
4. See [Sample Project for Confluent Terraform Provider](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/sample-project) that provides step-by-step instructions of running this example.
5. See [Quick Start for Confluent Cloud Provider Integration](https://docs.confluent.io/cloud/current/connectors/provider-integration/index.html) for more details.
6. Make sure to use `confluent_catalog_integration` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_catalog_integration) if you want to integrate Tableflow with AWS Glue Catalog or Snowflake Open Catalog.
  - [Integrate Tableflow with the AWS Glue Catalog in Confluent Cloud](https://docs.confluent.io/cloud/current/topics/tableflow/how-to-guides/catalog-integration/integrate-with-aws-glue-catalog.html)
  - [Integrate Tableflow with Snowflake Open Catalog or Apache Polaris in Confluent Cloud](https://docs.confluent.io/cloud/current/topics/tableflow/how-to-guides/catalog-integration/integrate-with-snowflake-open-catalog-or-apache-polaris.html)
