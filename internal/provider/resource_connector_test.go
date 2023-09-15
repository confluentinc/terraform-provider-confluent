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
	scenarioStateConnectorHasBeenValidated   = "The new connector config has been just validated"
	scenarioStateConnectorHasBeenCreating    = "The new connector has been creating"
	scenarioStateConnectorFetchingId         = "The new connector is in provisioning state, list all connectors"
	scenarioStateConnectorIsProvisioning     = "The new connector is in provisioning state"
	scenarioStateConnectorIsRunning1         = "The new connector is in running state #1"
	scenarioStateConnectorHasBeenCreated     = "The new connector has been just created"
	scenarioStateConnectorNameHasBeenUpdated = "The new connector's name has been just updated"
	scenarioStateConnectorHasBeenDeleted     = "The new connector has been deleted"
	connectorScenarioName                    = "confluent_connector Resource Lifecycle"
	sensitiveAttributeKey                    = "foo"
	sensitiveAttributeValue                  = "bar"
	sensitiveAttributeUpdatedValue           = "bar updated"
)

func TestAccConnector(t *testing.T) {
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
	validateConnectorResponse, _ := ioutil.ReadFile("../testdata/connector/validate.json")
	validateEnvStub := wiremock.Put(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connector-plugins/DatagenSourceInternal/config/validate")).
		InScenario(connectorScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateConnectorHasBeenValidated).
		WillReturn(
			string(validateConnectorResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(validateEnvStub)

	createConnectorStub := wiremock.Post(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connectors")).
		InScenario(connectorScenarioName).
		WhenScenarioStateIs(scenarioStateConnectorHasBeenValidated).
		WillSetStateTo(scenarioStateConnectorHasBeenCreating).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createConnectorStub)

	createdConnectorResponse, _ := ioutil.ReadFile("../testdata/connector/read_created_connectors.json")
	readCreatedConnectorsStub := wiremock.Get(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connectors")).
		WithQueryParam("expand", wiremock.EqualTo("info,status,id")).
		InScenario(connectorScenarioName).
		WhenScenarioStateIs(scenarioStateConnectorHasBeenCreating).
		WillSetStateTo(scenarioStateConnectorFetchingId).
		WillReturn(
			string(createdConnectorResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(readCreatedConnectorsStub)

	provisioningConnectorResponse, _ := ioutil.ReadFile("../testdata/connector/read_provisioning_connector.json")
	readProvisioningConnectorStub := wiremock.Get(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connectors/test_connector/status")).
		InScenario(connectorScenarioName).
		WhenScenarioStateIs(scenarioStateConnectorFetchingId).
		WillSetStateTo(scenarioStateConnectorIsProvisioning).
		WillReturn(
			string(provisioningConnectorResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(readProvisioningConnectorStub)

	runningConnectorResponse, _ := ioutil.ReadFile("../testdata/connector/read_running_connector.json")
	readRunningConnectorStub1 := wiremock.Get(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connectors/test_connector/status")).
		InScenario(connectorScenarioName).
		WhenScenarioStateIs(scenarioStateConnectorIsProvisioning).
		WillSetStateTo(scenarioStateConnectorIsRunning1).
		WillReturn(
			string(runningConnectorResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(readRunningConnectorStub1)

	readCreatedConnectorStub := wiremock.Get(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connectors")).
		WithQueryParam("expand", wiremock.EqualTo("info,status,id")).
		InScenario(connectorScenarioName).
		WhenScenarioStateIs(scenarioStateConnectorIsRunning1).
		WillSetStateTo(scenarioStateConnectorHasBeenCreated).
		WillReturn(
			string(createdConnectorResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(readCreatedConnectorStub)

	readCreatedConnectorStub2 := wiremock.Get(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connectors")).
		WithQueryParam("expand", wiremock.EqualTo("info,status,id")).
		InScenario(connectorScenarioName).
		WhenScenarioStateIs(scenarioStateConnectorHasBeenCreated).
		WillReturn(
			string(createdConnectorResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(readCreatedConnectorStub2)

	updateConnectorStub := wiremock.Put(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connectors/test_connector/config")).
		InScenario(connectorScenarioName).
		WhenScenarioStateIs(scenarioStateConnectorHasBeenCreated).
		WillSetStateTo(scenarioStateConnectorNameHasBeenUpdated).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(updateConnectorStub)

	updatedConnectorResponse, _ := ioutil.ReadFile("../testdata/connector/read_updated_connectors.json")
	readUpdatedConnectorStub := wiremock.Get(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connectors")).
		WithQueryParam("expand", wiremock.EqualTo("info,status,id")).
		InScenario(connectorScenarioName).
		WhenScenarioStateIs(scenarioStateConnectorNameHasBeenUpdated).
		WillReturn(
			string(updatedConnectorResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(readUpdatedConnectorStub)

	deleteConnectorResponse, _ := ioutil.ReadFile("../testdata/connector/delete_connector.json")
	deleteConnectorStub := wiremock.Delete(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connectors/test_connector")).
		InScenario(connectorScenarioName).
		WhenScenarioStateIs(scenarioStateConnectorNameHasBeenUpdated).
		WillSetStateTo(scenarioStateConnectorHasBeenDeleted).
		WillReturn(
			string(deleteConnectorResponse),
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteConnectorStub)

	readDeletedConnectorResponse, _ := ioutil.ReadFile("../testdata/connector/read_deleted_connector.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/connect/v1/environments/env-1j3m9j/clusters/lkc-vnwdjz/connectors/test_connector")).
		InScenario(connectorScenarioName).
		WhenScenarioStateIs(scenarioStateConnectorHasBeenDeleted).
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
				Config: testAccCheckConnectorConfig(mockServerUrl, connectorResourceLabel, connectorDisplayName),
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
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:            fullConnectorResourceLabel,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{paramSensitiveConfig},
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					environmentId := resources[fullConnectorResourceLabel].Primary.Attributes["environment.0.id"]
					clusterId := resources[fullConnectorResourceLabel].Primary.Attributes["kafka_cluster.0.id"]
					name := resources[fullConnectorResourceLabel].Primary.Attributes[fmt.Sprintf("%s.%s", paramNonSensitiveConfig, connectorConfigAttributeName)]
					return environmentId + "/" + clusterId + "/" + name, nil
				},
			},
			{
				Config: testAccCheckUpdatedConnectorConfig(mockServerUrl, connectorResourceLabel, connectorDisplayName),
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
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:            fullConnectorResourceLabel,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{paramSensitiveConfig},
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
		req := c.connectClient.ConnectorsV1Api.ReadConnectv1Connector(c.connectApiContext(context.Background()), deletedConnectorName, deletedConnectorEnvId, deletedConnectorKafkaClusterId)
		_, response, _ := req.Execute()
		if isNonKafkaRestApiResourceNotFound(response) {
			return nil
		}
		return fmt.Errorf("connector %q still exists", deletedConnectorName)
	}
	return nil
}

func testAccCheckConnectorConfig(mockServerUrl, environmentConnectorLabel, connectorDisplayName string) string {
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

func testAccCheckUpdatedConnectorConfig(mockServerUrl, environmentConnectorLabel, connectorDisplayName string) string {
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
		  "max.interval" = "123",
		  "tasks.max" = "1"
		  "quickstart" = "ORDERS"
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
