// Copyright 2025 Confluent Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"

	tableflowv1 "github.com/confluentinc/ccloud-sdk-go-v2/tableflow/v1"
)

func TestAccTableflowTopicByobAws(t *testing.T) {
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

	createTableflowTopicResponse, _ := os.ReadFile("../testdata/tableflow_topic/create_byob_aws_tt.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(tableflowTopicUrlPath)).
		InScenario(byobAwsTableflowTopicScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		//WillSetStateTo(scenarioStateTableflowTopicIsProvisioning).
		WillSetStateTo(scenarioStateTableflowTopicHasBeenCreated).
		WillReturn(
			string(createTableflowTopicResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	tableflowTopicReadUrlPath := fmt.Sprintf("%s/topic_1", tableflowTopicUrlPath)
	/*_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
	InScenario(byobAwsTableflowTopicScenarioName).
	WhenScenarioStateIs(scenarioStateTableflowTopicIsProvisioning).
	WillSetStateTo(scenarioStateTableflowTopicHasBeenCreated).
	WillReturn(
		string(createTableflowTopicResponse),
		contentTypeJSONHeader,
		http.StatusOK,
	))*/

	readCreatedTableflowTopicResponse, _ := os.ReadFile("../testdata/tableflow_topic/read_created_byob_aws_tt.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(byobAwsTableflowTopicScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowTopicHasBeenCreated).
		WillReturn(
			string(readCreatedTableflowTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	updatedTableflowTopicResponse, _ := os.ReadFile("../testdata/tableflow_topic/update_byob_aws_tt.json")
	_ = wiremockClient.StubFor(wiremock.Patch(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(byobAwsTableflowTopicScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowTopicHasBeenCreated).
		WillSetStateTo(scenarioStateTableflowTopicHasBeenUpdated).
		WillReturn(
			string(updatedTableflowTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(byobAwsTableflowTopicScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowTopicHasBeenUpdated).
		WillReturn(
			string(updatedTableflowTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(byobAwsTableflowTopicScenarioName).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceTableflowTopicByobAws(mockServerUrl, 100000000),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "id", "topic_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "display_name", "topic_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "kafka_cluster.0.id", "lkc-00000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "enable_compaction", "true"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "enable_partitioning", "true"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "suspended", "false"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "retention_ms", "100000000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "data_retention_ms", "2592000000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "record_failure_strategy", "SUSPEND"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "metadata_column_naming_scheme", "DEFAULT"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "error_handling.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "error_handling.0.mode", "SUSPEND"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.0", "ICEBERG"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "byob_aws.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "managed_storage.#", "0"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "azure_data_lake_storage_gen_2.#", "0"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "byob_aws.0.bucket_name", "bucket_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "byob_aws.0.bucket_region", "us-east-1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "byob_aws.0.provider_integration_id", "cspi-stgce89r7"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_path", "s3://dummy-bucket-name-1//10011010/11101100/org-1/env-2/lkc-3/v1/tableId"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "write_mode", "UPSERT"),
				),
			},
			{
				// Rotating credentials.key / credentials.secret alone must update in place rather
				// than sending an unnecessary UpdateTableflowV1TableflowTopic request to the backend.
				Config: testAccCheckResourceTableflowTopicByobAwsCredentialsRotated(mockServerUrl, 100000000),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "id", "topic_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "retention_ms", "100000000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "credentials.0.key", kafkaApiKeyRotated),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "credentials.0.secret", kafkaApiSecretRotated),
				),
			},
			{
				// Rotate back to the original credentials to confirm the reverse direction also updates
				// in place, and to keep the remaining steps in sync.
				Config: testAccCheckResourceTableflowTopicByobAws(mockServerUrl, 100000000),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "credentials.0.key", "test_key"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "credentials.0.secret", "test_secret"),
				),
			},
			{
				Config: testAccCheckResourceTableflowTopicByobAwsUpdate(mockServerUrl, 200000000),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "id", "topic_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "display_name", "topic_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "kafka_cluster.0.id", "lkc-00000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "enable_compaction", "true"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "enable_partitioning", "true"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "suspended", "false"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "retention_ms", "200000000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "data_retention_ms", "2592000000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "record_failure_strategy", "SKIP"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "metadata_column_naming_scheme", "PORTABLE"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "error_handling.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "error_handling.0.mode", "SKIP"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.0", "ICEBERG"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "byob_aws.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "managed_storage.#", "0"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "azure_data_lake_storage_gen_2.#", "0"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "byob_aws.0.bucket_name", "bucket_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "byob_aws.0.bucket_region", "us-east-1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "byob_aws.0.provider_integration_id", "cspi-stgce89r7"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_path", "s3://dummy-bucket-name-1//10011010/11101100/org-1/env-2/lkc-3/v1/tableId"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "write_mode", "UPSERT"),
				),
			},
		},
	})
}

func TestAccTableflowTopicAzure(t *testing.T) {
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

	createTableflowTopicResponse, _ := os.ReadFile("../testdata/tableflow_topic/create_azure_tt.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(tableflowTopicUrlPath)).
		InScenario(azureTableflowTopicScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		//WillSetStateTo(scenarioStateTableflowTopicIsProvisioning).
		WillSetStateTo(scenarioStateTableflowTopicHasBeenCreated).
		WillReturn(
			string(createTableflowTopicResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	tableflowTopicReadUrlPath := fmt.Sprintf("%s/topic_1", tableflowTopicUrlPath)

	readCreatedTableflowTopicResponse, _ := os.ReadFile("../testdata/tableflow_topic/read_created_azure_tt.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(azureTableflowTopicScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowTopicHasBeenCreated).
		WillReturn(
			string(readCreatedTableflowTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	updatedTableflowTopicResponse, _ := os.ReadFile("../testdata/tableflow_topic/update_azure_tt.json")
	_ = wiremockClient.StubFor(wiremock.Patch(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(azureTableflowTopicScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowTopicHasBeenCreated).
		WillSetStateTo(scenarioStateTableflowTopicHasBeenUpdated).
		WillReturn(
			string(updatedTableflowTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(azureTableflowTopicScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowTopicHasBeenUpdated).
		WillReturn(
			string(updatedTableflowTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(azureTableflowTopicScenarioName).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceTableflowTopicAzure(mockServerUrl, 100000000),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "id", "topic_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "display_name", "topic_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "kafka_cluster.0.id", "lkc-00000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "enable_compaction", "true"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "enable_partitioning", "true"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "suspended", "false"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "retention_ms", "100000000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "data_retention_ms", "2592000000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "record_failure_strategy", "SUSPEND"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.0", "ICEBERG"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "byob_aws.#", "0"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "managed_storage.#", "0"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "azure_data_lake_storage_gen_2.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "azure_data_lake_storage_gen_2.0.container_name", "Container1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "azure_data_lake_storage_gen_2.0.storage_account_name", "Acc1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "azure_data_lake_storage_gen_2.0.storage_region", "US1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "azure_data_lake_storage_gen_2.0.provider_integration_id", "cspi-stgce89r7"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_path", "s3://dummy-bucket-name-1//10011010/11101100/org-1/env-2/lkc-3/v1/tableId2"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "write_mode", "UPSERT"),
				),
			},
			{
				Config: testAccCheckResourceTableflowTopicAzureStorageUpdate(mockServerUrl, 200000000),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "id", "topic_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "display_name", "topic_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "kafka_cluster.0.id", "lkc-00000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "enable_compaction", "true"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "enable_partitioning", "true"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "suspended", "false"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "retention_ms", "200000000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "data_retention_ms", "2592000000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "record_failure_strategy", "SKIP"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.0", "ICEBERG"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "byob_aws.#", "0"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "managed_storage.#", "0"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "azure_data_lake_storage_gen_2.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "azure_data_lake_storage_gen_2.0.container_name", "Container1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "azure_data_lake_storage_gen_2.0.storage_account_name", "Acc1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "azure_data_lake_storage_gen_2.0.storage_region", "US1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "azure_data_lake_storage_gen_2.0.provider_integration_id", "cspi-stgce89r7"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_path", "s3://dummy-bucket-name-1//10011010/11101100/org-1/env-2/lkc-3/v1/tableId2"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "write_mode", "UPSERT"),
				),
			},
		},
	})
}

func TestAccTableflowTopicGcs(t *testing.T) {
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

	createTableflowTopicResponse, _ := os.ReadFile("../testdata/tableflow_topic/create_gcs_tt.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(tableflowTopicUrlPath)).
		InScenario(gcsTableflowTopicScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateTableflowTopicHasBeenCreated).
		WillReturn(
			string(createTableflowTopicResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	tableflowTopicReadUrlPath := fmt.Sprintf("%s/topic_1", tableflowTopicUrlPath)

	readCreatedTableflowTopicResponse, _ := os.ReadFile("../testdata/tableflow_topic/read_created_gcs_tt.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(gcsTableflowTopicScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowTopicHasBeenCreated).
		WillReturn(
			string(readCreatedTableflowTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	updatedTableflowTopicResponse, _ := os.ReadFile("../testdata/tableflow_topic/update_gcs_tt.json")
	_ = wiremockClient.StubFor(wiremock.Patch(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(gcsTableflowTopicScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowTopicHasBeenCreated).
		WillSetStateTo(scenarioStateTableflowTopicHasBeenUpdated).
		WillReturn(
			string(updatedTableflowTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(gcsTableflowTopicScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowTopicHasBeenUpdated).
		WillReturn(
			string(updatedTableflowTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(gcsTableflowTopicScenarioName).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceTableflowTopicGcs(mockServerUrl, 100000000),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "id", "topic_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "display_name", "topic_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "kafka_cluster.0.id", "lkc-00000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "enable_compaction", "true"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "enable_partitioning", "true"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "suspended", "false"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "retention_ms", "100000000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "data_retention_ms", "2592000000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "record_failure_strategy", "SUSPEND"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.0", "ICEBERG"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "byob_aws.#", "0"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "managed_storage.#", "0"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "azure_data_lake_storage_gen_2.#", "0"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "google_cloud_storage.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "google_cloud_storage.0.bucket_name", "bucket_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "google_cloud_storage.0.bucket_region", "us-central1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "google_cloud_storage.0.provider_integration_id", "cspi-stgce89r7"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_path", "gs://bucket_1/10011010/11101100/org-1/env-2/lkc-3/v1/tableId"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "write_mode", "UPSERT"),
				),
			},
			{
				Config: testAccCheckResourceTableflowTopicGcsStorageUpdate(mockServerUrl, 200000000),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "id", "topic_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "retention_ms", "200000000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "record_failure_strategy", "SKIP"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "google_cloud_storage.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "google_cloud_storage.0.bucket_name", "bucket_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "google_cloud_storage.0.bucket_region", "us-central1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "google_cloud_storage.0.provider_integration_id", "cspi-stgce89r7"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_path", "gs://bucket_1/10011010/11101100/org-1/env-2/lkc-3/v1/tableId"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "write_mode", "UPSERT"),
				),
			},
		},
	})
}

func testAccCheckResourceTableflowTopicGcs(mockServerUrl string, retention int) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_tableflow_topic" "main" {
		display_name = "topic_1"
		retention_ms = %d
		environment {
			id = "env-abc123"
		}
		kafka_cluster {
			id = "lkc-00000"
		}

		google_cloud_storage {
			provider_integration_id = "cspi-stgce89r7"
      		bucket_name = "bucket_1"
		}
		credentials {
			key = "test_key"
			secret = "test_secret"
		}
	}
	`, mockServerUrl, retention)
}

func testAccCheckResourceTableflowTopicGcsStorageUpdate(mockServerUrl string, retention int) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_tableflow_topic" "main" {
		display_name = "topic_1"
		retention_ms = %d
		environment {
			id = "env-abc123"
		}
		kafka_cluster {
			id = "lkc-00000"
		}
		google_cloud_storage {
			provider_integration_id = "cspi-stgce89r7"
      		bucket_name = "bucket_1"
		}
		credentials {
			key = "test_key"
			secret = "test_secret"
		}
	}
	`, mockServerUrl, retention)
}

func TestAccTableflowTopicManagedStorage(t *testing.T) {
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

	createTableflowTopicResponse, _ := os.ReadFile("../testdata/tableflow_topic/create_managed_storage_tt.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(tableflowTopicUrlPath)).
		InScenario(managedStorageTableflowTopicScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		//WillSetStateTo(scenarioStateTableflowTopicIsProvisioning).
		WillSetStateTo(scenarioStateTableflowTopicHasBeenCreated).
		WillReturn(
			string(createTableflowTopicResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	tableflowTopicReadUrlPath := fmt.Sprintf("%s/topic_1", tableflowTopicUrlPath)
	/*_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
	InScenario(managedStorageTableflowTopicScenarioName).
	WhenScenarioStateIs(scenarioStateTableflowTopicIsProvisioning).
	WillSetStateTo(scenarioStateTableflowTopicHasBeenCreated).
	WillReturn(
		string(createTableflowTopicResponse),
		contentTypeJSONHeader,
		http.StatusOK,
	))*/

	readCreatedTableflowTopicResponse, _ := os.ReadFile("../testdata/tableflow_topic/read_created_managed_storage_tt.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(managedStorageTableflowTopicScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowTopicHasBeenCreated).
		WillReturn(
			string(readCreatedTableflowTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	updatedTableflowTopicResponse, _ := os.ReadFile("../testdata/tableflow_topic/update_managed_storage_tt.json")
	// The update changes table_formats without table_format_properties in the configuration. The plan marks
	// table_format_properties as known after apply, and the request must not send it.
	_ = wiremockClient.StubFor(wiremock.Patch(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(managedStorageTableflowTopicScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowTopicHasBeenCreated).
		WithBodyPattern(wiremock.MatchingJsonPath(`$.spec.table_formats[?(@ == 'DELTA')]`)).
		WithBodyPattern(wiremock.NotMatching(`.*"table_format_properties".*`)).
		WillSetStateTo(scenarioStateTableflowTopicHasBeenUpdated).
		WillReturn(
			string(updatedTableflowTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(managedStorageTableflowTopicScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowTopicHasBeenUpdated).
		WillReturn(
			string(updatedTableflowTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(managedStorageTableflowTopicScenarioName).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceTableflowTopicManagedStorage(mockServerUrl, 100000000),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "id", "topic_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "display_name", "topic_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "kafka_cluster.0.id", "lkc-00000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "enable_compaction", "true"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "enable_partitioning", "true"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "suspended", "false"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "retention_ms", "100000000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "data_retention_ms", "2592000000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "record_failure_strategy", "SKIP"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "error_handling.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "error_handling.0.mode", "SKIP"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.0", "ICEBERG"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "byob_aws.#", "0"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "managed_storage.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "azure_data_lake_storage_gen_2.#", "0"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_path", "s3://dummy-bucket-name-1//10011010/11101100/org-1/env-2/lkc-3/v1/tableId"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "write_mode", "UPSERT"),
				),
			},
			{
				Config: testAccCheckResourceTableflowTopicManagedStorageUpdate(mockServerUrl, 200000000),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "id", "topic_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "display_name", "topic_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "kafka_cluster.0.id", "lkc-00000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "enable_compaction", "true"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "enable_partitioning", "true"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "suspended", "false"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "retention_ms", "200000000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "data_retention_ms", "2592000000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "error_handling.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "error_handling.0.mode", "LOG"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "error_handling.0.log_target", "log_topic"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.#", "2"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.0", "DELTA"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.1", "ICEBERG"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "byob_aws.#", "0"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "managed_storage.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "azure_data_lake_storage_gen_2.#", "0"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_path", "s3://dummy-bucket-name-1//10011010/11101100/org-1/env-2/lkc-3/v1/tableId"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "write_mode", "UPSERT"),
				),
			},
		},
	})
}

func TestAccTableflowTopicErrorHandlingLog(t *testing.T) {
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

	createTableflowTopicResponse, _ := os.ReadFile("../testdata/tableflow_topic/create_error_handling.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(tableflowTopicUrlPath)).
		InScenario(managedStorageTableflowTopicScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateTableflowTopicHasBeenCreated).
		WillReturn(
			string(createTableflowTopicResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	tableflowTopicReadUrlPath := fmt.Sprintf("%s/topic_1", tableflowTopicUrlPath)
	readCreatedTableflowTopicResponse, _ := os.ReadFile("../testdata/tableflow_topic/read_created_error_handling.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(managedStorageTableflowTopicScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowTopicHasBeenCreated).
		WillReturn(
			string(readCreatedTableflowTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	updatedTableflowTopicResponse, _ := os.ReadFile("../testdata/tableflow_topic/update_error_handling.json")
	_ = wiremockClient.StubFor(wiremock.Patch(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(managedStorageTableflowTopicScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowTopicHasBeenCreated).
		WillSetStateTo(scenarioStateTableflowTopicHasBeenUpdated).
		WillReturn(
			string(updatedTableflowTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(managedStorageTableflowTopicScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowTopicHasBeenUpdated).
		WillReturn(
			string(updatedTableflowTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(managedStorageTableflowTopicScenarioName).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceTableflowTopicErrorHandling(mockServerUrl, "LOG", "log_topic"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "id", "topic_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "display_name", "topic_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "kafka_cluster.0.id", "lkc-00000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "enable_compaction", "true"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "enable_partitioning", "true"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "suspended", "false"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "retention_ms", "100000000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "data_retention_ms", "2592000000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "error_handling.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "error_handling.0.mode", "LOG"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "error_handling.0.log_target", "log_topic"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.0", "ICEBERG"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "byob_aws.#", "0"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "managed_storage.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_path", "s3://dummy-bucket-name-1//10011010/11101100/org-1/env-2/lkc-3/v1/tableId"),
				),
			},
			{
				Config: testAccCheckResourceTableflowTopicErrorHandling(mockServerUrl, "SUSPEND", "log_topic"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "id", "topic_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "display_name", "topic_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "kafka_cluster.0.id", "lkc-00000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "enable_compaction", "true"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "enable_partitioning", "true"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "suspended", "false"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "retention_ms", "100000000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "data_retention_ms", "2592000000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "error_handling.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "error_handling.0.mode", "SUSPEND"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.0", "ICEBERG"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "byob_aws.#", "0"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "managed_storage.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_path", "s3://dummy-bucket-name-1//10011010/11101100/org-1/env-2/lkc-3/v1/tableId"),
				),
			},
		},
	})
}

func testAccCheckResourceTableflowTopicByobAws(mockServerUrl string, retention int) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_tableflow_topic" "main" {
		display_name = "topic_1"
		retention_ms = %d
		metadata_column_naming_scheme = "DEFAULT"
		error_handling {
			mode = "SUSPEND"
		}
		environment {
			id = "env-abc123"
		}
		kafka_cluster {
			id = "lkc-00000"
		}
		byob_aws {
			bucket_name = "bucket_1"
			provider_integration_id = "cspi-stgce89r7"
		}
		credentials {
			key = "test_key"
			secret = "test_secret"
		}
	}
	`, mockServerUrl, retention)
}

func testAccCheckResourceTableflowTopicByobAwsCredentialsRotated(mockServerUrl string, retention int) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_tableflow_topic" "main" {
		display_name = "topic_1"
		retention_ms = %d
		metadata_column_naming_scheme = "DEFAULT"
		error_handling {
			mode = "SUSPEND"
		}
		environment {
			id = "env-abc123"
		}
		kafka_cluster {
			id = "lkc-00000"
		}
		byob_aws {
			bucket_name = "bucket_1"
			provider_integration_id = "cspi-stgce89r7"
		}
		credentials {
			key = "%s"
			secret = "%s"
		}
	}
	`, mockServerUrl, retention, kafkaApiKeyRotated, kafkaApiSecretRotated)
}

func testAccCheckResourceTableflowTopicByobAwsUpdate(mockServerUrl string, retention int) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_tableflow_topic" "main" {
		display_name = "topic_1"
		retention_ms = %d
		metadata_column_naming_scheme = "PORTABLE"
		error_handling {
			mode = "SKIP"
		}
		environment {
			id = "env-abc123"
		}
		kafka_cluster {
			id = "lkc-00000"
		}
		byob_aws {
			bucket_name = "bucket_1"
			provider_integration_id = "cspi-stgce89r7"
		}
		credentials {
			key = "test_key"
			secret = "test_secret"
		}
	}
	`, mockServerUrl, retention)
}

func testAccCheckResourceTableflowTopicManagedStorage(mockServerUrl string, retention int) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_tableflow_topic" "main" {
		display_name = "topic_1"
		retention_ms = %d
		error_handling {
			mode = "SKIP"
		}
		environment {
			id = "env-abc123"
		}
		kafka_cluster {
			id = "lkc-00000"
		}
		managed_storage {}
		credentials {
			key = "test_key"
			secret = "test_secret"
		}
	}
	`, mockServerUrl, retention)
}

func testAccCheckResourceTableflowTopicManagedStorageUpdate(mockServerUrl string, retention int) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_tableflow_topic" "main" {
		display_name = "topic_1"
		retention_ms = %d
		error_handling {
			mode = "LOG"
			log_target = "log_topic"
		}
		environment {
			id = "env-abc123"
		}
		kafka_cluster {
			id = "lkc-00000"
		}
		table_formats = ["ICEBERG", "DELTA"]
		managed_storage {}
		credentials {
			key = "test_key"
			secret = "test_secret"
		}
	}
	`, mockServerUrl, retention)
}

func testAccCheckResourceTableflowTopicAzure(mockServerUrl string, retention int) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_tableflow_topic" "main" {
		display_name = "topic_1"
		retention_ms = %d
		environment {
			id = "env-abc123"
		}
		kafka_cluster {
			id = "lkc-00000"
		}

		azure_data_lake_storage_gen_2 {
			provider_integration_id = "cspi-stgce89r7"
      		container_name = "Container1"
      		storage_account_name = "Acc1"
		}
		credentials {
			key = "test_key"
			secret = "test_secret"
		}
	}
	`, mockServerUrl, retention)
}

func testAccCheckResourceTableflowTopicAzureStorageUpdate(mockServerUrl string, retention int) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_tableflow_topic" "main" {
		display_name = "topic_1"
		retention_ms = %d
		environment {
			id = "env-abc123"
		}
		kafka_cluster {
			id = "lkc-00000"
		}
		azure_data_lake_storage_gen_2 {
			provider_integration_id = "cspi-stgce89r7"
      		container_name = "Container1"
      		storage_account_name = "Acc1"
		}
		credentials {
			key = "test_key"
			secret = "test_secret"
		}
	}
	`, mockServerUrl, retention)
}

func testAccCheckResourceTableflowTopicErrorHandling(mockServerUrl, mode, logTarget string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_tableflow_topic" "main" {
		display_name = "topic_1"
		retention_ms = 100000000
		environment {
			id = "env-abc123"
		}
		kafka_cluster {
			id = "lkc-00000"
		}
		managed_storage {}
		error_handling {
			mode = "%s"
			log_target = "%s"
		}
		credentials {
			key = "test_key"
			secret = "test_secret"
		}
	}
	`, mockServerUrl, mode, logTarget)
}

func TestAccTableflowTopicIcebergV3(t *testing.T) {
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

	// The body patterns make sure that the create request sends only the configured V3 fields: the iceberg sub-block
	// and the VARIANT entry without shredding. A request that does not match gets the WireMock 404, and the test fails.
	createTableflowTopicResponse, _ := os.ReadFile("../testdata/tableflow_topic/create_iceberg_v3_tt.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(tableflowTopicUrlPath)).
		InScenario(byobAwsTableflowTopicScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WithBodyPattern(wiremock.MatchingJsonPath(`$.spec.table_format_properties[?(@.format == 'ICEBERG' && @.enabled == true && @.format_version == 3)]`)).
		WithBodyPattern(wiremock.MatchingJsonPath(`$.spec.config.features[?(@.feature == 'VARIANT' && @.enabled == true)]`)).
		WithBodyPattern(wiremock.NotMatching(`.*"DELTA".*`)).
		WithBodyPattern(wiremock.NotMatching(`.*"shredding".*`)).
		WillSetStateTo(scenarioStateTableflowTopicHasBeenCreated).
		WillReturn(
			string(createTableflowTopicResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	tableflowTopicReadUrlPath := fmt.Sprintf("%s/topic_1", tableflowTopicUrlPath)
	readCreatedTableflowTopicResponse, _ := os.ReadFile("../testdata/tableflow_topic/read_created_iceberg_v3_tt.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(byobAwsTableflowTopicScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowTopicHasBeenCreated).
		WillReturn(
			string(readCreatedTableflowTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// The update request sends the changed V3 fields, and does not send table_formats, because table_formats is not
	// in the configuration and did not change.
	updatedTableflowTopicResponse, _ := os.ReadFile("../testdata/tableflow_topic/update_iceberg_v3_tt.json")
	_ = wiremockClient.StubFor(wiremock.Patch(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(byobAwsTableflowTopicScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowTopicHasBeenCreated).
		WithBodyPattern(wiremock.MatchingJsonPath(`$.spec.table_format_properties[?(@.format == 'DELTA' && @.enabled == true)]`)).
		WithBodyPattern(wiremock.MatchingJsonPath(`$.spec.config.features[?(@.feature == 'VARIANT' && @.enabled == true && @.shredding.enabled == true)]`)).
		WithBodyPattern(wiremock.NotMatching(`.*"table_formats".*`)).
		WillSetStateTo(scenarioStateTableflowTopicHasBeenUpdated).
		WillReturn(
			string(updatedTableflowTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedTableflowTopicResponse, _ := os.ReadFile("../testdata/tableflow_topic/read_updated_iceberg_v3_tt.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(byobAwsTableflowTopicScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowTopicHasBeenUpdated).
		WillReturn(
			string(readUpdatedTableflowTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(byobAwsTableflowTopicScenarioName).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	createConfig := testAccCheckResourceTableflowTopicIcebergV3(mockServerUrl, 100000000,
		testAccTableFormatPropertiesConfig(3, nil)+testAccVariantFeatureConfig(true, nil))
	updateTableFormatProperties := testAccTableFormatPropertiesConfig(3, testBoolPtr(true))
	updateConfig := testAccCheckResourceTableflowTopicIcebergV3(mockServerUrl, 100000000,
		updateTableFormatProperties+testAccVariantFeatureConfig(true, testBoolPtr(true)))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: createConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "id", "topic_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.0", "ICEBERG"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_format_properties.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_format_properties.0.iceberg.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_format_properties.0.iceberg.0.enabled", "true"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_format_properties.0.iceberg.0.format_version", "3"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_format_properties.0.delta.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_format_properties.0.delta.0.enabled", "false"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "features.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "features.0.feature", "VARIANT"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "features.0.enabled", "true"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "features.0.shredding.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "features.0.shredding.0.enabled", "false"),
				),
			},
			{
				// The computed shredding value does not cause a diff.
				Config:             createConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: testAccCheckResourceTableflowTopicIcebergV3(mockServerUrl, 100000000,
					`table_formats = ["ICEBERG"]`+testAccTableFormatPropertiesConfig(3, nil)),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`"table_formats": conflicts with table_format_properties`),
			},
			{
				Config: testAccCheckResourceTableflowTopicIcebergV3(mockServerUrl, 100000000,
					testAccTableFormatPropertiesConfig(2, nil)+testAccVariantFeatureConfig(true, nil)),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("format_version can't be lowered from 3 to 2"),
			},
			{
				Config: testAccCheckResourceTableflowTopicIcebergV3(mockServerUrl, 100000000,
					testAccTableFormatPropertiesConfig(3, nil)+testAccVariantFeatureConfig(false, testBoolPtr(true))),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("shredding can't be enabled while VARIANT is disabled"),
			},
			{
				Config: updateConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "id", "topic_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.#", "2"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_format_properties.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_format_properties.0.iceberg.0.enabled", "true"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_format_properties.0.iceberg.0.format_version", "3"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_format_properties.0.delta.0.enabled", "true"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "features.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "features.0.feature", "VARIANT"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "features.0.enabled", "true"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "features.0.shredding.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "features.0.shredding.0.enabled", "true"),
				),
			},
			{
				// The removal of the features block stops the tracking of features. It does not disable VARIANT.
				Config:             testAccCheckResourceTableflowTopicIcebergV3(mockServerUrl, 100000000, updateTableFormatProperties),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// TestAccTableflowTopicTableFormatPropertiesDefaults covers configurations without format_version: the server value
// gives no diff, an update of another attribute sends no V3 field, a disabled ICEBERG keeps the stored version, and a
// configuration with only table_formats gives no diff.
func TestAccTableflowTopicTableFormatPropertiesDefaults(t *testing.T) {
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

	// The create request does not send format_version, because it is not in the configuration. The server sets 3.
	createTableflowTopicResponse, _ := os.ReadFile("../testdata/tableflow_topic/create_iceberg_v3_tt.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(tableflowTopicUrlPath)).
		InScenario(byobAwsTableflowTopicScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WithBodyPattern(wiremock.MatchingJsonPath(`$.spec.table_format_properties[?(@.format == 'ICEBERG' && @.enabled == true)]`)).
		WithBodyPattern(wiremock.MatchingJsonPath(`$.spec.config.features[?(@.feature == 'VARIANT' && @.enabled == true)]`)).
		WithBodyPattern(wiremock.NotMatching(`.*"format_version".*`)).
		WillSetStateTo(scenarioStateTableflowTopicHasBeenCreated).
		WillReturn(
			string(createTableflowTopicResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	tableflowTopicReadUrlPath := fmt.Sprintf("%s/topic_1", tableflowTopicUrlPath)
	readCreatedTableflowTopicResponse, _ := os.ReadFile("../testdata/tableflow_topic/read_created_iceberg_v3_tt.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(byobAwsTableflowTopicScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowTopicHasBeenCreated).
		WillReturn(
			string(readCreatedTableflowTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// An update of only retention_ms sends no V3 field, because the V3 blocks did not change.
	readUpdatedRetentionResponse, _ := os.ReadFile("../testdata/tableflow_topic/read_updated_retention_iceberg_v3_tt.json")
	_ = wiremockClient.StubFor(wiremock.Patch(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(byobAwsTableflowTopicScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowTopicHasBeenCreated).
		WithBodyPattern(wiremock.MatchingJsonPath(`$.spec.config[?(@.retention_ms == '200000000')]`)).
		WithBodyPattern(wiremock.NotMatching(`.*"table_format_properties".*`)).
		WithBodyPattern(wiremock.NotMatching(`.*"features".*`)).
		WithBodyPattern(wiremock.NotMatching(`.*"table_formats".*`)).
		WillSetStateTo(scenarioStateTableflowTopicHasBeenUpdated).
		WillReturn(
			string(readUpdatedRetentionResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(byobAwsTableflowTopicScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowTopicHasBeenUpdated).
		WillReturn(
			string(readUpdatedRetentionResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// The update request disables ICEBERG and enables DELTA. It does not send format_version, features or
	// table_formats, because they are not in the configuration or did not change.
	readIcebergDisabledResponse, _ := os.ReadFile("../testdata/tableflow_topic/read_iceberg_disabled_iceberg_v3_tt.json")
	_ = wiremockClient.StubFor(wiremock.Patch(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(byobAwsTableflowTopicScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowTopicHasBeenUpdated).
		WithBodyPattern(wiremock.MatchingJsonPath(`$.spec.table_format_properties[?(@.format == 'ICEBERG' && @.enabled == false)]`)).
		WithBodyPattern(wiremock.MatchingJsonPath(`$.spec.table_format_properties[?(@.format == 'DELTA' && @.enabled == true)]`)).
		WithBodyPattern(wiremock.NotMatching(`.*"format_version".*`)).
		WithBodyPattern(wiremock.NotMatching(`.*"features".*`)).
		WithBodyPattern(wiremock.NotMatching(`.*"table_formats".*`)).
		WillSetStateTo(scenarioStateTableflowTopicIcebergHasBeenDisabled).
		WillReturn(
			string(readIcebergDisabledResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(byobAwsTableflowTopicScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowTopicIcebergHasBeenDisabled).
		WillReturn(
			string(readIcebergDisabledResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(tableflowTopicReadUrlPath)).
		InScenario(byobAwsTableflowTopicScenarioName).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	icebergWithoutVersion := `
		table_format_properties {
			iceberg {
				enabled = true
			}
		}` + testAccVariantFeatureConfig(true, nil)
	icebergDisabled := `
		table_format_properties {
			iceberg {
				enabled = false
			}
			delta {
				enabled = true
			}
		}` + testAccVariantFeatureConfig(true, nil)
	createConfig := testAccCheckResourceTableflowTopicIcebergV3(mockServerUrl, 100000000, icebergWithoutVersion)
	retentionConfig := testAccCheckResourceTableflowTopicIcebergV3(mockServerUrl, 200000000, icebergWithoutVersion)
	icebergDisabledConfig := testAccCheckResourceTableflowTopicIcebergV3(mockServerUrl, 200000000, icebergDisabled)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: createConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "id", "topic_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_format_properties.0.iceberg.0.enabled", "true"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_format_properties.0.iceberg.0.format_version", "3"),
				),
			},
			{
				// The server value of format_version does not cause a diff.
				Config:             createConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: retentionConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "retention_ms", "200000000"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_format_properties.0.iceberg.0.enabled", "true"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_format_properties.0.iceberg.0.format_version", "3"),
				),
			},
			{
				Config: icebergDisabledConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.0", "DELTA"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_format_properties.0.iceberg.0.enabled", "false"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_format_properties.0.iceberg.0.format_version", "3"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_format_properties.0.delta.0.enabled", "true"),
				),
			},
			{
				// The stored format_version of a disabled ICEBERG does not cause a diff.
				Config:             icebergDisabledConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				// A configuration with only table_formats, as before Iceberg V3, does not cause a diff.
				Config:             testAccCheckResourceTableflowTopicIcebergV3(mockServerUrl, 200000000, `table_formats = ["DELTA"]`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestTableflowTopicJsonForLog(t *testing.T) {
	t.Run("a topic with known entries gives its JSON", func(t *testing.T) {
		topic := tableflowv1.TableflowV1TableflowTopic{Kind: tableflowv1.PtrString("TableflowTopic")}
		if got, want := tableflowTopicJsonForLog(topic), `{"kind":"TableflowTopic"}`; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
	t.Run("a topic with an unknown feature gives the marshal error", func(t *testing.T) {
		var topic tableflowv1.TableflowV1TableflowTopic
		body := `{"spec":{"config":{"features":[{"feature":"NON_NULL_DEFAULTS","enabled":true}]}}}`
		if err := json.Unmarshal([]byte(body), &topic); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got := tableflowTopicJsonForLog(topic); !strings.HasPrefix(got, "<can't marshal to JSON: ") {
			t.Errorf("got %q, want the marshal error", got)
		}
	})
}

// testAccCheckResourceTableflowTopicIcebergV3 returns the configuration of testAccCheckResourceTableflowTopicByobAws
// with retentionMs and the attributes in extraConfig.
func testAccCheckResourceTableflowTopicIcebergV3(mockServerUrl string, retentionMs int, extraConfig string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_tableflow_topic" "main" {
		display_name = "topic_1"
		retention_ms = %d
		metadata_column_naming_scheme = "DEFAULT"
		error_handling {
			mode = "SUSPEND"
		}
		environment {
			id = "env-abc123"
		}
		kafka_cluster {
			id = "lkc-00000"
		}
		byob_aws {
			bucket_name = "bucket_1"
			provider_integration_id = "cspi-stgce89r7"
		}
		credentials {
			key = "test_key"
			secret = "test_secret"
		}
		%s
	}
	`, mockServerUrl, retentionMs, extraConfig)
}

// testAccTableFormatPropertiesConfig returns a table_format_properties block with an enabled iceberg sub-block at
// formatVersion. It has a delta sub-block only when deltaEnabled is not nil.
func testAccTableFormatPropertiesConfig(formatVersion int, deltaEnabled *bool) string {
	delta := ""
	if deltaEnabled != nil {
		delta = fmt.Sprintf(`
			delta {
				enabled = %t
			}`, *deltaEnabled)
	}
	return fmt.Sprintf(`
		table_format_properties {
			iceberg {
				enabled        = true
				format_version = %d
			}%s
		}`, formatVersion, delta)
}

// testAccVariantFeatureConfig returns a features block for VARIANT. It has a shredding block only when shredding is
// not nil.
func testAccVariantFeatureConfig(enabled bool, shredding *bool) string {
	shreddingBlock := ""
	if shredding != nil {
		shreddingBlock = fmt.Sprintf(`
			shredding {
				enabled = %t
			}`, *shredding)
	}
	return fmt.Sprintf(`
		features {
			feature = "VARIANT"
			enabled = %t%s
		}`, enabled, shreddingBlock)
}

func testBoolPtr(v bool) *bool {
	return &v
}

// checkTableflowTopicError reports a test error when err is not the error want. An empty want means no error. call is
// the name of the checked call in the report.
func checkTableflowTopicError(t *testing.T, call string, err error, want string) {
	t.Helper()
	switch {
	case want == "" && err != nil:
		t.Errorf("%s: got %q, want nil", call, err)
	case want != "" && err == nil:
		t.Errorf("%s: got nil, want %q", call, want)
	case want != "" && err.Error() != want:
		t.Errorf("%s: got %q, want %q", call, err, want)
	}
}

// tableflowTopicTestResource has the attributes that the CustomizeDiff functions of table_format_properties, features
// and table_formats read, and display_name as a ForceNew attribute. It has no CustomizeDiff. Each test sets the
// CustomizeDiff that it tests.
func tableflowTopicTestResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			paramTableFormats: {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			paramTableFormatProperties: tableFormatPropertiesSchema(),
			paramFeatures:              featuresSchema(),
			paramKafkaCluster:          requiredKafkaClusterBlockSchema(),
			paramEnvironment:           environmentSchema(),
			paramByobAws:               byobAwsSchema(),
			paramAzureStorage:          azureStorageSchema(),
			paramGcsStorage:            gcsStorageSchema(),
		},
	}
}

// planTableflowTopicTestResource plans config against state with r, and returns the diff. A nil state is a create.
func planTableflowTopicTestResource(t *testing.T, r *schema.Resource, state map[string]interface{}, config map[string]cty.Value) (*terraform.InstanceDiff, error) {
	t.Helper()

	// The raw config has every attribute of the schema. An attribute that is not configured is null, and a block
	// that is not configured is an empty collection. Each list in the test resource is a block.
	attrs := map[string]cty.Value{}
	for name, attrType := range r.CoreConfigSchema().ImpliedType().AttributeTypes() {
		value, configured := config[name]
		switch {
		case configured:
			attrs[name] = value
		case attrType.IsListType():
			attrs[name] = cty.ListValEmpty(attrType.ElementType())
		default:
			attrs[name] = cty.NullVal(attrType)
		}
	}
	raw := cty.ObjectVal(attrs)

	instanceState := &terraform.InstanceState{}
	if state != nil {
		d := schema.TestResourceDataRaw(t, r.Schema, state)
		d.SetId("tt-abc123")
		instanceState = d.State()
	}
	instanceState.RawConfig = raw

	// SimpleDiff is the call of the SDK plan path (PlanResourceChange). It calls CustomizeDiff one time, with the prior
	// state, also for a replacement.
	return r.SimpleDiff(context.Background(), instanceState, terraform.NewResourceConfigShimmed(raw, r.CoreConfigSchema()), nil)
}

// withAttribute returns a copy of state with key set to value.
func withAttribute(state map[string]interface{}, key string, value interface{}) map[string]interface{} {
	result := maps.Clone(state)
	result[key] = value
	return result
}

// testBlock returns a config value with one element for the block key of r. values has the configured attributes of
// the element. The other attributes are null.
func testBlock(r *schema.Resource, key string, values map[string]string) cty.Value {
	elementType := r.CoreConfigSchema().ImpliedType().AttributeType(key).ElementType()
	attrs := map[string]cty.Value{}
	for name, attrType := range elementType.AttributeTypes() {
		if value, ok := values[name]; ok {
			attrs[name] = cty.StringVal(value)
		} else {
			attrs[name] = cty.NullVal(attrType)
		}
	}
	return cty.ListVal([]cty.Value{cty.ObjectVal(attrs)})
}

// testBlockState returns the state value with one element for a block. values has the attributes of the element.
func testBlockState(values map[string]string) []interface{} {
	element := map[string]interface{}{}
	for name, value := range values {
		element[name] = value
	}
	return []interface{}{element}
}

func TestTableflowTopicSchemasAreValid(t *testing.T) {
	resource := &schema.Resource{Schema: map[string]*schema.Schema{
		// table_formats is necessary, because the ConflictsWith of table_format_properties refers to it.
		paramTableFormats: {
			Type:     schema.TypeSet,
			Optional: true,
			Computed: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		paramTableFormatProperties: tableFormatPropertiesSchema(),
		paramFeatures:              featuresSchema(),
	}}
	if err := resource.InternalValidate(nil, true); err != nil {
		t.Errorf("resource schemas: %v", err)
	}
	dataSource := &schema.Resource{Schema: map[string]*schema.Schema{
		paramTableFormatProperties: tableFormatPropertiesDataSourceSchema(),
		paramFeatures:              featuresDataSourceSchema(),
	}}
	if err := dataSource.InternalValidate(nil, false); err != nil {
		t.Errorf("data source schemas: %v", err)
	}

	// features must be a list without MaxItems. See featuresSchema.
	for name, s := range map[string]*schema.Schema{"resource": featuresSchema(), "data source": featuresDataSourceSchema()} {
		if s.Type != schema.TypeList || s.Set != nil || s.MaxItems != 0 {
			t.Errorf("%s features: got Type %v, Set %v, MaxItems %d, want TypeList, no Set, no MaxItems", name, s.Type, s.Set != nil, s.MaxItems)
		}
	}
}
