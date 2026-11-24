# Client Request Policy example

This example manages a [`confluent_client_request_policy`](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_client_request_policy)
that defines a server-side rule Confluent Cloud enforces on Kafka client requests.

Client Request Policies let operators audit, deny, or override non-compliant client behavior (for
example, requiring a minimum client version) without any client-side changes. See the
[Client Request Policies documentation](https://docs.confluent.io/cloud/current/) for details on
supported policy types, scopes, and enforcement actions.

## Usage

1. Set your Confluent Cloud API credentials, either as `cloud_api_key` / `cloud_api_secret`
   variables or via the `CONFLUENT_CLOUD_API_KEY` and `CONFLUENT_CLOUD_API_SECRET` environment
   variables.
2. Set `kafka_cluster_crn` to the Confluent Resource Name (CRN) of the Kafka cluster the policy
   should be bound to.
3. Run:

   ```shell
   terraform init
   terraform plan
   terraform apply
   ```
