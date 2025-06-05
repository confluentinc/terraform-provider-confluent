### Notes

1. This example demonstrates how to create and use a custom SMT in Confluent Cloud.
2. The example creates:
   - A Connect Artifact resource that uploads and manages the custom SMT
   - A fully-managed Datagen Source connector that uses the custom SMT to dynamically route records to topics based on the `itemid` field
3. Before running this example:
   - Prepare your custom SMT artifact (JAR or ZIP file)
   - Create a `terraform.tfvars` file with the following variables:
     ```hcl
     confluent_cloud_api_key    = "your-cloud-api-key"
     confluent_cloud_api_secret = "your-cloud-api-secret"
     artifact_file             = "path/to/your/artifact.jar"  # Can be relative or absolute path
     environment_id            = "your-environment-id"
     kafka_cluster_id          = "your-kafka-cluster-id"
     kafka_api_key             = "your-kafka-api-key"
     kafka_api_secret          = "your-kafka-api-secret"
     ```
   - Optionally customize other variables like `artifact_display_name`, `artifact_description`, etc.
4. The example transforms records by routing them to topics named after their `itemid` field value instead of the configured topic.

Note: The `artifact_file` path can be either relative to your Terraform working directory or an absolute system path. For example:
- Relative path: `"target/custom-smt-1.0-SNAPSHOT-jar-with-dependencies.jar"`
- Absolute path: `"/Users/username/projects/my-smt/target/custom-smt-1.0-SNAPSHOT-jar-with-dependencies.jar"` 