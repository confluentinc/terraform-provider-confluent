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
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccClusterConfigWithEnhancedProviderBlock(t *testing.T) {
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

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckConfigDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckConfigConfigWithEnhancedProviderBlock(confluentCloudBaseUrl, mockConfigTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConfigExists(fullConfigResourceLabel),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "kafka_cluster.0.id", clusterId),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "id", fmt.Sprintf("%s", clusterId)),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "%", "5"),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "config.%", "3"),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, fmt.Sprintf("config.%s", firstClusterConfigName), firstClusterConfigValue),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, fmt.Sprintf("config.%s", secondClusterConfigName), secondClusterConfigValue),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, fmt.Sprintf("config.%s", thirdClusterConfigName), thirdClusterConfigValue),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "credentials.#", "0"),
				),
			},
			{
				Config: testAccCheckConfigUpdatedConfigWithEnhancedProviderBlock(confluentCloudBaseUrl, mockConfigTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConfigExists(fullConfigResourceLabel),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "kafka_cluster.0.id", clusterId),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "id", fmt.Sprintf("%s", clusterId)),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "%", "5"),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "config.%", "5"),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, fmt.Sprintf("config.%s", firstClusterConfigName), firstClusterConfigUpdatedValue),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, fmt.Sprintf("config.%s", secondClusterConfigName), secondClusterConfigValue),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, fmt.Sprintf("config.%s", thirdClusterConfigName), thirdClusterConfigUpdatedValue),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, fmt.Sprintf("config.%s", fourthClusterConfigName), fourthClusterConfigAddedValue),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, fmt.Sprintf("config.%s", fifthClusterConfigName), fifthClusterConfigAddedValue),
					resource.TestCheckResourceAttr(fullConfigResourceLabel, "credentials.#", "0"),
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

func testAccCheckConfigConfigWithEnhancedProviderBlock(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
      endpoint = "%s"
	  kafka_api_key = "%s"
	  kafka_api_secret = "%s"
	  kafka_rest_endpoint = "%s"
    }
	resource "confluent_kafka_cluster_config" "%s" {
	  kafka_cluster {
        id = "%s"
      }
	
	  config = {
		"%s" = "%s"
		"%s" = "%s"
		"%s" = "%s"
	  }
	}
	`, confluentCloudBaseUrl, kafkaApiKey, kafkaApiSecret, mockServerUrl, configResourceLabel, clusterId,
		firstClusterConfigName, firstClusterConfigValue, secondClusterConfigName, secondClusterConfigValue, thirdClusterConfigName, thirdClusterConfigValue,
	)
}

func testAccCheckConfigUpdatedConfigWithEnhancedProviderBlock(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
      endpoint = "%s"
	  kafka_api_key = "%s"
	  kafka_api_secret = "%s"
	  kafka_rest_endpoint = "%s"
    }
	resource "confluent_kafka_cluster_config" "%s" {
	  kafka_cluster {
        id = "%s"
      }
	
	  config = {
		"%s" = "%s"
		"%s" = "%s"
		"%s" = "%s"
		"%s" = "%s"
		"%s" = "%s"
	  }
	}
	`, confluentCloudBaseUrl, kafkaApiKey, kafkaApiSecret, mockServerUrl, configResourceLabel, clusterId,
		firstClusterConfigName, firstClusterConfigUpdatedValue, secondClusterConfigName, secondClusterConfigValue,
		thirdClusterConfigName, thirdClusterConfigUpdatedValue, fourthClusterConfigName, fourthClusterConfigAddedValue,
		fifthClusterConfigName, fifthClusterConfigAddedValue)
}
