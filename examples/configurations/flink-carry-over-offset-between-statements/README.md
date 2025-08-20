### Notes

1. Make sure to create a Flink table called `customers_source` before running this example.
    ```bash
    CREATE TABLE customers_source (
        customer_id INT,
        name STRING,
        address STRING,
        postcode STRING,
        city STRING,
        email STRING,
        PRIMARY KEY (customer_id) NOT ENFORCED
    );
   ```
2. Then run the following 2 statements:
    ```bash
    INSERT INTO customers_source (
        customer_id,
        name,
        address,
        postcode,
        city,
        email
    )
    SELECT
        customer_id,
        name,  
        address,
        postcode,
        city,
        email
    FROM examples.marketplace.customers;
    ```
    ```bash
    CREATE TABLE customers_sink (
        customer_id INT,
        name STRING,
        address STRING,
        postcode STRING,
        city STRING,
        email STRING,
        PRIMARY KEY (customer_id) NOT ENFORCED
    );
    ```
3. Apply this Terraform configuration by following the [Sample Project for Confluent Terraform Provider](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/sample-project) to start the initial statement.
4. Make the following changes to update the statement to the new version and restart automatically where the previous one stops:
   * Step #1: Uncomment `confluent_flink_statement.new` and run `terraform apply`.
   * Step #2: Stop `confluent_flink_statement.old` by setting `stopped = true` and run `terraform apply`.
   * Note: the new statement will automatically start from the last offsets of `confluent_flink_statement.old` once it's stopped.
5. In Confluent Cloud for Apache Flink®, an environment is mapped to a Flink catalog, a Kafka cluster is mapped to a Flink database, and a Kafka topic is mapped to a Flink table.
6. See [Sample Project for Confluent Terraform Provider](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/sample-project) that provides step-by-step instructions of running this example.
7. See [Flink SQL Quick Start with Confluent Cloud Console](https://docs.confluent.io/cloud/current/flink/get-started/quick-start-cloud-console.html#flink-sql-quick-start-with-ccloud-console) for more details about Flink Statements.
8. See [Grant Role-Based Access in Flink SQL](https://docs.confluent.io/cloud/current/flink/operate-and-deploy/flink-rbac.html) for more details about Grant Role-Based Access in Flink SQL.
9. See [Deploy a Flink SQL Statement using CI/CD](https://docs.confluent.io/cloud/current/flink/operate-and-deploy/deploy-flink-sql-statement.html) for more details about deploying a CI/CD workflow with GitHub Actions.
10. See [Example Data Streams](https://docs.confluent.io/cloud/current/flink/reference/example-data.html) to find more mock data streams that you can use for experimenting with Flink SQL queries.
