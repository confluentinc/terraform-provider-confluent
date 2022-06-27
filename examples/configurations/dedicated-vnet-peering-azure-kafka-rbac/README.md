### Note

When using this example, you must execute `terraform` on a system with connectivity to the Kafka REST API. Check the [Kafka REST API docs](https://docs.confluent.io/cloud/current/api.html#tag/Topic-(v3)) to learn more about it.

Go to the following URL using your AD tenant ID (`<tenant-id>`) and approve:

```
https://login.microsoftonline.com/<tenant-id>/oauth2/authorize?client_id=f0955e3a-9013-4cf4-a1ea-21587621c9cc&response_type=code
```

before applying the configuration.

Also make sure you service principal has got "Directory Readers" role assigned. Otherwise, you might receive the following error:
```bash
Error: Listing service principals for filter "appId eq 'f0955e3a-9013-4cf4-a1ea-21587621c9cc'"

  on main.tf line 248, in data "azuread_service_principal" "peering_creator":
 248: data "azuread_service_principal" "peering_creator" {

ServicePrincipalsClient.BaseClient.Get(): unexpected status 403 with OData
error: Authorization_RequestDenied: Insufficient privileges to complete the
operation.
```

See [VNet Peering on Azure](https://docs.confluent.io/cloud/current/networking/peering/azure-peering.html) for more details.