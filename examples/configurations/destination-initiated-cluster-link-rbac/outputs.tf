output "resource-ids" {
  value = <<-EOT
  Source Kafka Cluster's Environment ID:   ${data.confluent_kafka_cluster.source.environment.0.id}
  Source Kafka Cluster ID: ${data.confluent_kafka_cluster.source.id}
  Source topic name: ${var.source_topic_name}

  Destination Kafka Cluster's Environment ID:   ${data.confluent_kafka_cluster.destination.environment.0.id}
  Destination Kafka Cluster ID: ${data.confluent_kafka_cluster.destination.id}
  Destination mirror topic name: ${confluent_kafka_mirror_topic.test.mirror_topic_name}

  It's worth mentioning that mirror topics have these unique properties:
  * Mirror topics are created by and owned by a cluster link.
  * Mirror topics get their messages from their source topic. They are byte-for-byte, offset-preserving asynchronous copies of their source topics.
  * Mirror topics are read-only; you can consume them the same as any other topic, but you cannot produce into them. If a producer tries to produce a message into a mirror topic, the action will fail. The only way to get a message into a mirror topic is to produce the message to the mirror topic’s source topic.
  * Many of the mirror topic’s configurations are copied and synced from the source topic. A full list is at the end of this page.

  Service Accounts and their Kafka API Keys (API Keys inherit the permissions granted to the owner):
  ${confluent_service_account.app-manager-source-cluster.display_name}:                     ${confluent_service_account.app-manager-source-cluster.id}
  ${confluent_service_account.app-manager-source-cluster.display_name}'s Kafka API Key:     "${confluent_api_key.app-manager-source-cluster-api-key.id}"
  ${confluent_service_account.app-manager-source-cluster.display_name}'s Kafka API Secret:  "${confluent_api_key.app-manager-source-cluster-api-key.secret}"

  ${confluent_service_account.app-manager-destination-cluster.display_name}:                    ${confluent_service_account.app-manager-destination-cluster.id}
  ${confluent_service_account.app-manager-destination-cluster.display_name}'s Kafka API Key:    "${confluent_api_key.app-manager-destination-cluster-api-key.id}"
  ${confluent_service_account.app-manager-destination-cluster.display_name}'s Kafka API Secret: "${confluent_api_key.app-manager-destination-cluster-api-key.secret}"

  In order to use the Confluent CLI v2 to produce to source topic '${var.source_topic_name}' and consume from mirror topic '${var.source_topic_name}' using Kafka API Keys
  of ${confluent_service_account.app-manager-source-cluster.display_name} and ${confluent_service_account.app-manager-destination-cluster.display_name} service accounts
  run the following commands:

  # 1. Log in to Confluent Cloud
  $ confluent login

  # 2. Produce key-value records to source topic '${var.source_topic_name}' by using ${confluent_service_account.app-manager-source-cluster.display_name}'s Kafka API Key
  $ confluent kafka topic produce ${var.source_topic_name} --environment ${data.confluent_kafka_cluster.source.environment.0.id} --cluster ${data.confluent_kafka_cluster.source.id} --api-key "${confluent_api_key.app-manager-source-cluster-api-key.id}" --api-secret "${confluent_api_key.app-manager-source-cluster-api-key.secret}"
  # Enter a few records and then press 'Ctrl-C' when you're done.
  # Sample records:
  # {"number":1,"date":18500,"shipping_address":"899 W Evelyn Ave, Mountain View, CA 94041, USA","cost":15.00}
  # {"number":2,"date":18501,"shipping_address":"1 Bedford St, London WC2E 9HG, United Kingdom","cost":5.00}
  # {"number":3,"date":18502,"shipping_address":"3307 Northland Dr Suite 400, Austin, TX 78731, USA","cost":10.00}

  # 3. Consume records from a mirror topic '${var.source_topic_name}' (created by and owned by a cluster link '${confluent_cluster_link.destination-outbound.link_name}') by using ${confluent_service_account.app-manager-destination-cluster.display_name}'s Kafka API Key
  $ confluent kafka topic consume ${var.source_topic_name} --from-beginning --environment ${data.confluent_kafka_cluster.destination.environment.0.id} --cluster ${data.confluent_kafka_cluster.destination.id} --api-key "${confluent_api_key.app-manager-destination-cluster-api-key.id}" --api-secret "${confluent_api_key.app-manager-destination-cluster-api-key.secret}"
  # When you are done, press 'Ctrl-C'.
  EOT

  sensitive = true
}
