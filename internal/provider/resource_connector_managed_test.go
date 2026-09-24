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
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
)

func TestAccManagedConnector(t *testing.T) {
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
	validateConnectorResponse, _ := os.ReadFile("../testdata/connector/managed/validate.json")
	validateEnvStub := wiremock.Put(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connector-plugins/DatagenSourceInternal/config/validate")).
		InScenario(connectorScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateManagedConnectorHasBeenValidated).
		WillReturn(
			string(validateConnectorResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(validateEnvStub)

	createConnectorStub := wiremock.Post(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connectors")).
		InScenario(connectorScenarioName).
		WhenScenarioStateIs(scenarioStateManagedConnectorHasBeenValidated).
		WillSetStateTo(scenarioStateManagedConnectorHasBeenCreating).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createConnectorStub)

	createdConnectorResponse, _ := os.ReadFile("../testdata/connector/managed/read_created_connectors.json")
	readCreatedConnectorsStub := wiremock.Get(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connectors")).
		WithQueryParam("expand", wiremock.EqualTo("info,status,id")).
		InScenario(connectorScenarioName).
		WhenScenarioStateIs(scenarioStateManagedConnectorHasBeenCreating).
		WillSetStateTo(scenarioStateManagedConnectorFetchingId).
		WillReturn(
			string(createdConnectorResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(readCreatedConnectorsStub)

	provisioningConnectorResponse, _ := os.ReadFile("../testdata/connector/managed/read_provisioning_connector.json")
	readProvisioningConnectorStub := wiremock.Get(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connectors/test_connector/status")).
		InScenario(connectorScenarioName).
		WhenScenarioStateIs(scenarioStateManagedConnectorFetchingId).
		WillSetStateTo(scenarioStateManagedConnectorIsProvisioning).
		WillReturn(
			string(provisioningConnectorResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(readProvisioningConnectorStub)

	runningConnectorResponse, _ := os.ReadFile("../testdata/connector/managed/read_running_connector.json")
	readRunningConnectorStub1 := wiremock.Get(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connectors/test_connector/status")).
		InScenario(connectorScenarioName).
		WhenScenarioStateIs(scenarioStateManagedConnectorIsProvisioning).
		WillSetStateTo(scenarioStateManagedConnectorIsRunning1).
		WillReturn(
			string(runningConnectorResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(readRunningConnectorStub1)

	readCreatedConnectorStub := wiremock.Get(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connectors")).
		WithQueryParam("expand", wiremock.EqualTo("info,status,id")).
		InScenario(connectorScenarioName).
		WhenScenarioStateIs(scenarioStateManagedConnectorIsRunning1).
		WillSetStateTo(scenarioStateManagedConnectorHasBeenCreated).
		WillReturn(
			string(createdConnectorResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(readCreatedConnectorStub)

	readCreatedConnectorStub2 := wiremock.Get(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connectors")).
		WithQueryParam("expand", wiremock.EqualTo("info,status,id")).
		InScenario(connectorScenarioName).
		WhenScenarioStateIs(scenarioStateManagedConnectorHasBeenCreated).
		WillReturn(
			string(createdConnectorResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(readCreatedConnectorStub2)

	// Routine refreshes of an already-existing connector (i.e. any Read where
	// d.IsNewResource() is false) go through the cheap by-name config+status endpoints
	// instead of re-listing every connector (see readConnectorAndSetAttributes). Stub those
	// for every scenario state where such a refresh can happen: right after creation/import,
	// right after a config update, and right after an offsets update.
	createdConnectorConfigResponse, _ := os.ReadFile("../testdata/connector/managed/read_connector_config.json")
	readCreatedConnectorConfigStub := wiremock.Get(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connectors/test_connector/config")).
		InScenario(connectorScenarioName).
		WhenScenarioStateIs(scenarioStateManagedConnectorHasBeenCreated).
		WillReturn(
			string(createdConnectorConfigResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(readCreatedConnectorConfigStub)

	readCreatedConnectorStatusStub := wiremock.Get(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connectors/test_connector/status")).
		InScenario(connectorScenarioName).
		WhenScenarioStateIs(scenarioStateManagedConnectorHasBeenCreated).
		WillReturn(
			string(runningConnectorResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(readCreatedConnectorStatusStub)

	updateConnectorStub := wiremock.Put(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connectors/test_connector/config")).
		InScenario(connectorScenarioName).
		WhenScenarioStateIs(scenarioStateManagedConnectorHasBeenCreated).
		WillSetStateTo(scenarioStateManagedConnectorNameHasBeenUpdated).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(updateConnectorStub)

	updateConnectorOffsetStub := wiremock.Post(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connectors/test_connector/offsets/request")).
		WhenScenarioStateIs(scenarioStateManagedConnectorNameHasBeenUpdated).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusAccepted,
		)
	_ = wiremockClient.StubFor(updateConnectorOffsetStub)

	updatedConnectorOffsetsResponse, _ := os.ReadFile("../testdata/connector/managed/read_updated_connector_offset_status.json")
	updatedConnectorOffsetStatusStub := wiremock.Get(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connectors/test_connector/offsets/request/status")).
		WhenScenarioStateIs(scenarioStateManagedConnectorNameHasBeenUpdated).
		WillSetStateTo(scenarioStateManagedConnectorOffsetHasBeenUpdated).
		WillReturn(
			string(updatedConnectorOffsetsResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(updatedConnectorOffsetStatusStub)

	updatedConnectorResponse, _ := os.ReadFile("../testdata/connector/managed/read_updated_connectors.json")
	readUpdatedConnectorStub := wiremock.Get(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connectors")).
		WithQueryParam("expand", wiremock.EqualTo("info,status,id")).
		InScenario(connectorScenarioName).
		WhenScenarioStateIs(scenarioStateManagedConnectorNameHasBeenUpdated).
		WillReturn(
			string(updatedConnectorResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(readUpdatedConnectorStub)

	// Same as above: cover routine by-name refreshes after the config update, and again
	// after the offsets update (offsets don't change config/status, so both states reuse
	// the same "updated" config+status fixtures). Kept as separate named stubs (rather than
	// looped) so the counts below can assert on each scenario state independently.
	updatedConnectorConfigResponse, _ := os.ReadFile("../testdata/connector/managed/read_updated_connector_config.json")
	readUpdatedConnectorConfigStubAfterNameUpdate := wiremock.Get(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connectors/test_connector/config")).
		InScenario(connectorScenarioName).
		WhenScenarioStateIs(scenarioStateManagedConnectorNameHasBeenUpdated).
		WillReturn(
			string(updatedConnectorConfigResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(readUpdatedConnectorConfigStubAfterNameUpdate)

	readUpdatedConnectorStatusStubAfterNameUpdate := wiremock.Get(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connectors/test_connector/status")).
		InScenario(connectorScenarioName).
		WhenScenarioStateIs(scenarioStateManagedConnectorNameHasBeenUpdated).
		WillReturn(
			string(runningConnectorResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(readUpdatedConnectorStatusStubAfterNameUpdate)

	readUpdatedConnectorConfigStubAfterOffsetUpdate := wiremock.Get(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connectors/test_connector/config")).
		InScenario(connectorScenarioName).
		WhenScenarioStateIs(scenarioStateManagedConnectorOffsetHasBeenUpdated).
		WillReturn(
			string(updatedConnectorConfigResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(readUpdatedConnectorConfigStubAfterOffsetUpdate)

	readUpdatedConnectorStatusStubAfterOffsetUpdate := wiremock.Get(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connectors/test_connector/status")).
		InScenario(connectorScenarioName).
		WhenScenarioStateIs(scenarioStateManagedConnectorOffsetHasBeenUpdated).
		WillReturn(
			string(runningConnectorResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(readUpdatedConnectorStatusStubAfterOffsetUpdate)

	deleteConnectorResponse, _ := os.ReadFile("../testdata/connector/managed/delete_connector.json")
	deleteConnectorStub := wiremock.Delete(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connectors/test_connector")).
		InScenario(connectorScenarioName).
		WhenScenarioStateIs(scenarioStateManagedConnectorNameHasBeenUpdated).
		WillSetStateTo(scenarioStateManagedConnectorHasBeenDeleted).
		WillReturn(
			string(deleteConnectorResponse),
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteConnectorStub)

	readDeletedConnectorResponse, _ := os.ReadFile("../testdata/connector/managed/read_deleted_connector.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connectors/test_connector")).
		InScenario(connectorScenarioName).
		WhenScenarioStateIs(scenarioStateManagedConnectorHasBeenDeleted).
		WillReturn(
			string(readDeletedConnectorResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	connectorResourceLabel := "test_connector_resource_label"
	fullConnectorResourceLabel := fmt.Sprintf("confluent_connector.%s", connectorResourceLabel)
	connectorDisplayName := "test_connector"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckConnectorDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckManagedConnectorConfig(mockServerUrl, connectorResourceLabel, connectorDisplayName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConnectorExists(fullConnectorResourceLabel),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, paramId, "lcc-abc123"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), "env-1j3m9j"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.#", paramKafkaCluster), "1"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.0.%s", paramKafkaCluster, paramId), "lkc-vnwdjz"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, paramStatus, "RUNNING"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.%%", paramSensitiveConfig), "1"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.%s", paramSensitiveConfig, sensitiveAttributeKey), sensitiveAttributeValue),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.%%", paramNonSensitiveConfig), "6"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.%s", paramNonSensitiveConfig, connectorConfigAttributeClass), "DatagenSourceInternal"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.%s", paramNonSensitiveConfig, "kafka.topic"), "test_topic"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.%s", paramNonSensitiveConfig, connectorConfigAttributeName), "test_connector"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.%s", paramNonSensitiveConfig, "output.data.format"), "JSON"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.%s", paramNonSensitiveConfig, "quickstart"), "ORDERS"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.%s", paramNonSensitiveConfig, "tasks.max"), "1"),
					// Ensure these attributes (from ignoredConnectorConfigs) are not visible in the output
					resource.TestCheckNoResourceAttr(fullConnectorResourceLabel, "cloud.environment"),
					resource.TestCheckNoResourceAttr(fullConnectorResourceLabel, "cloud.provider"),
					resource.TestCheckNoResourceAttr(fullConnectorResourceLabel, "kafka.endpoint"),
					resource.TestCheckNoResourceAttr(fullConnectorResourceLabel, "kafka.max.partition.validation.disable"),
					resource.TestCheckNoResourceAttr(fullConnectorResourceLabel, "kafka.region"),
					resource.TestCheckNoResourceAttr(fullConnectorResourceLabel, "kafka.dedicated"),
					resource.TestCheckNoResourceAttr(fullConnectorResourceLabel, "schema.registry.url"),
					resource.TestCheckNoResourceAttr(fullConnectorResourceLabel, "valid.kafka.api.key"),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:            fullConnectorResourceLabel,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{paramSensitiveConfig, paramOffsetsConfig},
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					environmentId := resources[fullConnectorResourceLabel].Primary.Attributes["environment.0.id"]
					clusterId := resources[fullConnectorResourceLabel].Primary.Attributes["kafka_cluster.0.id"]
					name := resources[fullConnectorResourceLabel].Primary.Attributes[fmt.Sprintf("%s.%s", paramNonSensitiveConfig, connectorConfigAttributeName)]
					return environmentId + "/" + clusterId + "/" + name, nil
				},
			},
			{
				Config: testAccCheckUpdatedManagedConnectorConfig(mockServerUrl, connectorResourceLabel, connectorDisplayName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConnectorExists(fullConnectorResourceLabel),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, paramId, "lcc-abc123"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), "env-1j3m9j"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.#", paramKafkaCluster), "1"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.0.%s", paramKafkaCluster, paramId), "lkc-vnwdjz"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, paramStatus, "RUNNING"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.%%", paramSensitiveConfig), "1"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.%s", paramSensitiveConfig, sensitiveAttributeKey), sensitiveAttributeUpdatedValue),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.%%", paramNonSensitiveConfig), "7"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.%s", paramNonSensitiveConfig, connectorConfigAttributeClass), "DatagenSourceInternal"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.%s", paramNonSensitiveConfig, "kafka.topic"), "test_topic"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.%s", paramNonSensitiveConfig, connectorConfigAttributeName), "test_connector"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.%s", paramNonSensitiveConfig, "output.data.format"), "AVRO"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.%s", paramNonSensitiveConfig, "quickstart"), "ORDERS"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.%s", paramNonSensitiveConfig, "tasks.max"), "1"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.%s", paramNonSensitiveConfig, "max.interval"), "123"),
					// Ensure these attributes (from ignoredConnectorConfigs) are not visible in the output
					resource.TestCheckNoResourceAttr(fullConnectorResourceLabel, "cloud.environment"),
					resource.TestCheckNoResourceAttr(fullConnectorResourceLabel, "cloud.provider"),
					resource.TestCheckNoResourceAttr(fullConnectorResourceLabel, "kafka.endpoint"),
					resource.TestCheckNoResourceAttr(fullConnectorResourceLabel, "kafka.max.partition.validation.disable"),
					resource.TestCheckNoResourceAttr(fullConnectorResourceLabel, "kafka.region"),
					resource.TestCheckNoResourceAttr(fullConnectorResourceLabel, "kafka.dedicated"),
					resource.TestCheckNoResourceAttr(fullConnectorResourceLabel, "schema.registry.url"),
					resource.TestCheckNoResourceAttr(fullConnectorResourceLabel, "valid.kafka.api.key"),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:            fullConnectorResourceLabel,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{paramSensitiveConfig, paramOffsetsConfig},
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					environmentId := resources[fullConnectorResourceLabel].Primary.Attributes["environment.0.id"]
					clusterId := resources[fullConnectorResourceLabel].Primary.Attributes["kafka_cluster.0.id"]
					name := resources[fullConnectorResourceLabel].Primary.Attributes[fmt.Sprintf("%s.%s", paramNonSensitiveConfig, connectorConfigAttributeName)]
					return environmentId + "/" + clusterId + "/" + name, nil
				},
			},
			{
				Config: testAccCheckManagedConnectorOffsetsConfig(mockServerUrl, connectorResourceLabel, connectorDisplayName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConnectorExists(fullConnectorResourceLabel),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, paramId, "lcc-abc123"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), "env-1j3m9j"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.#", paramKafkaCluster), "1"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.0.%s", paramKafkaCluster, paramId), "lkc-vnwdjz"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, paramStatus, "RUNNING"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.%%", paramSensitiveConfig), "1"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.%%", paramNonSensitiveConfig), "7"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.%s", paramNonSensitiveConfig, connectorConfigAttributeClass), "DatagenSourceInternal"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.%s", paramNonSensitiveConfig, "kafka.topic"), "test_topic"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.%s", paramNonSensitiveConfig, connectorConfigAttributeName), "test_connector"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.%s", paramNonSensitiveConfig, "output.data.format"), "AVRO"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.%s", paramNonSensitiveConfig, "quickstart"), "ORDERS"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.%s", paramNonSensitiveConfig, "tasks.max"), "1"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.#", paramOffsetsConfig), "1"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.0.%s.%s", paramOffsetsConfig, paramPartition, "kafka_partition"), "0"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.0.%s.%s", paramOffsetsConfig, paramPartition, "kafka_topic"), "test_topic"),
					resource.TestCheckResourceAttr(fullConnectorResourceLabel, fmt.Sprintf("%s.0.%s.%s", paramOffsetsConfig, paramOffset, "kafka_offset"), "500"),
					// Ensure these attributes (from ignoredConnectorConfigs) are not visible in the output
					resource.TestCheckNoResourceAttr(fullConnectorResourceLabel, "cloud.environment"),
					resource.TestCheckNoResourceAttr(fullConnectorResourceLabel, "cloud.provider"),
					resource.TestCheckNoResourceAttr(fullConnectorResourceLabel, "kafka.endpoint"),
					resource.TestCheckNoResourceAttr(fullConnectorResourceLabel, "kafka.max.partition.validation.disable"),
					resource.TestCheckNoResourceAttr(fullConnectorResourceLabel, "kafka.region"),
					resource.TestCheckNoResourceAttr(fullConnectorResourceLabel, "kafka.dedicated"),
					resource.TestCheckNoResourceAttr(fullConnectorResourceLabel, "schema.registry.url"),
					resource.TestCheckNoResourceAttr(fullConnectorResourceLabel, "valid.kafka.api.key"),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:            fullConnectorResourceLabel,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{paramSensitiveConfig, paramOffsetsConfig},
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					environmentId := resources[fullConnectorResourceLabel].Primary.Attributes["environment.0.id"]
					clusterId := resources[fullConnectorResourceLabel].Primary.Attributes["kafka_cluster.0.id"]
					name := resources[fullConnectorResourceLabel].Primary.Attributes[fmt.Sprintf("%s.%s", paramNonSensitiveConfig, connectorConfigAttributeName)]
					return environmentId + "/" + clusterId + "/" + name, nil
				},
			},
		},
	})

	// The whole point of the fix in readConnectorAndSetAttributes is that a routine refresh
	// (d.IsNewResource() == false) uses the cheap by-name config/status endpoints instead of
	// re-listing every connector in the cluster. The full-list stubs above stay registered at
	// every scenario state precisely so a regression back to "always list" would still produce
	// a passing plan (same fixture data either way) -- only a request-count check catches that,
	// which is why we assert on actual counts here rather than only on resulting state values.
	//
	// GetCountRequests matches by request pattern (method + URL), not by which scenario-state
	// stub variable we reference, so any of the full-list (or by-name) stub variables above
	// report the same total regardless of which one we pass in here.
	//
	// Exactly 5 full-list (?expand=info,status,id) calls are structurally expected no matter how
	// many extra refresh passes the test framework does internally: 2 from Create (the
	// ID-discovery call, plus Create's own trailing Read -- both while IsNewResource() is still
	// true) + 1 per ImportState step (3 in this test), since Import is the only other path that
	// legitimately needs the list endpoint to discover the connector's LCC id. Every other Read
	// in this test must go through the by-name endpoints instead, so this count must never grow.
	checkStubCount(t, wiremockClient, readCreatedConnectorStub2, "GET .../connectors?expand=info,status,id (full-list)", 5)

	// By-name config is only ever reached via the routine-refresh path, so its count is a direct
	// measure of how many non-Create/non-Import refreshes happened across all three Config steps
	// (create, name/config update, offsets update) plus the test framework's own
	// plan-consistency refreshes. By-name status shares its URL with two extra calls from the
	// create-time provisioning poll (waitForConnectorToProvision), hence the +2 versus config.
	checkStubCount(t, wiremockClient, readCreatedConnectorConfigStub, "GET .../connectors/test_connector/config (by-name)", 10)
	checkStubCount(t, wiremockClient, readCreatedConnectorStatusStub, "GET .../connectors/test_connector/status (by-name + provisioning poll)", 12)
}

func testAccCheckConnectorDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each connector is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_connector" {
			continue
		}
		deletedConnectorName := rs.Primary.Attributes["config_nonsensitive.name"]
		deletedConnectorEnvId := rs.Primary.Attributes["environment.0.id"]
		deletedConnectorKafkaClusterId := rs.Primary.Attributes["kafka_cluster.0.id"]
		req := c.connectV1Client.ConnectorsConnectV1Api.ReadConnectv1Connector(c.connectV1ApiContext(context.Background()), deletedConnectorName, deletedConnectorEnvId, deletedConnectorKafkaClusterId)
		_, response, _ := req.Execute()
		if isNonKafkaRestApiResourceNotFound(response) {
			return nil
		}
		return fmt.Errorf("connector %q still exists", deletedConnectorName)
	}
	return nil
}

func testAccCheckManagedConnectorConfig(mockServerUrl, environmentConnectorLabel, connectorDisplayName string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_connector" "%s" {
		environment {
		  id = "env-1j3m9j"
		}
		kafka_cluster {
		  id = "lkc-vnwdjz"
		}
		config_sensitive = {
		  "%s"             = "%s"
		}
		config_nonsensitive = {
		  "name"            = "%s"
		  "connector.class" = "DatagenSourceInternal"
		  "kafka.topic" = "test_topic"
		  "output.data.format" = "JSON"
		  "tasks.max" = "1"
		  "quickstart" = "ORDERS"
		}
	}
	`, mockServerUrl, environmentConnectorLabel, sensitiveAttributeKey, sensitiveAttributeValue, connectorDisplayName)
}

func testAccCheckUpdatedManagedConnectorConfig(mockServerUrl, environmentConnectorLabel, connectorDisplayName string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_connector" "%s" {
		environment {
		  id = "env-1j3m9j"
		}
		kafka_cluster {
		  id = "lkc-vnwdjz"
		}
		config_sensitive = {
		  "%s"             = "%s"
		}
		config_nonsensitive = {
		  "name"            = "%s"
		  "connector.class" = "DatagenSourceInternal"
		  "kafka.topic" = "test_topic"
		  "output.data.format" = "AVRO"
		  "max.interval" = "123"
		  "tasks.max" = "1"
		  "quickstart" = "ORDERS"
		}
	}
	`, mockServerUrl, environmentConnectorLabel, sensitiveAttributeKey, sensitiveAttributeUpdatedValue, connectorDisplayName)
}

func testAccCheckManagedConnectorOffsetsConfig(mockServerUrl, environmentConnectorLabel, connectorDisplayName string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_connector" "%s" {
		environment {
		  id = "env-1j3m9j"
		}
		kafka_cluster {
		  id = "lkc-vnwdjz"
		}
		config_sensitive = {
		  "%s"             = "%s"
		}
		config_nonsensitive = {
		  "name"            = "%s"
		  "connector.class" = "DatagenSourceInternal"
		  "kafka.topic" = "test_topic"
		  "output.data.format" = "AVRO"
		  "max.interval" = "123"
		  "tasks.max" = "1"
		  "quickstart" = "ORDERS"
		}
		offsets {
			partition = {
				"kafka_partition" = 0,
				"kafka_topic" = "test_topic"
			}
			offset = {
				"kafka_offset" = 500
			}
		}
	}
	`, mockServerUrl, environmentConnectorLabel, sensitiveAttributeKey, sensitiveAttributeUpdatedValue, connectorDisplayName)
}

func testAccCheckConnectorExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("%s connector has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s environment", n)
		}

		return nil
	}
}

// TestConvertMapTypes tests the map type conversion function to ensure connector offset values are converted from strings to proper types
func TestConvertMapTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "Offsets with LSN",
			input: map[string]interface{}{
				"lsn": "123456789",
			},
			expected: map[string]interface{}{
				"lsn": int64(123456789),
			},
		},
		{
			name: "Mixed types in offset map",
			input: map[string]interface{}{
				"lsn_proc":    "true",
				"messageType": "INSERT",
				"lsn":         "123456789",
			},
			expected: map[string]interface{}{
				"lsn_proc":    bool(true),
				"messageType": "INSERT",
				"lsn":         int64(123456789),
			},
		},
		{
			name: "All string values",
			input: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
			expected: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name: "Mixed string and non-string values",
			input: map[string]interface{}{
				"lsn":          "123456789",
				"already_int":  int64(123),
				"already_bool": true,
			},
			expected: map[string]interface{}{
				"lsn":          int64(123456789),
				"already_int":  int64(123),
				"already_bool": true,
			},
		},
		{
			name: "Decimal numbers",
			input: map[string]interface{}{
				"key1": "99.99",
				"key2": "42",
			},
			expected: map[string]interface{}{
				"key1": float64(99.99),
				"key2": int64(42),
			},
		},
		{
			name:     "Empty map",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertMapTypes(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("convertMapTypes() returned map with length %d, expected %d",
					len(result), len(tt.expected))
				return
			}
			for key, expectedValue := range tt.expected {
				actualValue, ok := result[key]
				if !ok {
					t.Errorf("convertMapTypes() missing key %q", key)
					continue
				}
				if actualValue != expectedValue {
					t.Errorf("convertMapTypes() [%q] = %v (type %T), expected %v (type %T)",
						key, actualValue, actualValue, expectedValue, expectedValue)
				}
			}
		})
	}
}
