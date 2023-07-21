output "resource-ids" {
  value = <<-EOT
  'east' Kafka Cluster's Environment ID:   ${data.confluent_kafka_cluster.east.environment.0.id}
  'east' Kafka Cluster ID: ${data.confluent_kafka_cluster.east.id}
  'east' topic name: ${var.east_topic_name}
  'east' mirror topic name: ${confluent_kafka_mirror_topic.from-west.mirror_topic_name}

  'west' Kafka Cluster's Environment ID:   ${data.confluent_kafka_cluster.west.environment.0.id}
  'west' Kafka Cluster ID: ${data.confluent_kafka_cluster.west.id}
  'west' topic name: ${var.west_topic_name}
  'west' mirror topic name: ${confluent_kafka_mirror_topic.from-east.mirror_topic_name}

  It's worth mentioning that mirror topics have these unique properties:
  * Mirror topics are created by and owned by a cluster link.
  * Mirror topics get their messages from their source topic. They are byte-for-byte, offset-preserving asynchronous copies of their source topics.
  * Mirror topics are read-only; you can consume them the same as any other topic, but you cannot produce into them. If a producer tries to produce a message into a mirror topic, the action will fail. The only way to get a message into a mirror topic is to produce the message to the mirror topic’s source topic.
  * Many of the mirror topic’s configurations are copied and synced from the source topic. A full list is at the end of this page.

  Service Accounts and their Kafka API Keys (API Keys inherit the permissions granted to the owner):
  ${confluent_service_account.app-manager-east-cluster.display_name}:                     ${confluent_service_account.app-manager-east-cluster.id}
  ${confluent_service_account.app-manager-east-cluster.display_name}'s Kafka API Key:     "${confluent_api_key.app-manager-east-cluster-api-key.id}"
  ${confluent_service_account.app-manager-east-cluster.display_name}'s Kafka API Secret:  "${confluent_api_key.app-manager-east-cluster-api-key.secret}"

  ${confluent_service_account.app-manager-west-cluster.display_name}:                    ${confluent_service_account.app-manager-west-cluster.id}
  ${confluent_service_account.app-manager-west-cluster.display_name}'s Kafka API Key:    "${confluent_api_key.app-manager-west-cluster-api-key.id}"
  ${confluent_service_account.app-manager-west-cluster.display_name}'s Kafka API Secret: "${confluent_api_key.app-manager-west-cluster-api-key.secret}"

  In order to use the Confluent CLI v2 to

  A. Produce to east topic '${var.east_topic_name}' and consume from its mirror topic '${confluent_kafka_mirror_topic.from-east.mirror_topic_name}' using Kafka API Keys
  of ${confluent_service_account.app-manager-east-cluster.display_name} and ${confluent_service_account.app-manager-west-cluster.display_name} service accounts

  B. Produce to west topic '${var.west_topic_name}' and consume from its mirror topic '${confluent_kafka_mirror_topic.from-west.mirror_topic_name}' using Kafka API Keys
  of ${confluent_service_account.app-manager-west-cluster.display_name} and ${confluent_service_account.app-manager-east-cluster.display_name} service accounts

  run the following commands:

  # 1. Log in to Confluent Cloud
  $ confluent login

  # 2 (A). Produce key-value records to east topic '${var.east_topic_name}' by using ${confluent_service_account.app-manager-east-cluster.display_name}'s Kafka API Key
  $ confluent kafka topic produce ${var.east_topic_name} --environment ${data.confluent_kafka_cluster.east.environment.0.id} --cluster ${data.confluent_kafka_cluster.east.id} --api-key "${confluent_api_key.app-manager-east-cluster-api-key.id}" --api-secret "${confluent_api_key.app-manager-east-cluster-api-key.secret}"
  # Enter a few records and then press 'Ctrl-C' when you're done.
  # Sample records:
  # {"number":1,"date":18500,"shipping_address":"899 W Evelyn Ave, Mountain View, CA 94041, USA","cost":15.00}
  # {"number":2,"date":18501,"shipping_address":"1 Bedford St, London WC2E 9HG, United Kingdom","cost":5.00}
  # {"number":3,"date":18502,"shipping_address":"3307 Northland Dr Suite 400, Austin, TX 78731, USA","cost":10.00}

  # 3 (A). Consume records from its mirror topic '${confluent_kafka_mirror_topic.from-east.mirror_topic_name}' (created by and owned by a cluster link '${confluent_cluster_link.east-to-west.link_name}') by using ${confluent_service_account.app-manager-west-cluster.display_name}'s Kafka API Key
  $ confluent kafka topic consume ${confluent_kafka_mirror_topic.from-east.mirror_topic_name} --from-beginning --environment ${data.confluent_kafka_cluster.west.environment.0.id} --cluster ${data.confluent_kafka_cluster.west.id} --api-key "${confluent_api_key.app-manager-west-cluster-api-key.id}" --api-secret "${confluent_api_key.app-manager-west-cluster-api-key.secret}"
  # When you are done, press 'Ctrl-C'.

  # 4 (B). Produce key-value records to west topic '${var.west_topic_name}' by using ${confluent_service_account.app-manager-west-cluster.display_name}'s Kafka API Key
  $ confluent kafka topic produce ${var.west_topic_name} --environment ${data.confluent_kafka_cluster.west.environment.0.id} --cluster ${data.confluent_kafka_cluster.west.id} --api-key "${confluent_api_key.app-manager-west-cluster-api-key.id}" --api-secret "${confluent_api_key.app-manager-west-cluster-api-key.secret}"
  # Enter a few records and then press 'Ctrl-C' when you're done.
  # Sample records:
  # {"number":4,"date":18500,"shipping_address":"899 W Evelyn Ave, Mountain View, CA 94041, USA","cost":15.00}
  # {"number":5,"date":18501,"shipping_address":"1 Bedford St, London WC2E 9HG, United Kingdom","cost":5.00}
  # {"number":6,"date":18502,"shipping_address":"3307 Northland Dr Suite 400, Austin, TX 78731, USA","cost":10.00}

  # 4 (B). Consume records from its mirror topic '${confluent_kafka_mirror_topic.from-west.mirror_topic_name}' (created by and owned by a cluster link '${confluent_cluster_link.west-to-east.link_name}') by using ${confluent_service_account.app-manager-east-cluster.display_name}'s Kafka API Key
  $ confluent kafka topic consume ${confluent_kafka_mirror_topic.from-west.mirror_topic_name} --from-beginning --environment ${data.confluent_kafka_cluster.east.environment.0.id} --cluster ${data.confluent_kafka_cluster.east.id} --api-key "${confluent_api_key.app-manager-east-cluster-api-key.id}" --api-secret "${confluent_api_key.app-manager-east-cluster-api-key.secret}"
  # When you are done, press 'Ctrl-C'.
  EOT

  sensitive = true
}
