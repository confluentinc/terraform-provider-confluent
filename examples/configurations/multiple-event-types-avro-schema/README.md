### Notes

1. See [Sample Project for Confluent Terraform Provider](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/sample-project) that provides step-by-step instructions of running this example.
2. This example should be run in multiple steps since `confluent_api_key` doesn't support Schema Registry API keys and `confluent_role_binding` doesn't support Schema Registry clusters yet:

    * Follow instructions from [Sample Project for Confluent Terraform Provider](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/sample-project) as usual.

    * [Create](https://docs.confluent.io/cloud/current/access-management/authenticate/api-keys/api-keys.html#create-a-resource-specific-api-key) Schema Registry API Key via the Confluent Cloud Console or [Confluent CLI v2](https://docs.confluent.io/confluent-cli/current/migrate.html#directly-install-confluent-cli-v2-x)

    * Copy Schema Registry API Key and Schema Registry API Secret in a `locals` block in `main.tf`.

    * Uncomment `outputs.tf` file and  the remainder of `main.tf` file that contains definitions of `confluent_schema` resources.

    * Run `terraform plan` and `terraform apply` to create new schemas.

    * Run `terraform output resource-ids` to print out generated Confluent CLI v2 commands with the correct resource IDs injected for running a quick test of producing / consuming messages using the schema associated with the topic.

    * TODO: fix "Error: failed to load schema: avro reference not supported in cloud CLI"
3. See the following docs for more details:

    * [Hands On: Configure, Build and Register Protobuf and Avro Schemas](https://developer.confluent.io/learn-kafka/schema-registry/configure-schemas-hands-on/).

    * [Integrating Schema Registry with Client Applications](https://developer.confluent.io/learn-kafka/schema-registry/integrate-schema-registry-with-clients/).

    * [Managing multiple event types in a single topic with Schema Registry](https://www.confluent.io/events/kafka-summit-europe-2021/managing-multiple-event-types-in-a-single-topic-with-schema-registry/).

    * [Multiple Event Types in the Same Kafka Topic - Revisited](https://www.confluent.io/blog/multiple-event-types-in-the-same-kafka-topic/).
