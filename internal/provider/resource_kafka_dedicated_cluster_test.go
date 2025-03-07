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

func TestAccDedicatedCluster(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	//mockServerUrl := wiremockContainer.URI
	mockServerUrl := "http://localhost:8080"
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	createClusterResponse, _ := ioutil.ReadFile("../testdata/kafka/create_kafka_dedicated.json")
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

	readCreatedClusterResponse, _ := ioutil.ReadFile("../testdata/kafka/read_created_kafka_dedicated.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(kafkaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(scenarioStateKafkaHasBeenCreated).
		WillReturn(
			string(readCreatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readEnvironmentResponse, _ := ioutil.ReadFile("../testdata/environment/read_created_env.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readEnvPath)).
		InScenario(kafkaScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaHasBeenCreated).
		WillSetStateTo(scenarioStateKafkaHasBeenCreatedButZeroSRClusters).
		WillReturn(
			string(readEnvironmentResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	listZeroSchemaRegistryClusterResponse, _ := ioutil.ReadFile("../testdata/schema_registry_cluster/read_zero_clusters.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(listSchemaRegistryClusterUrlPath)).
		InScenario(kafkaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(scenarioStateKafkaHasBeenCreatedButZeroSRClusters).
		WillSetStateTo(scenarioStateKafkaHasBeenCreatedButSRClusterIsProvisioning).
		WillReturn(
			string(listZeroSchemaRegistryClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	listProvisioningSchemaRegistryClusterResponse, _ := ioutil.ReadFile("../testdata/schema_registry_cluster/read_provisioning_clusters.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(listSchemaRegistryClusterUrlPath)).
		InScenario(kafkaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(scenarioStateKafkaHasBeenCreatedButSRClusterIsProvisioning).
		WillSetStateTo(scenarioStateKafkaHasBeenCreatedAndSRClusterIsProvisioned).
		WillReturn(
			string(listProvisioningSchemaRegistryClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedSchemaRegistryClustersResponse, _ := ioutil.ReadFile("../testdata/schema_registry_cluster/read_provisioned_clusters.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(listSchemaRegistryClusterUrlPath)).
		InScenario(kafkaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(scenarioStateKafkaHasBeenCreatedAndSRClusterIsProvisioned).
		WillSetStateTo(scenarioStateKafkaHasBeenCreatedAndSyncIsComplete).
		WillReturn(
			string(readCreatedSchemaRegistryClustersResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedClusterResponse, _ = ioutil.ReadFile("../testdata/kafka/read_created_kafka_dedicated.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(kafkaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(scenarioStateKafkaHasBeenCreatedAndSyncIsComplete).
		WillReturn(
			string(readCreatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	//readUpdatedClusterResponse, _ := ioutil.ReadFile("../testdata/kafka/read_updated_kafka.json")
	//updateClusterStub := wiremock.Patch(wiremock.URLPathEqualTo(readKafkaPath)).
	//	InScenario(kafkaScenarioName).
	//	WhenScenarioStateIs(scenarioStateKafkaHasBeenCreatedAndSyncIsComplete).
	//	WillSetStateTo(scenarioStateKafkaHasBeenUpdated).
	//	WillReturn(
	//		string(readUpdatedClusterResponse),
	//		contentTypeJSONHeader,
	//		http.StatusOK,
	//	)
	//_ = wiremockClient.StubFor(updateClusterStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(kafkaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(scenarioStateKafkaHasBeenCreatedAndSyncIsComplete).
		WillReturn(
			string(readCreatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedEnvResponse, _ := ioutil.ReadFile("../testdata/kafka/read_deleted_kafka.json")

	deleteClusterStub := wiremock.Delete(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(kafkaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(scenarioStateKafkaHasBeenCreated).
		WillSetStateTo(scenarioStateKafkaHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteClusterStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(kafkaScenarioName).
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
		CheckDestroy:      testAccCheckDedicatedClusterDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDedicatedClusterConfig(mockServerUrl, paramDedicatedCluster),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClusterExists(fullKafkaResourceLabel),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "id", kafkaClusterId),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "api_version", kafkaApiVersion),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "kind", kafkaKind),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "display_name", kafkaDisplayName),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "availability", kafkaAvailability),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "bootstrap_endpoint", kafkaBootstrapEndpoint),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "cloud", kafkaCloud),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "basic.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "standard.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "environment.0.id", testEnvironmentId),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "dedicated.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "dedicated.0.zones.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "dedicated.0.zones.0", kafkaZones),
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
	//checkStubCount(t, wiremockClient, updateClusterStub, fmt.Sprintf("PATCH %s", readKafkaPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteClusterStub, fmt.Sprintf("DELETE %s", readKafkaPath), expectedCountOne)
}

func testAccCheckDedicatedClusterDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each environment is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_kafka_cluster" {
			continue
		}
		deletedClusterId := rs.Primary.ID
		req := c.cmkClient.ClustersCmkV2Api.GetCmkV2Cluster(c.cmkApiContext(context.Background()), deletedClusterId).Environment(testEnvironmentId)
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

func testAccCheckDedicatedClusterConfig(mockServerUrl, clusterType string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_kafka_cluster" "basic-cluster" {
		display_name = "%s"
		availability = "%s"
		cloud = "%s"
		region = "%s"
		%s {
			cku = 1			
			zones = ["%s"]		
		}
	
	  	environment {
			id = "%s"
	  	}
	}
	`, mockServerUrl, kafkaDisplayName, kafkaAvailability, kafkaCloud, kafkaRegion, clusterType, kafkaZones, testEnvironmentId)
}
