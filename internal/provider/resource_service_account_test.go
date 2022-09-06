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
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	scenarioStateSaHasBeenCreated             = "The new service account has been just created"
	scenarioStateSaDescriptionHaveBeenUpdated = "The new service account's description have been just updated"
	scenarioStateSaHasBeenDeleted             = "The new service account has been deleted"
	saScenarioName                            = "confluent_service_account Resource Lifecycle"
)

func TestAccServiceAccount(t *testing.T) {
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
	createSaResponse, _ := ioutil.ReadFile("../testdata/service_account/create_sa.json")
	createSaStub := wiremock.Post(wiremock.URLPathEqualTo("/iam/v2/service-accounts")).
		InScenario(saScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateSaHasBeenCreated).
		WillReturn(
			string(createSaResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createSaStub)

	readCreatedSaResponse, _ := ioutil.ReadFile("../testdata/service_account/read_created_sa.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/service-accounts/sa-1jjv26")).
		InScenario(saScenarioName).
		WhenScenarioStateIs(scenarioStateSaHasBeenCreated).
		WillReturn(
			string(readCreatedSaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedSaResponse, _ := ioutil.ReadFile("../testdata/service_account/read_updated_sa.json")
	patchSaStub := wiremock.Patch(wiremock.URLPathEqualTo("/iam/v2/service-accounts/sa-1jjv26")).
		InScenario(saScenarioName).
		WhenScenarioStateIs(scenarioStateSaHasBeenCreated).
		WillSetStateTo(scenarioStateSaDescriptionHaveBeenUpdated).
		WillReturn(
			string(readUpdatedSaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(patchSaStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/service-accounts/sa-1jjv26")).
		InScenario(saScenarioName).
		WhenScenarioStateIs(scenarioStateSaDescriptionHaveBeenUpdated).
		WillReturn(
			string(readUpdatedSaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedSaResponse, _ := ioutil.ReadFile("../testdata/service_account/read_deleted_sa.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/service-accounts/sa-1jjv26")).
		InScenario(saScenarioName).
		WhenScenarioStateIs(scenarioStateSaHasBeenDeleted).
		WillReturn(
			string(readDeletedSaResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	deleteSaStub := wiremock.Delete(wiremock.URLPathEqualTo("/iam/v2/service-accounts/sa-1jjv26")).
		InScenario(saScenarioName).
		WhenScenarioStateIs(scenarioStateSaDescriptionHaveBeenUpdated).
		WillSetStateTo(scenarioStateSaHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteSaStub)

	saDisplayName := "test_service_account_display_name"
	saDescription := "The initial description of service account"
	// in order to test tf update (step #3)
	saUpdatedDescription := "The updated description of service account"
	saResourceLabel := "test_sa_resource_label"
	fullSaResourceLabel := fmt.Sprintf("confluent_service_account.%s", saResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckServiceAccountDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckServiceAccountConfig(mockServerUrl, saResourceLabel, saDisplayName, saDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceAccountExists(fullSaResourceLabel),
					resource.TestCheckResourceAttr(fullSaResourceLabel, "id", "sa-1jjv26"),
					resource.TestCheckResourceAttr(fullSaResourceLabel, "api_version", saApiVersion),
					resource.TestCheckResourceAttr(fullSaResourceLabel, "kind", saKind),
					resource.TestCheckResourceAttr(fullSaResourceLabel, "display_name", saDisplayName),
					resource.TestCheckResourceAttr(fullSaResourceLabel, "description", saDescription),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullSaResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCheckServiceAccountConfig(mockServerUrl, saResourceLabel, saDisplayName, saUpdatedDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceAccountExists(fullSaResourceLabel),
					resource.TestCheckResourceAttr(fullSaResourceLabel, "id", "sa-1jjv26"),
					resource.TestCheckResourceAttr(fullSaResourceLabel, "api_version", saApiVersion),
					resource.TestCheckResourceAttr(fullSaResourceLabel, "kind", saKind),
					resource.TestCheckResourceAttr(fullSaResourceLabel, "display_name", saDisplayName),
					resource.TestCheckResourceAttr(fullSaResourceLabel, "description", saUpdatedDescription),
				),
			},
			{
				ResourceName:      fullSaResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, createSaStub, "POST /iam/v2/service-accounts", expectedCountOne)
	checkStubCount(t, wiremockClient, patchSaStub, "PATCH /iam/v2/service-accounts/sa-1jjv26", expectedCountOne)
	checkStubCount(t, wiremockClient, deleteSaStub, "DELETE /iam/v2/service-accounts/sa-1jjv26", expectedCountOne)
}

func testAccCheckServiceAccountDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each service account is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_service_account" {
			continue
		}
		deletedServiceAccountId := rs.Primary.ID
		req := c.iamClient.ServiceAccountsIamV2Api.GetIamV2ServiceAccount(c.iamApiContext(context.Background()), deletedServiceAccountId)
		deletedServiceAccount, response, err := req.Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			// v2/service-accounts/{nonExistentSaId/deletedSaID} returns http.StatusForbidden instead of http.StatusNotFound
			// If the error is equivalent to http.StatusNotFound, the service account is destroyed.
			return nil
		} else if err == nil && deletedServiceAccount.Id != nil {
			// Otherwise return the error
			if *deletedServiceAccount.Id == rs.Primary.ID {
				return fmt.Errorf("service account (%q) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckServiceAccountConfig(mockServerUrl, saResourceLabel, saDisplayName, saDescription string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_service_account" "%s" {
		display_name = "%s"
		description = "%s"
	}
	`, mockServerUrl, saResourceLabel, saDisplayName, saDescription)
}

func testAccCheckServiceAccountExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s service account has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s service account", n)
		}

		return nil
	}
}
