### Quick Start

This example displays how TF config can be split between Kafka Ops team and Product team.

```bash
➜  kafka-ops-team git:(master) ✗ export TF_VAR_confluent_cloud_api_key="***REDACTED***" # with OrganizationAdmin permissions
➜  kafka-ops-team git:(master) ✗ export TF_VAR_confluent_cloud_api_secret="***REDACTED***"
➜  kafka-ops-team git:(master) ✗ terraform apply --auto-approve
...
Apply complete! Resources: 6 added, 0 changed, 0 destroyed.

Outputs:

resource-ids = <sensitive>
➜  kafka-ops-team git:(master) ✗ terraform output resource-ids
<<EOT
Environment ID:   env-gqq321

Service Accounts and their Cloud API Keys (API Keys inherit the permissions granted to the owner):
env-manager:                     sa-v7gnk5
env-manager's Cloud API Key:     "HUBPX5LV6CVWPPOL"
env-manager's Cloud API Secret:  "I8367Bs2VXTVKe+nvc54NlmWDOdyYAiuoAX0ioG3pz1f/o394viYwVaWFYnDHaEv"

Service Accounts with no roles assigned:
app-consumer:                    sa-gq6w51
app-producer:                    sa-pgy965

EOT
➜  env-admin-product-team git:(master) ✗ export TF_VAR_environment_id="env-gqq321"
➜  env-admin-product-team git:(master) ✗ export TF_VAR_confluent_cloud_api_key="HUBPX5LV6CVWPPOL" # created by Kafka Ops team
➜  env-admin-product-team git:(master) ✗ export TF_VAR_confluent_cloud_api_secret="I8367Bs2VXTVKe+nvc54NlmWDOdyYAiuoAX0ioG3pz1f/o394viYwVaWFYnDHaEv"
➜  env-admin-product-team git:(master) ✗ export TF_VAR_env_manager_id="sa-v7gnk5"
➜  env-admin-product-team git:(master) ✗ export TF_VAR_app_consumer_id="sa-pgy965"
➜  env-admin-product-team git:(master) ✗ export TF_VAR_app_producer_id="sa-gq6w51"
➜  env-admin-product-team git:(master) ✗ terraform apply --auto-approve
...
Apply complete! Resources: 8 added, 0 changed, 0 destroyed.

Outputs:

resource-ids = <sensitive>
➜  env-admin-product-team git:(master) ✗ terraform output resource-ids
<<EOT
Kafka Cluster ID: lkc-6kg1oq
Kafka topic name: orders

Service Accounts and their Kafka API Keys (API Keys inherit the permissions granted to the owner):
env-manager:                     sa-v7gnk5
env-manager's Kafka API Key:     "BVN4BNU3VECMBNWU"
env-manager's Kafka API Secret:  "9QcsPfjZYXMxH5AlxoKLnjOj6jFpuvqWFPdoTOz8Anf9wLOGOeZRK/0wggwtRfFv"

app-consumer:                    sa-gq6w51
app-consumer's Kafka API Key:    "OYLDWG7HYDRKAKPM"
app-consumer's Kafka API Secret: "6NqyCyQx1muA2QxJB1V6GBtBAE7bsn3DU0Kj4vAixc8TAEjmI+KZQGTp7znO/IQp"

app-producer:                    sa-pgy965
app-producer's Kafka API Key:    "MVFNKXPAKCSJ2W7C"
app-producer's Kafka API Secret: "RuIGnDrufGMZcL7Sv1uU+con3NJv5iaIGZE+NxNOPoJ9upoZzJ3cCxOCoH6fLk0W"

In order to use the Confluent CLI v2 to produce and consume messages from topic 'orders' using Kafka API Keys
of app-consumer and app-producer service accounts
run the following commands:

# 1. Log in to Confluent Cloud
$ confluent login

# 2. Produce key-value records to topic 'orders' by using app-consumer's Kafka API Key
$ confluent kafka topic produce orders --environment env-gqq321 --cluster lkc-6kg1oq --api-key "OYLDWG7HYDRKAKPM" --api-secret "6NqyCyQx1muA2QxJB1V6GBtBAE7bsn3DU0Kj4vAixc8TAEjmI+KZQGTp7znO/IQp"
# Enter a few records and then press 'Ctrl-C' when you're done.
# Sample records:
# {"number":1,"date":18500,"shipping_address":"899 W Evelyn Ave, Mountain View, CA 94041, USA","cost":15.00}
# {"number":2,"date":18501,"shipping_address":"1 Bedford St, London WC2E 9HG, United Kingdom","cost":5.00}
# {"number":3,"date":18502,"shipping_address":"3307 Northland Dr Suite 400, Austin, TX 78731, USA","cost":10.00}

# 3. Consume records from topic 'orders' by using app-producer's Kafka API Key
$ confluent kafka topic consume orders --from-beginning --environment env-gqq321 --cluster lkc-6kg1oq --api-key "MVFNKXPAKCSJ2W7C" --api-secret "RuIGnDrufGMZcL7Sv1uU+con3NJv5iaIGZE+NxNOPoJ9upoZzJ3cCxOCoH6fLk0W"
# When you are done, press 'Ctrl-C'.

EOT
```
