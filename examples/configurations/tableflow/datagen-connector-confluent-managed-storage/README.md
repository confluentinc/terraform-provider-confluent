### Notes

1. This example is a copy of the [`byob-aws-storage`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/tableflow/confluent-managed-storage) example. However, instead of enabling Tableflow for an empty Kafka topic, this example also creates a Managed Datagen Source connector that produces a large number of messages to the `orders` topic. As a result, users can almost immediately see populated data in their S3 bucket.
2. See [Sample Project for Confluent Terraform Provider](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/sample-project) that provides step-by-step instructions of running this example.
3. See [Get Started with Tableflow in Confluent Cloud: Quick Start with Custom Storage](https://docs.confluent.io/cloud/current/topics/tableflow/get-started/quick-start-custom-storage-glue.html#cloud-tableflow-quick-start) for more details.
