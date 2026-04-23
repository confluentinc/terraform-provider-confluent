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

func TestAccProviderIntegrationSetupAzure(t *testing.T) {
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

	createAzureProviderIntegrationV2Response, _ := ioutil.ReadFile("../testdata/provider_integration_setup/create_azure_provider_integration_setup.json")
	createAzureProviderIntegrationV2Stub := wiremock.Post(wiremock.URLPathEqualTo("/pim/v2/integrations")).
		InScenario(providerIntegrationSetupScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateProviderIntegrationV2HasBeenCreated).
		WillReturn(
			string(createAzureProviderIntegrationV2Response),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	if err := wiremockClient.StubFor(createAzureProviderIntegrationV2Stub); err != nil {
		t.Errorf("StubFor failed: %v", err)
	}

	readCreatedAzureProviderIntegrationV2Response, _ := ioutil.ReadFile("../testdata/provider_integration_setup/create_azure_provider_integration_setup.json")
	if err := wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/pim/v2/integrations/%s", azureProviderIntegrationV2Id))).
		InScenario(providerIntegrationSetupScenarioName).
		WhenScenarioStateIs(scenarioStateProviderIntegrationV2HasBeenCreated).
		WillReturn(
			string(readCreatedAzureProviderIntegrationV2Response),
			contentTypeJSONHeader,
			http.StatusOK,
		)); err != nil {
		t.Errorf("StubFor failed: %v", err)
	}

	deleteAzureProviderIntegrationV2Stub := wiremock.Delete(wiremock.URLPathEqualTo(fmt.Sprintf("/pim/v2/integrations/%s", azureProviderIntegrationV2Id))).
		InScenario(providerIntegrationSetupScenarioName).
		WhenScenarioStateIs(scenarioStateProviderIntegrationV2HasBeenCreated).
		WillSetStateTo(scenarioStateProviderIntegrationV2HasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	if err := wiremockClient.StubFor(deleteAzureProviderIntegrationV2Stub); err != nil {
		t.Errorf("StubFor failed: %v", err)
	}

	readDeletedAzureProviderIntegrationV2Response, _ := ioutil.ReadFile("../testdata/provider_integration/read_deleted_aws_provider_integration.json")
	if err := wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/pim/v2/integrations/%s", azureProviderIntegrationV2Id))).
		InScenario(providerIntegrationSetupScenarioName).
		WhenScenarioStateIs(scenarioStateProviderIntegrationV2HasBeenDeleted).
		WillReturn(
			string(readDeletedAzureProviderIntegrationV2Response),
			contentTypeJSONHeader,
			http.StatusNotFound,
		)); err != nil {
		t.Errorf("StubFor failed: %v", err)
	}

	fullAzureProviderIntegrationV2ResourceLabel := fmt.Sprintf("confluent_provider_integration_setup.%s", azureProviderIntegrationV2ResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckProviderIntegrationSetupMockDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckProviderIntegrationSetupAzureConfig(confluentCloudBaseUrl, mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProviderIntegrationSetupMockExists(fullAzureProviderIntegrationV2ResourceLabel),
					resource.TestCheckResourceAttr(fullAzureProviderIntegrationV2ResourceLabel, paramId, azureProviderIntegrationV2Id),
					resource.TestCheckResourceAttr(fullAzureProviderIntegrationV2ResourceLabel, paramDisplayName, azureProviderIntegrationV2DisplayName),
					resource.TestCheckResourceAttr(fullAzureProviderIntegrationV2ResourceLabel, paramCloud, "AZURE"),
					resource.TestCheckResourceAttr(fullAzureProviderIntegrationV2ResourceLabel, paramStatus, "DRAFT"),
					resource.TestCheckResourceAttr(fullAzureProviderIntegrationV2ResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), azureProviderIntegrationV2EnvironmentId),
				),
			},
			{
				ResourceName:      fullAzureProviderIntegrationV2ResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					providerIntegrationId := resources[fullAzureProviderIntegrationV2ResourceLabel].Primary.ID
					environmentId := resources[fullAzureProviderIntegrationV2ResourceLabel].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + providerIntegrationId, nil
				},
			},
		},
	})
}

