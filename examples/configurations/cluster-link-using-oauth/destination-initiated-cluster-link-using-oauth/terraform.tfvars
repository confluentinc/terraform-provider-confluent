# The OAuth 2.0 token endpoint from external identity provider
oauth_external_token_url = "https://ccloud-sso-sandbox.okta.com/oauth2/ausod37qoaxy2xfjI697/v1/token"

# The Application (client) ID registered in external identity provider for OAuth purpose
oauth_external_client_id = "0oaod69tu7yYnbrMn697"

# The Application (client) ID registered in external identity provider for OAuth purpose
oauth_external_client_secret = "QN83b_1JscAAep7JTEdXvdloEUqKBXwk4_K00VyzXnYYBVFbCxdZN4Vy6NUDdo04"

# The OAuth identity pool id with external identity provider, registered with Confluent Cloud
# https://docs.confluent.io/cloud/current/security/authenticate/workload-identities/identity-providers/oauth/identity-providers.html
oauth_identity_pool_id = "pool-W5Qe"

# ID of the destination Kafka Cluster
destination_kafka_cluster_id = "lkc-x81kwg"

# ID of the Environment that the destination Kafka Cluster belongs to
destination_kafka_cluster_environment_id = "env-y2omyj"

# ID of the source Kafka Cluster
source_kafka_cluster_id = "lkc-og066y"

# ID of the Environment that the source Kafka Cluster belongs to
source_kafka_cluster_environment_id = "env-8y1wv7"

# Name of the Topic on the source Kafka Cluster to create a Mirror Topic for
source_topic_name = "cluster_link_topic_test1"

# Name of the Cluster Link to create
cluster_link_name = "destination-initiated-terraform-using-oauth"
