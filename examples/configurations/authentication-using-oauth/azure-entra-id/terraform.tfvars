# The instructions to setup Azure Entra ID (formerly Azure AD) as an OAuth identity provider for Confluent Cloud can be found here:
# https://www.confluent.io/blog/configuring-azure-ad-ds-with-oauth-for-confluent/
# Below are the required variables with dummy values to configure Azure Entra ID as an OAuth identity provider for Confluent Cloud.

# The Microsoft Azure Entra ID OAuth 2.0 token endpoint (v2)
oauth_external_token_url = "https://login.microsoftonline.com/{tenant_id}/oauth2/v2.0/token"

# The Application (client) ID registered in Azure Entra ID for OAuth purpose
oauth_external_client_id = "11111111-2222-3333-4444-555555555555"

# The Application (client) secret registered in Azure Entra ID for OAuth purpose
oauth_external_client_secret = "abcde~12345-fghijklm-67890~iukRhtC_Xb0t"

# The OAuth client application scope, Azure Entra ID serves may reject a request that doesn't include a scope.
oauth_external_token_scope = "api://{client_id}/.default"

# The OAuth identity pool id with Azure Entra ID as identity provider, registered with Confluent Cloud
# https://docs.confluent.io/cloud/current/security/authenticate/workload-identities/identity-providers/oauth/identity-pools.html
oauth_identity_pool_id = "pool-abC12"
