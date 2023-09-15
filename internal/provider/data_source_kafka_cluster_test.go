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

const (
	dataSourceKafkaScenarioName = "confluent_kafka Data Source Lifecycle"
)

var fullKafkaDataSourceLabel = fmt.Sprintf("data.confluent_kafka_cluster.%s", kafkaResourceLabel)

func TestAccDataSourceCluster(t *testing.T) {
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

	readCreatedClusterResponse, _ := ioutil.ReadFile("../testdata/kafka/read_created_kafka.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(dataSourceKafkaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(kafkaEnvId)).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readClustersResponse, _ := ioutil.ReadFile("../testdata/kafka/read_kafkas.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/cmk/v2/clusters")).
		InScenario(dataSourceKafkaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(kafkaEnvId)).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readClustersResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceClusterConfigWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClusterExists(fullKafkaDataSourceLabel),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "id", kafkaClusterId),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "api_version", kafkaApiVersion),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "kind", kafkaKind),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "display_name", kafkaDisplayName),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "availability", kafkaAvailability),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "bootstrap_endpoint", kafkaBootstrapEndpoint),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "cloud", kafkaCloud),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "basic.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "basic.0.%", "0"),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "standard.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "environment.0.id", kafkaEnvId),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "network.0.id", kafkaNetworkId),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "rest_endpoint", kafkaHttpEndpoint),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "rbac_crn", kafkaRbacCrn),
				),
			},
			{
				Config: testAccCheckDataSourceClusterConfigWithDisplayNameSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClusterExists(fullKafkaDataSourceLabel),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "id", kafkaClusterId),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "api_version", kafkaApiVersion),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "kind", kafkaKind),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "display_name", kafkaDisplayName),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "availability", kafkaAvailability),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "bootstrap_endpoint", kafkaBootstrapEndpoint),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "cloud", kafkaCloud),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "basic.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "basic.0.%", "0"),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "standard.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "environment.0.id", kafkaEnvId),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "rest_endpoint", kafkaHttpEndpoint),
					resource.TestCheckResourceAttr(fullKafkaDataSourceLabel, "rbac_crn", kafkaRbacCrn),
				),
			},
		},
	})
}

func testAccCheckDataSourceClusterConfigWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_kafka_cluster" "basic-cluster" {
		id = "%s"
	  	environment {
			id = "%s"
	  	}
	}
	`, mockServerUrl, kafkaClusterId, kafkaEnvId)
}

func testAccCheckDataSourceClusterConfigWithDisplayNameSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_kafka_cluster" "basic-cluster" {
		display_name = "%s"
	  	environment {
			id = "%s"
	  	}
	}
	`, mockServerUrl, kafkaDisplayName, kafkaEnvId)
}
