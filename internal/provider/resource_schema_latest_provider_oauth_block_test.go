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
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

//
// This test validates that SetSchemaDiff normalization works correctly with OAuth authentication.
// The bug: old code exits early if clusterApiKey == "", skipping schemaLookupCheck() normalization.
// The fix: checks isOAuthEnabled and only requires API keys when OAuth is not enabled.
//
// Test scenario:
// Step 1: Create schema with "foobar"
// Step 2 (plan-only): Config changes to "foobar  " (with whitespace) → should not show non-empty terraform plan (aka terraform drift)
//        - SetSchemaDiff detects config change from "foobar" to "foobar  "
//        - Buggy code: exits early when clusterApiKey=="", no normalization, shows non-empty terraform plan (FAIL)
//        - Fixed code: checks isOAuthEnabled, runs schemaLookupCheck, finds equivalent schema, empty terraform plan (PASS)
//

func TestAccLatestSchemaWithEnhancedProviderBlockOAuth(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockSchemaTestServerUrl := wiremockContainer.URI
	mockOAuthServerUrl := wiremockContainer.URI
	confluentCloudBaseUrl := mockSchemaTestServerUrl
	wiremockClient := wiremock.NewClient(mockSchemaTestServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	// Add constants for scenario states
	const (
		scenarioStateOAuthSchemaCreated = "OAuth Schema Created"
	)

	// Temporarily unset API key environment variables to avoid conflict with OAuth logic in provider_test.go
	oldApiKey := os.Getenv("CONFLUENT_CLOUD_API_KEY")
	oldApiSecret := os.Getenv("CONFLUENT_CLOUD_API_SECRET")
	os.Unsetenv("CONFLUENT_CLOUD_API_KEY")
	os.Unsetenv("CONFLUENT_CLOUD_API_SECRET")

	defer func() {
		if oldApiKey != "" {
			os.Setenv("CONFLUENT_CLOUD_API_KEY", oldApiKey)
		}
		if oldApiSecret != "" {
			os.Setenv("CONFLUENT_CLOUD_API_SECRET", oldApiSecret)
		}
	}()

	// Mock OAuth token endpoint
	oauthTokenResponse := `{
        "access_token": "mock-external-access-token",
        "token_type": "Bearer",
        "expires_in": 3600
    }`
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo("/oauth/token")).
		WillReturn(
			oauthTokenResponse,
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// Mock STS token exchange endpoint
	stsTokenResponse := `{
        "access_token": "mock-confluent-access-token",
        "token_type": "Bearer",
        "expires_in": 3600,
        "scope": "schema-registry"
    }`
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo("/sts/v1/oauth2/token")).
		WillReturn(
			stsTokenResponse,
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// ============================================================
	// Step 1: Create schema - server returns normal "foobar"
	// ============================================================

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

	// For Step 1: Server returns normal "foobar"
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
		WillSetStateTo(scenarioStateOAuthSchemaCreated).
		WillReturn(
			string(readSchemasResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// ============================================================
	// Step 2 (plan-only): Config changes to add whitespace
	// Server will return the original schema (lookup will find it)
	// ============================================================

	// For Step 2: Keep returning the original "foobar" schema
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readLatestSchemaPath)).
		InScenario(schemaScenarioName).
		WhenScenarioStateIs(scenarioStateOAuthSchemaCreated).
		WillReturn(
			string(readLatestSchemaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readSchemasPath)).
		InScenario(schemaScenarioName).
		WhenScenarioStateIs(scenarioStateOAuthSchemaCreated).
		WillReturn(
			string(readSchemasResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// Mock validation endpoint for the schema with whitespace
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(validateSchemaPath)).
		InScenario(schemaScenarioName).
		WhenScenarioStateIs(scenarioStateOAuthSchemaCreated).
		WillReturn(
			string(validateSchemaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// Mock schema lookup - uses /subjects/{subject}?normalize=false endpoint (different from creation)
	// The code tries normalize=false first, if it fails (404), it tries normalize=true
	// We return success on normalize=true to simulate finding the schema after normalization
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo("/subjects/test2")).
		WithQueryParam("normalize", wiremock.EqualTo("true")).
		InScenario(schemaScenarioName).
		WhenScenarioStateIs(scenarioStateOAuthSchemaCreated).
		WillReturn(
			string(readLatestSchemaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// Cleanup
	deleteSchemaStub := wiremock.Delete(wiremock.URLPathEqualTo(deleteSchemaPath)).
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
		PreCheck:          func() { testAccPreCheckOAuth(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return testAccCheckSchemaDestroy(s, mockSchemaTestServerUrl)
		},
		Steps: []resource.TestStep{
			{
				// Step 1: Create schema with "foobar" - normal creation with OAuth
				Config: testAccCheckSchemaConfigWithOAuthProviderBlock(confluentCloudBaseUrl, mockSchemaTestServerUrl, mockOAuthServerUrl, testSchemaContent),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaExists(fullSchemaResourceLabel),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "id", fmt.Sprintf("%s/%s/%s", testStreamGovernanceClusterId, testSubjectName, latestSchemaVersionAndPlaceholderForSchemaIdentifier)),
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "schema_registry_cluster.#", "0"),
					resource.TestCheckNoResourceAttr(fullSchemaResourceLabel, "schema_registry_cluster.0.id"),
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
					resource.TestCheckResourceAttr(fullSchemaResourceLabel, "credentials.#", "0"),
					resource.TestCheckNoResourceAttr(fullSchemaResourceLabel, "credentials.0.key"),
					resource.TestCheckNoResourceAttr(fullSchemaResourceLabel, "credentials.0.secret"),
					resource.TestCheckNoResourceAttr(fullSchemaResourceLabel, "rest_endpoint"),
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
				// Step 2 (plan-only): Config changes to add whitespace "foobar  "
				// SetSchemaDiff will detect the change from "foobar" to "foobar  "
				// Buggy code: exits early when clusterApiKey=="", skips schemaLookupCheck, shows terraform drift (FAIL)
				// Fixed code: checks isOAuthEnabled, runs schemaLookupCheck, finds schema ID 100001 matches state, no terraform drift (PASS)
				Config:             testAccCheckSchemaConfigWithOAuthProviderBlock(confluentCloudBaseUrl, mockSchemaTestServerUrl, mockOAuthServerUrl, testSchemaContent+"  "),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false, // Should NOT show terraform drift after normalization
			},
		},
	})

	checkStubCount(t, wiremockClient, deleteSchemaStub, fmt.Sprintf("DELETE %s", deleteSchemaPath), expectedCountOne)
}

func testAccPreCheckOAuth(t *testing.T) {
	// For OAuth tests, API keys should already be unset by the test setup
	// This precheck is essentially a no-op, but kept for consistency
}

func testAccCheckSchemaConfigWithOAuthProviderBlock(confluentCloudBaseUrl, mockServerUrl, mockOAuthServerUrl string, schemaContent string) string {
	return fmt.Sprintf(`
    provider "confluent" {
      endpoint = "%s"
      oauth {
        oauth_external_token_url = "%s/oauth/token"
        oauth_external_client_id = "test-client-id"
        oauth_external_client_secret = "test-client-secret"
        oauth_external_token_scope = "test-scope"
        oauth_identity_pool_id = "test-pool-id"
      }
      schema_registry_rest_endpoint = "%s"
      schema_registry_id = "%s"
    }
    resource "confluent_schema" "%s" {
      subject_name = "%s"
      format = "%s"
      schema = "%s"
      recreate_on_update = false
      skip_validation_during_plan = false
      
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

      ruleset {
        domain_rules {
          name = "encryptPII"
          kind = "TRANSFORM"
          type = "ENCRYPT"
          mode = "WRITEREAD"
          tags = ["PII"]
          params = {
              "encrypt.kek.name" = "testkek2"
          }
        }
        domain_rules  {
          name = "encrypt"
          kind = "TRANSFORM"
          type = "ENCRYPT"
          mode = "WRITEREAD"
          tags = ["PIIIII"]
          params = {
              "encrypt.kek.name" = "testkek2"
          }
        }
        migration_rules  {
          name = "encrypt"
          kind = "TRANSFORM"
          type = "ENCRYPT"
          mode = "WRITEREAD"
          tags = ["PIm"]
          params = {
              "encrypt.kek.name" = "testkekM"
          }
        }
      }
    }
    `, confluentCloudBaseUrl, mockOAuthServerUrl, mockServerUrl, testStreamGovernanceClusterId,
		testSchemaResourceLabel, testSubjectName, testFormat, schemaContent,
		testFirstSchemaReferenceDisplayName, testFirstSchemaReferenceSubject, testFirstSchemaReferenceVersion,
		testSecondSchemaReferenceDisplayName, testSecondSchemaReferenceSubject, testSecondSchemaReferenceVersion)
}
