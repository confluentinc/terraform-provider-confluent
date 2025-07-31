# The OAuth 2.0 token endpoint from external identity provider
oauth_external_token_url = "https://ccloud-sso-sandbox.okta.com/oauth2/ausod37qoaxy2xfjI697/v1/token"

# The Application (client) ID registered in external identity provider for OAuth purpose
oauth_external_client_id = "0oaod69tu7yYnbrMn697"

# The Application (client) ID registered in external identity provider for OAuth purpose
oauth_external_client_secret = "QN83b_1JscAAep7JTEdXvdloEUqKBXwk4_K00VyzXnYYBVFbCxdZN4Vy6NUDdo04"

# The OAuth identity pool id with external identity provider, registered with Confluent Cloud
# https://docs.confluent.io/cloud/current/security/authenticate/workload-identities/identity-providers/oauth/identity-providers.html
oauth_identity_pool_id = "pool-W5Qe"

# ID of the 'west' Kafka Cluster
west_kafka_cluster_id = "lkc-mo3j77"

# ID of the Environment that the 'west' Kafka Cluster belongs to
west_kafka_cluster_environment_id = "env-y2omyj"

# Name of the Topic on the 'west' Kafka Cluster to create a Mirror Topic for
west_topic_name = "public.topic-on-west"

# ID of the 'east' Kafka Cluster
east_kafka_cluster_id = "lkc-og118j"

# ID of the Environment that the 'east' Kafka Cluster belongs to
east_kafka_cluster_environment_id = "env-8y1wv7"

# Name of the Topic on the 'east' Kafka Cluster to create a Mirror Topic for
east_topic_name = "public.topic-on-east"

# Name of the Cluster Link to create
cluster_link_name = "bidirectional-link"
