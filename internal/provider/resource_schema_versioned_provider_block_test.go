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
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccVersionedSchemaWithEnhancedProviderBlock(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockSchemaTestServerUrl = wiremockContainer.URI
	confluentCloudBaseUrl := ""
	wiremockClient := wiremock.NewClient(mockSchemaTestServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()
	validateSchemaResponse, _ := ioutil.ReadFile("../testdata/schema_registry_schema/validate_schema.json")
	validateSchemaStub := wiremock.Post(wiremock.URLPathEqualTo(validateSchemaPath)).
		InScenario(schemaScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(validateSchemaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(validateSchemaStub)

	createSchemaResponse, _ := ioutil.ReadFile("../testdata/schema_registry_schema/create_schema.json")
	createSchemaStub := wiremock.Post(wiremock.URLPathEqualTo(createSchemaPath)).
		InScenario(schemaScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateSchemaHasBeenCreated).
		WillReturn(
			string(createSchemaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(createSchemaStub)

	readCreatedSchemasResponse, _ := ioutil.ReadFile("../testdata/schema_registry_schema/read_schemas.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readSchemasPath)).
		InScenario(schemaScenarioName).
		WhenScenarioStateIs(scenarioStateSchemaHasBeenCreated).
		WillReturn(
			string(readCreatedSchemasResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	checkSchemaExistsResponse, _ := ioutil.ReadFile("../testdata/schema_registry_schema/create_schema.json")
	checkSchemaExistsStub := wiremock.Post(wiremock.URLPathEqualTo(createSchemaPath)).
		InScenario(schemaScenarioName).
		WhenScenarioStateIs(scenarioStateSchemaHasBeenCreated).
		WillReturn(
			string(checkSchemaExistsResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(checkSchemaExistsStub)

	deleteSchemaStub := wiremock.Delete(wiremock.URLPathEqualTo(deleteSchemaPath)).
		InScenario(schemaScenarioName).
		WhenScenarioStateIs(scenarioStateSchemaHasBeenCreated).
		WillSetStateTo(scenarioStateSchemaHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteSchemaStub)

	readDeletedSaResponse, _ := ioutil.ReadFile("../testdata/schema_registry_schema/read_schemas_after_delete.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readSchemasPath)).
		InScenario(schemaScenarioName).
		WhenScenarioStateIs(scenarioStateSchemaHasBeenDeleted).
		WillReturn(
			string(readDeletedSaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// Set fake values for schema content since it's required for importing
	_ = os.Setenv("SCHEMA_CONTENT", testSchemaContent)
	defer func() {
		_ = os.Unsetenv("SCHEMA_CONTENT")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSchemaDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckVersionedSchemaConfigWithEnhancedProviderBlock(confluentCloudBaseUrl, mockSchemaTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaExists(fullSchemaResourceLabel),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "id", fmt.Sprintf("%s/%s/%d", testStreamGovernanceClusterId, testSubjectName, testSchemaIdentifier)),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_registry_cluster.#", "0"),
					resource.TestCheckNoResourceAttr(fullSchemaResourceLabel, "schema_registry_cluster.0.id"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "subject_name", testSubjectName),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "format", testFormat),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema", testSchemaContent),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "version", strconv.Itoa(testSchemaVersion)),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_identifier", strconv.Itoa(testSchemaIdentifier)),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "hard_delete", testHardDelete),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "recreate_on_update", testRecreateOnUpdateTrue),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_reference.#", "2"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_reference.0.%", "3"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_reference.0.name", testFirstSchemaReferenceDisplayName),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_reference.0.subject_name", testFirstSchemaReferenceSubject),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_reference.0.version", strconv.Itoa(testFirstSchemaReferenceVersion)),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_reference.1.%", "3"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_reference.1.name", testSecondSchemaReferenceDisplayName),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_reference.1.subject_name", testSecondSchemaReferenceSubject),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_reference.1.version", strconv.Itoa(testSecondSchemaReferenceVersion)),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "credentials.#", "0"),
					resource.TestCheckNoResourceAttr(fullSchemaResourceLabel, "credentials.0.key"),
					resource.TestCheckNoResourceAttr(fullSchemaResourceLabel, "credentials.0.secret"),
					resource.TestCheckNoResourceAttr(fullSchemaResourceLabel, "rest_endpoint"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "%", strconv.Itoa(testNumberOfSchemaRegistrySchemaResourceAttributes)),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullSchemaResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, deleteSchemaStub, fmt.Sprintf("DELETE %s", readSchemasPath), expectedCountOne)
}

func testAccCheckVersionedSchemaConfigWithEnhancedProviderBlock(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	  schema_registry_api_key = "%s"
	  schema_registry_api_secret = "%s"
	  schema_registry_rest_endpoint = "%s"
	  schema_registry_id = "%s"
	}
	resource "confluent_schema" "%s" {
	  subject_name = "%s"
	  format = "%s"
      schema = "%s"

      hard_delete = "%s"
      recreate_on_update = "%s"
	  
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
	`, confluentCloudBaseUrl, kafkaApiKey, kafkaApiSecret, mockServerUrl, testStreamGovernanceClusterId, testSchemaResourceLabel, testSubjectName, testFormat, testSchemaContent,
		testHardDelete, testRecreateOnUpdateTrue,
		testFirstSchemaReferenceDisplayName, testFirstSchemaReferenceSubject, testFirstSchemaReferenceVersion,
		testSecondSchemaReferenceDisplayName, testSecondSchemaReferenceSubject, testSecondSchemaReferenceVersion)
}
