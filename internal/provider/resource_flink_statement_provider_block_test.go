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
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	scenarioStateStatementHasBeenCreated = "A new statement has been just created"
	scenarioStateStatementIsPending      = "A new statement is pending"
	scenarioStateStatementIsUpdating     = "A new statement is updating"
	scenarioStateStatementIsDeleting     = "The statement is being deleted"
	scenarioStateStatementHasBeenDeleted = "The statement has been deleted"
	scenarioStateStatementHasBeenUpdated = "The statement has been updated"
	statementScenarioName                = "confluent_flink_statement Resource Lifecycle"

	flinkPrincipalIdTest        = "u-yo9j87"
	flinkComputePoolIdTest      = "lfcp-x7rgx1"
	flinkStatementTest          = "SELECT CURRENT_TIMESTAMP;"
	flinkStatementNameTest      = "workspace-2023-11-15-030109-0408d52d-eaff-4d50-a246-f822a29f2eb9"
	flinkFirstPropertyKeyTest   = "sql.local-time-zone"
	flinkFirstPropertyValueTest = "GMT-08:00"
	flinkStatementResourceLabel = "example"
)

var fullFlinkStatementResourceLabel = fmt.Sprintf("confluent_flink_statement.%s", flinkStatementResourceLabel)
var createFlinkStatementPath = fmt.Sprintf("/sql/v1beta1/organizations/%s/environments/%s/statements", flinkOrganizationIdTest, flinkEnvironmentIdTest)
var readFlinkStatementPath = fmt.Sprintf("/sql/v1beta1/organizations/%s/environments/%s/statements/%s", flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkStatementNameTest)

func TestAccFlinkStatementWithEnhancedProviderBlock(t *testing.T) {
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
				Config: testAccCheckFlinkStatementWithEnhancedProviderBlock(confluentCloudBaseUrl, mockFlinkStatementTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkStatementExists(fullFlinkStatementResourceLabel),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "id", fmt.Sprintf("%s/%s/%s", flinkEnvironmentIdTest, flinkComputePoolIdTest, flinkStatementNameTest)),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "compute_pool.#", "0"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "compute_pool.0.id"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "principal.#", "0"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "principal.0.id"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "statement_name", flinkStatementNameTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "statement", flinkStatementTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "stopped", "false"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "properties.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, fmt.Sprintf("properties.%s", flinkFirstPropertyKeyTest), flinkFirstPropertyValueTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "credentials.#", "0"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "credentials.0.key"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "credentials.0.secret"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "rest_endpoint"),
				),
			},
			{
				Config: testAccCheckFlinkStatementUpdatedWithEnhancedProviderBlock(confluentCloudBaseUrl, mockFlinkStatementTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkStatementExists(fullFlinkStatementResourceLabel),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "id", fmt.Sprintf("%s/%s/%s", flinkEnvironmentIdTest, flinkComputePoolIdTest, flinkStatementNameTest)),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "compute_pool.#", "0"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "compute_pool.0.id"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "principal.#", "0"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "principal.0.id"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "organization.#", "0"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "organization.0.id"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "environment.#", "0"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "environment.0.id"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "statement_name", flinkStatementNameTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "statement", flinkStatementTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "stopped", "true"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "properties.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, fmt.Sprintf("properties.%s", flinkFirstPropertyKeyTest), flinkFirstPropertyValueTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "credentials.#", "0"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "credentials.0.key"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "credentials.0.secret"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "rest_endpoint"),
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

func testAccCheckFlinkStatementWithEnhancedProviderBlock(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
      endpoint = "%s"
      flink_api_key = "%s"
      flink_api_secret = "%s"
      flink_rest_endpoint = "%s"
      flink_principal_id = "%s"
      organization_id = "%s"
      environment_id = "%s"
      flink_compute_pool_id = "%s"
    }
	resource "confluent_flink_statement" "%s" {
	  statement_name = "%s"
	  statement = "%s"
	
	  properties = {
		"%s" = "%s"
	  }
	}
	`, confluentCloudBaseUrl, kafkaApiKey, kafkaApiSecret, mockServerUrl, flinkPrincipalIdTest,
		flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkComputePoolIdTest,
		flinkStatementResourceLabel, flinkStatementNameTest, flinkStatementTest, flinkFirstPropertyKeyTest, flinkFirstPropertyValueTest)
}

func testAccCheckFlinkStatementUpdatedWithEnhancedProviderBlock(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
     endpoint = "%s"
     flink_api_key = "%s"
     flink_api_secret = "%s"
     flink_rest_endpoint = "%s"
     flink_principal_id = "%s"
     organization_id = "%s"
     environment_id = "%s"
     flink_compute_pool_id = "%s"
   }
	resource "confluent_flink_statement" "%s" {
	  statement_name = "%s"
	  statement = "%s"
	  stopped = true

	  properties = {
		"%s" = "%s"
	  }
	}
	`, confluentCloudBaseUrl, kafkaApiKey, kafkaApiSecret, mockServerUrl, flinkPrincipalIdTest,
		flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkComputePoolIdTest,
		flinkStatementResourceLabel, flinkStatementNameTest, flinkStatementTest, flinkFirstPropertyKeyTest, flinkFirstPropertyValueTest)
}

func testAccCheckFlinkStatementDestroy(s *terraform.State, url string) error {
	c := testAccProvider.Meta().(*Client).flinkRestClientFactory.CreateFlinkRestClient(url, flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkComputePoolIdTest, flinkPrincipalIdTest, kafkaApiKey, kafkaApiSecret, false)
	// Loop through the resources in state, verifying each Kafka topic is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_flink_statement" {
			continue
		}
		deletedTopicId := rs.Primary.ID
		_, response, err := c.apiClient.StatementsSqlV1beta1Api.GetSqlv1beta1Statement(c.apiContext(context.Background()), flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkStatementNameTest).Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			return nil
		} else if err == nil && deletedTopicId != "" {
			// Otherwise return the error
			if deletedTopicId == rs.Primary.ID {
				return fmt.Errorf("topic (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckFlinkStatementExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s flink statement has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s flink statement", n)
		}

		return nil
	}
}
