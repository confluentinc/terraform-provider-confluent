### Notes

1. When using this example, you must execute `terraform` on a system with connectivity to the Kafka REST API. Check the [Kafka REST API docs](https://docs.confluent.io/cloud/current/api.html#tag/Topic-(v3)) to learn more about it. Otherwise, you might see errors like:
   ```
   Error: error waiting for Kafka API Key "[REDACTED]" to sync: error listing Kafka Topics using Kafka API Key "[REDACTED]": Get "[https://[REDACTED]/kafka/v3/clusters/[REDACTED]/topics](https://[REDACTED]/kafka/v3/clusters/[REDACTED]/topics)": GET [https://[REDACTED]/kafka/v3/clusters/[REDACTED]/topics](https://[REDACTED]/kafka/v3/clusters/[REDACTED]/topics) giving up after 5 attempt(s): Get "[https://[REDACTED]/kafka/v3/clusters/[REDACTED]/topics](https://[REDACTED]/kafka/v3/clusters/[REDACTED/topics)": dial tcp [REDACTED]:443: i/o timeout
   ```

2. Apply Terraform configuration in 2 steps:

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

   You may need to run the following command:

    ```
    az network vnet subnet update \
      --disable-private-endpoint-network-policies true \
      --name default \
      --resource-group myResourceGroup \
      --vnet-name myVirtualNetwork
    ```
   For more information, see [Disable network policy](https://docs.microsoft.com/en-us/azure/private-link/disable-private-endpoint-network-policy).

   In order to apply remaining changes run
    ```
    # Creates others resources (except already created environment and network) declared in main.tf
    terraform apply
    ```

3. See [Use Azure Private Link](https://docs.confluent.io/cloud/current/networking/private-links/azure-privatelink.html) for more details.
