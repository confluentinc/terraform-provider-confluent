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
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

const (
	scenarioStateTableflowTopicIsProvisioning = "The new tableflow topic is provisioning"
	scenarioStateTableflowTopicHasBeenCreated = "The new tableflow topic has been just created"
	scenarioStateTableflowTopicHasBeenUpdated = "The new tableflow topic has been updated"
	byobAwsTableflowTopicScenarioName         = "confluent_tableflow_topic Byob Aws Resource Lifecycle"
	managedStorageTableflowTopicScenarioName  = "confluent_tableflow_topic Managed Storage Resource Lifecycle"

	tableflowTopicUrlPath       = "/tableflow/v1/tableflow-topics"
	tableflowTopicResourceLabel = "confluent_tableflow_topic.main"
)

func TestAccTableflowTopicByobAws(t *testing.T) {
	ctx := context.Background()

	time.Sleep(5 * time.Second)
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
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "record_failure_strategy", "SUSPEND"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.0", "ICEBERG"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "byob_aws.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "managed_storage.#", "0"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "byob_aws.0.bucket_name", "bucket_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "byob_aws.0.bucket_region", "us-east-1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "byob_aws.0.provider_integration_id", "cspi-stgce89r7"),
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
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "record_failure_strategy", "SKIP"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.0", "ICEBERG"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "byob_aws.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "managed_storage.#", "0"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "byob_aws.0.bucket_name", "bucket_1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "byob_aws.0.bucket_region", "us-east-1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "byob_aws.0.provider_integration_id", "cspi-stgce89r7"),
				),
			},
		},
	})
}

func TestAccTableflowTopicManagedStorage(t *testing.T) {
	ctx := context.Background()

	time.Sleep(5 * time.Second)
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
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "record_failure_strategy", "SUSPEND"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.#", "1"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.0", "ICEBERG"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "byob_aws.#", "0"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "managed_storage.#", "1"),
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
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "record_failure_strategy", "SUSPEND"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.#", "2"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.0", "DELTA"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "table_formats.1", "ICEBERG"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "byob_aws.#", "0"),
					resource.TestCheckResourceAttr(tableflowTopicResourceLabel, "managed_storage.#", "1"),
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

func testAccCheckResourceTableflowTopicByobAwsUpdate(mockServerUrl string, retention int) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_tableflow_topic" "main" {
		display_name = "topic_1"
		retention_ms = %d
		record_failure_strategy = "SKIP"
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
