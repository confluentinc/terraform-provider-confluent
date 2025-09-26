# Please find more details on how to configure OAuth authentication Terraform Provider variables for Confluent Cloud in the examples link below:
# Azure Entra Id: https://github.com/confluentinc/terraform-provider-confluent/blob/master/examples/configurations/authentication-using-oauth/azure-entra-id/terraform.tfvars
# Okta: https://github.com/confluentinc/terraform-provider-confluent/blob/master/examples/configurations/authentication-using-oauth/okta/terraform.tfvars

# The OAuth 2.0 token endpoint (v2)
oauth_external_token_url = ""

# The Application (client) ID registered in Identity Provider for OAuth purpose
oauth_external_client_id = ""

# The Application (client) secret registered in Identity Provider for OAuth purpose
oauth_external_client_secret = ""

# The optional OAuth client application scope, Azure Entra ID server will reject a API request that doesn't include the scope
oauth_external_token_scope = ""

# The OAuth identity pool id, registered with Confluent Cloud
oauth_identity_pool_id = ""
