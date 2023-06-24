### Notes

1. See [Sample Project for Confluent Terraform Provider](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/sample-project) that provides step-by-step instructions of running this example.

2. This example is a slightly rewritten version of [basic-kafka-acls](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/basic-kafka-acls) that hides most of the complexity from a user: it takes topic name and Kafka Cluster ID as an input and returns 2 Kafka API Keys for producing and consuming from that topic so a user doesn't have to worry about all the internal infrastructure like service accounts and ACLs.
