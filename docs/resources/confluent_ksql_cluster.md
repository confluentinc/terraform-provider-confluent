# confluent_ksql_cluster Resource

[![Open Preview](https://img.shields.io/badge/Lifecycle%20Stage-Open%20Preview-%2300afba)](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy)

-> **Note:** `confluent_ksql_cluster` resource is available in **Open Preview** for early adopters. Open Preview features are introduced to gather customer feedback. This feature should be used only for evaluation and non-production testing purposes or to provide feedback to Confluent, particularly as it becomes more widely available in follow-on editions.  
**Open Preview** features are intended for evaluation use in development and testing environments only, and not for production use. The warranty, SLA, and Support Services provisions of your agreement with Confluent do not apply to Open Preview features. Open Preview features are considered to be a Proof of Concept as defined in the Confluent Cloud Terms of Service. Confluent may discontinue providing preview releases of the Open Preview features at any time in Confluent’s sole discretion.

!> **Warning:**  It is strongly recommended that you provision a `confluent_stream_governance_cluster` resource before you provision a `confluent_ksql_cluster` resource in a given environment. If you're provisioning the `confluent_stream_governance_cluster` and the `confluent_ksql_cluster` resource in the same Terraform apply command, reference the `confluent_stream_governance_cluster` from the `depends_on` argument inside the `confluent_ksql_cluster` resource. This ensures that the `confluent_stream_governance_cluster` resource is created before the `confluent_ksql_cluster` resource. If you provision a `confluent_ksql_cluster` resource without a `confluent_stream_governance_cluster` resource, and later, you want to add a `confluent_stream_governance_cluster` resource, you must destroy and re-create your `confluent_ksql_cluster` resource after provisioning a `confluent_stream_governance_cluster` resource.

`confluent_ksql_cluster` provides a ksqlDB cluster resource that enables creating, editing, and deleting ksqlDB clusters on Confluent Cloud.

## Example Usage

```terraform
resource "confluent_environment" "development" {
  display_name = "Development"
}

resource "confluent_stream_governance_cluster" "essentials" {
  package = "ESSENTIALS"

  environment {
    id = confluent_environment.development.id
  }

  region {
    # See https://docs.confluent.io/cloud/current/stream-governance/packages.html#stream-governance-regions
    id = "sgreg-1"
  }
}

resource "confluent_kafka_cluster" "basic" {
  display_name = "basic_kafka_cluster"
  availability = "SINGLE_ZONE"
  cloud        = "AWS"
  region       = "us-east-1"
  basic {}

  environment {
    id = confluent_environment.development.id
  }
}

resource "confluent_service_account" "app-ksql" {
  display_name = "app-ksql"
  description  = "Service account to manage 'example' ksqlDB cluster"
}

resource "confluent_role_binding" "app-ksql-kafka-cluster-admin" {
  principal   = "User:${confluent_service_account.app-ksql.id}"
  role_name   = "CloudClusterAdmin"
  crn_pattern = confluent_kafka_cluster.basic.rbac_crn
}

resource "confluent_ksql_cluster" "example" {
  display_name = "example"
  csu          = 1
  kafka_cluster {
    id = confluent_kafka_cluster.basic.id
  }
  credential_identity {
    id = confluent_service_account.app-ksql.id
  }
  environment {
    id = confluent_environment.staging.id
  }
  depends_on = [
    confluent_role_binding.app-ksql-kafka-cluster-admin,
    confluent_stream_governance_cluster.essentials
  ]
}
```

## Argument Reference

The following arguments are supported:

- `display_name` - (Required String) The name of the ksqlDB cluster.
- `csu` - (Required Number) The number of Confluent Streaming Units (CSUs) for the ksqlDB cluster.
- `use_detailed_processing_log` (Optional Boolean) Controls whether the row data should be included in the processing log topic. Set it to `false` if you don't want to emit sensitive information to the processing log. Defaults to `true`.
- `environment` (Required Configuration Block) supports the following:
    - `id` - (Required String) The ID of the associated Environment, for example, `env-xyz456`.
- `kafka_cluster` (Required Configuration Block) supports the following:
    - `id` - (Required String) The ID of the associated Kafka cluster, for example, `lkc-abc123`.
- `credential_identity` (Required Configuration Block) supports the following:
    - `id` - (Required String) The ID of the associated service or user account, for example, `sa-abc123`.
  

## Attributes Reference

In addition to the preceding arguments, the following attributes are exported:

- `id` - (Required String) The ID of the ksqlDB cluster, for example, `lksqlc-abc123`.
- `api_version` - (Required String) An API Version of the schema version of the ksqlDB cluster, for example, `ksqldbcm/v2`.
- `kind` - (Required String) A kind of the ksqlDB cluster, for example, `Cluster`.
- `topic_prefix` - (Required String) Topic name prefix used by this ksqlDB cluster. Used to assign ACLs for this ksqlDB cluster to use, for example, `pksqlc-00000`.
- `http_endpoint` - (Required String) The API endpoint of the ksqlDB cluster, for example, `https://pksqlc-00000.us-central1.gcp.glb.confluent.cloud`.
- `storage` - (Required Integer) The amount of storage (in GB) provisioned to the ksqlDB cluster.

## Import

-> **Note:** `CONFLUENT_CLOUD_API_KEY` and `CONFLUENT_CLOUD_API_SECRET` environment variables must be set before importing a ksqlDB cluster.

You can import a ksqlDB cluster by using Environment ID and ksqlDB cluster ID, in the format `<Environment ID>/<ksqlDB cluster ID>`, for example:

```shell
$ export CONFLUENT_CLOUD_API_KEY="<cloud_api_key>"
$ export CONFLUENT_CLOUD_API_SECRET="<cloud_api_secret>"
$ terraform import confluent_ksql_cluster.example env-abc123/lksqlc-abc123
```

!> **Warning:**  Do not forget to delete the terminal's command history afterward for security purposes.


## Getting Started

The following end-to-end examples might help to get started with `confluent_ksql_cluster` resource:
* [`ksql-acls`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/ksql-acls)
* [`ksql-rbac`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/ksql-rbac)
