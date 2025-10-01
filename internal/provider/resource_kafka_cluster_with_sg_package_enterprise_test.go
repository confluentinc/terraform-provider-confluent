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
	enterpriseKafkaCloud             = "AWS"
	enterpriseKafkaRegion            = "us-east-2"
	enterpriseKafkaBootstrapEndpoint = "lkc-19ynpv.us-east-2.aws.private.confluent.cloud:9092"
	enterpriseKafkaHttpEndpoint      = "https://lkc-19ynpv.us-east-2.aws.private.confluent.cloud:443"
	enterpriseKafkaScenarioName      = "confluent_kafka Resource Lifecycle"
	kafkaDisplayNameUpdated          = "TestClusterUpdated"
	fullEnterpriseKafkaResourceLabel = "confluent_kafka_cluster.enterprise-cluster"
)

func TestAccEnterpriseClusterWithSGPackage(t *testing.T) {
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

	createClusterResponse, _ := ioutil.ReadFile("../testdata/enterprise_kafka/create_kafka.json")
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

	readCreatedClusterResponse, _ := ioutil.ReadFile("../testdata/enterprise_kafka/read_created_kafka.json")
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

	readCreatedClusterResponse, _ = ioutil.ReadFile("../testdata/enterprise_kafka/read_created_kafka.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(kafkaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(scenarioStateKafkaHasBeenCreatedAndSyncIsComplete).
		WillReturn(
			string(readCreatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedClusterResponse, _ := ioutil.ReadFile("../testdata/enterprise_kafka/read_updated_kafka.json")
	updateClusterStub := wiremock.Patch(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(kafkaScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaHasBeenCreatedAndSyncIsComplete).
		WillSetStateTo(scenarioStateKafkaHasBeenUpdated).
		WillReturn(
			string(readUpdatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(updateClusterStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(kafkaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(scenarioStateKafkaHasBeenUpdated).
		WillReturn(
			string(readUpdatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedEnvResponse, _ := ioutil.ReadFile("../testdata/enterprise_kafka/read_deleted_kafka.json")

	deleteClusterStub := wiremock.Delete(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(kafkaScenarioName).
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
		CheckDestroy:      testAccCheckClusterDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckEnterpriseClusterConfig(mockServerUrl, paramEnterpriseCluster, kafkaDisplayName, kafkaMaxEcku),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClusterExists(fullEnterpriseKafkaResourceLabel),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "id", kafkaClusterId),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "api_version", kafkaApiVersion),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "kind", kafkaKind),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "display_name", kafkaDisplayName),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "availability", lowAvailability),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "bootstrap_endpoint", enterpriseKafkaBootstrapEndpoint),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "cloud", enterpriseKafkaCloud),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "basic.#", "0"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "standard.#", "0"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "enterprise.#", "1"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "enterprise.0.%", "1"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "enterprise.0.max_ecku", "5"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "environment.0.id", testEnvironmentId),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "network.0.%", "1"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "rest_endpoint", enterpriseKafkaHttpEndpoint),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "rbac_crn", kafkaRbacCrn),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "endpoints.#", "2"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "endpoints.0.access_point_id", "ap1pni123"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "endpoints.0.connection_type", "PRIVATENETWORKINTERFACE"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "endpoints.0.rest_endpoint", "https://lkc-s1232.us-central1.gcp.private.confluent.cloud:443"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "endpoints.0.bootstrap_endpoint", "lkc-s1232-00000.us-central1.gcp.private.confluent.cloud:9092"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "endpoints.1.access_point_id", "ap2platt67890"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "endpoints.1.connection_type", "PRIVATELINK"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "endpoints.1.rest_endpoint", "https://lkc-00000-00000.us-central1.gcp.glb.confluent.cloud:443"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "endpoints.1.bootstrap_endpoint", "lkc-00000-00000.us-central1.gcp.glb.confluent.cloud:9092"),
				),
			},
			{
				Config: testAccCheckEnterpriseClusterConfig(mockServerUrl, paramEnterpriseCluster, kafkaDisplayNameUpdated, kafkaMaxEckuUpdated),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClusterExists(fullEnterpriseKafkaResourceLabel),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "id", kafkaClusterId),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "api_version", kafkaApiVersion),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "kind", kafkaKind),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "display_name", kafkaDisplayNameUpdated),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "availability", lowAvailability),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "bootstrap_endpoint", enterpriseKafkaBootstrapEndpoint),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "cloud", enterpriseKafkaCloud),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "basic.#", "0"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "standard.#", "0"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "enterprise.#", "1"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "enterprise.0.%", "1"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "enterprise.0.max_ecku", "3"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "environment.0.id", testEnvironmentId),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "network.0.%", "1"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "rest_endpoint", enterpriseKafkaHttpEndpoint),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "rbac_crn", kafkaRbacCrn),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "endpoints.#", "2"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "endpoints.0.access_point_id", "ap1pni123"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "endpoints.0.connection_type", "PRIVATENETWORKINTERFACE"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "endpoints.0.rest_endpoint", "https://lkc-s1232.us-central1.gcp.private.confluent.cloud:443"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "endpoints.0.bootstrap_endpoint", "lkc-s1232-00000.us-central1.gcp.private.confluent.cloud:9092"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "endpoints.1.access_point_id", "ap2platt67890"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "endpoints.1.connection_type", "PRIVATELINK"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "endpoints.1.rest_endpoint", "https://lkc-00000-00000.us-central1.gcp.glb.confluent.cloud:443"),
					resource.TestCheckResourceAttr(fullEnterpriseKafkaResourceLabel, "endpoints.1.bootstrap_endpoint", "lkc-00000-00000.us-central1.gcp.glb.confluent.cloud:9092"),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullEnterpriseKafkaResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					clusterId := resources[fullEnterpriseKafkaResourceLabel].Primary.ID
					environmentId := resources[fullEnterpriseKafkaResourceLabel].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + clusterId, nil
				},
			},
		},
	})

	checkStubCount(t, wiremockClient, createClusterStub, fmt.Sprintf("POST %s", createKafkaPath), expectedCountOne)
	checkStubCount(t, wiremockClient, updateClusterStub, fmt.Sprintf("PATCH %s", readKafkaPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteClusterStub, fmt.Sprintf("DELETE %s", readKafkaPath), expectedCountOne)
}

func testAccCheckEnterpriseClusterConfig(mockServerUrl, clusterType, displayName string, maxEcku int32) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_kafka_cluster" "enterprise-cluster" {
		display_name = "%s"
		availability = "%s"
		cloud = "%s"
		region = "%s"
		%s {
			max_ecku = %v
		}
	
	  	environment {
			id = "%s"
	  	}
	}
	`, mockServerUrl, displayName, lowAvailability, enterpriseKafkaCloud, enterpriseKafkaRegion, clusterType, maxEcku, testEnvironmentId)
}
