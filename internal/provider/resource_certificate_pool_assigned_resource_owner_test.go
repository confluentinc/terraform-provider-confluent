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
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
)

// TestAccCertificatePoolAssignedResourceOwner covers assigned_resource_owner, which the API
// accepts only as a query parameter on create and never returns on read.
//
// Five things are asserted, each of which would otherwise fail silently:
//
//   - The create request actually carries ?assigned_resource_owner=<principal>. The create stub
//     matches on that query parameter, so a provider that accepts the attribute and then drops it
//     gets no stub match, a 404, and a failed apply — rather than a green test.
//   - The configured value survives in state even though no read path sets it. The read fixture
//     does not carry the field (the API does not return it), and the framework fails a step whose
//     post-apply plan is non-empty, so this step passing is what proves the attribute does not
//     drift on refresh.
//   - Import seeds the attribute from IMPORT_CERTIFICATE_POOL_ASSIGNED_RESOURCE_OWNER. Without that
//     env var an import leaves it empty, and because the attribute is ForceNew the first
//     post-import plan would want to *replace* the pool. The import step therefore runs with
//     ImportStateVerify and no ImportStateVerifyIgnore, exactly as
//     resource_identity_pool_assigned_resource_owner_test.go does for identity_pool.
//   - Changing the attribute replaces the resource, and the replacement's create sends the new
//     value. The stub counts at the end assert a POST matching the new owner exactly once, and a
//     second DELETE beyond the final destroy.
//   - Removing the attribute from configuration does not replace the resource. The last step drops
//     it, so a replacement would need a third DELETE and a POST with no owner, for which no stub
//     exists; the stub counts at the end are unchanged by that step.
//
// Kept separate from TestAccCertificatePool, which never configures this attribute.
// TestAccCertificatePoolAssignedResourceOwnerAddedToExisting covers the unset path: that the create
// request then carries no query parameter at all.
func TestAccCertificatePoolAssignedResourceOwner(t *testing.T) {
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

	const assignedResourceOwnerScenario = "confluent_certificate_pool assigned_resource_owner"
	const stateCreated = "assigned-resource-owner-created"
	// The spec's own example for the parameter.
	const testAssignedResourceOwner = "u-a83k9b"
	// Changing the owner replaces the resource (the attribute is ForceNew): Terraform deletes it,
	// then creates it again with the new owner. Delete does not poll, so that create is the next
	// request.
	const testReplacementAssignedResourceOwner = "sa-r3pl4c"
	const stateDeletedForReplacement = "assigned-resource-owner-deleted-for-replacement"
	const stateRecreated = "assigned-resource-owner-recreated"

	itemUrlPath := fmt.Sprintf("%s/%s", certificatePoolUrlPath, certificatePoolId)

	// The query-param matcher is the assertion: without it on the request, this stub does not
	// match and the apply fails.
	createCertificatePoolResponse, _ := ioutil.ReadFile("../testdata/certificate_pool/create_certificate_pool.json")
	createStub := wiremock.Post(wiremock.URLPathEqualTo(certificatePoolUrlPath)).
		WithQueryParam(paramAssignedResourceOwner, wiremock.EqualTo(testAssignedResourceOwner)).
		InScenario(assignedResourceOwnerScenario).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(stateCreated).
		WillReturn(string(createCertificatePoolResponse), contentTypeJSONHeader, http.StatusCreated)
	_ = wiremockClient.StubFor(createStub)

	// The read response deliberately does not carry assigned_resource_owner, matching the real
	// API. The value in state comes from configuration, not from this payload.
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(itemUrlPath)).
		InScenario(assignedResourceOwnerScenario).
		WhenScenarioStateIs(stateCreated).
		WillReturn(string(createCertificatePoolResponse), contentTypeJSONHeader, http.StatusOK))

	deleteForReplacementStub := wiremock.Delete(wiremock.URLPathEqualTo(itemUrlPath)).
		InScenario(assignedResourceOwnerScenario).
		WhenScenarioStateIs(stateCreated).
		WillSetStateTo(stateDeletedForReplacement).
		WillReturn("", contentTypeJSONHeader, http.StatusNoContent)
	_ = wiremockClient.StubFor(deleteForReplacementStub)

	// The replacement's create must carry the new owner, or this stub does not match.
	recreateStub := wiremock.Post(wiremock.URLPathEqualTo(certificatePoolUrlPath)).
		WithQueryParam(paramAssignedResourceOwner, wiremock.EqualTo(testReplacementAssignedResourceOwner)).
		InScenario(assignedResourceOwnerScenario).
		WhenScenarioStateIs(stateDeletedForReplacement).
		WillSetStateTo(stateRecreated).
		WillReturn(string(createCertificatePoolResponse), contentTypeJSONHeader, http.StatusCreated)
	_ = wiremockClient.StubFor(recreateStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(itemUrlPath)).
		InScenario(assignedResourceOwnerScenario).
		WhenScenarioStateIs(stateRecreated).
		WillReturn(string(createCertificatePoolResponse), contentTypeJSONHeader, http.StatusOK))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(itemUrlPath)).
		InScenario(assignedResourceOwnerScenario).
		WhenScenarioStateIs(stateRecreated).
		WillReturn("", contentTypeJSONHeader, http.StatusNoContent))

	// The import step below reads this: the API does not return the value, so without it the
	// imported state would hold "" and ImportStateVerify would fail. t.Setenv restores any value
	// already exported in the developer's shell when the test ends.
	t.Setenv("IMPORT_CERTIFICATE_POOL_ASSIGNED_RESOURCE_OWNER", testAssignedResourceOwner)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckCertificatePoolAssignedResourceOwnerConfig(mockServerUrl, testAssignedResourceOwner),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(certificatePoolResourceLabel, paramId, certificatePoolId),
					resource.TestCheckResourceAttr(certificatePoolResourceLabel, paramAssignedResourceOwner, testAssignedResourceOwner),
				),
			},
			{
				ResourceName:      certificatePoolResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					poolId := resources[certificatePoolResourceLabel].Primary.ID
					certificateAuthorityId := resources[certificatePoolResourceLabel].Primary.Attributes["certificate_authority.0.id"]
					return certificateAuthorityId + "/" + poolId, nil
				},
			},
			{
				// assigned_resource_owner is ForceNew, so changing it replaces the resource.
				Config: testAccCheckCertificatePoolAssignedResourceOwnerConfig(mockServerUrl, testReplacementAssignedResourceOwner),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(certificatePoolResourceLabel, paramId, certificatePoolId),
					resource.TestCheckResourceAttr(certificatePoolResourceLabel, paramAssignedResourceOwner, testReplacementAssignedResourceOwner),
				),
			},
			{
				// Removing the attribute must not plan anything: the plan after this apply must be
				// empty, and the configured value from the previous step stays in state.
				Config: testAccCheckCertificatePoolAssignedResourceOwnerConfig(mockServerUrl, ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(certificatePoolResourceLabel, paramId, certificatePoolId),
					resource.TestCheckResourceAttr(certificatePoolResourceLabel, paramAssignedResourceOwner, testReplacementAssignedResourceOwner),
				),
			},
		},
	})

	checkStubCount(t, wiremockClient, createStub, fmt.Sprintf("POST %s?%s=%s", certificatePoolUrlPath, paramAssignedResourceOwner, testAssignedResourceOwner), expectedCountOne)
	// Request counts match on method and URL, not scenario state, so this also counts the final
	// destroy: two DELETEs (replacement, then destroy) where a test with no replacement sees one.
	checkStubCount(t, wiremockClient, deleteForReplacementStub, fmt.Sprintf("DELETE %s", itemUrlPath), expectedCountTwo)
	checkStubCount(t, wiremockClient, recreateStub, fmt.Sprintf("POST %s?%s=%s", certificatePoolUrlPath, paramAssignedResourceOwner, testReplacementAssignedResourceOwner), expectedCountOne)
}

