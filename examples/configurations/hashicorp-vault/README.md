### Notes

1. See [Sample Project for Confluent Terraform Provider](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/sample-project) that provides step-by-step instructions of running this example.

2. This example shows how to save API Key credentials to HashiCorp Vault.

3. This example assumes that KV v2 Engine is enabled under `secret/` path. Otherwise, `vault_mount` [resource](https://registry.terraform.io/providers/hashicorp/vault/latest/docs/resources/mount) should be created too.
