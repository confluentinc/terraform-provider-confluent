// Copyright 2021 Confluent Inc. All Rights Reserved.
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
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	scenarioStateConfigHasBeenCreated = "A new config has been just created"
	scenarioStateConfigHasBeenUpdated = "A new config has been just updated"
	configScenarioName                = "confluent_kafka_cluster_config Resource Lifecycle"
	firstClusterConfigName            = "auto.create.topics.enable"
	firstClusterConfigValue           = "false"
	firstClusterConfigUpdatedValue    = "true"
	secondClusterConfigName           = "ssl.cipher.suites"
	secondClusterConfigValue          = "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
	thirdClusterConfigName            = "num.partitions"
	thirdClusterConfigValue           = "6"
	thirdClusterConfigUpdatedValue    = "8"
	fourthClusterConfigName           = "log.cleaner.max.compaction.lag.ms"
	fourthClusterConfigAddedValue     = "9223372036854775807"
	fifthClusterConfigName            = "log.retention.ms"
	fifthClusterConfigAddedValue      = "604800001"
	configResourceLabel               = "test_config_resource_label"
)

var fullConfigResourceLabel = fmt.Sprintf("confluent_kafka_cluster_config.%s", configResourceLabel)
var readKafkaConfigPath = fmt.Sprintf("/kafka/v3/clusters/%s/broker-configs", clusterId)
var updateKafkaConfigPath = fmt.Sprintf("/kafka/v3/clusters/%s/broker-configs:alter", clusterId)

// TODO: APIF-1990
var mockConfigTestServerUrl = ""

func TestAccClusterConfig(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockConfigTestServerUrl = wiremockContainer.URI
	confluentCloudBaseUrl := ""
	wiremockClient := wiremock.NewClient(mockConfigTestServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()
	createConfigStub := wiremock.Post(wiremock.URLPathEqualTo(updateKafkaConfigPath)).
		InScenario(configScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateConfigHasBeenCreated).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createConfigStub)

	readCreatedConfigResponse, _ := ioutil.ReadFile("../testdata/kafka_config/read_created_kafka_config.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaConfigPath)).
		InScenario(configScenarioName).
		WhenScenarioStateIs(scenarioStateConfigHasBeenCreated).
		WillReturn(
			string(readCreatedConfigResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedConfigResponse, _ := ioutil.ReadFile("../testdata/kafka_config/read_updated_kafka_config.json")
	patchConfigStub := wiremock.Post(wiremock.URLPathEqualTo(updateKafkaConfigPath)).
		InScenario(configScenarioName).
		WhenScenarioStateIs(scenarioStateConfigHasBeenCreated).
		WillSetStateTo(scenarioStateConfigHasBeenUpdated).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(patchConfigStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaConfigPath)).
		InScenario(configScenarioName).
		WhenScenarioStateIs(scenarioStateConfigHasBeenUpdated).
		WillReturn(
			string(readUpdatedConfigResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// Set fake values for secrets since those are required for importing
	_ = os.Setenv("IMPORT_KAFKA_API_KEY", kafkaApiKey)
	_ = os.Setenv("IMPORT_KAFKA_API_SECRET", kafkaApiSecret)
	_ = os.Setenv("IMPORT_KAFKA_REST_ENDPOINT", mockConfigTestServerUrl)
	defer func() {
		_ = os.Unsetenv("IMPORT_KAFKA_API_KEY")
		_ = os.Unsetenv("IMPORT_KAFKA_API_SECRET")
		_ = os.Unsetenv("IMPORT_KAFKA_REST_ENDPOINT")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckConfigDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckConfigConfig(confluentCloudBaseUrl, mockConfigTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConfigExists(fullConfigResourceLabel),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "kafka_cluster.0.id", clusterId),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "id", fmt.Sprintf("%s", clusterId)),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "%", "5"),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "rest_endpoint", mockConfigTestServerUrl),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "config.%", "3"),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, fmt.Sprintf("config.%s", firstClusterConfigName), firstClusterConfigValue),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, fmt.Sprintf("config.%s", secondClusterConfigName), secondClusterConfigValue),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, fmt.Sprintf("config.%s", thirdClusterConfigName), thirdClusterConfigValue),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "credentials.0.key", kafkaApiKey),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "credentials.0.secret", kafkaApiSecret),
				),
			},
			{
				Config: testAccCheckConfigUpdatedConfig(confluentCloudBaseUrl, mockConfigTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConfigExists(fullConfigResourceLabel),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "kafka_cluster.0.id", clusterId),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "id", fmt.Sprintf("%s", clusterId)),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "%", "5"),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "rest_endpoint", mockConfigTestServerUrl),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "config.%", "5"),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, fmt.Sprintf("config.%s", firstClusterConfigName), firstClusterConfigUpdatedValue),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, fmt.Sprintf("config.%s", secondClusterConfigName), secondClusterConfigValue),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, fmt.Sprintf("config.%s", thirdClusterConfigName), thirdClusterConfigUpdatedValue),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, fmt.Sprintf("config.%s", fourthClusterConfigName), fourthClusterConfigAddedValue),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, fmt.Sprintf("config.%s", fifthClusterConfigName), fifthClusterConfigAddedValue),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "credentials.0.key", kafkaApiKey),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "credentials.0.secret", kafkaApiSecret),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullConfigResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, createConfigStub, fmt.Sprintf("POST %s", updateKafkaConfigPath), 2)
}

func testAccCheckConfigDestroy(s *terraform.State) error {
	return nil
}

func testAccCheckConfigConfig(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
      endpoint = "%s"
    }
	resource "confluent_kafka_cluster_config" "%s" {
	  kafka_cluster {
        id = "%s"
      }

	  rest_endpoint = "%s"
	
	  config = {
		"%s" = "%s"
		"%s" = "%s"
		"%s" = "%s"
	  }

	  credentials {
		key = "%s"
		secret = "%s"
	  }
	}
	`, confluentCloudBaseUrl, configResourceLabel, clusterId, mockServerUrl,
		firstClusterConfigName, firstClusterConfigValue, secondClusterConfigName, secondClusterConfigValue, thirdClusterConfigName, thirdClusterConfigValue,
		kafkaApiKey, kafkaApiSecret)
}

func testAccCheckConfigUpdatedConfig(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
      endpoint = "%s"
    }
	resource "confluent_kafka_cluster_config" "%s" {
	  kafka_cluster {
        id = "%s"
      }

	  rest_endpoint = "%s"
	
	  config = {
		"%s" = "%s"
		"%s" = "%s"
		"%s" = "%s"
		"%s" = "%s"
		"%s" = "%s"
	  }

	  credentials {
		key = "%s"
		secret = "%s"
	  }
	}
	`, confluentCloudBaseUrl, configResourceLabel, clusterId, mockServerUrl,
		firstClusterConfigName, firstClusterConfigUpdatedValue, secondClusterConfigName, secondClusterConfigValue,
		thirdClusterConfigName, thirdClusterConfigUpdatedValue, fourthClusterConfigName, fourthClusterConfigAddedValue,
		fifthClusterConfigName, fifthClusterConfigAddedValue,
		kafkaApiKey, kafkaApiSecret)
}

func testAccCheckConfigExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s config has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s config", n)
		}

		return nil
	}
}
