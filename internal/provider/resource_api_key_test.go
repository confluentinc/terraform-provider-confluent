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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/walkerus/go-wiremock"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	scenarioStateKafkaApiKeyHasBeenCreated                = "The new kafka api key has been just created"
	scenarioStateKafkaApiKeyHasBeenSyncedFirstRead        = "The new kafka api key has been just synced (read #1)"
	scenarioStateKafkaApiKeyHasBeenSyncedConfirmationRead = "The new kafka api key has been just synced (final read)"
	scenarioStateKafkaApiKeyHasBeenUpdated                = "The new kafka api key's description and display_name have been just updated"
	scenarioStateKafkaApiKeyHasBeenDeleted                = "The new kafka api key has been deleted"
	kafkaApiKeyScenarioName                               = "confluent_api_key (Kafka API Key) Resource Lifecycle"

	scenarioStateFlinkApiKeyHasBeenCreated                = "The new flink api key has been just created"
	scenarioStateFlinkApiKeyHasBeenSyncedFirstRead        = "The new flink api key has been just synced (read #1)"
	scenarioStateFlinkApiKeyHasBeenSyncedConfirmationRead = "The new flink api key has been just synced (final read)"
	scenarioStateFlinkApiKeyHasBeenUpdated                = "The new flink api key's description and display_name have been just updated"
	scenarioStateFlinkApiKeyHasBeenDeleted                = "The new flink api key has been deleted"
	flinkApiKeyScenarioName                               = "confluent_api_key (Flink API Key) Resource Lifecycle"

	scenarioStateCloudApiKeyHasBeenCreated = "The new cloud api key has been just created"
	scenarioStateCloudApiKeyHasBeenSynced  = "The new cloud api key has been just synced"
	scenarioStateCloudApiKeyHasBeenUpdated = "The new cloud api key's description and display_name have been just updated"
	scenarioStateCloudApiKeyHasBeenDeleted = "The new cloud api key has been deleted"
	cloudApiKeyScenarioName                = "confluent_api_key (Cloud API Key) Resource Lifecycle"

	scenarioStateTableflowApiKeyHasBeenCreated = "The new tableflow api key has been just created"
	scenarioStateTableflowApiKeyHasBeenSynced  = "The new tableflow api key has been just synced"
	scenarioStateTableflowApiKeyHasBeenUpdated = "The new tableflow api key's description and display_name have been just updated"
	scenarioStateTableflowApiKeyHasBeenDeleted = "The new tableflow api key has been deleted"
	tableflowApiKeyScenarioName                = "confluent_api_key (Tableflow API Key) Resource Lifecycle"
)

