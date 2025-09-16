// Copyright 2025 Confluent Inc. All Rights Reserved.
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
	providerIntegrationV2AuthScenarioName                = "confluent_provider_integration_v2_authorization Resource Lifecycle"
	scenarioStateProviderIntegrationV2AuthHasBeenCreated = "The new provider_integration_v2_authorization has been just created"
	scenarioStateProviderIntegrationV2AuthHasBeenDeleted = "The provider_integration_v2_authorization has been deleted"
	
	// Azure authorization constants
	azureProviderIntegrationV2AuthId           = "cspi-abc123"
	azureProviderIntegrationV2AuthTenantId     = "12345678-1234-1234-1234-123456789abc"
	azureProviderIntegrationV2AuthEnvironmentId = "env-00000"
	
	// GCP authorization constants
	gcpProviderIntegrationV2AuthId           = "cspi-def456"
	gcpProviderIntegrationV2AuthEnvironmentId = "env-00000"
)

func TestAccProviderIntegrationV2AuthorizationAzure(t *testing.T) {
	ctx := context.Background()
	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockServerUrl := wiremockContainer.URI
	confluentCloudBaseUrl := ""
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	// Mock the initial integration read (DRAFT status)
	createAzureProviderIntegrationV2Response, _ := ioutil.ReadFile("../testdata/provider_integration_v2/create_azure_provider_integration_v2.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/pim/v2/integrations/%s", azureProviderIntegrationV2AuthId))).
		InScenario(providerIntegrationV2AuthScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(createAzureProviderIntegrationV2Response),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// Mock the PATCH operation (DRAFT -> CREATED)
	updateAzureProviderIntegrationV2AuthResponse, _ := ioutil.ReadFile("../testdata/provider_integration_v2_authorization/update_azure_provider_integration_v2_authorization.json")
	updateAzureProviderIntegrationV2AuthStub := wiremock.Patch(wiremock.URLPathEqualTo(fmt.Sprintf("/pim/v2/integrations/%s", azureProviderIntegrationV2AuthId))).
		InScenario(providerIntegrationV2AuthScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateProviderIntegrationV2AuthHasBeenCreated).
		WillReturn(
			string(updateAzureProviderIntegrationV2AuthResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(updateAzureProviderIntegrationV2AuthStub)

	// Mock the integration read after PATCH (CREATED status)
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/pim/v2/integrations/%s", azureProviderIntegrationV2AuthId))).
		InScenario(providerIntegrationV2AuthScenarioName).
		WhenScenarioStateIs(scenarioStateProviderIntegrationV2AuthHasBeenCreated).
		WillReturn(
			string(updateAzureProviderIntegrationV2AuthResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// Mock validation failure (returns error to trigger warning)
	validateAzureProviderIntegrationV2AuthStub := wiremock.Post(wiremock.URLPathEqualTo("/pim/v2/integrations:validate")).
		InScenario(providerIntegrationV2AuthScenarioName).
		WhenScenarioStateIs(scenarioStateProviderIntegrationV2AuthHasBeenCreated).
		WillReturn(
			`{"errors":[{"status":"400","code":"bad_request","detail":"Azure setup required"}]}`,
			contentTypeJSONHeader,
			http.StatusBadRequest,
		)
	_ = wiremockClient.StubFor(validateAzureProviderIntegrationV2AuthStub)

	fullAzureProviderIntegrationV2AuthResourceLabel := fmt.Sprintf("confluent_provider_integration_v2_authorization.%s", azureProviderIntegrationV2AuthResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckProviderIntegrationV2AuthDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckProviderIntegrationV2AuthorizationAzureConfig(confluentCloudBaseUrl, mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProviderIntegrationV2AuthExists(fullAzureProviderIntegrationV2AuthResourceLabel),
					resource.TestCheckResourceAttr(fullAzureProviderIntegrationV2AuthResourceLabel, paramId, azureProviderIntegrationV2AuthId),
					resource.TestCheckResourceAttr(fullAzureProviderIntegrationV2AuthResourceLabel, paramProviderIntegrationIdAuth, azureProviderIntegrationV2AuthId),
					resource.TestCheckResourceAttr(fullAzureProviderIntegrationV2AuthResourceLabel, fmt.Sprintf("azure.0.%s", paramAzureCustomerTenantId), azureProviderIntegrationV2AuthTenantId),
					resource.TestCheckResourceAttrSet(fullAzureProviderIntegrationV2AuthResourceLabel, fmt.Sprintf("azure.0.%s", paramAzureConfluentMultiTenantAppId)),
					resource.TestCheckResourceAttr(fullAzureProviderIntegrationV2AuthResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), azureProviderIntegrationV2AuthEnvironmentId),
				),
			},
			{
				ResourceName:      fullAzureProviderIntegrationV2AuthResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					providerIntegrationId := resources[fullAzureProviderIntegrationV2AuthResourceLabel].Primary.ID
					environmentId := resources[fullAzureProviderIntegrationV2AuthResourceLabel].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + providerIntegrationId, nil
				},
			},
		},
	})
}

