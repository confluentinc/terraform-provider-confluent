// Copyright 2026 Confluent Inc. All Rights Reserved.
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
	clientRequestPolicyResourceScenarioName        = "confluent_client_request_policy Resource Lifecycle"
	scenarioStateClientRequestPolicyHasBeenCreated = "The new client request policy has been just created"
	scenarioStateClientRequestPolicyHasBeenUpdated = "The client request policy has been updated"
	scenarioStateClientRequestPolicyHasBeenDeleted = "The client request policy has been deleted"

	clientRequestPolicyResourceLabel = "test_client_request_policy_resource_label"
	testClientRequestPolicyName      = "restrict-client-versions"
	testClientRequestPolicyType      = "VersionPolicy"
	testClientRequestPolicyScopeCRN  = "crn://confluent.cloud/organization=abc/environment=env-123/cloud-cluster=lkc-abc123"
	testClientRequestPolicyRuleName  = "require-recent-java-client"
)

func TestAccClientRequestPolicy(t *testing.T) {
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

	createResponse, _ := ioutil.ReadFile("../testdata/client_request_policy/create_client_request_policy.json")
	createStub := wiremock.Post(wiremock.URLPathEqualTo("/configurationcontrol/v1/policies")).
		InScenario(clientRequestPolicyResourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateClientRequestPolicyHasBeenCreated).
		WillReturn(
			string(createResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createStub)

	readCreatedResponse, _ := ioutil.ReadFile("../testdata/client_request_policy/read_created_client_request_policy.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/configurationcontrol/v1/policies/%s", testClientRequestPolicyName))).
		InScenario(clientRequestPolicyResourceScenarioName).
		WhenScenarioStateIs(scenarioStateClientRequestPolicyHasBeenCreated).
		WillReturn(
			string(readCreatedResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedResponse, _ := ioutil.ReadFile("../testdata/client_request_policy/read_updated_client_request_policy.json")
	patchStub := wiremock.Patch(wiremock.URLPathEqualTo(fmt.Sprintf("/configurationcontrol/v1/policies/%s", testClientRequestPolicyName))).
		InScenario(clientRequestPolicyResourceScenarioName).
		WhenScenarioStateIs(scenarioStateClientRequestPolicyHasBeenCreated).
		WillSetStateTo(scenarioStateClientRequestPolicyHasBeenUpdated).
		WillReturn(
			string(readUpdatedResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(patchStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/configurationcontrol/v1/policies/%s", testClientRequestPolicyName))).
		InScenario(clientRequestPolicyResourceScenarioName).
		WhenScenarioStateIs(scenarioStateClientRequestPolicyHasBeenUpdated).
		WillReturn(
			string(readUpdatedResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedResponse, _ := ioutil.ReadFile("../testdata/client_request_policy/read_deleted_client_request_policy.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/configurationcontrol/v1/policies/%s", testClientRequestPolicyName))).
		InScenario(clientRequestPolicyResourceScenarioName).
		WhenScenarioStateIs(scenarioStateClientRequestPolicyHasBeenDeleted).
		WillReturn(
			string(readDeletedResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	deleteStub := wiremock.Delete(wiremock.URLPathEqualTo(fmt.Sprintf("/configurationcontrol/v1/policies/%s", testClientRequestPolicyName))).
		InScenario(clientRequestPolicyResourceScenarioName).
		WhenScenarioStateIs(scenarioStateClientRequestPolicyHasBeenUpdated).
		WillSetStateTo(scenarioStateClientRequestPolicyHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteStub)

	fullResourceLabel := fmt.Sprintf("confluent_client_request_policy.%s", clientRequestPolicyResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckClientRequestPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckClientRequestPolicyConfig(mockServerUrl, clientRequestPolicyResourceLabel, "AUDIT", "request.client.version >= '3.5.0'"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClientRequestPolicyExists(fullResourceLabel),
					resource.TestCheckResourceAttr(fullResourceLabel, paramId, testClientRequestPolicyName),
					resource.TestCheckResourceAttr(fullResourceLabel, paramName, testClientRequestPolicyName),
					resource.TestCheckResourceAttr(fullResourceLabel, paramPolicyType, testClientRequestPolicyType),
					resource.TestCheckResourceAttr(fullResourceLabel, paramResourceName, testClientRequestPolicyScopeCRN),
					resource.TestCheckResourceAttr(fullResourceLabel, paramMode, "AUDIT"),
					resource.TestCheckResourceAttr(fullResourceLabel, paramAction, "DENY"),
					resource.TestCheckResourceAttr(fullResourceLabel, fmt.Sprintf("%s.#", paramRules), "1"),
					resource.TestCheckResourceAttr(fullResourceLabel, fmt.Sprintf("%s.0.%s", paramRules, paramName), testClientRequestPolicyRuleName),
					resource.TestCheckResourceAttr(fullResourceLabel, fmt.Sprintf("%s.0.%s", paramRules, paramMatch), "request.client.version >= '3.5.0'"),
					resource.TestCheckResourceAttrSet(fullResourceLabel, paramUid),
					resource.TestCheckResourceAttr(fullResourceLabel, paramPhase, "PROVISIONED"),
				),
			},
			{
				ResourceName:      fullResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCheckClientRequestPolicyConfig(mockServerUrl, clientRequestPolicyResourceLabel, "ACTIVE", "request.client.version >= '3.6.0'"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClientRequestPolicyExists(fullResourceLabel),
					resource.TestCheckResourceAttr(fullResourceLabel, paramId, testClientRequestPolicyName),
					resource.TestCheckResourceAttr(fullResourceLabel, paramMode, "ACTIVE"),
					resource.TestCheckResourceAttr(fullResourceLabel, fmt.Sprintf("%s.0.%s", paramRules, paramMatch), "request.client.version >= '3.6.0'"),
					resource.TestCheckResourceAttr(fullResourceLabel, paramResourceVersion, "v2"),
				),
			},
			{
				ResourceName:      fullResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, createStub, "POST /configurationcontrol/v1/policies", expectedCountOne)
	checkStubCount(t, wiremockClient, patchStub, fmt.Sprintf("PATCH /configurationcontrol/v1/policies/%s", testClientRequestPolicyName), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteStub, fmt.Sprintf("DELETE /configurationcontrol/v1/policies/%s", testClientRequestPolicyName), expectedCountOne)
}

func testAccCheckClientRequestPolicyDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_client_request_policy" {
			continue
		}
		deletedPolicyName := rs.Primary.ID
		_, response, err := c.configurationControlV1Client.PoliciesConfigurationcontrolV1Api.GetConfigurationcontrolV1Policy(c.configurationControlV1ApiContext(context.Background()), deletedPolicyName).Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			return nil
		} else if err == nil {
			return fmt.Errorf("client request policy (%q) still exists", rs.Primary.ID)
		}
		return err
	}
	return nil
}

func testAccCheckClientRequestPolicyConfig(mockServerUrl, resourceLabel, mode, match string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_client_request_policy" "%s" {
		name          = "%s"
		policy_type   = "%s"
		resource_name = "%s"
		mode          = "%s"
		action        = "DENY"
		rules {
			name  = "%s"
			match = "%s"
		}
	}
	`, mockServerUrl, resourceLabel, testClientRequestPolicyName, testClientRequestPolicyType, testClientRequestPolicyScopeCRN, mode, testClientRequestPolicyRuleName, match)
}

func testAccCheckClientRequestPolicyExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("%s client request policy has not been found", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s client request policy", n)
		}
		return nil
	}
}