func TestAccKafkaApiKey(t *testing.T) {
	ctx := context.Background()

	time.Sleep(5 * time.Second)
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
	createKafkaApiKeyResponse, _ := ioutil.ReadFile("../testdata/apikey/create_kafka_api_key.json")
	createKafkaApiKeyStub := wiremock.Post(wiremock.URLPathEqualTo("/iam/v2/api-keys")).
		InScenario(kafkaApiKeyScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(createKafkaApiKeyResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createKafkaApiKeyStub)

	createKafkaCmkApiResponse, _ := ioutil.ReadFile("../testdata/apikey/read_kafka.json")
	var createKafkaCmkApiResponseMap map[string]interface{}
	_ = json.Unmarshal(createKafkaCmkApiResponse, &createKafkaCmkApiResponseMap)
	// Override http endpoint to mock Kafka REST API responses
	createKafkaCmkApiResponseMap["spec"].(map[string]interface{})["http_endpoint"] = mockServerUrl
	createKafkaCmkApiResponseWithUpdatedHttpEndpoint, _ := json.Marshal(createKafkaCmkApiResponseMap)
	createCmkApiStub := wiremock.Get(wiremock.URLPathEqualTo("/cmk/v2/clusters/lkc-zmmq63")).
		InScenario(kafkaApiKeyScenarioName).
		WithQueryParam("environment", wiremock.EqualTo("env-12345")).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(createKafkaCmkApiResponseWithUpdatedHttpEndpoint),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(createCmkApiStub)

	kafkaRestApi401Response, _ := ioutil.ReadFile("../testdata/apikey/read_list_topics_401.html")
	listTopicsKafkaRestApi401Stub := wiremock.Get(wiremock.URLPathEqualTo("/kafka/v3/clusters/lkc-zmmq63/topics")).
		InScenario(kafkaApiKeyScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateKafkaApiKeyHasBeenSyncedFirstRead).
		WillReturn(
			string(kafkaRestApi401Response),
			contentTypeJSONHeader,
			http.StatusUnauthorized,
		)
	_ = wiremockClient.StubFor(listTopicsKafkaRestApi401Stub)

	kafkaRestApi200Response, _ := ioutil.ReadFile("../testdata/apikey/read_list_topics_200.json")
	listTopicsKafkaRestApi200Stub := wiremock.Get(wiremock.URLPathEqualTo("/kafka/v3/clusters/lkc-zmmq63/topics")).
		InScenario(kafkaApiKeyScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaApiKeyHasBeenSyncedFirstRead).
		WillSetStateTo(scenarioStateKafkaApiKeyHasBeenSyncedConfirmationRead).
		WillReturn(
			string(kafkaRestApi200Response),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(listTopicsKafkaRestApi200Stub)

	listTopicsKafkaRestApi200ConfirmationStub := wiremock.Get(wiremock.URLPathEqualTo("/kafka/v3/clusters/lkc-zmmq63/topics")).
		InScenario(kafkaApiKeyScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaApiKeyHasBeenSyncedConfirmationRead).
		WillSetStateTo(scenarioStateKafkaApiKeyHasBeenCreated).
		WillReturn(
			string(kafkaRestApi200Response),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(listTopicsKafkaRestApi200ConfirmationStub)

	readCreatedKafkaApiKeyResponse, _ := ioutil.ReadFile("../testdata/apikey/read_created_kafka_api_key.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/api-keys/7FJIYKQ4SGQDQ72H")).
		InScenario(kafkaApiKeyScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaApiKeyHasBeenCreated).
		WillReturn(
			string(readCreatedKafkaApiKeyResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedKafkaApiKeyResponse, _ := ioutil.ReadFile("../testdata/apikey/read_updated_kafka_api_key.json")
	patchKafkaApiKeyStub := wiremock.Patch(wiremock.URLPathEqualTo("/iam/v2/api-keys/7FJIYKQ4SGQDQ72H")).
		InScenario(kafkaApiKeyScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaApiKeyHasBeenCreated).
		WillSetStateTo(scenarioStateKafkaApiKeyHasBeenUpdated).
		WillReturn(
			string(readUpdatedKafkaApiKeyResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(patchKafkaApiKeyStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/api-keys/7FJIYKQ4SGQDQ72H")).
		InScenario(kafkaApiKeyScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaApiKeyHasBeenUpdated).
		WillReturn(
			string(readUpdatedKafkaApiKeyResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedKafkaApiKeyResponse, _ := ioutil.ReadFile("../testdata/apikey/read_deleted_kafka_api_key.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/api-keys/7FJIYKQ4SGQDQ72H")).
		InScenario(kafkaApiKeyScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaApiKeyHasBeenDeleted).
		WillReturn(
			string(readDeletedKafkaApiKeyResponse),
			contentTypeJSONHeader,
			http.StatusForbidden,
		))
	deleteKafkaApiKeyStub := wiremock.Delete(wiremock.URLPathEqualTo("/iam/v2/api-keys/7FJIYKQ4SGQDQ72H")).
		InScenario(kafkaApiKeyScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaApiKeyHasBeenUpdated).
		WillSetStateTo(scenarioStateKafkaApiKeyHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteKafkaApiKeyStub)

	kafkaApiKeyDisplayName := "CI Kafka API Key"
	kafkaApiKeyDescription := "This API key provides kafka access to cluster x"
	// in order to test tf update (step #3)
	kafkaApiKeyUpdatedDisplayName := "CI Kafka API Key updated"
	kafkaApiKeyUpdatedDescription := "This API key provides kafka access to cluster x updated"
	kafkaApiKeyResourceLabel := "test_cluster_api_key_resource_label"
	fullKafkaApiKeyResourceLabel := fmt.Sprintf("confluent_api_key.%s", kafkaApiKeyResourceLabel)
	ownerId := "sa-12mgdv"
	ownerApiVersion := "iam/v2"
	ownerKind := "ServiceAccount"
	resourceId := "lkc-zmmq63"
	resourceApiVersion := "cmk/v2"
	resourceKind := "Cluster"
	environmentId := "env-12345"

	// Set fake values for secrets since those are required for importing
	os.Setenv("API_KEY_SECRET", "gtH2gI504c0rqSppdMPqFu7BypmleQVImiJGNxlCNlhR2kNhGY86XGi49Rp3bmaY")
	defer func() {
		os.Unsetenv("API_KEY_SECRET")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckApiKeyDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckKafkaApiKeyConfig(mockServerUrl, kafkaApiKeyResourceLabel, kafkaApiKeyDisplayName, kafkaApiKeyDescription, ownerId, ownerApiVersion, ownerKind, resourceId, resourceApiVersion, resourceKind, environmentId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckApiKeyExists(fullKafkaApiKeyResourceLabel),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "id", "7FJIYKQ4SGQDQ72H"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "display_name", kafkaApiKeyDisplayName),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "description", kafkaApiKeyDescription),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "owner.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "owner.0.%", "3"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "owner.0.api_version", "iam/v2"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "owner.0.id", "sa-12mgdv"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "owner.0.kind", "ServiceAccount"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "managed_resource.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "managed_resource.0.%", "4"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "managed_resource.0.api_version", "cmk/v2"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "managed_resource.0.id", "lkc-zmmq63"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "managed_resource.0.kind", "Cluster"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "managed_resource.0.environment.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "managed_resource.0.environment.0.%", "1"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "managed_resource.0.environment.0.id", "env-12345"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "secret", "gtH2gI504c0rqSppdMPqFu7BypmleQVImiJGNxlCNlhR2kNhGY86XGi49Rp3bmaY"),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullKafkaApiKeyResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					kafkaApiKeyId := resources[fullKafkaApiKeyResourceLabel].Primary.ID
					environmentId := resources[fullKafkaApiKeyResourceLabel].Primary.Attributes["managed_resource.0.environment.0.id"]
					return environmentId + "/" + kafkaApiKeyId, nil
				},
			},
			{
				Config: testAccCheckKafkaApiKeyConfig(mockServerUrl, kafkaApiKeyResourceLabel, kafkaApiKeyUpdatedDisplayName, kafkaApiKeyUpdatedDescription, ownerId, ownerApiVersion, ownerKind, resourceId, resourceApiVersion, resourceKind, environmentId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckApiKeyExists(fullKafkaApiKeyResourceLabel),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "id", "7FJIYKQ4SGQDQ72H"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "display_name", kafkaApiKeyUpdatedDisplayName),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "description", kafkaApiKeyUpdatedDescription),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "disable_wait_for_ready", "false"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "owner.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "owner.0.%", "3"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "owner.0.api_version", "iam/v2"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "owner.0.id", "sa-12mgdv"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "owner.0.kind", "ServiceAccount"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "managed_resource.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "managed_resource.0.%", "4"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "managed_resource.0.api_version", "cmk/v2"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "managed_resource.0.id", "lkc-zmmq63"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "managed_resource.0.kind", "Cluster"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "managed_resource.0.environment.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "managed_resource.0.environment.0.%", "1"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "managed_resource.0.environment.0.id", "env-12345"),
					resource.TestCheckResourceAttr(fullKafkaApiKeyResourceLabel, "secret", "gtH2gI504c0rqSppdMPqFu7BypmleQVImiJGNxlCNlhR2kNhGY86XGi49Rp3bmaY"),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullKafkaApiKeyResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					kafkaApiKeyId := resources[fullKafkaApiKeyResourceLabel].Primary.ID
					environmentId := resources[fullKafkaApiKeyResourceLabel].Primary.Attributes["managed_resource.0.environment.0.id"]
					return environmentId + "/" + kafkaApiKeyId, nil
				},
			},
		},
	})

	checkStubCount(t, wiremockClient, createKafkaApiKeyStub, "POST /iam/v2/api-keys", expectedCountOne)
	checkStubCount(t, wiremockClient, patchKafkaApiKeyStub, "PATCH /iam/v2/api-keys/NUYYQXLNGKLJLTWT", expectedCountOne)
	checkStubCount(t, wiremockClient, createCmkApiStub, "GET /cmk/v2/clusters/lkc-zmmq63", expectedCountOne)
}

