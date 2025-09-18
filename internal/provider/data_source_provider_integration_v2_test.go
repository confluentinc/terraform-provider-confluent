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
	dataSourceProviderIntegrationV2ScenarioName = "confluent_provider_integration_v2 Data Source Lifecycle"
)

func TestAccDataSourceProviderIntegrationV2Azure(t *testing.T) {
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

	// Mock the GET of an Azure provider integration v2
	readResponse, _ := ioutil.ReadFile("../testdata/provider_integration_v2/read_created_azure_provider_integration_v2.json")
	readStub := wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/pim/v2/integrations/%s", azureProviderIntegrationV2Id))).
		InScenario(dataSourceProviderIntegrationV2ScenarioName).
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
				Config: testAccCheckDataSourceProviderIntegrationV2AzureConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.confluent_provider_integration_v2.test_azure", paramId, azureProviderIntegrationV2Id),
					resource.TestCheckResourceAttr("data.confluent_provider_integration_v2.test_azure", paramDisplayName, azureProviderIntegrationV2DisplayName),
					resource.TestCheckResourceAttr("data.confluent_provider_integration_v2.test_azure", paramCloudProvider, "azure"),
					resource.TestCheckResourceAttr("data.confluent_provider_integration_v2.test_azure", paramStatus, "CREATED"),
					resource.TestCheckResourceAttr("data.confluent_provider_integration_v2.test_azure", fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), azureProviderIntegrationV2EnvironmentId),
				),
			},
		},
	})
}

func TestAccDataSourceProviderIntegrationV2Gcp(t *testing.T) {
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

	// Mock the GET of a GCP provider integration v2
	readResponse, _ := ioutil.ReadFile("../testdata/provider_integration_v2/read_created_gcp_provider_integration_v2.json")
	readStub := wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/pim/v2/integrations/%s", gcpProviderIntegrationV2Id))).
		InScenario(dataSourceProviderIntegrationV2ScenarioName).
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
				Config: testAccCheckDataSourceProviderIntegrationV2GcpConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.confluent_provider_integration_v2.test_gcp", paramId, gcpProviderIntegrationV2Id),
					resource.TestCheckResourceAttr("data.confluent_provider_integration_v2.test_gcp", paramDisplayName, gcpProviderIntegrationV2DisplayName),
					resource.TestCheckResourceAttr("data.confluent_provider_integration_v2.test_gcp", paramCloudProvider, "gcp"),
					resource.TestCheckResourceAttr("data.confluent_provider_integration_v2.test_gcp", paramStatus, "CREATED"),
					resource.TestCheckResourceAttr("data.confluent_provider_integration_v2.test_gcp", fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), gcpProviderIntegrationV2EnvironmentId),
				),
			},
		},
	})
}

func testAccCheckDataSourceProviderIntegrationV2AzureConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_provider_integration_v2" "test_azure" {
		id = "%s"
		environment {
			id = "%s"
		}
	}
	`, mockServerUrl, azureProviderIntegrationV2Id, azureProviderIntegrationV2EnvironmentId)
}

func testAccCheckDataSourceProviderIntegrationV2GcpConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_provider_integration_v2" "test_gcp" {
		id = "%s"
		environment {
			id = "%s"
		}
	}
	`, mockServerUrl, gcpProviderIntegrationV2Id, gcpProviderIntegrationV2EnvironmentId)
}