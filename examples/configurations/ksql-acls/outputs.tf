output "resource-ids" {
  value = <<-EOT
  Environment ID:                       ${confluent_environment.staging.id}
  Kafka Cluster ID:                     ${confluent_kafka_cluster.standard.id}
  ksqlDB Cluster ID:                    ${confluent_ksql_cluster.main.id}
  ksqlDB Cluster API Endpoint:          ${confluent_ksql_cluster.main.rest_endpoint}
  KSQL Service Account ID:              ${confluent_service_account.app-ksql.id}

  # 1. Log in to Confluent Cloud
  $ confluent login

  # 2. Start ksqlDB's interactive CLI and connect it to your ksqlDB cluster. You'll need the ksqlDB API credentials you created, as well as the ksqlDB endpoint.
  # Please note that the ksqlDB cluster might take a few minutes to accept connections.
  $ docker run --rm -it confluentinc/ksqldb-cli:0.27.1 ksql \
       -u "${confluent_api_key.app-ksqldb-api-key.id}" \
       -p "${confluent_api_key.app-ksqldb-api-key.secret}" \
       "${confluent_ksql_cluster.main.rest_endpoint}"

  # 3. Make sure you can see "Server Status: RUNNING", otherwise (for example, "Server Status: <unknown>") enter `exit` and repeat step #3 in a few minutes.

  # 4. Once you are connected, you can create a ksqlDB stream. A stream essentially associates a schema with an underlying Kafka topic.
  CREATE STREAM ${confluent_kafka_topic.users.topic_name}_stream (id INTEGER KEY, gender STRING, name STRING, age INTEGER) WITH (kafka_topic='${confluent_kafka_topic.users.topic_name}', partitions=${confluent_kafka_topic.users.partitions_count}, value_format='JSON');

  # 5. Insert some data into the stream you just created.
  INSERT INTO ${confluent_kafka_topic.users.topic_name}_stream (id, gender, name, age) VALUES (0, 'female', 'sarah', 42);
  INSERT INTO ${confluent_kafka_topic.users.topic_name}_stream (id, gender, name, age) VALUES (1, 'male', 'john', 28);

  # 6. To confirm your insertion was successful, run a SELECT statement on your stream:
  SELECT * FROM ${confluent_kafka_topic.users.topic_name}_stream;

  # When you are done, press 'Ctrl-C'.
  EOT

  sensitive = true
}