func TestAccFlinkApiKey(t *testing.T) {
	ctx := context.Background()

	time.Sleep(5 * time.Second)
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
	createFlinkApiKeyResponse, _ := ioutil.ReadFile("../testdata/apikey/create_flink_api_key.json")
	createFlinkApiKeyStub := wiremock.Post(wiremock.URLPathEqualTo("/iam/v2/api-keys")).
		InScenario(flinkApiKeyScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(createFlinkApiKeyResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createFlinkApiKeyStub)

	readCreatedFlinkApiKeyResponse, _ := ioutil.ReadFile("../testdata/apikey/read_created_flink_api_key.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/api-keys/AK4NBR7MUYHVJMHW")).
		InScenario(flinkApiKeyScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateFlinkApiKeyHasBeenCreated).
		WillReturn(
			string(readCreatedFlinkApiKeyResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readImportedFlinkApiKeyResponse, _ := ioutil.ReadFile("../testdata/apikey/read_created_flink_api_key.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/api-keys/AK4NBR7MUYHVJMHW")).
		InScenario(flinkApiKeyScenarioName).
		WhenScenarioStateIs(scenarioStateFlinkApiKeyHasBeenCreated).
		WillReturn(
			string(readImportedFlinkApiKeyResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedFlinkApiKeyResponse, _ := ioutil.ReadFile("../testdata/apikey/read_updated_flink_api_key.json")
	patchFlinkApiKeyStub := wiremock.Patch(wiremock.URLPathEqualTo("/iam/v2/api-keys/AK4NBR7MUYHVJMHW")).
		InScenario(flinkApiKeyScenarioName).
		WhenScenarioStateIs(scenarioStateFlinkApiKeyHasBeenCreated).
		WillSetStateTo(scenarioStateFlinkApiKeyHasBeenUpdated).
		WillReturn(
			string(readUpdatedFlinkApiKeyResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(patchFlinkApiKeyStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/api-keys/AK4NBR7MUYHVJMHW")).
		InScenario(flinkApiKeyScenarioName).
		WhenScenarioStateIs(scenarioStateFlinkApiKeyHasBeenUpdated).
		WillReturn(
			string(readUpdatedFlinkApiKeyResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedFlinkApiKeyResponse, _ := ioutil.ReadFile("../testdata/apikey/read_deleted_flink_api_key.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/api-keys/AK4NBR7MUYHVJMHW")).
		InScenario(flinkApiKeyScenarioName).
		WhenScenarioStateIs(scenarioStateFlinkApiKeyHasBeenDeleted).
		WillReturn(
			string(readDeletedFlinkApiKeyResponse),
			contentTypeJSONHeader,
			http.StatusForbidden,
		))
	deleteFlinkApiKeyStub := wiremock.Delete(wiremock.URLPathEqualTo("/iam/v2/api-keys/AK4NBR7MUYHVJMHW")).
		InScenario(flinkApiKeyScenarioName).
		WhenScenarioStateIs(scenarioStateFlinkApiKeyHasBeenUpdated).
		WillSetStateTo(scenarioStateFlinkApiKeyHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteFlinkApiKeyStub)

	flinkApiKeyDisplayName := "CI Flink API Key"
	flinkApiKeyDescription := "test description"
	// in order to test tf update (step #3)
	flinkApiKeyUpdatedDisplayName := "CI Flink API Key updated"
	flinkApiKeyUpdatedDescription := "test description updated"
	flinkApiKeyResourceLabel := "test_cluster_api_key_resource_label"
	fullFlinkApiKeyResourceLabel := fmt.Sprintf("confluent_api_key.%s", flinkApiKeyResourceLabel)
	ownerId := "sa-12mgdv"
	ownerApiVersion := "iam/v2"
	ownerKind := "ServiceAccount"
	resourceId := "aws.us-east-1"
	resourceApiVersion := "fcpm/v2"
	resourceKind := "Region"
	environmentId := "env-3732nw"

	// Set fake values for secrets since those are required for importing
	os.Setenv("API_KEY_SECRET", "wtxA2jNlEtBIb0Qb+FE+a/JQMe9xvjt9ijG9j4jwTpy4SzUQ8pPocSFC43loOWBn")
	defer func() {
		os.Unsetenv("API_KEY_SECRET")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckApiKeyDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckFlinkApiKeyConfig(mockServerUrl, flinkApiKeyResourceLabel, flinkApiKeyDisplayName, flinkApiKeyDescription, ownerId, ownerApiVersion, ownerKind, resourceId, resourceApiVersion, resourceKind, environmentId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckApiKeyExists(fullFlinkApiKeyResourceLabel),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "id", "AK4NBR7MUYHVJMHW"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "display_name", flinkApiKeyDisplayName),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "description", flinkApiKeyDescription),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "owner.#", "1"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "owner.0.%", "3"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "owner.0.api_version", "iam/v2"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "owner.0.id", "sa-12mgdv"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "owner.0.kind", "ServiceAccount"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "managed_resource.#", "1"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "managed_resource.0.%", "4"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "managed_resource.0.api_version", "fcpm/v2"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "managed_resource.0.id", "aws.us-east-1"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "managed_resource.0.kind", "Region"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "managed_resource.0.environment.#", "1"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "managed_resource.0.environment.0.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "managed_resource.0.environment.0.id", "env-3732nw"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "secret", "wtxA2jNlEtBIb0Qb+FE+a/JQMe9xvjt9ijG9j4jwTpy4SzUQ8pPocSFC43loOWBn"),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullFlinkApiKeyResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					flinkApiKeyId := resources[fullFlinkApiKeyResourceLabel].Primary.ID
					environmentId := resources[fullFlinkApiKeyResourceLabel].Primary.Attributes["managed_resource.0.environment.0.id"]
					return environmentId + "/" + flinkApiKeyId, nil
				},
			},
			{
				Config: testAccCheckFlinkApiKeyConfig(mockServerUrl, flinkApiKeyResourceLabel, flinkApiKeyUpdatedDisplayName, flinkApiKeyUpdatedDescription, ownerId, ownerApiVersion, ownerKind, resourceId, resourceApiVersion, resourceKind, environmentId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckApiKeyExists(fullFlinkApiKeyResourceLabel),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "id", "AK4NBR7MUYHVJMHW"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "display_name", flinkApiKeyUpdatedDisplayName),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "description", flinkApiKeyUpdatedDescription),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "disable_wait_for_ready", "false"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "owner.#", "1"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "owner.0.%", "3"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "owner.0.api_version", "iam/v2"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "owner.0.id", "sa-12mgdv"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "owner.0.kind", "ServiceAccount"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "managed_resource.#", "1"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "managed_resource.0.%", "4"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "managed_resource.0.api_version", "fcpm/v2"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "managed_resource.0.id", "aws.us-east-1"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "managed_resource.0.kind", "Region"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "managed_resource.0.environment.#", "1"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "managed_resource.0.environment.0.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "managed_resource.0.environment.0.id", "env-3732nw"),
					resource.TestCheckResourceAttr(fullFlinkApiKeyResourceLabel, "secret", "wtxA2jNlEtBIb0Qb+FE+a/JQMe9xvjt9ijG9j4jwTpy4SzUQ8pPocSFC43loOWBn"),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullFlinkApiKeyResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					flinkApiKeyId := resources[fullFlinkApiKeyResourceLabel].Primary.ID
					environmentId := resources[fullFlinkApiKeyResourceLabel].Primary.Attributes["managed_resource.0.environment.0.id"]
					return environmentId + "/" + flinkApiKeyId, nil
				},
			},
		},
	})

	checkStubCount(t, wiremockClient, createFlinkApiKeyStub, "POST /iam/v2/api-keys", expectedCountOne)
	//checkStubCount(t, wiremockClient, patchFlinkApiKeyStub, "PATCH /iam/v2/api-keys/AK4NBR7MUYHVJMHW", expectedCountOne)
}

