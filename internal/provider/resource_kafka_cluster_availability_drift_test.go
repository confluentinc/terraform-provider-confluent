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
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

const (
	scenarioStateKafkaCreatedWithSingleZone  = "Kafka cluster created with SINGLE_ZONE"
	scenarioStateKafkaProvisionedSingleZone  = "Kafka cluster provisioned with SINGLE_ZONE"
	scenarioStateKafkaReadyForLowTransition  = "Kafka cluster ready for LOW transition"
	scenarioStateKafkaApiReturnsLow          = "Kafka cluster API returns LOW (V2 billing model)"
	scenarioStateKafkaCreatedWithMultiZone   = "Kafka cluster created with MULTI_ZONE"
	scenarioStateKafkaProvisionedMultiZone   = "Kafka cluster provisioned with MULTI_ZONE"
	scenarioStateKafkaReadyForHighTransition = "Kafka cluster ready for HIGH transition"
	scenarioStateKafkaApiReturnsHigh         = "Kafka cluster API returns HIGH (V2 billing model)"
	availabilityDriftScenarioName            = "confluent_kafka Availability Drift"
)

// TestAccKafkaClusterAvailabilityDriftSingleZoneToLow tests that DiffSuppressFunc
// correctly suppresses drift when API returns LOW but config has SINGLE_ZONE
func TestAccKafkaClusterAvailabilityDriftSingleZoneToLow(t *testing.T) {
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

	// Step 1: Create cluster with SINGLE_ZONE
	createClusterResponse, _ := ioutil.ReadFile("../testdata/kafka/create_kafka.json")
	createClusterStub := wiremock.Post(wiremock.URLPathEqualTo(createKafkaPath)).
		InScenario(availabilityDriftScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateKafkaCreatedWithSingleZone).
		WillReturn(
			string(createClusterResponse),
			contentTypeJSONHeader,
			http.StatusAccepted,
		)
	_ = wiremockClient.StubFor(createClusterStub)

	// Step 2: First GET after creation returns SINGLE_ZONE and transitions to provisioned state
	readCreatedClusterResponse, _ := ioutil.ReadFile("../testdata/kafka/read_created_kafka.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(availabilityDriftScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(scenarioStateKafkaCreatedWithSingleZone).
		WillSetStateTo(scenarioStateKafkaProvisionedSingleZone).
		WillReturn(
			string(readCreatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// Step 3: Subsequent GETs during step 1 return SINGLE_ZONE and transition to ready state
	// This allows multiple GETs during step 1 to all return SINGLE_ZONE
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(availabilityDriftScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(scenarioStateKafkaProvisionedSingleZone).
		WillSetStateTo(scenarioStateKafkaReadyForLowTransition).
		WillReturn(
			string(readCreatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// Step 4: GETs from ready state return SINGLE_ZONE (stay in same state)
	// This handles any additional GETs during step 1
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(availabilityDriftScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(scenarioStateKafkaReadyForLowTransition).
		WillReturn(
			string(readCreatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// Step 5: Transition to returning LOW on GET (simulating API returning V2 billing model value)
	// This will be called during the plan refresh in step 2
	readClusterWithLowResponse, _ := ioutil.ReadFile("../testdata/kafka/read_created_kafka_with_low.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(availabilityDriftScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(scenarioStateKafkaReadyForLowTransition).
		WillSetStateTo(scenarioStateKafkaApiReturnsLow).
		WillReturn(
			string(readClusterWithLowResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// Step 5: Subsequent GETs return LOW (after state transition)
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(availabilityDriftScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(scenarioStateKafkaApiReturnsLow).
		WillReturn(
			string(readClusterWithLowResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readEnvironmentResponse, _ := ioutil.ReadFile("../testdata/environment/read_created_env_without_sg.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readEnvPath)).
		InScenario(availabilityDriftScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaCreatedWithSingleZone).
		WillReturn(
			string(readEnvironmentResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readEnvPath)).
		InScenario(availabilityDriftScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaProvisionedSingleZone).
		WillReturn(
			string(readEnvironmentResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readEnvPath)).
		InScenario(availabilityDriftScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaReadyForLowTransition).
		WillReturn(
			string(readEnvironmentResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readEnvPath)).
		InScenario(availabilityDriftScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaApiReturnsLow).
		WillReturn(
			string(readEnvironmentResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// DELETE stub for cleanup
	readDeletedKafkaResponse, _ := ioutil.ReadFile("../testdata/kafka/read_deleted_kafka.json")
	deleteClusterStub := wiremock.Delete(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(availabilityDriftScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(scenarioStateKafkaApiReturnsLow).
		WillSetStateTo("deleted").
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteClusterStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(availabilityDriftScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs("deleted").
		WillReturn(
			string(readDeletedKafkaResponse),
			contentTypeJSONHeader,
			http.StatusForbidden,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckClusterDestroy,
		Steps: []resource.TestStep{
			{
				// Step 1: Create cluster with SINGLE_ZONE
				Config: testAccCheckClusterConfig(mockServerUrl, paramBasicCluster),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClusterExists(fullKafkaResourceLabel),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "availability", "SINGLE_ZONE"),
				),
			},
			{
				// Step 2: API returns LOW, but config still has SINGLE_ZONE
				// DiffSuppressFunc should suppress the diff and show no changes
				Config:             testAccCheckClusterConfig(mockServerUrl, paramBasicCluster),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false, // Should NOT show drift after DiffSuppressFunc suppresses the diff
			},
		},
	})
}

// TestAccKafkaClusterAvailabilityDriftMultiZoneToHigh tests that DiffSuppressFunc
// correctly suppresses drift when API returns HIGH but config has MULTI_ZONE
func TestAccKafkaClusterAvailabilityDriftMultiZoneToHigh(t *testing.T) {
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

	// Step 1: Create cluster with MULTI_ZONE
	createClusterResponse, _ := ioutil.ReadFile("../testdata/kafka/create_kafka_standard.json")
	createClusterStub := wiremock.Post(wiremock.URLPathEqualTo(createKafkaPath)).
		InScenario(availabilityDriftScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateKafkaCreatedWithMultiZone).
		WillReturn(
			string(createClusterResponse),
			contentTypeJSONHeader,
			http.StatusAccepted,
		)
	_ = wiremockClient.StubFor(createClusterStub)

	// Step 2: First GET after creation returns MULTI_ZONE and transitions to provisioned state
	readCreatedClusterResponse, _ := ioutil.ReadFile("../testdata/kafka/read_created_kafka_standard.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(availabilityDriftScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(scenarioStateKafkaCreatedWithMultiZone).
		WillSetStateTo(scenarioStateKafkaProvisionedMultiZone).
		WillReturn(
			string(readCreatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// Step 3: Subsequent GETs during step 1 return MULTI_ZONE and transition to ready state
	// This allows multiple GETs during step 1 to all return MULTI_ZONE
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(availabilityDriftScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(scenarioStateKafkaProvisionedMultiZone).
		WillSetStateTo(scenarioStateKafkaReadyForHighTransition).
		WillReturn(
			string(readCreatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// Step 4: GETs from ready state return MULTI_ZONE (stay in same state)
	// This handles any additional GETs during step 1
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(availabilityDriftScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(scenarioStateKafkaReadyForHighTransition).
		WillReturn(
			string(readCreatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// Step 5: Transition to returning HIGH on GET (simulating API returning V2 billing model value)
	// This will be called during the plan refresh in step 2
	readClusterWithHighResponse, _ := ioutil.ReadFile("../testdata/kafka/read_created_kafka_standard_with_high.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(availabilityDriftScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(scenarioStateKafkaReadyForHighTransition).
		WillSetStateTo(scenarioStateKafkaApiReturnsHigh).
		WillReturn(
			string(readClusterWithHighResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// Step 5: Subsequent GETs return HIGH (after state transition)
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(availabilityDriftScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(scenarioStateKafkaApiReturnsHigh).
		WillReturn(
			string(readClusterWithHighResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readEnvironmentResponse, _ := ioutil.ReadFile("../testdata/environment/read_created_env_without_sg.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readEnvPath)).
		InScenario(availabilityDriftScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaCreatedWithMultiZone).
		WillReturn(
			string(readEnvironmentResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readEnvPath)).
		InScenario(availabilityDriftScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaProvisionedMultiZone).
		WillReturn(
			string(readEnvironmentResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readEnvPath)).
		InScenario(availabilityDriftScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaReadyForHighTransition).
		WillReturn(
			string(readEnvironmentResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readEnvPath)).
		InScenario(availabilityDriftScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaApiReturnsHigh).
		WillReturn(
			string(readEnvironmentResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// DELETE stub for cleanup
	readDeletedKafkaResponse, _ := ioutil.ReadFile("../testdata/kafka/read_deleted_kafka.json")
	deleteClusterStub := wiremock.Delete(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(availabilityDriftScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(scenarioStateKafkaApiReturnsHigh).
		WillSetStateTo("deleted").
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteClusterStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaPath)).
		InScenario(availabilityDriftScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs("deleted").
		WillReturn(
			string(readDeletedKafkaResponse),
			contentTypeJSONHeader,
			http.StatusForbidden,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckClusterDestroy,
		Steps: []resource.TestStep{
			{
				// Step 1: Create cluster with MULTI_ZONE
				Config: testAccCheckClusterConfigWithAvailability(mockServerUrl, paramStandardCluster, "MULTI_ZONE"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClusterExists(fullKafkaResourceLabel),
					resource.TestCheckResourceAttr(fullKafkaResourceLabel, "availability", "MULTI_ZONE"),
				),
			},
			{
				// Step 2: API returns HIGH, but config still has MULTI_ZONE
				// DiffSuppressFunc should suppress the diff and show no changes
				Config:             testAccCheckClusterConfigWithAvailability(mockServerUrl, paramStandardCluster, "MULTI_ZONE"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false, // Should NOT show drift after DiffSuppressFunc suppresses the diff
			},
		},
	})
}

// testAccCheckClusterConfigWithAvailability creates a cluster config with a specific availability
func testAccCheckClusterConfigWithAvailability(mockServerUrl, clusterType, availability string) string {
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
	`, mockServerUrl, kafkaDisplayName, availability, kafkaCloud, kafkaRegion, clusterType, testEnvironmentId)
}
