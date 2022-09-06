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
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	scenarioStateKafkaHasBeenCreated = "A new Kafka Basic cluster has been just created"
	scenarioStateKafkaHasBeenUpdated = "The new Kafka cluster's kind has been just updated to Standard"
	scenarioStateKafkaHasBeenDeleted = "The new Kafka cluster has been deleted"
	kafkaScenarioName                = "confluent_kafka Resource Lifecycle"
	kafkaClusterId                   = "lkc-19ynpv"
	kafkaEnvId                       = "env-1jrymj"
	kafkaNetworkId                   = "n-123abc"
	kafkaDisplayName                 = "TestCluster"
	kafkaApiVersion                  = "cmk/v2"
	kafkaKind                        = "Cluster"
	kafkaAvailability                = "SINGLE_ZONE"
	kafkaCloud                       = "GCP"
	kafkaRegion                      = "us-central1"
	kafkaResourceLabel               = "basic-cluster"
	kafkaHttpEndpoint                = "https://pkc-0wg55.us-central1.gcp.confluent.cloud:443"
	kafkaBootstrapEndpoint           = "SASL_SSL://pkc-0wg55.us-central1.gcp.confluent.cloud:9092"
	kafkaRbacCrn                     = "crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa/environment=env-1jrymj/cloud-cluster=lkc-19ynpv"
)

var createKafkaPath = "/cmk/v2/clusters"
var readKafkaPath = fmt.Sprintf("/cmk/v2/clusters/%s", kafkaClusterId)
var fullKafkaResourceLabel = fmt.Sprintf("confluent_kafka_cluster.%s", kafkaResourceLabel)

func TestAccCluster(t *testing.T) {
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
		InScenario(kafkaScenarioName).
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
		InScenario(kafkaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(kafkaEnvId)).
		WhenScenarioStateIs(scenarioStateKafkaHasBeenCreated).
		WillReturn(
			string(readCreatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedClusterResponse, _ := ioutil.ReadFile("../testdata/kafka/read_updated_kafka.json")
	updateClusterStub := wiremock.Patch(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(kafkaScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaHasBeenCreated).
		WillSetStateTo(scenarioStateKafkaHasBeenUpdated).
		WillReturn(
			string(readUpdatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(updateClusterStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(kafkaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(kafkaEnvId)).
		WhenScenarioStateIs(scenarioStateKafkaHasBeenUpdated).
		WillReturn(
			string(readUpdatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedEnvResponse, _ := ioutil.ReadFile("../testdata/kafka/read_deleted_kafka.json")

	deleteClusterStub := wiremock.Delete(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(kafkaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(kafkaEnvId)).
		WhenScenarioStateIs(scenarioStateKafkaHasBeenUpdated).
		WillSetStateTo(scenarioStateKafkaHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteClusterStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(kafkaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(kafkaEnvId)).
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
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "environment.0.id", kafkaEnvId),
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
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "standard.0.%", "0"),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "standard.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "display_name", kafkaDisplayName),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "environment.0.id", kafkaEnvId),
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

func testAccCheckClusterDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each environment is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_kafka_cluster" {
			continue
		}
		deletedClusterId := rs.Primary.ID
		req := c.cmkClient.ClustersCmkV2Api.GetCmkV2Cluster(c.cmkApiContext(context.Background()), deletedClusterId).Environment(kafkaEnvId)
		deletedCluster, response, err := req.Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			// cmk/v2/clusters/{nonExistentClusterId/deletedClusterID} returns http.StatusForbidden instead of http.StatusNotFound
			// If the error is equivalent to http.StatusNotFound, the environment is destroyed.
			return nil
		} else if err == nil && deletedCluster.Id != nil {
			// Otherwise return the error
			if *deletedCluster.Id == rs.Primary.ID {
				return fmt.Errorf("kafka cluster (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckClusterConfig(mockServerUrl, clusterType string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_kafka_cluster" "basic-cluster" {
		display_name = "%s"
		availability = "%s"
		cloud = "%s"
		region = "%s"
		%s {}
	
	  	environment {
			id = "%s"
	  	}
	}
	`, mockServerUrl, kafkaDisplayName, kafkaAvailability, kafkaCloud, kafkaRegion, clusterType, kafkaEnvId)
}

func testAccCheckClusterExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s kafka cluster has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s kafka cluster", n)
		}

		return nil
	}
}
