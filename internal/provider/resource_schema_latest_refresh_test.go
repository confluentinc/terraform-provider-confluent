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
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
)

const (
	scenarioStateSchemaTest1Part2 = "In Test 1 part 2"
	scenarioStateSchemaTest2      = "In test 2 part 1"
)

//
// This is a new test to check specific functionality whereby a schema is updated between Terraform runs.
// It's actually more to do with ensuring the state is refreshed - see issue #318
// The provider would compare the new schema with the old value in the new state.
// In the second step, this test returns updated version of the schema and ensures that Terraform creates a plan against it's initial schema.
// In the third step, this test returns updated version of the schema and ensures that Terraform does not create a plan against an updated schema.
//

func TestAccLatestRefreshSchema(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	mockSchemaTestServerUrl := wiremockContainer.URI
	confluentCloudBaseUrl := ""
	wiremockClient := wiremock.NewClient(mockSchemaTestServerUrl)

	// Schema has not yet been created. Used for test step 1.

	validateSchemaResponse, _ := os.ReadFile("../testdata/schema_registry_schema/validate_schema.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(validateSchemaPath)).
		InScenario(schemaScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(validateSchemaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	createSchemaResponse, _ := os.ReadFile("../testdata/schema_registry_schema/create_schema.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(createSchemaPath)).
		InScenario(schemaScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(createSchemaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readLatestSchemaResponse, _ := os.ReadFile("../testdata/schema_registry_schema/read_latest_schema.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readLatestSchemaPath)).
		InScenario(schemaScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readLatestSchemaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readSchemasResponse, _ := os.ReadFile("../testdata/schema_registry_schema/read_schemas.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readSchemasPath)).
		InScenario(schemaScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateSchemaTest1Part2).
		WillReturn(
			string(readSchemasResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// Created schema but still in test 1, we now have a plan to respond to.
	// We need to have an intermediate state so we know when we move on to the next tests.

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readLatestSchemaPath)).
		InScenario(schemaScenarioName).
		WhenScenarioStateIs(scenarioStateSchemaTest1Part2).
		WillReturn(
			string(readLatestSchemaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readSchemasPath)).
		InScenario(schemaScenarioName).
		WhenScenarioStateIs(scenarioStateSchemaTest1Part2).
		WillSetStateTo(scenarioStateSchemaTest2).
		WillReturn(
			string(readSchemasResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// On to test step 2 and 3. On refresh, we will return that something has changed compared to step 1.
	// Planning only o we do not need to create a schema but we do need to validate in step 2.

	readLatestSchemaResponseUpdated, _ := os.ReadFile("../testdata/schema_registry_schema/read_latest_schema_updated.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readLatestSchemaPath)).
		InScenario(schemaScenarioName).
		WhenScenarioStateIs(scenarioStateSchemaTest2).
		WillReturn(
			string(readLatestSchemaResponseUpdated),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readSchemasResponseUpdated, _ := os.ReadFile("../testdata/schema_registry_schema/read_schemas_updated.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readSchemasPath)).
		InScenario(schemaScenarioName).
		WhenScenarioStateIs(scenarioStateSchemaTest2).
		WillReturn(
			string(readSchemasResponseUpdated),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(validateSchemaPath)).
		InScenario(schemaScenarioName).
		WhenScenarioStateIs(scenarioStateSchemaTest2).
		WillReturn(
			string(validateSchemaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// Test 2 and 3 done. The delete path is unique so we don't need to worry about the state.

	deleteSchemaStub := wiremock.Delete(wiremock.URLPathEqualTo(deleteSchemaPathUpdated)).
		InScenario(schemaScenarioName).
		WillSetStateTo(scenarioStateSchemaHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteSchemaStub)

	readDeletedSaResponse, _ := os.ReadFile("../testdata/schema_registry_schema/read_schemas_after_delete.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readSchemasPath)).
		InScenario(schemaScenarioName).
		WhenScenarioStateIs(scenarioStateSchemaHasBeenDeleted).
		WillReturn(
			string(readDeletedSaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return testAccCheckSchemaDestroy(s, mockSchemaTestServerUrl)
		},
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckLatestRefreshSchemaConfig(confluentCloudBaseUrl, mockSchemaTestServerUrl, testSchemaContent),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaExists(fullSchemaResourceLabel),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "id", fmt.Sprintf("%s/%s/%s", testStreamGovernanceClusterId, testSubjectName, latestSchemaVersionAndPlaceholderForSchemaIdentifier)),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_registry_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_registry_cluster.0.%", "1"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_registry_cluster.0.id", testStreamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "rest_endpoint", mockSchemaTestServerUrl),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "credentials.0.key", testSchemaRegistryKey),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "credentials.0.secret", testSchemaRegistrySecret),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "subject_name", testSubjectName),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "format", testFormat),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema", testSchemaContent),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "version", strconv.Itoa(testSchemaVersion)),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_identifier", strconv.Itoa(testSchemaIdentifier)),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "hard_delete", testHardDelete),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "recreate_on_update", testRecreateOnUpdateFalse),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "skip_validation_during_plan", testSkipSchemaValidationDuringPlanTrue),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_reference.#", "2"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_reference.0.%", "3"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_reference.0.name", testFirstSchemaReferenceDisplayName),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_reference.0.subject_name", testFirstSchemaReferenceSubject),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_reference.0.version", strconv.Itoa(testFirstSchemaReferenceVersion)),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_reference.1.%", "3"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_reference.1.name", testSecondSchemaReferenceDisplayName),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_reference.1.subject_name", testSecondSchemaReferenceSubject),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_reference.1.version", strconv.Itoa(testSecondSchemaReferenceVersion)),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "%", strconv.Itoa(testNumberOfSchemaRegistrySchemaResourceAttributes)),
				),
			},
			{
				// For step 2 wiremock will send back an updated schema, which should cause Terraform to create a plan.
				Config:             testAccCheckLatestRefreshSchemaConfig(confluentCloudBaseUrl, mockSchemaTestServerUrl, testSchemaContent),
				PlanOnly:           true, // The plan is only checked if this is set!
				ExpectNonEmptyPlan: true,
			}, {
				// For step 3 wiremock will send back an updated schema, which should not cause Terraform to create a plan against the updated schema.
				Config:             testAccCheckLatestRefreshSchemaConfig(confluentCloudBaseUrl, mockSchemaTestServerUrl, testSchemaContentUpdated),
				PlanOnly:           true, // The plan is only checked if this is set and any other Check is ignored!
				ExpectNonEmptyPlan: false,
			},
		},
	})

	checkStubCount(t, wiremockClient, deleteSchemaStub, fmt.Sprintf("DELETE %s", readSchemasPath), expectedCountOne)

	t.Cleanup(func() {
		err := wiremockClient.Reset()
		if err != nil {
			t.Fatal(fmt.Sprintf("Failed to reset wiremock: %v", err))
		}

		err = wiremockClient.ResetAllScenarios()
		if err != nil {
			t.Fatal(fmt.Sprintf("Failed to reset scenarios: %v", err))
		}

		// Also add container termination here to ensure it happens
		err = wiremockContainer.Terminate(ctx)
		if err != nil {
			t.Fatal(fmt.Sprintf("Failed to terminate container: %v", err))
		}
	})
}

func testAccCheckLatestRefreshSchemaConfig(confluentCloudBaseUrl, mockServerUrl string, schemaContent string) string {
	return fmt.Sprintf(`
	provider "confluent" {
      endpoint = "%s"
    }
	resource "confluent_schema" "%s" {
	  schema_registry_cluster {
        id = "%s"
      }
      rest_endpoint = "%s"
      credentials {
        key = "%s"
        secret = "%s"
	  }
	
	  subject_name = "%s"
	  format = "%s"
      schema = "%s"
      hard_delete = "%s"
      skip_validation_during_plan = "%s"
	  
      schema_reference {
        name = "%s"
        subject_name = "%s"
        version = %d
      }
      schema_reference {
        name = "%s"
        subject_name = "%s"
        version = %d
      }
	}
	`, confluentCloudBaseUrl, testSchemaResourceLabel, testStreamGovernanceClusterId, mockServerUrl, testSchemaRegistryKey, testSchemaRegistrySecret, testSubjectName, testFormat, schemaContent,
		testHardDelete, testSkipSchemaValidationDuringPlanTrue,
		testFirstSchemaReferenceDisplayName, testFirstSchemaReferenceSubject, testFirstSchemaReferenceVersion,
		testSecondSchemaReferenceDisplayName, testSecondSchemaReferenceSubject, testSecondSchemaReferenceVersion)
}
