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
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccLatestSchema(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockSchemaTestServerUrl := wiremockContainer.URI
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

	readLatestSchemaResponse, _ := ioutil.ReadFile("../testdata/schema_registry_schema/read_latest_schema.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readLatestSchemaPath)).
		InScenario(schemaScenarioName).
		WhenScenarioStateIs(scenarioStateSchemaHasBeenCreated).
		WillReturn(
			string(readLatestSchemaResponse),
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

	// Set fake values for secrets since those are required for importing
	_ = os.Setenv("IMPORT_SCHEMA_REGISTRY_API_KEY", testSchemaRegistryUpdatedKey)
	_ = os.Setenv("IMPORT_SCHEMA_REGISTRY_API_SECRET", testSchemaRegistryUpdatedSecret)
	_ = os.Setenv("IMPORT_SCHEMA_REGISTRY_REST_ENDPOINT", mockSchemaTestServerUrl)
	defer func() {
		_ = os.Unsetenv("IMPORT_SCHEMA_REGISTRY_API_KEY")
		_ = os.Unsetenv("IMPORT_SCHEMA_REGISTRY_API_SECRET")
		_ = os.Unsetenv("IMPORT_SCHEMA_REGISTRY_REST_ENDPOINT")
	}()

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
				Config: testAccCheckLatestSchemaConfig(confluentCloudBaseUrl, mockSchemaTestServerUrl),
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
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.#", "1"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.%", "2"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.#", "2"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.0.%", "11"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.0.doc", ""),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.0.expr", ""),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.0.kind", "TRANSFORM"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.0.mode", "WRITEREAD"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.0.name", "encrypt"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.0.on_failure", "ERROR,ERROR"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.0.on_success", "NONE,NONE"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.0.params.%", "1"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.0.params.encrypt.kek.name", "testkek2"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.0.tags.#", "1"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.0.tags.0", "PIIIII"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.0.type", "ENCRYPT"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.0.disabled", "false"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.1.%", "11"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.1.doc", ""),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.1.expr", ""),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.1.kind", "TRANSFORM"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.1.mode", "WRITEREAD"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.1.name", "encryptPII"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.1.on_failure", "ERROR,ERROR"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.1.on_success", "NONE,NONE"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.1.params.%", "1"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.1.params.encrypt.kek.name", "testkek2"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.1.tags.#", "1"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.1.tags.0", "PII"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.1.type", "ENCRYPT"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "ruleset.0.domain_rules.1.disabled", "false"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "metadata.#", "1"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "metadata.0.%", "3"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "metadata.0.properties.%", "2"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "metadata.0.properties.email", "bob@acme.com"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "metadata.0.properties.owner", "Bob Jones"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "metadata.0.sensitive.#", "2"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "metadata.0.sensitive.0", "s1"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "metadata.0.sensitive.1", "s2"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "metadata.0.tags.#", "2"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "metadata.0.tags.0.%", "2"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "metadata.0.tags.0.key", "tag1"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "metadata.0.tags.0.value.#", "1"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "metadata.0.tags.0.value.0", "PII"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "metadata.0.tags.1.%", "2"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "metadata.0.tags.1.key", "tag2"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "metadata.0.tags.1.value.#", "1"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "metadata.0.tags.1.value.0", "PIIIII"),
				),
			},
			{
				Config: testAccCheckLatestSchemaConfigWithUpdatedCredentials(confluentCloudBaseUrl, mockSchemaTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaExists(fullSchemaResourceLabel),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "id", fmt.Sprintf("%s/%s/%s", testStreamGovernanceClusterId, testSubjectName, latestSchemaVersionAndPlaceholderForSchemaIdentifier)),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_registry_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_registry_cluster.0.%", "1"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_registry_cluster.0.id", testStreamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "rest_endpoint", mockSchemaTestServerUrl),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "credentials.0.key", testSchemaRegistryUpdatedKey),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "credentials.0.secret", testSchemaRegistryUpdatedSecret),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "subject_name", testSubjectName),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "format", testFormat),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema", testSchemaContent),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "version", strconv.Itoa(testSchemaVersion)),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_identifier", strconv.Itoa(testSchemaIdentifier)),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "hard_delete", testHardDelete),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "recreate_on_update", testRecreateOnUpdateFalse),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "skip_validation_during_plan", testSkipSchemaValidationDuringPlanFalse),
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
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullSchemaResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, deleteSchemaStub, fmt.Sprintf("DELETE %s", readSchemasPath), expectedCountOne)
}

func testAccCheckLatestSchemaConfig(confluentCloudBaseUrl, mockServerUrl string) string {
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
	`, confluentCloudBaseUrl, testSchemaResourceLabel, testStreamGovernanceClusterId, mockServerUrl, testSchemaRegistryKey, testSchemaRegistrySecret, testSubjectName, testFormat, testSchemaContent,
		testHardDelete, testSkipSchemaValidationDuringPlanTrue,
		testFirstSchemaReferenceDisplayName, testFirstSchemaReferenceSubject, testFirstSchemaReferenceVersion,
		testSecondSchemaReferenceDisplayName, testSecondSchemaReferenceSubject, testSecondSchemaReferenceVersion)
}

func testAccCheckLatestSchemaConfigWithUpdatedCredentials(confluentCloudBaseUrl, mockServerUrl string) string {
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
	`, confluentCloudBaseUrl, testSchemaResourceLabel, testStreamGovernanceClusterId, mockServerUrl, testSchemaRegistryUpdatedKey, testSchemaRegistryUpdatedSecret, testSubjectName, testFormat, testSchemaContent,
		testHardDelete, testRecreateOnUpdateFalse, testFirstSchemaReferenceDisplayName, testFirstSchemaReferenceSubject, testFirstSchemaReferenceVersion,
		testSecondSchemaReferenceDisplayName, testSecondSchemaReferenceSubject, testSecondSchemaReferenceVersion)
}