func TestAccProviderIntegrationSetupGcp(t *testing.T) {
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

	createGcpProviderIntegrationV2Response, _ := ioutil.ReadFile("../testdata/provider_integration_setup/create_gcp_provider_integration_setup.json")
	createGcpProviderIntegrationV2Stub := wiremock.Post(wiremock.URLPathEqualTo("/pim/v2/integrations")).
		InScenario(providerIntegrationSetupScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateProviderIntegrationV2HasBeenCreated).
		WillReturn(
			string(createGcpProviderIntegrationV2Response),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	if err := wiremockClient.StubFor(createGcpProviderIntegrationV2Stub); err != nil {
		t.Errorf("StubFor failed: %v", err)
	}

	readCreatedGcpProviderIntegrationV2Response, _ := ioutil.ReadFile("../testdata/provider_integration_setup/create_gcp_provider_integration_setup.json")
	if err := wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/pim/v2/integrations/%s", gcpProviderIntegrationV2Id))).
		InScenario(providerIntegrationSetupScenarioName).
		WhenScenarioStateIs(scenarioStateProviderIntegrationV2HasBeenCreated).
		WillReturn(
			string(readCreatedGcpProviderIntegrationV2Response),
			contentTypeJSONHeader,
			http.StatusOK,
		)); err != nil {
		t.Errorf("StubFor failed: %v", err)
	}

	deleteGcpProviderIntegrationV2Stub := wiremock.Delete(wiremock.URLPathEqualTo(fmt.Sprintf("/pim/v2/integrations/%s", gcpProviderIntegrationV2Id))).
		InScenario(providerIntegrationSetupScenarioName).
		WhenScenarioStateIs(scenarioStateProviderIntegrationV2HasBeenCreated).
		WillSetStateTo(scenarioStateProviderIntegrationV2HasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	if err := wiremockClient.StubFor(deleteGcpProviderIntegrationV2Stub); err != nil {
		t.Errorf("StubFor failed: %v", err)
	}

	readDeletedGcpProviderIntegrationV2Response, _ := ioutil.ReadFile("../testdata/provider_integration/read_deleted_aws_provider_integration.json")
	if err := wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/pim/v2/integrations/%s", gcpProviderIntegrationV2Id))).
		InScenario(providerIntegrationSetupScenarioName).
		WhenScenarioStateIs(scenarioStateProviderIntegrationV2HasBeenDeleted).
		WillReturn(
			string(readDeletedGcpProviderIntegrationV2Response),
			contentTypeJSONHeader,
			http.StatusNotFound,
		)); err != nil {
		t.Errorf("StubFor failed: %v", err)
	}

	fullGcpProviderIntegrationV2ResourceLabel := fmt.Sprintf("confluent_provider_integration_setup.%s", gcpProviderIntegrationV2ResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckProviderIntegrationSetupMockDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckProviderIntegrationSetupGcpConfig(confluentCloudBaseUrl, mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProviderIntegrationSetupMockExists(fullGcpProviderIntegrationV2ResourceLabel),
					resource.TestCheckResourceAttr(fullGcpProviderIntegrationV2ResourceLabel, paramId, gcpProviderIntegrationV2Id),
					resource.TestCheckResourceAttr(fullGcpProviderIntegrationV2ResourceLabel, paramDisplayName, gcpProviderIntegrationV2DisplayName),
					resource.TestCheckResourceAttr(fullGcpProviderIntegrationV2ResourceLabel, paramCloud, "GCP"),
					resource.TestCheckResourceAttr(fullGcpProviderIntegrationV2ResourceLabel, paramStatus, "DRAFT"),
					resource.TestCheckResourceAttr(fullGcpProviderIntegrationV2ResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), gcpProviderIntegrationV2EnvironmentId),
				),
			},
			{
				ResourceName:      fullGcpProviderIntegrationV2ResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					providerIntegrationId := resources[fullGcpProviderIntegrationV2ResourceLabel].Primary.ID
					environmentId := resources[fullGcpProviderIntegrationV2ResourceLabel].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + providerIntegrationId, nil
				},
			},
		},
	})
}

func testAccCheckProviderIntegrationSetupMockDestroy(s *terraform.State) error {
	// This is handled by wiremock scenarios
	return nil
}

func testAccCheckProviderIntegrationSetupMockExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("%s provider integration v2 has not been found", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s provider integration v2", n)
		}
		return nil
	}
}

func testAccCheckProviderIntegrationSetupAzureConfig(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_provider_integration_setup" "%s" {
		environment {
			id = "%s"
		}
		display_name   = "%s"
		cloud = "AZURE"
	}
	`, mockServerUrl, azureProviderIntegrationV2ResourceLabel, azureProviderIntegrationV2EnvironmentId, azureProviderIntegrationV2DisplayName)
}

func testAccCheckProviderIntegrationSetupGcpConfig(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_provider_integration_setup" "%s" {
		environment {
			id = "%s"
		}
		display_name   = "%s"
		cloud = "GCP"
	}
	`, mockServerUrl, gcpProviderIntegrationV2ResourceLabel, gcpProviderIntegrationV2EnvironmentId, gcpProviderIntegrationV2DisplayName)
}
