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
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	scenarioStateSchemaHasBeenValidated = "A new schema has been just validated"
	scenarioStateSchemaHasBeenCreated   = "A new schema has been just created"
	scenarioStateSchemaHasBeenDeleted   = "The schema has been deleted"
	schemaScenarioName                  = "confluent_schema Resource Lifecycle"

	testSubjectName               = "test2"
	testSchemaIdentifier          = 100001
	testSchemaVersion             = 8
	testFormat                    = "AVRO"
	testStreamGovernanceClusterId = "lsrc-abc123"
	testSchemaContent             = "foobar"
	testSchemaResourceLabel       = "test_schema_resource_label"

	testFirstSchemaReferenceDisplayName = "sampleRecord"
	testFirstSchemaReferenceSubject     = "test2"
	testFirstSchemaReferenceVersion     = 9

	testSecondSchemaReferenceDisplayName = "sampleRecord2"
	testSecondSchemaReferenceSubject     = "test3"
	testSecondSchemaReferenceVersion     = 3

	testNumberOfSchemaRegistrySchemaResourceAttributes = 12

	testSchemaRegistryKey           = "foo"
	testSchemaRegistrySecret        = "bar"
	testSchemaRegistryUpdatedKey    = "foo_new"
	testSchemaRegistryUpdatedSecret = "bar_new"

	testHardDelete = "false"

	testRecreateOnUpdateTrue  = "true"
	testRecreateOnUpdateFalse = "false"
)

var fullSchemaResourceLabel = fmt.Sprintf("confluent_schema.%s", testSchemaResourceLabel)
var validateSchemaPath = fmt.Sprintf("/compatibility/subjects/%s/versions", testSubjectName)
var createSchemaPath = fmt.Sprintf("/subjects/%s/versions", testSubjectName)
var readSchemasPath = fmt.Sprintf("/schemas")
var readLatestSchemaPath = fmt.Sprintf("/subjects/%s/versions/latest", testSubjectName)
var deleteSchemaPath = fmt.Sprintf("/subjects/%s/versions/%s", testSubjectName, strconv.Itoa(testSchemaVersion))

// TODO: APIF-1990
var mockSchemaTestServerUrl = ""

func TestAccVersionedSchema(t *testing.T) {
	containerPort := "8080"
	containerPortTcp := fmt.Sprintf("%s/tcp", containerPort)
	ctx := context.Background()
	listeningPort := wait.ForListeningPort(nat.Port(containerPortTcp))
	req := testcontainers.ContainerRequest{
		Image:        "rodolpheche/wiremock",
		ExposedPorts: []string{containerPortTcp},
		WaitingFor:   listeningPort,
	}
	wiremockContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	require.NoError(t, err)

	// nolint:errcheck
	defer wiremockContainer.Terminate(ctx)

	host, err := wiremockContainer.Host(ctx)
	require.NoError(t, err)

	wiremockHttpMappedPort, err := wiremockContainer.MappedPort(ctx, nat.Port(containerPort))
	require.NoError(t, err)

	mockSchemaTestServerUrl = fmt.Sprintf("http://%s:%s", host, wiremockHttpMappedPort.Port())
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
		WillSetStateTo(scenarioStateSchemaHasBeenValidated).
		WillReturn(
			string(validateSchemaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(validateSchemaStub)

	createSchemaResponse, _ := ioutil.ReadFile("../testdata/schema_registry_schema/create_schema.json")
	createSchemaStub := wiremock.Post(wiremock.URLPathEqualTo(createSchemaPath)).
		InScenario(schemaScenarioName).
		WhenScenarioStateIs(scenarioStateSchemaHasBeenValidated).
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

	// Set fake values for secrets since those are required for importing
	_ = os.Setenv("IMPORT_SCHEMA_REGISTRY_API_KEY", testSchemaRegistryUpdatedKey)
	_ = os.Setenv("IMPORT_SCHEMA_REGISTRY_API_SECRET", testSchemaRegistryUpdatedSecret)
	_ = os.Setenv("IMPORT_SCHEMA_REGISTRY_REST_ENDPOINT", mockSchemaTestServerUrl)
	_ = os.Setenv("SCHEMA_CONTENT", testSchemaContent)
	defer func() {
		_ = os.Unsetenv("IMPORT_SCHEMA_REGISTRY_API_KEY")
		_ = os.Unsetenv("IMPORT_SCHEMA_REGISTRY_API_SECRET")
		_ = os.Unsetenv("IMPORT_SCHEMA_REGISTRY_REST_ENDPOINT")
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
				Config: testAccCheckSchemaConfig(confluentCloudBaseUrl, mockSchemaTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaExists(fullSchemaResourceLabel),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "id", fmt.Sprintf("%s/%s/%d", testStreamGovernanceClusterId, testSubjectName, testSchemaIdentifier)),
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
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "%", strconv.Itoa(testNumberOfSchemaRegistrySchemaResourceAttributes)),
				),
			},
			{
				Config: testAccCheckSchemaConfigWithUpdatedCredentials(confluentCloudBaseUrl, mockSchemaTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaExists(fullSchemaResourceLabel),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "id", fmt.Sprintf("%s/%s/%d", testStreamGovernanceClusterId, testSubjectName, testSchemaIdentifier)),
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

func testAccCheckSchemaDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(mockSchemaTestServerUrl, clusterId, testSchemaRegistryKey, testSchemaRegistrySecret, false)
	// Loop through the resources in state, verifying each Schema is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_schema" {
			continue
		}
		deletedSchemaId := rs.Primary.ID
		schemaRegistrySchemas, _, err := c.apiClient.SchemasV1Api.GetSchemas(c.apiContext(context.Background())).Execute()
		_, exists := findSchemaById(schemaRegistrySchemas, strconv.Itoa(testSchemaIdentifier), testSubjectName)
		if err == nil {
			if exists {
				if deletedSchemaId == rs.Primary.ID {
					return fmt.Errorf("schema (%s) still exists", rs.Primary.ID)
				}
			} else {
				return nil
			}
		}
		return err
	}
	return nil
}

func testAccCheckSchemaConfig(confluentCloudBaseUrl, mockServerUrl string) string {
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
	`, confluentCloudBaseUrl, testSchemaResourceLabel, testStreamGovernanceClusterId, mockServerUrl, testSchemaRegistryKey, testSchemaRegistrySecret, testSubjectName, testFormat, testSchemaContent,
		testHardDelete, testRecreateOnUpdateTrue,
		testFirstSchemaReferenceDisplayName, testFirstSchemaReferenceSubject, testFirstSchemaReferenceVersion,
		testSecondSchemaReferenceDisplayName, testSecondSchemaReferenceSubject, testSecondSchemaReferenceVersion)
}

func testAccCheckSchemaConfigWithUpdatedCredentials(confluentCloudBaseUrl, mockServerUrl string) string {
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
		testHardDelete, testRecreateOnUpdateTrue, testFirstSchemaReferenceDisplayName, testFirstSchemaReferenceSubject, testFirstSchemaReferenceVersion,
		testSecondSchemaReferenceDisplayName, testSecondSchemaReferenceSubject, testSecondSchemaReferenceVersion)
}

func testAccCheckSchemaExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s schema has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s schema", n)
		}

		return nil
	}
}