// TestAccCertificatePoolAssignedResourceOwnerAddedToExisting covers a certificate pool created without
// assigned_resource_owner, the path every existing configuration takes, and the attribute being
// added to it later:
//
//   - The create request carries no assigned_resource_owner query parameter at all, so existing
//     configurations send exactly the request they always have. A higher-priority stub answers any
//     create that carries the parameter, whatever its value, with a 400, so a provider that sent it
//     for an unset attribute fails the first apply.
//   - Adding the attribute to the existing certificate pool replaces it, as the docs say: Terraform
//     deletes it, then creates it again with the owner. The stub counts at the end require exactly
//     one create carrying the parameter, the replacement's, out of two creates.
func TestAccCertificatePoolAssignedResourceOwnerAddedToExisting(t *testing.T) {
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

	const addedLaterScenario = "confluent_certificate_pool assigned_resource_owner added later"
	const stateCreated = "created-without-assigned-resource-owner"
	const stateDeletedForReplacement = "deleted-for-replacement"
	const stateRecreated = "recreated-with-assigned-resource-owner"
	const testAssignedResourceOwner = "u-a83k9b"

	createUrlPath := certificatePoolUrlPath
	itemUrlPath := fmt.Sprintf("%s/%s", certificatePoolUrlPath, certificatePoolId)

	createResponse, _ := ioutil.ReadFile("../testdata/certificate_pool/create_certificate_pool.json")
	readCreatedResponse, _ := ioutil.ReadFile("../testdata/certificate_pool/create_certificate_pool.json")

	// Any create carrying the parameter, even with an empty value, matches this stub ahead of the
	// plain create below (priority 1 beats the default) and fails the apply.
	createWithOwnerStub := wiremock.Post(wiremock.URLPathEqualTo(createUrlPath)).
		WithQueryParam(paramAssignedResourceOwner, wiremock.Matching(".*")).
		AtPriority(1).
		InScenario(addedLaterScenario).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(`{"errors":[{"status":"400","detail":"unexpected assigned_resource_owner query parameter"}]}`, contentTypeJSONHeader, http.StatusBadRequest)
	_ = wiremockClient.StubFor(createWithOwnerStub)

	createStub := wiremock.Post(wiremock.URLPathEqualTo(createUrlPath)).
		InScenario(addedLaterScenario).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(stateCreated).
		WillReturn(string(createResponse), contentTypeJSONHeader, http.StatusCreated)
	_ = wiremockClient.StubFor(createStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(itemUrlPath)).
		InScenario(addedLaterScenario).
		WhenScenarioStateIs(stateCreated).
		WillReturn(string(readCreatedResponse), contentTypeJSONHeader, http.StatusOK))

	deleteStub := wiremock.Delete(wiremock.URLPathEqualTo(itemUrlPath)).
		InScenario(addedLaterScenario).
		WhenScenarioStateIs(stateCreated).
		WillSetStateTo(stateDeletedForReplacement).
		WillReturn("", contentTypeJSONHeader, http.StatusNoContent)
	_ = wiremockClient.StubFor(deleteStub)

	// The replacement's create must carry the owner, or this stub does not match.
	recreateStub := wiremock.Post(wiremock.URLPathEqualTo(createUrlPath)).
		WithQueryParam(paramAssignedResourceOwner, wiremock.EqualTo(testAssignedResourceOwner)).
		InScenario(addedLaterScenario).
		WhenScenarioStateIs(stateDeletedForReplacement).
		WillSetStateTo(stateRecreated).
		WillReturn(string(createResponse), contentTypeJSONHeader, http.StatusCreated)
	_ = wiremockClient.StubFor(recreateStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(itemUrlPath)).
		InScenario(addedLaterScenario).
		WhenScenarioStateIs(stateRecreated).
		WillReturn(string(readCreatedResponse), contentTypeJSONHeader, http.StatusOK))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(itemUrlPath)).
		InScenario(addedLaterScenario).
		WhenScenarioStateIs(stateRecreated).
		WillReturn("", contentTypeJSONHeader, http.StatusNoContent))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckCertificatePoolAssignedResourceOwnerConfig(mockServerUrl, ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(certificatePoolResourceLabel, paramId, certificatePoolId),
					resource.TestCheckNoResourceAttr(certificatePoolResourceLabel, paramAssignedResourceOwner),
				),
			},
			{
				// assigned_resource_owner is ForceNew, so adding it to an existing certificate pool
				// replaces it.
				Config: testAccCheckCertificatePoolAssignedResourceOwnerConfig(mockServerUrl, testAssignedResourceOwner),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(certificatePoolResourceLabel, paramId, certificatePoolId),
					resource.TestCheckResourceAttr(certificatePoolResourceLabel, paramAssignedResourceOwner, testAssignedResourceOwner),
				),
			},
		},
	})

	// Request counts match on method, URL and query parameters, not scenario state.
	checkStubCount(t, wiremockClient, createStub, fmt.Sprintf("POST %s", createUrlPath), expectedCountTwo)
	checkStubCount(t, wiremockClient, createWithOwnerStub, fmt.Sprintf("POST %s?%s=<any>", createUrlPath, paramAssignedResourceOwner), expectedCountOne)
	checkStubCount(t, wiremockClient, recreateStub, fmt.Sprintf("POST %s?%s=%s", createUrlPath, paramAssignedResourceOwner, testAssignedResourceOwner), expectedCountOne)
	// The replacement, then the final destroy.
	checkStubCount(t, wiremockClient, deleteStub, fmt.Sprintf("DELETE %s", itemUrlPath), expectedCountTwo)
}