func TestAccTableflowApiKey(t *testing.T) {
	ctx := context.Background()

	time.Sleep(5 * time.Second)
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
	createTableflowApiKeyResponse, _ := ioutil.ReadFile("../testdata/apikey/create_tableflow_api_key.json")
	createTableflowApiKeyStub := wiremock.Post(wiremock.URLPathEqualTo("/iam/v2/api-keys")).
		InScenario(tableflowApiKeyScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(createTableflowApiKeyResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createTableflowApiKeyStub)

	tableflowApi401Response, _ := ioutil.ReadFile("../testdata/apikey/read_list_tableflow_regions_401.json")
	listRegionsTableflowApi401Stub := wiremock.Get(wiremock.URLPathEqualTo("/tableflow/v1/regions")).
		InScenario(tableflowApiKeyScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateTableflowApiKeyHasBeenSynced).
		WillReturn(
			string(tableflowApi401Response),
			contentTypeJSONHeader,
			http.StatusUnauthorized,
		)
	_ = wiremockClient.StubFor(listRegionsTableflowApi401Stub)

	tableflowApi200Response, _ := ioutil.ReadFile("../testdata/apikey/read_list_tableflow_regions_200.json")
	listRegionsTableflowApi200Stub := wiremock.Get(wiremock.URLPathEqualTo("/tableflow/v1/regions")).
		InScenario(tableflowApiKeyScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowApiKeyHasBeenSynced).
		WillSetStateTo(scenarioStateTableflowApiKeyHasBeenCreated).
		WillReturn(
			string(tableflowApi200Response),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(listRegionsTableflowApi200Stub)

	readCreatedTableflowApiKeyResponse, _ := ioutil.ReadFile("../testdata/apikey/read_created_tableflow_api_key.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/api-keys/HRVR6K4VMXYD2LDZ")).
		InScenario(tableflowApiKeyScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowApiKeyHasBeenCreated).
		WillReturn(
			string(readCreatedTableflowApiKeyResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedTableflowApiKeyResponse, _ := ioutil.ReadFile("../testdata/apikey/read_updated_tableflow_api_key.json")
	patchTableflowApiKeyStub := wiremock.Patch(wiremock.URLPathEqualTo("/iam/v2/api-keys/HRVR6K4VMXYD2LDZ")).
		InScenario(tableflowApiKeyScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowApiKeyHasBeenCreated).
		WillSetStateTo(scenarioStateTableflowApiKeyHasBeenUpdated).
		WillReturn(
			string(readUpdatedTableflowApiKeyResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(patchTableflowApiKeyStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/api-keys/HRVR6K4VMXYD2LDZ")).
		InScenario(tableflowApiKeyScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowApiKeyHasBeenUpdated).
		WillReturn(
			string(readUpdatedTableflowApiKeyResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteTableflowApiKeyStub := wiremock.Delete(wiremock.URLPathEqualTo("/iam/v2/api-keys/HRVR6K4VMXYD2LDZ")).
		InScenario(tableflowApiKeyScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowApiKeyHasBeenUpdated).
		WillSetStateTo(scenarioStateTableflowApiKeyHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteTableflowApiKeyStub)

	readDeletedTableflowApiKeyResponse, _ := ioutil.ReadFile("../testdata/apikey/read_deleted_tableflow_api_key.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/api-keys/HRVR6K4VMXYD2LDZ")).
		InScenario(tableflowApiKeyScenarioName).
		WhenScenarioStateIs(scenarioStateTableflowApiKeyHasBeenDeleted).
		WillReturn(
			string(readDeletedTableflowApiKeyResponse),
			contentTypeJSONHeader,
			http.StatusForbidden,
		))

	tableflowApiKeyDisplayName := "CI Tableflow API Key"
	tableflowApiKeyDescription := "temp description"
	// in order to test tf update (step #3)
	tableflowApiKeyUpdatedDisplayName := "CI Tableflow API Key updated"
	tableflowApiKeyUpdatedDescription := "temp description updated"
	tableflowApiKeyResourceLabel := "test_tableflow_api_key_resource_label"
	fullTableflowApiKeyResourceLabel := fmt.Sprintf("confluent_api_key.%s", tableflowApiKeyResourceLabel)
	ownerId := "sa-12mgdv"
	ownerApiVersion := "iam/v2"
	ownerKind := "ServiceAccount"
	resourceId := "tableflow"
	resourceKind := "Tableflow"
	resourceApiVersion := "tableflow/v1"

	// Set fake values for secrets since those are required for importing
	os.Setenv("API_KEY_SECRET", "p07o8EyjQvink5NmErBffigyynQXrTsYGKBzIgr3M10Mg+JOgnObYjlqCC1Q1id1")
	defer func() {
		os.Unsetenv("API_KEY_SECRET")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckApiKeyDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckTableflowApiKeyConfig(mockServerUrl, tableflowApiKeyResourceLabel, tableflowApiKeyDisplayName, tableflowApiKeyDescription, ownerId, ownerApiVersion, ownerKind, resourceApiVersion, resourceId, resourceKind),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckApiKeyExists(fullTableflowApiKeyResourceLabel),
					resource.TestCheckResourceAttr(fullTableflowApiKeyResourceLabel, "id", "HRVR6K4VMXYD2LDZ"),
					resource.TestCheckResourceAttr(fullTableflowApiKeyResourceLabel, "display_name", tableflowApiKeyDisplayName),
					resource.TestCheckResourceAttr(fullTableflowApiKeyResourceLabel, "description", tableflowApiKeyDescription),
					resource.TestCheckResourceAttr(fullTableflowApiKeyResourceLabel, "owner.#", "1"),
					resource.TestCheckResourceAttr(fullTableflowApiKeyResourceLabel, "owner.0.%", "3"),
					resource.TestCheckResourceAttr(fullTableflowApiKeyResourceLabel, "owner.0.api_version", "iam/v2"),
					resource.TestCheckResourceAttr(fullTableflowApiKeyResourceLabel, "owner.0.id", "sa-12mgdv"),
					resource.TestCheckResourceAttr(fullTableflowApiKeyResourceLabel, "owner.0.kind", "ServiceAccount"),
					resource.TestCheckResourceAttr(fullTableflowApiKeyResourceLabel, "managed_resource.0.api_version", "tableflow/v1"),
					resource.TestCheckResourceAttr(fullTableflowApiKeyResourceLabel, "managed_resource.0.kind", "Tableflow"),
					resource.TestCheckResourceAttr(fullTableflowApiKeyResourceLabel, "managed_resource.0.id", "tableflow"),
					resource.TestCheckResourceAttr(fullTableflowApiKeyResourceLabel, "secret", "p07o8EyjQvink5NmErBffigyynQXrTsYGKBzIgr3M10Mg+JOgnObYjlqCC1Q1id1"),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullTableflowApiKeyResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCheckTableflowApiKeyConfig(mockServerUrl, tableflowApiKeyResourceLabel, tableflowApiKeyUpdatedDisplayName, tableflowApiKeyUpdatedDescription, ownerId, ownerApiVersion, ownerKind, resourceApiVersion, resourceId, resourceKind),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckApiKeyExists(fullTableflowApiKeyResourceLabel),
					resource.TestCheckResourceAttr(fullTableflowApiKeyResourceLabel, "id", "HRVR6K4VMXYD2LDZ"),
					resource.TestCheckResourceAttr(fullTableflowApiKeyResourceLabel, "display_name", tableflowApiKeyUpdatedDisplayName),
					resource.TestCheckResourceAttr(fullTableflowApiKeyResourceLabel, "description", tableflowApiKeyUpdatedDescription),
					resource.TestCheckResourceAttr(fullTableflowApiKeyResourceLabel, "disable_wait_for_ready", "false"),
					resource.TestCheckResourceAttr(fullTableflowApiKeyResourceLabel, "owner.#", "1"),
					resource.TestCheckResourceAttr(fullTableflowApiKeyResourceLabel, "owner.0.%", "3"),
					resource.TestCheckResourceAttr(fullTableflowApiKeyResourceLabel, "owner.0.api_version", "iam/v2"),
					resource.TestCheckResourceAttr(fullTableflowApiKeyResourceLabel, "owner.0.id", "sa-12mgdv"),
					resource.TestCheckResourceAttr(fullTableflowApiKeyResourceLabel, "owner.0.kind", "ServiceAccount"),
					resource.TestCheckResourceAttr(fullTableflowApiKeyResourceLabel, "managed_resource.0.api_version", "tableflow/v1"),
					resource.TestCheckResourceAttr(fullTableflowApiKeyResourceLabel, "managed_resource.0.kind", "Tableflow"),
					resource.TestCheckResourceAttr(fullTableflowApiKeyResourceLabel, "managed_resource.0.id", "tableflow"),
					resource.TestCheckResourceAttr(fullTableflowApiKeyResourceLabel, "secret", "p07o8EyjQvink5NmErBffigyynQXrTsYGKBzIgr3M10Mg+JOgnObYjlqCC1Q1id1"),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullTableflowApiKeyResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, createTableflowApiKeyStub, "POST /iam/v2/api-keys", expectedCountOne)
	checkStubCount(t, wiremockClient, patchTableflowApiKeyStub, "PATCH /iam/v2/api-keys/HRVR6K4VMXYD2LDZ", expectedCountOne)
}

func TestAccCloudApiKey(t *testing.T) {
	ctx := context.Background()

	time.Sleep(5 * time.Second)
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
	createCloudApiKeyResponse, _ := ioutil.ReadFile("../testdata/apikey/create_cloud_api_key.json")
	createCloudApiKeyStub := wiremock.Post(wiremock.URLPathEqualTo("/iam/v2/api-keys")).
		InScenario(cloudApiKeyScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(createCloudApiKeyResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createCloudApiKeyStub)

	listEnvs401Response, _ := ioutil.ReadFile("../testdata/apikey/read_list_envs_401.json")
	listEnvsOrgApi401Stub := wiremock.Get(wiremock.URLPathEqualTo("/org/v2/environments")).
		InScenario(cloudApiKeyScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateCloudApiKeyHasBeenSynced).
		WillReturn(
			string(listEnvs401Response),
			contentTypeJSONHeader,
			http.StatusUnauthorized,
		)
	_ = wiremockClient.StubFor(listEnvsOrgApi401Stub)

	listEnvs200Response, _ := ioutil.ReadFile("../testdata/apikey/read_list_envs_200.json")
	listEnvsOrgApi200Stub := wiremock.Get(wiremock.URLPathEqualTo("/org/v2/environments")).
		InScenario(cloudApiKeyScenarioName).
		WhenScenarioStateIs(scenarioStateCloudApiKeyHasBeenSynced).
		WillSetStateTo(scenarioStateCloudApiKeyHasBeenCreated).
		WillReturn(
			string(listEnvs200Response),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(listEnvsOrgApi200Stub)

	readCreatedCloudApiKeyResponse, _ := ioutil.ReadFile("../testdata/apikey/read_created_cloud_api_key.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/api-keys/HRVR6K4VMXYD2LDZ")).
		InScenario(cloudApiKeyScenarioName).
		WhenScenarioStateIs(scenarioStateCloudApiKeyHasBeenCreated).
		WillReturn(
			string(readCreatedCloudApiKeyResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedCloudApiKeyResponse, _ := ioutil.ReadFile("../testdata/apikey/read_updated_cloud_api_key.json")
	patchCloudApiKeyStub := wiremock.Patch(wiremock.URLPathEqualTo("/iam/v2/api-keys/HRVR6K4VMXYD2LDZ")).
		InScenario(cloudApiKeyScenarioName).
		WhenScenarioStateIs(scenarioStateCloudApiKeyHasBeenCreated).
		WillSetStateTo(scenarioStateCloudApiKeyHasBeenUpdated).
		WillReturn(
			string(readUpdatedCloudApiKeyResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(patchCloudApiKeyStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/api-keys/HRVR6K4VMXYD2LDZ")).
		InScenario(cloudApiKeyScenarioName).
		WhenScenarioStateIs(scenarioStateCloudApiKeyHasBeenUpdated).
		WillReturn(
			string(readUpdatedCloudApiKeyResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedCloudApiKeyResponse, _ := ioutil.ReadFile("../testdata/apikey/read_deleted_cloud_api_key.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/api-keys/HRVR6K4VMXYD2LDZ")).
		InScenario(cloudApiKeyScenarioName).
		WhenScenarioStateIs(scenarioStateCloudApiKeyHasBeenDeleted).
		WillReturn(
			string(readDeletedCloudApiKeyResponse),
			contentTypeJSONHeader,
			http.StatusForbidden,
		))
	deleteCloudApiKeyStub := wiremock.Delete(wiremock.URLPathEqualTo("/iam/v2/api-keys/HRVR6K4VMXYD2LDZ")).
		InScenario(cloudApiKeyScenarioName).
		WhenScenarioStateIs(scenarioStateCloudApiKeyHasBeenUpdated).
		WillSetStateTo(scenarioStateCloudApiKeyHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteCloudApiKeyStub)

	cloudApiKeyDisplayName := "CI Cloud API Key"
	cloudApiKeyDescription := "temp description"
	// in order to test tf update (step #3)
	cloudApiKeyUpdatedDisplayName := "CI Cloud API Key updated"
	cloudApiKeyUpdatedDescription := "temp description updated"
	cloudApiKeyResourceLabel := "test_cloud_api_key_resource_label"
	fullCloudApiKeyResourceLabel := fmt.Sprintf("confluent_api_key.%s", cloudApiKeyResourceLabel)
	ownerId := "sa-12mgdv"
	ownerApiVersion := "iam/v2"
	ownerKind := "ServiceAccount"

	// Set fake values for secrets since those are required for importing
	os.Setenv("API_KEY_SECRET", "p07o8EyjQvink5NmErBffigyynQXrTsYGKBzIgr3M10Mg+JOgnObYjlqCC1Q1id1")
	defer func() {
		os.Unsetenv("API_KEY_SECRET")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckApiKeyDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckCloudApiKeyConfig(mockServerUrl, cloudApiKeyResourceLabel, cloudApiKeyDisplayName, cloudApiKeyDescription, ownerId, ownerApiVersion, ownerKind),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckApiKeyExists(fullCloudApiKeyResourceLabel),
					resource.TestCheckResourceAttr(fullCloudApiKeyResourceLabel, "id", "HRVR6K4VMXYD2LDZ"),
					resource.TestCheckResourceAttr(fullCloudApiKeyResourceLabel, "display_name", cloudApiKeyDisplayName),
					resource.TestCheckResourceAttr(fullCloudApiKeyResourceLabel, "description", cloudApiKeyDescription),
					resource.TestCheckResourceAttr(fullCloudApiKeyResourceLabel, "owner.#", "1"),
					resource.TestCheckResourceAttr(fullCloudApiKeyResourceLabel, "owner.0.%", "3"),
					resource.TestCheckResourceAttr(fullCloudApiKeyResourceLabel, "owner.0.api_version", "iam/v2"),
					resource.TestCheckResourceAttr(fullCloudApiKeyResourceLabel, "owner.0.id", "sa-12mgdv"),
					resource.TestCheckResourceAttr(fullCloudApiKeyResourceLabel, "owner.0.kind", "ServiceAccount"),
					resource.TestCheckResourceAttr(fullCloudApiKeyResourceLabel, "secret", "p07o8EyjQvink5NmErBffigyynQXrTsYGKBzIgr3M10Mg+JOgnObYjlqCC1Q1id1"),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullCloudApiKeyResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCheckCloudApiKeyConfig(mockServerUrl, cloudApiKeyResourceLabel, cloudApiKeyUpdatedDisplayName, cloudApiKeyUpdatedDescription, ownerId, ownerApiVersion, ownerKind),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckApiKeyExists(fullCloudApiKeyResourceLabel),
					resource.TestCheckResourceAttr(fullCloudApiKeyResourceLabel, "id", "HRVR6K4VMXYD2LDZ"),
					resource.TestCheckResourceAttr(fullCloudApiKeyResourceLabel, "display_name", cloudApiKeyUpdatedDisplayName),
					resource.TestCheckResourceAttr(fullCloudApiKeyResourceLabel, "description", cloudApiKeyUpdatedDescription),
					resource.TestCheckResourceAttr(fullCloudApiKeyResourceLabel, "disable_wait_for_ready", "false"),
					resource.TestCheckResourceAttr(fullCloudApiKeyResourceLabel, "owner.#", "1"),
					resource.TestCheckResourceAttr(fullCloudApiKeyResourceLabel, "owner.0.%", "3"),
					resource.TestCheckResourceAttr(fullCloudApiKeyResourceLabel, "owner.0.api_version", "iam/v2"),
					resource.TestCheckResourceAttr(fullCloudApiKeyResourceLabel, "owner.0.id", "sa-12mgdv"),
					resource.TestCheckResourceAttr(fullCloudApiKeyResourceLabel, "owner.0.kind", "ServiceAccount"),
					resource.TestCheckResourceAttr(fullCloudApiKeyResourceLabel, "secret", "p07o8EyjQvink5NmErBffigyynQXrTsYGKBzIgr3M10Mg+JOgnObYjlqCC1Q1id1"),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullCloudApiKeyResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, createCloudApiKeyStub, "POST /iam/v2/api-keys", expectedCountOne)
	checkStubCount(t, wiremockClient, patchCloudApiKeyStub, "PATCH /iam/v2/api-keys/HRVR6K4VMXYD2LDZ", expectedCountOne)
	// Combine both stubs into a single check since it doesn't differentiate between states
	checkStubCount(t, wiremockClient, listEnvsOrgApi401Stub, "GET /org/v2/environments", 2)
}

func testAccCheckApiKeyDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each kafka api key is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_api_key" {
			continue
		}
		deletedApiKeyId := rs.Primary.ID
		req := c.apiKeysClient.APIKeysIamV2Api.GetIamV2ApiKey(c.apiKeysApiContext(context.Background()), deletedApiKeyId)
		deletedApiKey, response, err := req.Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden) {
			return nil
		} else if err == nil && deletedApiKey.Id != nil {
			// Otherwise return the error
			if *deletedApiKey.Id == rs.Primary.ID {
				return fmt.Errorf("kafka / cloud api key (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckKafkaApiKeyConfig(mockServerUrl, kafkaApiKeyResourceLabel, kafkaApiKeyDisplayName, kafkaApiKeyDescription, ownerId, ownerApiVersion, ownerKind, resourceId, resourceApiVersion, resourceKind, environmentId string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_api_key" "%s" {
		display_name = "%s"
		description = "%s"
		owner {
			id = "%s"
			api_version = "%s"
			kind = "%s"
		}
		managed_resource {
			id = "%s"
            api_version = "%s"
			kind = "%s"
			environment {
				id = "%s"
			}
		}
	}
	`, mockServerUrl, kafkaApiKeyResourceLabel, kafkaApiKeyDisplayName, kafkaApiKeyDescription, ownerId, ownerApiVersion, ownerKind, resourceId, resourceApiVersion, resourceKind, environmentId)
}

func testAccCheckFlinkApiKeyConfig(mockServerUrl, flinkApiKeyResourceLabel, flinkApiKeyDisplayName,
	flinkApiKeyDescription, ownerId, ownerApiVersion, ownerKind, resourceId, resourceApiVersion,
	resourceKind, environmentId string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_api_key" "%s" {
		display_name = "%s"
		description = "%s"
		owner {
			id = "%s"
			api_version = "%s"
			kind = "%s"
		}
		managed_resource {
			id = "%s"
			api_version = "%s"
			kind = "%s"
			environment {
				id = "%s"
			}
		}
	}
	`, mockServerUrl, flinkApiKeyResourceLabel, flinkApiKeyDisplayName, flinkApiKeyDescription, ownerId, ownerApiVersion, ownerKind, resourceId, resourceApiVersion, resourceKind, environmentId)
}

func testAccCheckCloudApiKeyConfig(mockServerUrl, cloudApiKeyResourceLabel, cloudApiKeyDisplayName, cloudApiKeyDescription, ownerId, ownerApiVersion, ownerKind string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_api_key" "%s" {
		display_name = "%s"
		description = "%s"
		owner {
			id = "%s"
			api_version = "%s"
			kind = "%s"
		}
	}
	`, mockServerUrl, cloudApiKeyResourceLabel, cloudApiKeyDisplayName, cloudApiKeyDescription, ownerId, ownerApiVersion, ownerKind)
}

func testAccCheckTableflowApiKeyConfig(mockServerUrl, tableflowApiKeyResourceLabel, tableflowApiKeyDisplayName, tableflowApiKeyDescription, ownerId, ownerApiVersion, ownerKind, resourceApiVersion, resourceId, resourceKind string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_api_key" "%s" {
		display_name = "%s"
		description = "%s"
		owner {
			id = "%s"
			api_version = "%s"
			kind = "%s"
		}
		managed_resource {
			api_version = "%s"
			id = "%s"
			kind = "%s"
		}
	}
	`, mockServerUrl, tableflowApiKeyResourceLabel, tableflowApiKeyDisplayName, tableflowApiKeyDescription, ownerId, ownerApiVersion, ownerKind, resourceApiVersion, resourceId, resourceKind)
}

func testAccCheckApiKeyExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s cluster / cloud api key has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s cluster / cloud api key", n)
		}

		return nil
	}
}
