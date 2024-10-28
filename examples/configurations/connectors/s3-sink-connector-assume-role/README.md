### Notes

1. This example needs to be run in 2 steps: first run `terraform apply` and then uncomment lines for "step #2"
(update aws_iam_role.s3_access_role, confluent_connector.s3-sink instances) and run `terraform apply` again.
2. See [Sample Project for Confluent Terraform Provider](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/sample-project) that provides step-by-step instructions of running this example.
3. See [Quick Start for Confluent Cloud Provider Integration
   ](https://docs.confluent.io/cloud/current/connectors/provider-integration/index.html) for more details.