// TestAccCertificatePoolAssignedResourceOwnerCreateError covers a create the API rejects because of the owner,
// such as a principal that does not exist. The API's error detail must reach the user, and nothing
// may be left in state, so no read or delete of the certificate pool follows.
func TestAccCertificatePoolAssignedResourceOwnerCreateError(t *testing.T) {
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

	const invalidAssignedResourceOwner = "u-doesnotexist"
	const createErrorDetail = "Principal u-doesnotexist does not exist"

	createUrlPath := certificatePoolUrlPath
	itemUrlPath := fmt.Sprintf("%s/%s", certificatePoolUrlPath, certificatePoolId)

	createStub := wiremock.Post(wiremock.URLPathEqualTo(createUrlPath)).
		WithQueryParam(paramAssignedResourceOwner, wiremock.EqualTo(invalidAssignedResourceOwner)).
		WillReturn(fmt.Sprintf(`{"errors":[{"status":"400","detail":%q}]}`, createErrorDetail), contentTypeJSONHeader, http.StatusBadRequest)
	_ = wiremockClient.StubFor(createStub)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccCheckCertificatePoolAssignedResourceOwnerConfig(mockServerUrl, invalidAssignedResourceOwner),
				ExpectError: regexp.MustCompile(regexp.QuoteMeta(createErrorDetail)),
			},
		},
	})

	// A 400 is not retried, and a failed create leaves nothing to read or delete.
	checkStubCount(t, wiremockClient, createStub, fmt.Sprintf("POST %s?%s=%s", createUrlPath, paramAssignedResourceOwner, invalidAssignedResourceOwner), expectedCountOne)
	checkStubCount(t, wiremockClient, wiremock.Get(wiremock.URLPathEqualTo(itemUrlPath)), fmt.Sprintf("GET %s", itemUrlPath), expectedCountZero)
	checkStubCount(t, wiremockClient, wiremock.Delete(wiremock.URLPathEqualTo(itemUrlPath)), fmt.Sprintf("DELETE %s", itemUrlPath), expectedCountZero)
}

func testAccCheckCertificatePoolAssignedResourceOwnerConfig(mockServerUrl, assignedResourceOwner string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_certificate_pool" "main" {
		certificate_authority {
			id = "op-abc123"
		}
		display_name            = "my-certificate-pool"
		description             = "example-description"
		external_identifier     = "UID"
		filter                  = "C=='Canada' && O=='Confluent'"
		%s
	}
	`, mockServerUrl, assignedResourceOwnerConfigLine(assignedResourceOwner))
}
