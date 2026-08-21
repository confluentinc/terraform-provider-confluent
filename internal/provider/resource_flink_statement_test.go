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
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
)

func TestAccFlinkStatement2(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockFlinkStatementTestServerUrl := wiremockContainer.URI
	confluentCloudBaseUrl := ""
	wiremockClient := wiremock.NewClient(mockFlinkStatementTestServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()
	createFlinkStatementResponse, _ := ioutil.ReadFile("../testdata/flink_statement/create_flink_statement.json")
	createFlinkStatementStub := wiremock.Post(wiremock.URLPathEqualTo(createFlinkStatementPath)).
		InScenario(statementScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateStatementIsPending).
		WillReturn(
			string(createFlinkStatementResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createFlinkStatementStub)

	readPendingFlinkStatementResponse, _ := ioutil.ReadFile("../testdata/flink_statement/read_pending_flink_statement.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementScenarioName).
		WhenScenarioStateIs(scenarioStateStatementIsPending).
		WillSetStateTo(scenarioStateStatementHasBeenCreated).
		WillReturn(
			string(readPendingFlinkStatementResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedFlinkStatementResponse, _ := ioutil.ReadFile("../testdata/flink_statement/read_running_flink_statement.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementScenarioName).
		WhenScenarioStateIs(scenarioStateStatementHasBeenCreated).
		WillReturn(
			string(readCreatedFlinkStatementResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// Update the Flink statement stopped status false -> true to trigger a stop
	stopFlinkStatementResponse, _ := ioutil.ReadFile("../testdata/flink_statement/read_stopped_flink_statement.json")
	stopFlinkStatementStub := wiremock.Put(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementScenarioName).
		WhenScenarioStateIs(scenarioStateStatementHasBeenCreated).
		WillSetStateTo(scenarioStateStatementHasBeenStopped).
		WillReturn(
			string(stopFlinkStatementResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(stopFlinkStatementStub)

	readStoppedFlinkStatementResponse, _ := ioutil.ReadFile("../testdata/flink_statement/read_stopped_flink_statement.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementScenarioName).
		WhenScenarioStateIs(scenarioStateStatementHasBeenStopped).
		WillReturn(
			string(readStoppedFlinkStatementResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// Update the Flink statement stopped status true -> false to trigger a resume with different `principal` and `compute_pool`
	resumingFlinkStatementResponse, _ := ioutil.ReadFile("../testdata/flink_statement/read_resuming_flink_statement.json")
	resumingFlinkStatementStub := wiremock.Put(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementScenarioName).
		WhenScenarioStateIs(scenarioStateStatementHasBeenStopped).
		WillSetStateTo(scenarioStateStatementIsResuming).
		WillReturn(
			string(resumingFlinkStatementResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(resumingFlinkStatementStub)

	readResumedFlinkStatementResponse, _ := ioutil.ReadFile("../testdata/flink_statement/read_resumed_flink_statement.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementScenarioName).
		WhenScenarioStateIs(scenarioStateStatementIsResuming).
		WillSetStateTo(scenarioStateStatementHasBeenResumed).
		WillReturn(
			string(readResumedFlinkStatementResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readPostResumeFlinkStatementResponse, _ := ioutil.ReadFile("../testdata/flink_statement/read_resumed_flink_statement.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementScenarioName).
		WhenScenarioStateIs(scenarioStateStatementHasBeenResumed).
		WillReturn(
			string(readPostResumeFlinkStatementResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteFlinkStatementStub := wiremock.Delete(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementScenarioName).
		WhenScenarioStateIs(scenarioStateStatementHasBeenResumed).
		WillSetStateTo(scenarioStateStatementIsDeleting).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteFlinkStatementStub)

	readDeletingFlinkStatementStub := wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementScenarioName).
		WhenScenarioStateIs(scenarioStateStatementIsDeleting).
		WillSetStateTo(scenarioStateStatementHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(readDeletingFlinkStatementStub)

	readDeletedFlinkStatementResponse, _ := ioutil.ReadFile("../testdata/flink_statement/read_deleted_flink_statement.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementScenarioName).
		WhenScenarioStateIs(scenarioStateStatementHasBeenDeleted).
		WillReturn(
			string(readDeletedFlinkStatementResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	// Set fake values for secrets since those are required for importing
	_ = os.Setenv("IMPORT_FLINK_API_KEY", kafkaApiKey)
	_ = os.Setenv("IMPORT_FLINK_API_SECRET", kafkaApiSecret)
	_ = os.Setenv("IMPORT_FLINK_REST_ENDPOINT", mockFlinkStatementTestServerUrl)
	_ = os.Setenv("IMPORT_FLINK_PRINCIPAL_ID", flinkPrincipalIdTest)
	_ = os.Setenv("IMPORT_CONFLUENT_ORGANIZATION_ID", flinkOrganizationIdTest)
	_ = os.Setenv("IMPORT_CONFLUENT_ENVIRONMENT_ID", flinkEnvironmentIdTest)
	_ = os.Setenv("IMPORT_FLINK_COMPUTE_POOL_ID", flinkComputePoolIdTest)
	defer func() {
		_ = os.Unsetenv("IMPORT_FLINK_API_KEY")
		_ = os.Unsetenv("IMPORT_FLINK_API_SECRET")
		_ = os.Unsetenv("IMPORT_FLINK_REST_ENDPOINT")
		_ = os.Unsetenv("IMPORT_FLINK_PRINCIPAL_ID")
		_ = os.Unsetenv("IMPORT_CONFLUENT_ORGANIZATION_ID")
		_ = os.Unsetenv("IMPORT_CONFLUENT_ENVIRONMENT_ID")
		_ = os.Unsetenv("IMPORT_FLINK_COMPUTE_POOL_ID")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return testAccCheckFlinkStatementDestroy(s, mockFlinkStatementTestServerUrl)
		},
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckFlinkStatement(confluentCloudBaseUrl, mockFlinkStatementTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkStatementExists(fullFlinkStatementResourceLabel),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "id", fmt.Sprintf("%s/%s/%s", flinkEnvironmentIdTest, flinkComputePoolIdTest, flinkStatementNameTest)),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "organization.#", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "organization.0.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "organization.0.id", flinkOrganizationIdTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "environment.0.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "environment.0.id", flinkEnvironmentIdTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "compute_pool.#", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "compute_pool.0.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "compute_pool.0.id", flinkComputePoolIdTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "principal.#", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "principal.0.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "principal.0.id", flinkPrincipalIdTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "statement_name", flinkStatementNameTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "statement", flinkStatementTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "stopped", "false"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "latest_offsets.%", "0"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "latest_offsets_timestamp", latestOffsetsTimestampEmptyValueTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "properties.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, fmt.Sprintf("properties.%s", flinkFirstPropertyKeyTest), flinkFirstPropertyValueTest),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "sql.secrets.openaikey"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "credentials.0.key", kafkaApiKey),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "credentials.0.secret", kafkaApiSecret),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "rest_endpoint", mockFlinkStatementTestServerUrl),
				),
			},
			{
				Config: testAccCheckFlinkStatementWithoutComputePool(confluentCloudBaseUrl, mockFlinkStatementTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkStatementExists(fullFlinkStatementResourceLabel),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "id", fmt.Sprintf("%s/%s/%s", flinkEnvironmentIdTest, flinkComputePoolIdTest, flinkStatementNameTest)),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "organization.#", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "organization.0.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "organization.0.id", flinkOrganizationIdTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "environment.0.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "environment.0.id", flinkEnvironmentIdTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "compute_pool.#", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "compute_pool.0.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "compute_pool.0.id", flinkComputePoolIdTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "principal.#", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "principal.0.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "principal.0.id", flinkPrincipalIdTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "statement_name", flinkStatementNameTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "statement", flinkStatementTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "stopped", "false"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "latest_offsets.%", "0"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "latest_offsets_timestamp", latestOffsetsTimestampEmptyValueTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "properties.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, fmt.Sprintf("properties.%s", flinkFirstPropertyKeyTest), flinkFirstPropertyValueTest),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "sql.secrets.openaikey"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "credentials.0.key", kafkaApiKey),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "credentials.0.secret", kafkaApiSecret),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "rest_endpoint", mockFlinkStatementTestServerUrl),
				),
			},
			{
				Config: testAccCheckFlinkStatementStopped(confluentCloudBaseUrl, mockFlinkStatementTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkStatementExists(fullFlinkStatementResourceLabel),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "id", fmt.Sprintf("%s/%s/%s", flinkEnvironmentIdTest, flinkComputePoolIdTest, flinkStatementNameTest)),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "organization.#", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "organization.0.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "organization.0.id", flinkOrganizationIdTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "environment.0.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "environment.0.id", flinkEnvironmentIdTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "compute_pool.#", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "compute_pool.0.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "compute_pool.0.id", flinkComputePoolIdTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "principal.#", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "principal.0.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "principal.0.id", flinkPrincipalIdTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "statement_name", flinkStatementNameTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "statement", flinkStatementTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "stopped", "true"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, fmt.Sprintf("latest_offsets.%s", latestOffsetFirstKeyTest), latestOffsetFirstValueTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "latest_offsets_timestamp", latestOffsetsTimestampStoppedValueTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "properties.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, fmt.Sprintf("properties.%s", flinkFirstPropertyKeyTest), flinkFirstPropertyValueTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "credentials.0.key", kafkaApiKey),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "credentials.0.secret", kafkaApiSecret),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "rest_endpoint", mockFlinkStatementTestServerUrl),
				),
			},
			{
				Config: testAccCheckFlinkStatementResumed(confluentCloudBaseUrl, mockFlinkStatementTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkStatementExists(fullFlinkStatementResourceLabel),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "id", fmt.Sprintf("%s/%s/%s", flinkEnvironmentIdTest, flinkComputePoolUpdatedIdTest, flinkStatementNameTest)),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "organization.#", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "organization.0.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "organization.0.id", flinkOrganizationIdTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "environment.0.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "environment.0.id", flinkEnvironmentIdTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "compute_pool.#", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "compute_pool.0.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "compute_pool.0.id", flinkComputePoolUpdatedIdTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "principal.#", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "principal.0.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "principal.0.id", flinkPrincipalUpdatedIdTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "statement_name", flinkStatementNameTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "statement", flinkStatementTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "stopped", "false"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "latest_offsets.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, fmt.Sprintf("latest_offsets.%s", latestOffsetFirstKeyTest), latestOffsetFirstValueTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "latest_offsets_timestamp", latestOffsetsTimestampStoppedValueTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "properties.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, fmt.Sprintf("properties.%s", flinkFirstPropertyKeyTest), flinkFirstPropertyValueTest),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "sql.secrets.openaikey"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "credentials.0.key", kafkaApiKey),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "credentials.0.secret", kafkaApiSecret),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "rest_endpoint", mockFlinkStatementTestServerUrl),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullFlinkStatementResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					resourceId := resources[fullFlinkStatementResourceLabel].Primary.ID
					statementName, _ := parseStatementName(resourceId)
					return statementName, nil
				},
			},
		},
	})

	checkStubCount(t, wiremockClient, createFlinkStatementStub, fmt.Sprintf("POST %s", createFlinkStatementPath), expectedCountOne)
	checkStubCount(t, wiremockClient, stopFlinkStatementStub, fmt.Sprintf("PUT %s", readFlinkStatementPath), expectedCountTwo)
	checkStubCount(t, wiremockClient, deleteFlinkStatementStub, fmt.Sprintf("DELETE %s", readFlinkStatementPath), expectedCountOne)
}

func testAccCheckFlinkStatement(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
      endpoint = "%s"
    }
	resource "confluent_flink_statement" "%s" {
      credentials {
        key = "%s"
        secret = "%s"
      }
 
      rest_endpoint = "%s"
      principal {
         id = "%s"
      }
      organization {
         id = "%s"
      }
      environment {
         id = "%s"
      }
      compute_pool {
         id = "%s"
      }

	  statement_name = "%s"
	  statement = "%s"
	
	  properties = {
		"%s" = "%s"
	  }
	}
	`, confluentCloudBaseUrl, flinkStatementResourceLabel, kafkaApiKey, kafkaApiSecret, mockServerUrl, flinkPrincipalIdTest,
		flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkComputePoolIdTest,
		flinkStatementNameTest, flinkStatementTest, flinkFirstPropertyKeyTest, flinkFirstPropertyValueTest)
}

func testAccCheckFlinkStatementWithoutComputePool(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
      endpoint = "%s"
    }
	resource "confluent_flink_statement" "%s" {
      credentials {
        key = "%s"
        secret = "%s"
      }
 
      rest_endpoint = "%s"
      principal {
         id = "%s"
      }
      organization {
         id = "%s"
      }
      environment {
         id = "%s"
      }

	  statement_name = "%s"
	  statement = "%s"
	
	  properties = {
		"%s" = "%s"
	  }
	}
	`, confluentCloudBaseUrl, flinkStatementResourceLabel, kafkaApiKey, kafkaApiSecret, mockServerUrl, flinkPrincipalIdTest,
		flinkOrganizationIdTest, flinkEnvironmentIdTest,
		flinkStatementNameTest, flinkStatementTest, flinkFirstPropertyKeyTest, flinkFirstPropertyValueTest)
}

func testAccCheckFlinkStatementStopped(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
      endpoint = "%s"
    }
	resource "confluent_flink_statement" "%s" {
      credentials {
        key = "%s"
        secret = "%s"
      }
 
      rest_endpoint = "%s"
      principal {
         id = "%s"
      }
      organization {
         id = "%s"
      }
      environment {
         id = "%s"
      }
      compute_pool {
         id = "%s"
      }

	  statement_name = "%s"
	  statement = "%s"
	  stopped = true

	  properties = {
		"%s" = "%s"
	  }
	}
	`, confluentCloudBaseUrl, flinkStatementResourceLabel, kafkaApiKey, kafkaApiSecret, mockServerUrl, flinkPrincipalIdTest,
		flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkComputePoolIdTest,
		flinkStatementNameTest, flinkStatementTest, flinkFirstPropertyKeyTest, flinkFirstPropertyValueTest)
}

func TestAccFlinkStatementCredentialUpdate(t *testing.T) {
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

	scenarioName := "confluent_flink_statement Credential Rotation"
	statePending := "Statement pending"
	stateCreated := "Statement created"
	stateStopped := "Statement stopped"
	stateResumed := "Statement resumed"
	stateDeleting := "Statement deleting"
	stateDeleted := "Statement deleted"

	createResponse, _ := ioutil.ReadFile("../testdata/flink_statement/create_flink_statement.json")
	pendingResponse, _ := ioutil.ReadFile("../testdata/flink_statement/read_pending_flink_statement.json")
	runningResponse, _ := ioutil.ReadFile("../testdata/flink_statement/read_running_flink_statement.json")
	stoppedResponse, _ := ioutil.ReadFile("../testdata/flink_statement/read_stopped_flink_statement.json")
	deletedResponse, _ := ioutil.ReadFile("../testdata/flink_statement/read_deleted_flink_statement.json")

	// Step 1: Create
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(createFlinkStatementPath)).
		InScenario(scenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(statePending).
		WillReturn(string(createResponse), contentTypeJSONHeader, http.StatusCreated))
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(scenarioName).
		WhenScenarioStateIs(statePending).
		WillSetStateTo(stateCreated).
		WillReturn(string(pendingResponse), contentTypeJSONHeader, http.StatusOK))

	// Steps 1+2: Read in created/running state (credential-only update only reads, no PUT)
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(scenarioName).
		WhenScenarioStateIs(stateCreated).
		WillReturn(string(runningResponse), contentTypeJSONHeader, http.StatusOK))

	// Step 3: Credentials + stopped change (triggers stop PUT)
	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(scenarioName).
		WhenScenarioStateIs(stateCreated).
		WillSetStateTo(stateStopped).
		WillReturn(string(stoppedResponse), contentTypeJSONHeader, http.StatusOK))
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(scenarioName).
		WhenScenarioStateIs(stateStopped).
		WillReturn(string(stoppedResponse), contentTypeJSONHeader, http.StatusOK))

	// Step 4: Resume with rotated credentials (triggers resume PUT)
	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(scenarioName).
		WhenScenarioStateIs(stateStopped).
		WillSetStateTo(stateResumed).
		WillReturn(string(runningResponse), contentTypeJSONHeader, http.StatusOK))
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(scenarioName).
		WhenScenarioStateIs(stateResumed).
		WillReturn(string(runningResponse), contentTypeJSONHeader, http.StatusOK))

	// Cleanup
	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(scenarioName).
		WillSetStateTo(stateDeleting).
		WillReturn("", contentTypeJSONHeader, http.StatusNoContent))
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(scenarioName).
		WhenScenarioStateIs(stateDeleting).
		WillSetStateTo(stateDeleted).
		WillReturn("", contentTypeJSONHeader, http.StatusNoContent))
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(scenarioName).
		WhenScenarioStateIs(stateDeleted).
		WillReturn(string(deletedResponse), contentTypeJSONHeader, http.StatusNotFound))

	rotatedApiKey := "rotated_test_key"
	rotatedApiSecret := "rotated_test_secret"

	resourceLabel := "cred_rotation_test"
	fullResourceLabel := fmt.Sprintf("confluent_flink_statement.%s", resourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return testAccCheckFlinkStatementDestroy(s, mockServerUrl)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccCheckFlinkStatementWithCredentials("", mockServerUrl, resourceLabel, kafkaApiKey, kafkaApiSecret, false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkStatementExists(fullResourceLabel),
					resource.TestCheckResourceAttr(fullResourceLabel, "credentials.0.key", kafkaApiKey),
					resource.TestCheckResourceAttr(fullResourceLabel, "credentials.0.secret", kafkaApiSecret),
				),
			},
			{
				Config: testAccCheckFlinkStatementWithCredentials("", mockServerUrl, resourceLabel, rotatedApiKey, rotatedApiSecret, false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkStatementExists(fullResourceLabel),
					resource.TestCheckResourceAttr(fullResourceLabel, "credentials.0.key", rotatedApiKey),
					resource.TestCheckResourceAttr(fullResourceLabel, "credentials.0.secret", rotatedApiSecret),
					resource.TestCheckResourceAttr(fullResourceLabel, "statement", flinkStatementTest),
				),
			},
			{
				Config: testAccCheckFlinkStatementWithCredentials("", mockServerUrl, resourceLabel, rotatedApiKey, rotatedApiSecret, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkStatementExists(fullResourceLabel),
					resource.TestCheckResourceAttr(fullResourceLabel, "credentials.0.key", rotatedApiKey),
					resource.TestCheckResourceAttr(fullResourceLabel, "credentials.0.secret", rotatedApiSecret),
					resource.TestCheckResourceAttr(fullResourceLabel, "stopped", "true"),
				),
			},
			{
				Config: testAccCheckFlinkStatementWithCredentials("", mockServerUrl, resourceLabel, kafkaApiKey, kafkaApiSecret, false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkStatementExists(fullResourceLabel),
					resource.TestCheckResourceAttr(fullResourceLabel, "credentials.0.key", kafkaApiKey),
					resource.TestCheckResourceAttr(fullResourceLabel, "credentials.0.secret", kafkaApiSecret),
					resource.TestCheckResourceAttr(fullResourceLabel, "stopped", "false"),
				),
			},
		},
	})
}

func testAccCheckFlinkStatementWithCredentials(confluentCloudBaseUrl, mockServerUrl, resourceLabel, apiKey, apiSecret string, stopped bool) string {
	stoppedStr := "false"
	if stopped {
		stoppedStr = "true"
	}
	return fmt.Sprintf(`
	provider "confluent" {
      endpoint = "%s"
    }
	resource "confluent_flink_statement" "%s" {
      credentials {
        key = "%s"
        secret = "%s"
      }

      rest_endpoint = "%s"
      principal {
         id = "%s"
      }
      organization {
         id = "%s"
      }
      environment {
         id = "%s"
      }
      compute_pool {
         id = "%s"
      }

	  statement_name = "%s"
	  statement = "%s"
	  stopped = %s

	  properties = {
		"%s" = "%s"
	  }
	}
	`, confluentCloudBaseUrl, resourceLabel, apiKey, apiSecret, mockServerUrl, flinkPrincipalIdTest,
		flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkComputePoolIdTest,
		flinkStatementNameTest, flinkStatementTest, stoppedStr, flinkFirstPropertyKeyTest, flinkFirstPropertyValueTest)
}

func testAccCheckFlinkStatementResumed(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
      endpoint = "%s"
    }
	resource "confluent_flink_statement" "%s" {
      credentials {
        key = "%s"
        secret = "%s"
      }
 
      rest_endpoint = "%s"
      principal {
         id = "%s"
      }
      organization {
         id = "%s"
      }
      environment {
         id = "%s"
      }
      compute_pool {
         id = "%s"
      }

	  statement_name = "%s"
	  statement = "%s"
	  stopped = false

	  properties = {
		"%s" = "%s"
	  }
	}
	`, confluentCloudBaseUrl, flinkStatementResourceLabel, kafkaApiKey, kafkaApiSecret, mockServerUrl, flinkPrincipalUpdatedIdTest,
		flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkComputePoolUpdatedIdTest,
		flinkStatementNameTest, flinkStatementTest, flinkFirstPropertyKeyTest, flinkFirstPropertyValueTest)
}
