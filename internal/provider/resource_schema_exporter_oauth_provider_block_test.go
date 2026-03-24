// Copyright 2023 Confluent Inc. All Rights Reserved.
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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"testing"
)

const (
	testOAuthBearerClientId          = "test-client-id"
	testOAuthBearerClientSecret      = "test-oauth-secret"
	testOAuthBearerIdentityPoolId    = "pool-abc123"
	testOAuthBearerScope             = "test-client-id/.default"
	testOAuthBearerCredentialsSource = "OAUTHBEARER"
	testOAuthBearerIssuerEndpointUrl = "https://login.example.com/oauth2/v2.0/token"

	testOAuthBearerUpdatedIdentityPoolId = "pool-updated"
	testOAuthBearerUpdatedScope          = "test-client-id/.default-updated"
)

func TestAccSchemaExporterOAuthWithEnhancedProviderBlock(t *testing.T) {
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

	generalResponse, _ := ioutil.ReadFile("../testdata/schema_exporter_oauth/general_response.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(createSchemaExporterUrlPath)).
		InScenario(schemaExporterResourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateSchemaExporterHasBeenCreated).
		WillReturn(
			string(generalResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	createdExporter, _ := ioutil.ReadFile("../testdata/schema_exporter_oauth/created_exporter.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readCreatedSchemaExporterUrlPath)).
		InScenario(schemaExporterResourceScenarioName).
		WhenScenarioStateIs(scenarioStateSchemaExporterHasBeenCreated).
		WillReturn(
			string(createdExporter),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(readCreatedSchemaExporterUrlPath)).
		InScenario(schemaExporterResourceScenarioName).
		WillSetStateTo(scenarioStateSchemaExporterHasBeenUpdated).
		WillReturn(
			string(generalResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	updatedExporter, _ := ioutil.ReadFile("../testdata/schema_exporter_oauth/updated_exporter.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readCreatedSchemaExporterUrlPath)).
		InScenario(schemaExporterResourceScenarioName).
		WhenScenarioStateIs(scenarioStateSchemaExporterHasBeenUpdated).
		WillReturn(
			string(updatedExporter),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(readCreatedSchemaExporterUrlPath+"/pause")).
		InScenario(schemaExporterResourceScenarioName).
		WillReturn(
			string(generalResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(readCreatedSchemaExporterUrlPath+"/resume")).
		InScenario(schemaExporterResourceScenarioName).
		WillReturn(
			string(generalResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(readCreatedSchemaExporterUrlPath+"/reset")).
		InScenario(schemaExporterResourceScenarioName).
		WillReturn(
			string(generalResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	runningStatusResponse, _ := ioutil.ReadFile("../testdata/schema_exporter_oauth/running_status.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readCreatedSchemaExporterStatusUrlPath)).
		InScenario(schemaExporterResourceScenarioName).
		WhenScenarioStateIs(scenarioStateSchemaExporterHasBeenCreated).
		WillReturn(
			string(runningStatusResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	pausedStatusResponse, _ := ioutil.ReadFile("../testdata/schema_exporter_oauth/pause_status.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readCreatedSchemaExporterStatusUrlPath)).
		InScenario(schemaExporterResourceScenarioName).
		WhenScenarioStateIs(scenarioStateSchemaExporterHasBeenUpdated).
		WillReturn(
			string(pausedStatusResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(readCreatedSchemaExporterUrlPath)).
		InScenario(schemaExporterResourceScenarioName).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: schemaExporterOAuthResourceConfigWithEnhancedProviderBlock(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(schemaExporterLabel, "name", "exporter1"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "context", "tc"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "context_type", "CUSTOM"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "status", "RUNNING"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "subjects.#", "1"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "subjects.0", "foo"),
					// Verify exactly 6 user-specified config keys are preserved (no boilerplate)
					resource.TestCheckResourceAttr(schemaExporterLabel, fmt.Sprintf("%s.%%", paramConfigs), "6"),
					resource.TestCheckResourceAttr(schemaExporterLabel, fmt.Sprintf("%s.bearer.auth.client.id", paramConfigs), testOAuthBearerClientId),
					resource.TestCheckResourceAttr(schemaExporterLabel, fmt.Sprintf("%s.bearer.auth.client.secret", paramConfigs), testOAuthBearerClientSecret),
					resource.TestCheckResourceAttr(schemaExporterLabel, fmt.Sprintf("%s.bearer.auth.identity.pool.id", paramConfigs), testOAuthBearerIdentityPoolId),
					resource.TestCheckResourceAttr(schemaExporterLabel, fmt.Sprintf("%s.bearer.auth.scope", paramConfigs), testOAuthBearerScope),
					resource.TestCheckResourceAttr(schemaExporterLabel, fmt.Sprintf("%s.bearer.auth.credentials.source", paramConfigs), testOAuthBearerCredentialsSource),
					resource.TestCheckResourceAttr(schemaExporterLabel, fmt.Sprintf("%s.bearer.auth.issuer.endpoint.url", paramConfigs), testOAuthBearerIssuerEndpointUrl),
					// Verify boilerplate keys are NOT in config
					resource.TestCheckNoResourceAttr(schemaExporterLabel, fmt.Sprintf("%s.schema.registry.url", paramConfigs)),
					resource.TestCheckNoResourceAttr(schemaExporterLabel, fmt.Sprintf("%s.bearer.auth.logical.cluster", paramConfigs)),
					// Verify destination_schema_registry_cluster block
					resource.TestCheckResourceAttr(schemaExporterLabel, "destination_schema_registry_cluster.#", "1"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "destination_schema_registry_cluster.0.rest_endpoint", testOriginalDestinationSchemaRegistryRestEndpoint),
					resource.TestCheckResourceAttr(schemaExporterLabel, "destination_schema_registry_cluster.0.credentials.#", "1"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "destination_schema_registry_cluster.0.credentials.0.key", testDestinationSchemaRegistryKey),
					resource.TestCheckResourceAttr(schemaExporterLabel, "destination_schema_registry_cluster.0.credentials.0.secret", testDestinationSchemaRegistrySecret),
				),
			},
			{
				Config: schemaExporterOAuthResourceUpdatedConfigWithEnhancedProviderBlock(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(schemaExporterLabel, "name", "exporter1"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "context", "tc-3"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "context_type", "CUSTOM"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "status", "PAUSED"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "subjects.#", "1"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "subjects.0", "foo3"),
					// Verify exactly 6 user-specified config keys are preserved after update
					resource.TestCheckResourceAttr(schemaExporterLabel, fmt.Sprintf("%s.%%", paramConfigs), "6"),
					resource.TestCheckResourceAttr(schemaExporterLabel, fmt.Sprintf("%s.bearer.auth.client.id", paramConfigs), testOAuthBearerClientId),
					resource.TestCheckResourceAttr(schemaExporterLabel, fmt.Sprintf("%s.bearer.auth.client.secret", paramConfigs), testOAuthBearerClientSecret),
					resource.TestCheckResourceAttr(schemaExporterLabel, fmt.Sprintf("%s.bearer.auth.identity.pool.id", paramConfigs), testOAuthBearerUpdatedIdentityPoolId),
					resource.TestCheckResourceAttr(schemaExporterLabel, fmt.Sprintf("%s.bearer.auth.scope", paramConfigs), testOAuthBearerUpdatedScope),
					resource.TestCheckResourceAttr(schemaExporterLabel, fmt.Sprintf("%s.bearer.auth.credentials.source", paramConfigs), testOAuthBearerCredentialsSource),
					resource.TestCheckResourceAttr(schemaExporterLabel, fmt.Sprintf("%s.bearer.auth.issuer.endpoint.url", paramConfigs), testOAuthBearerIssuerEndpointUrl),
					// Verify boilerplate keys are still NOT in config after update
					resource.TestCheckNoResourceAttr(schemaExporterLabel, fmt.Sprintf("%s.schema.registry.url", paramConfigs)),
					resource.TestCheckNoResourceAttr(schemaExporterLabel, fmt.Sprintf("%s.bearer.auth.logical.cluster", paramConfigs)),
					// Verify destination_schema_registry_cluster block
					resource.TestCheckResourceAttr(schemaExporterLabel, "destination_schema_registry_cluster.#", "1"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "destination_schema_registry_cluster.0.rest_endpoint", testDestinationSchemaRegistryRestEndpoint),
					resource.TestCheckResourceAttr(schemaExporterLabel, "destination_schema_registry_cluster.0.credentials.#", "1"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "destination_schema_registry_cluster.0.credentials.0.key", testDestinationSchemaRegistryKey),
					resource.TestCheckResourceAttr(schemaExporterLabel, "destination_schema_registry_cluster.0.credentials.0.secret", testDestinationSchemaRegistrySecret),
				),
			},
		},
	})
}

func schemaExporterOAuthResourceConfigWithEnhancedProviderBlock(mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
 	    schema_registry_id = "%s"
	    schema_registry_rest_endpoint = "%s"
	    schema_registry_api_key       = "%s"
	    schema_registry_api_secret = "%s"
 	}
 	resource "confluent_schema_exporter" "main" {
		name = "exporter1"
		context = "tc"
		context_type = "CUSTOM"
		subjects = ["foo"]

		destination_schema_registry_cluster {
		  rest_endpoint = "%s"
		  credentials {
			key    = "%s"
			secret = "%s"
		  }
		}

		config = {
			"bearer.auth.client.id"          = "%s"
			"bearer.auth.client.secret"      = "%s"
			"bearer.auth.identity.pool.id"   = "%s"
			"bearer.auth.scope"              = "%s"
			"bearer.auth.credentials.source" = "%s"
			"bearer.auth.issuer.endpoint.url" = "%s"
		}
	}

 	`, testStreamGovernanceClusterId, mockServerUrl, testSchemaRegistryKey, testSchemaRegistrySecret,
		testOriginalDestinationSchemaRegistryRestEndpoint, testDestinationSchemaRegistryKey, testDestinationSchemaRegistrySecret,
		testOAuthBearerClientId, testOAuthBearerClientSecret, testOAuthBearerIdentityPoolId,
		testOAuthBearerScope, testOAuthBearerCredentialsSource, testOAuthBearerIssuerEndpointUrl)
}

func schemaExporterOAuthResourceUpdatedConfigWithEnhancedProviderBlock(mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
 	    schema_registry_id = "%s"
	    schema_registry_rest_endpoint = "%s"
	    schema_registry_api_key       = "%s"
	    schema_registry_api_secret = "%s"
 	}
 	resource "confluent_schema_exporter" "main" {
	    name = "exporter1"
		context = "tc-3"
		context_type = "CUSTOM"
		subjects = ["foo3"]

        status = "PAUSED"

		destination_schema_registry_cluster {
		  rest_endpoint = "%s"
		  credentials {
			key    = "%s"
			secret = "%s"
		  }
		}

		config = {
			"bearer.auth.client.id"          = "%s"
			"bearer.auth.client.secret"      = "%s"
			"bearer.auth.identity.pool.id"   = "%s"
			"bearer.auth.scope"              = "%s"
			"bearer.auth.credentials.source" = "%s"
			"bearer.auth.issuer.endpoint.url" = "%s"
		}
	}
 	`, testStreamGovernanceClusterId, mockServerUrl, testSchemaRegistryKey, testSchemaRegistrySecret,
		testDestinationSchemaRegistryRestEndpoint, testDestinationSchemaRegistryKey, testDestinationSchemaRegistrySecret,
		testOAuthBearerClientId, testOAuthBearerClientSecret, testOAuthBearerUpdatedIdentityPoolId,
		testOAuthBearerUpdatedScope, testOAuthBearerCredentialsSource, testOAuthBearerIssuerEndpointUrl)
}
