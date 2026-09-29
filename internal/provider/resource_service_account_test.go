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
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
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
	// Declared before the stubs because the create stub matches on it. The spec's own example.
	saAssignedResourceOwner := "u-a83k9b"

	createSaResponse, _ := ioutil.ReadFile("../testdata/service_account/create_sa.json")
	// The query-param matcher is the assertion: assigned_resource_owner is never returned by the
	// API, so without matching on it here the test would still pass if the provider accepted the
	// attribute and then dropped it from the create request.
	createSaStub := wiremock.Post(wiremock.URLPathEqualTo("/iam/v2/service-accounts")).
		WithQueryParam(paramAssignedResourceOwner, wiremock.EqualTo(saAssignedResourceOwner)).
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
	saUpdatedDisplayName := "test_service_account_updated_display_name"
	saUpdatedDescription := "The updated description of service account"
	saResourceLabel := "test_sa_resource_label"
	fullSaResourceLabel := fmt.Sprintf("confluent_service_account.%s", saResourceLabel)

	// The import steps below read this. assigned_resource_owner is create-only and never returned,
	// so serviceAccountImport seeds it from this variable; without it the imported state would hold
	// "" against a configured "u-a83k9b" and ImportStateVerify would fail.
	_ = os.Setenv("IMPORT_ASSIGNED_RESOURCE_OWNER", saAssignedResourceOwner)
	defer func() {
		_ = os.Unsetenv("IMPORT_ASSIGNED_RESOURCE_OWNER")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckServiceAccountDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckServiceAccountConfig(mockServerUrl, saResourceLabel, saDisplayName, saDescription, saAssignedResourceOwner),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceAccountExists(fullSaResourceLabel),
					resource.TestCheckResourceAttr(fullSaResourceLabel, "id", "sa-1jjv26"),
					resource.TestCheckResourceAttr(fullSaResourceLabel, "api_version", saApiVersion),
					resource.TestCheckResourceAttr(fullSaResourceLabel, "kind", saKind),
					resource.TestCheckResourceAttr(fullSaResourceLabel, "display_name", saDisplayName),
					resource.TestCheckResourceAttr(fullSaResourceLabel, "description", saDescription),
					resource.TestCheckResourceAttr(fullSaResourceLabel, "assigned_resource_owner", saAssignedResourceOwner),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullSaResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCheckServiceAccountConfig(mockServerUrl, saResourceLabel, saUpdatedDisplayName, saUpdatedDescription, saAssignedResourceOwner),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceAccountExists(fullSaResourceLabel),
					resource.TestCheckResourceAttr(fullSaResourceLabel, "id", "sa-1jjv26"),
					resource.TestCheckResourceAttr(fullSaResourceLabel, "api_version", saApiVersion),
					resource.TestCheckResourceAttr(fullSaResourceLabel, "kind", saKind),
					resource.TestCheckResourceAttr(fullSaResourceLabel, "display_name", saUpdatedDisplayName),
					resource.TestCheckResourceAttr(fullSaResourceLabel, "description", saUpdatedDescription),
					resource.TestCheckResourceAttr(fullSaResourceLabel, "assigned_resource_owner", saAssignedResourceOwner),
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
		req := c.iamV2Client.ServiceAccountsIamV2Api.GetIamV2ServiceAccount(c.iamV2ApiContext(context.Background()), deletedServiceAccountId)
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

func testAccCheckServiceAccountConfig(mockServerUrl, saResourceLabel, saDisplayName, saDescription, saAssignedResourceOwner string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_service_account" "%s" {
		display_name = "%s"
		description = "%s"
		assigned_resource_owner = %q
	}
	`, mockServerUrl, saResourceLabel, saDisplayName, saDescription, saAssignedResourceOwner)
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
