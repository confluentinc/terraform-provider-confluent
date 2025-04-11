// Copyright 2022 Confluent Inc. All Rights Reserved.
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
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

const (
	dataSourceProviderIntegrationScenarioName = "confluent_provider_integration Data Source Lifecycle"
)

func TestAccDataSourceProviderIntegration(t *testing.T) {
	ctx := context.Background()
	time.Sleep(5 * time.Second)
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

	// Mock the GET of a provider integration
	readResponse, _ := ioutil.ReadFile("../testdata/provider_integration/read_created_aws_provider_integration.json")
	readStub := wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/pim/v1/integrations/%s", providerIntegrationId))).
		InScenario(dataSourceProviderIntegrationScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(providerIntegrationEnvironmentId)).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(readStub)

	// Mock the LIST of the provider integrations
	listResponse, _ := ioutil.ReadFile("../testdata/provider_integration/read_aws_provider_integrations.json")
	listStub := wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/pim/v1/integrations"))).
		InScenario(dataSourceProviderIntegrationScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(providerIntegrationEnvironmentId)).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(listResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(listStub)

	dataSourceLabel := "test"
	fullLabel := fmt.Sprintf("data.confluent_provider_integration.%s", dataSourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceProviderIntegrationWithIdSet(mockServerUrl, dataSourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProviderIntegrationExists(fullLabel),
					resource.TestCheckResourceAttr(fullLabel, paramId, providerIntegrationId),
					resource.TestCheckResourceAttr(fullLabel, paramDisplayName, providerIntegrationDisplayName),
					resource.TestCheckResourceAttr(fullLabel, "aws.#", "1"),
					resource.TestCheckResourceAttr(fullLabel, "aws.0.customer_role_arn", providerIntegrationCustomerRoleARN),
					resource.TestCheckResourceAttr(fullLabel, "aws.0.iam_role_arn", providerIntegrationIamRoleArn),
					resource.TestCheckResourceAttr(fullLabel, "aws.0.external_id", providerIntegrationExternalId),
					resource.TestCheckResourceAttr(fullLabel, "usages.#", "1"),
					resource.TestCheckResourceAttr(fullLabel, "usages.0", providerIntegrationUsage),
					resource.TestCheckResourceAttr(fullLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), providerIntegrationEnvironmentId),
				),
			},
			{
				Config: testAccCheckDataSourceProviderIntegrationWithDisplayNameSet(mockServerUrl, dataSourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProviderIntegrationExists(fullLabel),
					resource.TestCheckResourceAttr(fullLabel, paramId, providerIntegrationId),
					resource.TestCheckResourceAttr(fullLabel, paramDisplayName, providerIntegrationDisplayName),
					resource.TestCheckResourceAttr(fullLabel, "aws.#", "1"),
					resource.TestCheckResourceAttr(fullLabel, "aws.0.customer_role_arn", providerIntegrationCustomerRoleARN),
					resource.TestCheckResourceAttr(fullLabel, "aws.0.iam_role_arn", providerIntegrationIamRoleArn),
					resource.TestCheckResourceAttr(fullLabel, "aws.0.external_id", providerIntegrationExternalId),
					resource.TestCheckResourceAttr(fullLabel, "usages.#", "1"),
					resource.TestCheckResourceAttr(fullLabel, "usages.0", providerIntegrationUsage),
					resource.TestCheckResourceAttr(fullLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), providerIntegrationEnvironmentId),
				),
			},
		},
	})
}

func testAccCheckDataSourceProviderIntegrationWithIdSet(mockServerUrl, dataSourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_provider_integration" "%s" {
		environment {
    		id = "%s"
  		}
		id = "%s"
	}
	`, mockServerUrl, dataSourceLabel, providerIntegrationEnvironmentId, providerIntegrationId)
}

func testAccCheckDataSourceProviderIntegrationWithDisplayNameSet(mockServerUrl, dataSourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_provider_integration" "%s" {
		environment {
    		id = "%s"
  		}
		display_name = "%s"
	}
	`, mockServerUrl, dataSourceLabel, providerIntegrationEnvironmentId, providerIntegrationDisplayName)
}
