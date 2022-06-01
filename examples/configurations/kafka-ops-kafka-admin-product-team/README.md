### Quick Start

This example displays how TF config can be split between Kafka Ops team and Product team.

```bash
➜  kafka-ops-team git:(master) ✗ export TF_VAR_confluent_cloud_api_key="***REDACTED***" # with OrganizationAdmin permissions
➜  kafka-ops-team git:(master) ✗ export TF_VAR_confluent_cloud_api_secret="***REDACTED***"
➜  kafka-ops-team git:(master) ✗ terraform apply --auto-approve
...
Apply complete! Resources: 8 added, 0 changed, 0 destroyed.

Outputs:

resource-ids = <sensitive>
➜  kafka-ops-team git:(master) ✗ terraform output resource-ids
<<EOT
Environment ID:     env-k88o1p
Kafka cluster ID:   lkc-575nzg

Service Accounts with CloudClusterAdmin role and their API Keys (API Keys inherit the permissions granted to the owner):
app-manager:                     sa-570qrq
app-manager's Cloud API Key:     "IX3O2Q7QRWD4GSQD"
app-manager's Cloud API Secret:  "TKSrm4isgZhgPo4qWveuLCA3DqNSl6F4zXsy6Rboh7wrtp12GTeW/+Bmso0ZzONm"

app-manager's Kafka API Key:     "LNXKPG2IIT3IRYZ6"
app-manager's Kafka API Secret:  "ReNOUjwnCRlDspxkyD+uiOsf6v9vvoIsPM4vhcb8GyAcN+0EAh0uiSymKA/V4A5q"

Service Accounts with no roles assigned:
app-consumer:                    sa-6kmw28
app-producer:                    sa-domvq1


EOT
➜  kafka-admin-product-team git:(master) ✗ export TF_VAR_environment_id="env-k88o1p"
➜  kafka-admin-product-team git:(master) ✗ export TF_VAR_kafka_cluster_id="lkc-575nzg"
➜  kafka-admin-product-team git:(master) ✗ export TF_VAR_confluent_cloud_api_key="IX3O2Q7QRWD4GSQD" # created by Kafka Ops team
➜  kafka-admin-product-team git:(master) ✗ export TF_VAR_confluent_cloud_api_secret="TKSrm4isgZhgPo4qWveuLCA3DqNSl6F4zXsy6Rboh7wrtp12GTeW/+Bmso0ZzONm"
➜  kafka-admin-product-team git:(master) ✗ export TF_VAR_kafka_api_key="LNXKPG2IIT3IRYZ6" # created by Kafka Ops team
➜  kafka-admin-product-team git:(master) ✗ export TF_VAR_kafka_api_secret="ReNOUjwnCRlDspxkyD+uiOsf6v9vvoIsPM4vhcb8GyAcN+0EAh0uiSymKA/V4A5q"
➜  kafka-admin-product-team git:(master) ✗ export TF_VAR_app_manager_id="sa-570qrq"
➜  kafka-admin-product-team git:(master) ✗ export TF_VAR_app_consumer_id="sa-6kmw28"
➜  kafka-admin-product-team git:(master) ✗ export TF_VAR_app_producer_id="sa-domvq1"
➜  kafka-admin-product-team git:(master) ✗ terraform apply --auto-approve
...
Apply complete! Resources: 6 added, 0 changed, 0 destroyed.

Outputs:

resource-ids = <sensitive>
➜  kafka-admin-product-team git:(master) ✗ terraform output resource-ids
<<EOT
Kafka Cluster ID: lkc-575nzg
Kafka topic name: orders

Service Accounts and their Kafka API Keys (API Keys inherit the permissions granted to the owner):
app-producer:                    sa-domvq1
app-producer's Kafka API Key:    "ECPLDCE3L6XX6HWV"
app-producer's Kafka API Secret: "fT/66uULTJEN6ftT2FllvuJYPovvTiohz9ScHm3Cjws+QlvQgMCrJEPssntuDgd3"

app-consumer:                    sa-6kmw28
app-consumer's Kafka API Key:    "CDHNIZ3RBFK2VOE3"
app-consumer's Kafka API Secret: "semhBbPLv4+QIxe2ER3xaNMd8I7ZeSkPDwxRg0fmIsTqkO6VcW5JMjXeilg335v9"

In order to use the Confluent CLI v2 to produce and consume messages from topic 'orders' using Kafka API Keys
of app-producer and app-consumer service accounts
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
