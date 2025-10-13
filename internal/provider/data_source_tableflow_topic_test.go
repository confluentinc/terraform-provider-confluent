package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

const (
	TableflowTopicDataSourceScenarioName = "confluent_tableflow_topic Data Source Lifecycle"
)

func TestAccDataSourceTableflowTopic(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockServerUrl := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	readTableflowTopicResponse, _ := os.ReadFile("../testdata/tableflow_topic/read_created_byob_aws_tt.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/tableflow/v1/tableflow-topics/topic_1")).
		InScenario(TableflowTopicDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readTableflowTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	TableflowTopicResourceName := "data.confluent_tableflow_topic.main"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceTableflowTopic(mockServerUrl, "topic_1"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "id", "topic_1"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "display_name", "topic_1"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "environment.#", "1"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "kafka_cluster.0.id", "lkc-00000"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "enable_compaction", "true"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "enable_partitioning", "true"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "suspended", "false"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "retention_ms", "100000000"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "record_failure_strategy", "SUSPEND"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "table_formats.#", "1"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "table_formats.0", "ICEBERG"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "byob_aws.#", "1"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "managed_storage.#", "0"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "byob_aws.0.bucket_name", "bucket_1"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "byob_aws.0.bucket_region", "us-east-1"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "byob_aws.0.provider_integration_id", "cspi-stgce89r7"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "table_path", "s3://dummy-bucket-name-1//10011010/11101100/org-1/env-2/lkc-3/v1/tableId"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "write_mode", "UPSERT"),
				),
			},
		},
	})
}

func TestAccDataSourceTableflowTopicErrorHandlingLog(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockServerUrl := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	readTableflowTopicResponse, _ := os.ReadFile("../testdata/tableflow_topic/read_created_error_handling_log_tt.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/tableflow/v1/tableflow-topics/topic_1")).
		InScenario(TableflowTopicDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readTableflowTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	TableflowTopicResourceName := "data.confluent_tableflow_topic.main"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceTableflowTopic(mockServerUrl, "topic_1"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "id", "topic_1"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "display_name", "topic_1"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "environment.#", "1"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "kafka_cluster.0.id", "lkc-00000"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "enable_compaction", "true"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "enable_partitioning", "true"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "suspended", "false"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "retention_ms", "100000000"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "error_handling_log.#", "1"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "error_handling_log.0.target", "dlq_topic_1"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "table_formats.#", "1"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "table_formats.0", "ICEBERG"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "byob_aws.#", "0"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "managed_storage.#", "1"),
					resource.TestCheckResourceAttr(TableflowTopicResourceName, "table_path", "s3://dummy-bucket-name-1//10011010/11101100/org-1/env-2/lkc-3/v1/tableId"),
				),
			},
		},
	})

}

func testAccCheckDataSourceTableflowTopic(mockServerUrl, resourceName string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	data "confluent_tableflow_topic" "main" {
	    display_name = "%s"
		environment {
			id = "env-abc123"
		}
		kafka_cluster {
			id = "lkc-00000"
		}
		credentials {
			key = "test_key"
			secret = "test_secret"
		}
	}
	`, mockServerUrl, resourceName)
}
