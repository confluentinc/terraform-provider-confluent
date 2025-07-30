# The instructions to setup Okta as an OAuth identity provider for Confluent Cloud can be found here:
# https://docs.confluent.io/cloud/current/security/authenticate/workload-identities/identity-providers/oauth/identity-providers.html
# Below are the required variables with dummy values to configure Okta as an OAuth identity provider for Confluent Cloud.

# The Okta OAuth 2.0 token endpoint (v1)
oauth_external_token_url = "https://<company-domain>.okta.com/oauth2/abcdefg1234567/v1/token"

# The Application (client) ID registered in Okta for OAuth purpose
oauth_external_client_id = "123456789abcdef12345"

# The Application (client) secret registered in Okta for OAuth purpose
oauth_external_client_secret = "ABCde_1JscAAep7JTEdXvdloEUqKBXwk4_K00VyzXnYYBVFbCxdZN4Vy6NUD0123"

# The OAuth identity pool id with Okta as identity provider, registered with Confluent Cloud
# https://docs.confluent.io/cloud/current/security/authenticate/workload-identities/identity-providers/oauth/identity-providers.html
oauth_identity_pool_id = "pool-abC12"
