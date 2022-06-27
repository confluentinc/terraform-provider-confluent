### Note

When using this example, you must execute `terraform` on a system with connectivity to the Kafka REST API. Check the [Kafka REST API docs](https://docs.confluent.io/cloud/current/api.html#tag/Topic-(v3)) to learn more about it.

Apply Terraform configuration in 2 steps:

```
# Creates an environment and a network
terraform apply -target=confluent_network.private-link
```

If you run into

```
Error: Invalid function argument

  on main.tf line 249, in locals:
 249:   ]) > 0 ? file("\n\nerror: private link endpoint network policies must be disabled https://docs.microsoft.com/en-us/azure/private-link/disable-private-endpoint-network-policy") : ""

Invalid value for "path" parameter: no file exists at

error: private link endpoint network policies must be disabled
https:/docs.microsoft.com/en-us/azure/private-link/disable-private-endpoint-network-policy;
this function works only with files that are distributed as part of the
configuration source code, so if this file will be created by a resource in
this configuration you must instead obtain this result from an attribute of
that resource.
```

Please follow [Disable network policies for Private Link service source IP](https://docs.microsoft.com/en-us/azure/private-link/disable-private-link-service-network-policy) to fix the issue.

In order to apply remaining changes run
```
# Creates others resources (except already created environment and network) declared in main.tf
terraform apply
```

See [Use Azure Private Link](https://docs.confluent.io/cloud/current/networking/private-links/azure-privatelink.html) for more details.
