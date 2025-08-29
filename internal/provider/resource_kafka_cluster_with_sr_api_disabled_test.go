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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"testing"
)

const (
	kafkaWithDisabledSRApiScenarioName                     = "confluent_kafka Resource (with disabled SR API) Lifecycle"
	SRApiScenarioStateKafkaHasBeenCreatedWithDisabledSRApi = "A new Kafka Basic cluster has been just created with disabled SR API"
)

func TestAccClusterWithSGPackageAndDisabledSRApi(t *testing.T) {
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

	createClusterResponse, _ := ioutil.ReadFile("../testdata/kafka/create_kafka.json")
	createClusterStub := wiremock.Post(wiremock.URLPathEqualTo(createKafkaPath)).
		InScenario(kafkaWithDisabledSRApiScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateKafkaHasBeenCreated).
		WillReturn(
			string(createClusterResponse),
			contentTypeJSONHeader,
			http.StatusAccepted,
		)
	_ = wiremockClient.StubFor(createClusterStub)

	readCreatedClusterResponse, _ := ioutil.ReadFile("../testdata/kafka/read_created_kafka.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(kafkaWithDisabledSRApiScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(scenarioStateKafkaHasBeenCreated).
		WillReturn(
			string(readCreatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readEnvironmentResponse, _ := ioutil.ReadFile("../testdata/environment/read_created_env.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readEnvPath)).
		InScenario(kafkaWithDisabledSRApiScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaHasBeenCreated).
		WillSetStateTo(SRApiScenarioStateKafkaHasBeenCreatedWithDisabledSRApi).
		WillReturn(
			string(readEnvironmentResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	schemaRegistryApiNotAvailableResponse, _ := ioutil.ReadFile("../testdata/schema_registry_cluster/not_available.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(listSchemaRegistryClusterUrlPath)).
		InScenario(kafkaWithDisabledSRApiScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(SRApiScenarioStateKafkaHasBeenCreatedWithDisabledSRApi).
		WillSetStateTo(scenarioStateKafkaHasBeenCreatedAndSyncIsComplete).
		WillReturn(
			string(schemaRegistryApiNotAvailableResponse),
			contentTypeJSONHeader,
			http.StatusForbidden,
		))

	readCreatedClusterResponse, _ = ioutil.ReadFile("../testdata/kafka/read_created_kafka.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(kafkaWithDisabledSRApiScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(scenarioStateKafkaHasBeenCreatedAndSyncIsComplete).
		WillReturn(
			string(readCreatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedClusterResponse, _ := ioutil.ReadFile("../testdata/kafka/read_updated_kafka.json")
	updateClusterStub := wiremock.Patch(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(kafkaWithDisabledSRApiScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaHasBeenCreatedAndSyncIsComplete).
		WillSetStateTo(scenarioStateKafkaHasBeenUpdated).
		WillReturn(
			string(readUpdatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(updateClusterStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(kafkaWithDisabledSRApiScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(scenarioStateKafkaHasBeenUpdated).
		WillReturn(
			string(readUpdatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedEnvResponse, _ := ioutil.ReadFile("../testdata/kafka/read_deleted_kafka.json")

	deleteClusterStub := wiremock.Delete(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(kafkaWithDisabledSRApiScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(scenarioStateKafkaHasBeenUpdated).
		WillSetStateTo(scenarioStateKafkaHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteClusterStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(kafkaWithDisabledSRApiScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(scenarioStateKafkaHasBeenDeleted).
		WillReturn(
			string(readDeletedEnvResponse),
			contentTypeJSONHeader,
			http.StatusForbidden,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckClusterDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckClusterConfig(mockServerUrl, paramBasicCluster),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClusterExists(fullKafkaResourceLabel),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "id", kafkaClusterId),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "api_version", kafkaApiVersion),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "kind", kafkaKind),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "display_name", kafkaDisplayName),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "availability", kafkaAvailability),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "bootstrap_endpoint", kafkaBootstrapEndpoint),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "cloud", kafkaCloud),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "basic.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "basic.0.%", "0"),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "standard.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "environment.0.id", testEnvironmentId),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "network.0.id", kafkaNetworkId),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "rest_endpoint", kafkaHttpEndpoint),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "rbac_crn", kafkaRbacCrn),
				),
			},
			{
				Config: testAccCheckClusterConfig(mockServerUrl, paramStandardCluster),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClusterExists(fullKafkaResourceLabel),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "id", kafkaClusterId),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "api_version", kafkaApiVersion),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "kind", kafkaKind),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "availability", kafkaAvailability),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "bootstrap_endpoint", kafkaBootstrapEndpoint),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "cloud", kafkaCloud),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "basic.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "enterprise.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "freight.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "standard.0.%", "0"),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "standard.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "display_name", kafkaDisplayName),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "environment.0.id", testEnvironmentId),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "network.0.id", kafkaNetworkId),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "rest_endpoint", kafkaHttpEndpoint),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "rbac_crn", kafkaRbacCrn),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullKafkaResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					clusterId := resources[fullKafkaResourceLabel].Primary.ID
					environmentId := resources[fullKafkaResourceLabel].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + clusterId, nil
				},
			},
		},
	})

	checkStubCount(t, wiremockClient, createClusterStub, fmt.Sprintf("POST %s", createKafkaPath), expectedCountOne)
	checkStubCount(t, wiremockClient, updateClusterStub, fmt.Sprintf("PATCH %s", readKafkaPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteClusterStub, fmt.Sprintf("DELETE %s", readKafkaPath), expectedCountOne)
}
