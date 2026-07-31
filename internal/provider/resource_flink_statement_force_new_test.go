// Copyright 2026 Confluent Inc. All Rights Reserved.
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
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
)

const (
	scenarioStateStatementV2IsPending       = "A new statement (v2) is pending"
	scenarioStateStatementV2HasBeenCreated  = "A new statement (v2) has been just created"
	scenarioStateStatementV2IsDeleting      = "The statement (v2) is being deleted"
	scenarioStateStatementV2HasBeenDeleted  = "The statement (v2) has been deleted"
	statementPropertiesForceNewScenarioName = "confluent_flink_statement Properties ForceNew Lifecycle"
)

// Verifies that changing `properties` drives a real destroy-then-recreate, not an in-place
// update, and that the resulting state reflects the new properties value.
func TestAccFlinkStatementPropertiesForceNew(t *testing.T) {
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

	// v1: initial create with the first properties value
	createFlinkStatementResponse, _ := ioutil.ReadFile("../testdata/flink_statement/create_flink_statement.json")
	createFlinkStatementStub := wiremock.Post(wiremock.URLPathEqualTo(createFlinkStatementPath)).
		InScenario(statementPropertiesForceNewScenarioName).
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
		InScenario(statementPropertiesForceNewScenarioName).
		WhenScenarioStateIs(scenarioStateStatementIsPending).
		WillSetStateTo(scenarioStateStatementHasBeenCreated).
		WillReturn(
			string(readPendingFlinkStatementResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedFlinkStatementResponse, _ := ioutil.ReadFile("../testdata/flink_statement/read_running_flink_statement.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementPropertiesForceNewScenarioName).
		WhenScenarioStateIs(scenarioStateStatementHasBeenCreated).
		WillReturn(
			string(readCreatedFlinkStatementResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// Replace: destroy the v1 statement...
	deleteFlinkStatementStub := wiremock.Delete(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementPropertiesForceNewScenarioName).
		WhenScenarioStateIs(scenarioStateStatementHasBeenCreated).
		WillSetStateTo(scenarioStateStatementIsDeleting).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteFlinkStatementStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementPropertiesForceNewScenarioName).
		WhenScenarioStateIs(scenarioStateStatementIsDeleting).
		WillSetStateTo(scenarioStateStatementHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	readDeletedFlinkStatementResponse, _ := ioutil.ReadFile("../testdata/flink_statement/read_deleted_flink_statement.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementPropertiesForceNewScenarioName).
		WhenScenarioStateIs(scenarioStateStatementHasBeenDeleted).
		WillReturn(
			string(readDeletedFlinkStatementResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	// ...then create the v2 statement with the new properties value.
	createFlinkStatementV2Response, _ := ioutil.ReadFile("../testdata/flink_statement/create_flink_statement_v2.json")
	createFlinkStatementV2Stub := wiremock.Post(wiremock.URLPathEqualTo(createFlinkStatementPath)).
		InScenario(statementPropertiesForceNewScenarioName).
		WhenScenarioStateIs(scenarioStateStatementHasBeenDeleted).
		WillSetStateTo(scenarioStateStatementV2IsPending).
		WillReturn(
			string(createFlinkStatementV2Response),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createFlinkStatementV2Stub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementPropertiesForceNewScenarioName).
		WhenScenarioStateIs(scenarioStateStatementV2IsPending).
		WillSetStateTo(scenarioStateStatementV2HasBeenCreated).
		WillReturn(
			string(createFlinkStatementV2Response),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedFlinkStatementV2Response, _ := ioutil.ReadFile("../testdata/flink_statement/read_running_flink_statement_v2.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementPropertiesForceNewScenarioName).
		WhenScenarioStateIs(scenarioStateStatementV2HasBeenCreated).
		WillReturn(
			string(readCreatedFlinkStatementV2Response),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// Final teardown (CheckDestroy, below) deletes the v2 statement.
	deleteFlinkStatementV2Stub := wiremock.Delete(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementPropertiesForceNewScenarioName).
		WhenScenarioStateIs(scenarioStateStatementV2HasBeenCreated).
		WillSetStateTo(scenarioStateStatementV2IsDeleting).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteFlinkStatementV2Stub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementPropertiesForceNewScenarioName).
		WhenScenarioStateIs(scenarioStateStatementV2IsDeleting).
		WillSetStateTo(scenarioStateStatementV2HasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementPropertiesForceNewScenarioName).
		WhenScenarioStateIs(scenarioStateStatementV2HasBeenDeleted).
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
		Steps: []resource.TestStep{
			{
				Config: testAccCheckFlinkStatementWithEnhancedProviderBlock(confluentCloudBaseUrl, mockFlinkStatementTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkStatementExists(fullFlinkStatementResourceLabel),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "properties.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, fmt.Sprintf("properties.%s", flinkFirstPropertyKeyTest), flinkFirstPropertyValueTest),
				),
			},
			{
				// Changing `properties` alone must be proposed (and executed) as a replace, not an
				// in-place update, now that the attribute is ForceNew.
				Config: testAccCheckFlinkStatementWithSecondProperty(confluentCloudBaseUrl, mockFlinkStatementTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkStatementExists(fullFlinkStatementResourceLabel),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "properties.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, fmt.Sprintf("properties.%s", flinkSecondPropertyKeyTest), flinkSecondPropertyValueTest),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, fmt.Sprintf("properties.%s", flinkFirstPropertyKeyTest)),
				),
			},
		},
	})

	// checkStubCount matches by request pattern (method+URL), not by which specific stub/scenario-
	// state handled it, so the v1 and v2 create (and delete) stubs share one count each: one create
	// for the original statement, one more after the forced replace; one delete for the replace's
	// destroy, one more for this test's own final teardown.
	checkStubCount(t, wiremockClient, createFlinkStatementStub, fmt.Sprintf("POST %s", createFlinkStatementPath), expectedCountTwo)
	checkStubCount(t, wiremockClient, deleteFlinkStatementStub, fmt.Sprintf("DELETE %s", readFlinkStatementPath), expectedCountTwo)
}

func testAccCheckFlinkStatementWithSecondProperty(confluentCloudBaseUrl, mockServerUrl string) string {
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
		flinkStatementResourceLabel, flinkStatementNameTest, flinkStatementTest, flinkSecondPropertyKeyTest, flinkSecondPropertyValueTest)
}
