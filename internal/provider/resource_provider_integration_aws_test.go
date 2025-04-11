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

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
)

const (
	providerIntegrationScenarioName                = "confluent_provider_integration Resource Lifecycle"
	scenarioStateProviderIntegrationHasBeenCreated = "The new provider_integration has been just created"
	scenarioStateProviderIntegrationHasBeenDeleted = "The provider_integration has been deleted"
	providerIntegrationId                          = "dlz-f3a90de"
	providerIntegrationDisplayName                 = "s3_provider_integration"
	providerIntegrationEnvironmentId               = "env-00000"
	providerIntegrationIamRoleArn                  = "arn:aws:iam::000000000000:role/my-test-aws-role"
	providerIntegrationExternalId                  = "95c35493-41aa-44f8-9154-5a25cbbc1865"
	providerIntegrationCustomerRoleARN             = "arn:aws:iam::000000000000:role/my-test-aws-role"
	providerIntegrationUsage                       = "crn://confluent.cloud/organization=9bb441c4-edef-46ac-8a41-c49e44a3fd9a/environment=env-456xy/cloud-cluster=lkc-123abc/connector=my_datagen_connector"
)

func TestAccProviderIntegration(t *testing.T) {
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

	// Mock the POST of a provider integration
	createResponse, _ := ioutil.ReadFile("../testdata/provider_integration/create_aws_provider_integration.json")
	createStub := wiremock.Post(wiremock.URLPathEqualTo("/pim/v1/integrations")).
		InScenario(providerIntegrationScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateProviderIntegrationHasBeenCreated).
		WillReturn(
			string(createResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createStub)

	// Mock the GET of a provider integration
	readResponse, _ := ioutil.ReadFile("../testdata/provider_integration/read_created_aws_provider_integration.json")
	readStub := wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/pim/v1/integrations/%s", providerIntegrationId))).
		InScenario(providerIntegrationScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(providerIntegrationEnvironmentId)).
		WhenScenarioStateIs(scenarioStateProviderIntegrationHasBeenCreated).
		WillReturn(
			string(readResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(readStub)

	// Mock the DELETE of a provider integration
	deleteStub := wiremock.Delete(wiremock.URLPathEqualTo(fmt.Sprintf("/pim/v1/integrations/%s", providerIntegrationId))).
		InScenario(providerIntegrationScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(providerIntegrationEnvironmentId)).
		WhenScenarioStateIs(scenarioStateProviderIntegrationHasBeenCreated).
		WillSetStateTo(scenarioStateProviderIntegrationHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteStub)

	// Mock the GET of a deleted provider integration during terraform destroy
	readDeletedResponse, _ := ioutil.ReadFile("../testdata/provider_integration/read_deleted_aws_provider_integration.json")
	readDeletedStub := wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/pim/v1/integrations/%s", providerIntegrationId))).
		InScenario(providerIntegrationScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(providerIntegrationEnvironmentId)).
		WhenScenarioStateIs(scenarioStateProviderIntegrationHasBeenDeleted).
		WillReturn(
			string(readDeletedResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		)
	_ = wiremockClient.StubFor(readDeletedStub)

	resourceLabel := "test"
	fullLabel := fmt.Sprintf("confluent_provider_integration.%s", resourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckProviderIntegrationConfig(mockServerUrl, resourceLabel),
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
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					pimId := resources[fullLabel].Primary.ID
					environmentId := resources[fullLabel].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + pimId, nil
				},
			},
		},
		CheckDestroy: testAccCheckProviderIntegrationDestroy,
	})
}

func testAccCheckProviderIntegrationDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each provider integration is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_provider_integration" {
			continue
		}
		deletedProviderIntegrationId := rs.Primary.ID
		req := c.piClient.IntegrationsPimV1Api.GetPimV1Integration(c.netApiContext(context.Background()), deletedProviderIntegrationId).Environment(providerIntegrationEnvironmentId)
		deletedProviderIntegration, response, err := req.Execute()
		if response != nil && response.StatusCode == http.StatusNotFound {
			return nil
		} else if err == nil && deletedProviderIntegration.Id != nil {
			// Otherwise return the error
			if *deletedProviderIntegration.Id == rs.Primary.ID {
				return fmt.Errorf("provider integration (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckProviderIntegrationConfig(mockServerUrl, resourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_provider_integration" "%s" {
		environment {
    		id = "%s"
  		}
		display_name = "%s"
		aws {
    		customer_role_arn = "%s"
  		}
	}
	`, mockServerUrl, resourceLabel, providerIntegrationEnvironmentId, providerIntegrationDisplayName, providerIntegrationCustomerRoleARN)
}

func testAccCheckProviderIntegrationExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s provider integration has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s provider integration", n)
		}

		return nil
	}
}
