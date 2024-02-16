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
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccFlinkStatement(t *testing.T) {
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

	updateFlinkStatementResponse := wiremock.Put(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementScenarioName).
		WhenScenarioStateIs(scenarioStateStatementHasBeenCreated).
		WillSetStateTo(scenarioStateStatementIsUpdating).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(updateFlinkStatementResponse)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementScenarioName).
		WhenScenarioStateIs(scenarioStateStatementIsUpdating).
		WillSetStateTo(scenarioStateStatementHasBeenUpdated).
		WillReturn(
			string(readCreatedFlinkStatementResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedFlinkStatementResponse, _ := ioutil.ReadFile("../testdata/flink_statement/read_stopped_flink_statement.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementScenarioName).
		WhenScenarioStateIs(scenarioStateStatementHasBeenUpdated).
		WillReturn(
			string(readUpdatedFlinkStatementResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteFlinkStatementStub := wiremock.Delete(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementScenarioName).
		WhenScenarioStateIs(scenarioStateStatementHasBeenUpdated).
		WillSetStateTo(scenarioStateStatementIsDeleting).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteFlinkStatementStub)

	readDeletingFlinkStatementStub := wiremock.Delete(wiremock.URLPathEqualTo(readFlinkStatementPath)).
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
	_ = os.Setenv("IMPORT_ORGANIZATION_ID", flinkOrganizationIdTest)
	_ = os.Setenv("IMPORT_ENVIRONMENT_ID", flinkEnvironmentIdTest)
	_ = os.Setenv("IMPORT_FLINK_COMPUTE_POOL_ID", flinkComputePoolIdTest)
	defer func() {
		_ = os.Unsetenv("IMPORT_FLINK_API_KEY")
		_ = os.Unsetenv("IMPORT_FLINK_API_SECRET")
		_ = os.Unsetenv("IMPORT_FLINK_REST_ENDPOINT")
		_ = os.Unsetenv("IMPORT_FLINK_PRINCIPAL_ID")
		_ = os.Unsetenv("IMPORT_ORGANIZATION_ID")
		_ = os.Unsetenv("IMPORT_ENVIRONMENT_ID")
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
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "resource_version", flinkResourceVersionTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "stopped", "false"),
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
				Config: testAccCheckFlinkStatementUpdated(confluentCloudBaseUrl, mockFlinkStatementTestServerUrl),
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
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "resource_version", flinkResourceVersionTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "stopped", "true"),
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

func testAccCheckFlinkStatementUpdated(confluentCloudBaseUrl, mockServerUrl string) string {
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