func TestAccProviderIntegrationV2AuthorizationGcp(t *testing.T) {
	ctx := context.Background()
	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockServerUrl := wiremockContainer.URI
	confluentCloudBaseUrl := ""
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	// Mock the initial integration read (DRAFT status)
	createGcpProviderIntegrationV2Response, _ := ioutil.ReadFile("../testdata/provider_integration_v2/create_gcp_provider_integration_v2.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/pim/v2/integrations/%s", gcpProviderIntegrationV2AuthId))).
		InScenario(providerIntegrationV2AuthScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(createGcpProviderIntegrationV2Response),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// Mock the PATCH operation (DRAFT -> CREATED)
	updateGcpProviderIntegrationV2AuthResponse, _ := ioutil.ReadFile("../testdata/provider_integration_v2_authorization/update_gcp_provider_integration_v2_authorization.json")
	updateGcpProviderIntegrationV2AuthStub := wiremock.Patch(wiremock.URLPathEqualTo(fmt.Sprintf("/pim/v2/integrations/%s", gcpProviderIntegrationV2AuthId))).
		InScenario(providerIntegrationV2AuthScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateProviderIntegrationV2AuthHasBeenCreated).
		WillReturn(
			string(updateGcpProviderIntegrationV2AuthResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(updateGcpProviderIntegrationV2AuthStub)

	// Mock the integration read after PATCH (CREATED status)
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/pim/v2/integrations/%s", gcpProviderIntegrationV2AuthId))).
		InScenario(providerIntegrationV2AuthScenarioName).
		WhenScenarioStateIs(scenarioStateProviderIntegrationV2AuthHasBeenCreated).
		WillReturn(
			string(updateGcpProviderIntegrationV2AuthResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// Mock validation failure (returns error to trigger warning)
	validateGcpProviderIntegrationV2AuthStub := wiremock.Post(wiremock.URLPathEqualTo("/pim/v2/integrations:validate")).
		InScenario(providerIntegrationV2AuthScenarioName).
		WhenScenarioStateIs(scenarioStateProviderIntegrationV2AuthHasBeenCreated).
		WillReturn(
			`{"errors":[{"status":"400","code":"bad_request","detail":"missing 'iam.serviceAccounts.getAccessToken' permission"}]}`,
			contentTypeJSONHeader,
			http.StatusBadRequest,
		)
	_ = wiremockClient.StubFor(validateGcpProviderIntegrationV2AuthStub)

	fullGcpProviderIntegrationV2AuthResourceLabel := fmt.Sprintf("confluent_provider_integration_v2_authorization.%s", gcpProviderIntegrationV2AuthResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckProviderIntegrationV2AuthDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckProviderIntegrationV2AuthorizationGcpConfig(confluentCloudBaseUrl, mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProviderIntegrationV2AuthExists(fullGcpProviderIntegrationV2AuthResourceLabel),
					resource.TestCheckResourceAttr(fullGcpProviderIntegrationV2AuthResourceLabel, paramId, gcpProviderIntegrationV2AuthId),
					resource.TestCheckResourceAttr(fullGcpProviderIntegrationV2AuthResourceLabel, paramProviderIntegrationIdAuth, gcpProviderIntegrationV2AuthId),
					resource.TestCheckResourceAttr(fullGcpProviderIntegrationV2AuthResourceLabel, fmt.Sprintf("gcp.0.%s", paramGcpCustomerServiceAccount), gcpProviderIntegrationV2CustomerSA),
					resource.TestCheckResourceAttrSet(fullGcpProviderIntegrationV2AuthResourceLabel, fmt.Sprintf("gcp.0.%s", paramGcpGoogleServiceAccount)),
					resource.TestCheckResourceAttr(fullGcpProviderIntegrationV2AuthResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), gcpProviderIntegrationV2AuthEnvironmentId),
				),
			},
			{
				ResourceName:      fullGcpProviderIntegrationV2AuthResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					providerIntegrationId := resources[fullGcpProviderIntegrationV2AuthResourceLabel].Primary.ID
					environmentId := resources[fullGcpProviderIntegrationV2AuthResourceLabel].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + providerIntegrationId, nil
				},
			},
		},
	})
}

const (
	azureProviderIntegrationV2AuthResourceLabel = "test_azure_auth"
	gcpProviderIntegrationV2AuthResourceLabel   = "test_gcp_auth"
)

func testAccCheckProviderIntegrationV2AuthDestroy(s *terraform.State) error {
	// Authorization resource delete only removes from state, doesn't call API
	return nil
}

func testAccCheckProviderIntegrationV2AuthExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("%s provider integration v2 authorization has not been found", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s provider integration v2 authorization", n)
		}
		return nil
	}
}

func testAccCheckProviderIntegrationV2AuthorizationAzureConfig(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_provider_integration_v2_authorization" "%s" {
		provider_integration_id = "%s"
		environment {
			id = "%s"
		}
		azure {
			customer_azure_tenant_id = "%s"
		}
	}
	`, mockServerUrl, azureProviderIntegrationV2AuthResourceLabel, azureProviderIntegrationV2AuthId, azureProviderIntegrationV2AuthEnvironmentId, azureProviderIntegrationV2AuthTenantId)
}

func testAccCheckProviderIntegrationV2AuthorizationGcpConfig(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_provider_integration_v2_authorization" "%s" {
		provider_integration_id = "%s"
		environment {
			id = "%s"
		}
		gcp {
			customer_google_service_account = "%s"
		}
	}
	`, mockServerUrl, gcpProviderIntegrationV2AuthResourceLabel, gcpProviderIntegrationV2AuthId, gcpProviderIntegrationV2AuthEnvironmentId, gcpProviderIntegrationV2CustomerSA)
}