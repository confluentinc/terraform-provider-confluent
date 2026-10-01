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
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

// TestAccServiceAccountAssignedResourceOwner covers assigned_resource_owner, which the API accepts
// only as a query parameter on create and never returns on read.
//
// Three things are asserted, each of which would otherwise fail silently:
//
//   - The create request actually carries ?assigned_resource_owner=<principal>. The create stub
//     matches on that query parameter, so a provider that accepts the attribute and then drops it
//     gets no stub match, a 404, and a failed apply — rather than a green test.
//   - The configured value survives in state even though no read path sets it. The read fixture
//     does not carry the field (the API does not return it), and the framework fails a step whose
//     post-apply plan is non-empty, so this step passing is what proves the attribute does not
//     drift on refresh.
//   - Import seeds the attribute from IMPORT_ASSIGNED_RESOURCE_OWNER. Without that env var an
//     import leaves it empty, and because the attribute is ForceNew the first post-import plan
//     would want to *replace* the service account — from a `terraform import`, which is meant to
//     be read-only. The import step therefore runs with ImportStateVerify and no
//     ImportStateVerifyIgnore: the verification passing is what proves the env var closed the gap,
//     exactly as resource_identity_pool_assigned_resource_owner_test.go does for identity_pool.
//
// Kept separate from TestAccServiceAccount (which never configures this attribute) so that test
// keeps covering the default, omitted-attribute path a WireMock-level test is uniquely placed to
// verify: that the create request carries no query parameter at all when unset.
func TestAccServiceAccountAssignedResourceOwner(t *testing.T) {
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

	const assignedResourceOwnerScenario = "confluent_service_account assigned_resource_owner"
	const stateCreated = "assigned-resource-owner-created"
	const stateDeleted = "assigned-resource-owner-deleted"
	// The spec's own example for the parameter.
	const testAssignedResourceOwner = "u-a83k9b"

	const createUrlPath = "/iam/v2/service-accounts"
	const itemUrlPath = "/iam/v2/service-accounts/sa-1jjv26"

	// The query-param matcher is the assertion: without it on the request, this stub does not
	// match and the apply fails.
	createSaResponse, _ := ioutil.ReadFile("../testdata/service_account/create_sa.json")
	createStub := wiremock.Post(wiremock.URLPathEqualTo(createUrlPath)).
		WithQueryParam(paramAssignedResourceOwner, wiremock.EqualTo(testAssignedResourceOwner)).
		InScenario(assignedResourceOwnerScenario).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(stateCreated).
		WillReturn(string(createSaResponse), contentTypeJSONHeader, http.StatusCreated)
	_ = wiremockClient.StubFor(createStub)

	// The read response deliberately does not carry assigned_resource_owner, matching the real
	// API. The value in state comes from configuration, not from this payload.
	readCreatedSaResponse, _ := ioutil.ReadFile("../testdata/service_account/read_created_sa.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(itemUrlPath)).
		InScenario(assignedResourceOwnerScenario).
		WhenScenarioStateIs(stateCreated).
		WillReturn(string(readCreatedSaResponse), contentTypeJSONHeader, http.StatusOK))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(itemUrlPath)).
		InScenario(assignedResourceOwnerScenario).
		WhenScenarioStateIs(stateCreated).
		WillSetStateTo(stateDeleted).
		WillReturn("", contentTypeJSONHeader, http.StatusNoContent))

	readDeletedSaResponse, _ := ioutil.ReadFile("../testdata/service_account/read_deleted_sa.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(itemUrlPath)).
		InScenario(assignedResourceOwnerScenario).
		WhenScenarioStateIs(stateDeleted).
		WillReturn(string(readDeletedSaResponse), contentTypeJSONHeader, http.StatusNotFound))

	saResourceLabel := "test_sa_assigned_resource_owner"
	fullSaResourceLabel := fmt.Sprintf("confluent_service_account.%s", saResourceLabel)

	// The import step below reads this: the API does not return the value, so without it the
	// imported state would hold "" and ImportStateVerify would fail.
	_ = os.Setenv("IMPORT_ASSIGNED_RESOURCE_OWNER", testAssignedResourceOwner)
	defer func() {
		_ = os.Unsetenv("IMPORT_ASSIGNED_RESOURCE_OWNER")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckServiceAccountDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckServiceAccountAssignedResourceOwnerConfig(mockServerUrl, saResourceLabel, testAssignedResourceOwner),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceAccountExists(fullSaResourceLabel),
					resource.TestCheckResourceAttr(fullSaResourceLabel, paramId, "sa-1jjv26"),
					resource.TestCheckResourceAttr(fullSaResourceLabel, paramAssignedResourceOwner, testAssignedResourceOwner),
				),
			},
			{
				ResourceName:      fullSaResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, createStub, fmt.Sprintf("POST %s?%s=%s", createUrlPath, paramAssignedResourceOwner, testAssignedResourceOwner), expectedCountOne)
}

func testAccCheckServiceAccountAssignedResourceOwnerConfig(mockServerUrl, saResourceLabel, assignedResourceOwner string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_service_account" "%s" {
		display_name            = "test_service_account_display_name"
		description             = "The initial description of service account"
		assigned_resource_owner = "%s"
	}
	`, mockServerUrl, saResourceLabel, assignedResourceOwner)
}
