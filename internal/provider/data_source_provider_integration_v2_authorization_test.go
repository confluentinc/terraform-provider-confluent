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
	"github.com/walkerus/go-wiremock"
)

const (
	dataSourceProviderIntegrationV2AuthScenarioName = "confluent_provider_integration_v2_authorization Data Source Lifecycle"
)

func TestAccDataSourceProviderIntegrationV2AuthorizationAzure(t *testing.T) {
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

	// Mock the GET of an Azure provider integration v2 authorization
	readResponse, _ := ioutil.ReadFile("../testdata/provider_integration_v2_authorization/update_azure_provider_integration_v2_authorization.json")
	readStub := wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/pim/v2/integrations/%s", azureProviderIntegrationV2AuthId))).
		InScenario(dataSourceProviderIntegrationV2AuthScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(readStub)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceProviderIntegrationV2AuthorizationAzureConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.confluent_provider_integration_v2_authorization.test_azure_auth", paramId, azureProviderIntegrationV2AuthId),
					resource.TestCheckResourceAttr("data.confluent_provider_integration_v2_authorization.test_azure_auth", paramProviderIntegrationIdAuth, azureProviderIntegrationV2AuthId),
					resource.TestCheckResourceAttr("data.confluent_provider_integration_v2_authorization.test_azure_auth", fmt.Sprintf("azure.0.%s", paramAzureCustomerTenantId), azureProviderIntegrationV2AuthTenantId),
					resource.TestCheckResourceAttrSet("data.confluent_provider_integration_v2_authorization.test_azure_auth", fmt.Sprintf("azure.0.%s", paramAzureConfluentMultiTenantAppId)),
					resource.TestCheckResourceAttr("data.confluent_provider_integration_v2_authorization.test_azure_auth", fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), azureProviderIntegrationV2AuthEnvironmentId),
				),
			},
		},
	})
}

func TestAccDataSourceProviderIntegrationV2AuthorizationGcp(t *testing.T) {
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

	// Mock the GET of a GCP provider integration v2 authorization
	readResponse, _ := ioutil.ReadFile("../testdata/provider_integration_v2_authorization/update_gcp_provider_integration_v2_authorization.json")
	readStub := wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/pim/v2/integrations/%s", gcpProviderIntegrationV2AuthId))).
		InScenario(dataSourceProviderIntegrationV2AuthScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(readStub)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceProviderIntegrationV2AuthorizationGcpConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.confluent_provider_integration_v2_authorization.test_gcp_auth", paramId, gcpProviderIntegrationV2AuthId),
					resource.TestCheckResourceAttr("data.confluent_provider_integration_v2_authorization.test_gcp_auth", paramProviderIntegrationIdAuth, gcpProviderIntegrationV2AuthId),
					resource.TestCheckResourceAttr("data.confluent_provider_integration_v2_authorization.test_gcp_auth", fmt.Sprintf("gcp.0.%s", paramGcpCustomerServiceAccount), gcpProviderIntegrationV2CustomerSA),
					resource.TestCheckResourceAttrSet("data.confluent_provider_integration_v2_authorization.test_gcp_auth", fmt.Sprintf("gcp.0.%s", paramGcpGoogleServiceAccount)),
					resource.TestCheckResourceAttr("data.confluent_provider_integration_v2_authorization.test_gcp_auth", fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), gcpProviderIntegrationV2AuthEnvironmentId),
				),
			},
		},
	})
}

func testAccCheckDataSourceProviderIntegrationV2AuthorizationAzureConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_provider_integration_v2_authorization" "test_azure_auth" {
		id = "%s"
		environment {
			id = "%s"
		}
	}
	`, mockServerUrl, azureProviderIntegrationV2AuthId, azureProviderIntegrationV2AuthEnvironmentId)
}

func testAccCheckDataSourceProviderIntegrationV2AuthorizationGcpConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_provider_integration_v2_authorization" "test_gcp_auth" {
		id = "%s"
		environment {
			id = "%s"
		}
	}
	`, mockServerUrl, gcpProviderIntegrationV2AuthId, gcpProviderIntegrationV2AuthEnvironmentId)
}