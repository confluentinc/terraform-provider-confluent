### Notes

1. Make sure to use Stream Governance [Advanced package](https://docs.confluent.io/cloud/current/stream-governance/packages.html#packages) and create a Kafka topic called "purchase" before running this example.
2. Apply this Terraform configuration by following [Sample Project for Confluent Terraform Provider](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/sample-project).
3. Download [Confluent Platform archive package](https://docs.confluent.io/platform/current/installation/installing_cp/zip-tar.html) that contains `bin/kafka-avro-console-producer` and `bin/kafka-avro-console-consumer` binaries.
4. 

N. See the following docs for more details:

    * [Hands On: Configure, Build and Register Protobuf and Avro Schemas](https://developer.confluent.io/learn-kafka/schema-registry/configure-schemas-hands-on/).

    * [Integrating Schema Registry with Client Applications](https://developer.confluent.io/learn-kafka/schema-registry/integrate-schema-registry-with-clients/).
