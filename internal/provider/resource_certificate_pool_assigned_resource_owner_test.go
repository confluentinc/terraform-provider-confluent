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

// TestAccCertificatePoolAssignedResourceOwner covers assigned_resource_owner, which the API
// accepts only as a query parameter on create and never returns on read.
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
//   - Import seeds the attribute from IMPORT_CERTIFICATE_POOL_ASSIGNED_RESOURCE_OWNER. Without that
//     env var an import leaves it empty, and because the attribute is ForceNew the first
//     post-import plan would want to *replace* the pool. The import step therefore runs with
//     ImportStateVerify and no ImportStateVerifyIgnore, exactly as
//     resource_identity_pool_assigned_resource_owner_test.go does for identity_pool.
//
// Kept separate from TestAccCertificatePool, which never configures this attribute, so that test
// keeps covering the default path where the create request carries no query parameter at all.
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

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(itemUrlPath)).
		InScenario(assignedResourceOwnerScenario).
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
		},
	})

	checkStubCount(t, wiremockClient, createStub, fmt.Sprintf("POST %s?%s=%s", certificatePoolUrlPath, paramAssignedResourceOwner, testAssignedResourceOwner), expectedCountOne)
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
		assigned_resource_owner = "%s"
	}
	`, mockServerUrl, assignedResourceOwner)
}
