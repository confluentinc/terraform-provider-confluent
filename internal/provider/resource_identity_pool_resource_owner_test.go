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
	"github.com/walkerus/go-wiremock"
)

const (
	scenarioStateIdentityPoolOwnerV2IsCreated = "The new identity_pool (v2 owner) has been just created"
	identityPoolResourceOwnerScenarioName     = "confluent_identity_pool Resource Owner Lifecycle"
	identityPoolResourceOwnerV1               = "sa-owner1v1"
	identityPoolResourceOwnerV2               = "sa-owner2v2"
)

// Verifies that resource_owner is sent as the assigned_resource_owner query param on create (the
// create stub only matches on the exact expected value) and that changing it drives a real
// destroy-then-recreate rather than an in-place update.
func TestAccIdentityPoolResourceOwner(t *testing.T) {
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

	createPath := fmt.Sprintf("/iam/v2/identity-providers/%s/identity-pools", identityProviderId)
	readPath := fmt.Sprintf("/iam/v2/identity-providers/%s/identity-pools/%s", identityProviderId, identityPoolId)

	createResponse, _ := ioutil.ReadFile("../testdata/identity_pool/create_identity_pool.json")
	readCreatedResponse, _ := ioutil.ReadFile("../testdata/identity_pool/read_created_identity_pool.json")
	readDeletedResponse, _ := ioutil.ReadFile("../testdata/identity_pool/read_deleted_identity_pool.json")

	// v1: create with the first resource_owner value.
	createStubV1 := wiremock.Post(wiremock.URLPathEqualTo(createPath)).
		WithQueryParam("assigned_resource_owner", wiremock.EqualTo(identityPoolResourceOwnerV1)).
		InScenario(identityPoolResourceOwnerScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateIdentityPoolHasBeenCreated).
		WillReturn(
			string(createResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createStubV1)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readPath)).
		InScenario(identityPoolResourceOwnerScenarioName).
		WhenScenarioStateIs(scenarioStateIdentityPoolHasBeenCreated).
		WillReturn(
			string(readCreatedResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// Replace: destroy the v1 pool...
	deleteStub := wiremock.Delete(wiremock.URLPathEqualTo(readPath)).
		InScenario(identityPoolResourceOwnerScenarioName).
		WhenScenarioStateIs(scenarioStateIdentityPoolHasBeenCreated).
		WillSetStateTo(scenarioStateIdentityPoolHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readPath)).
		InScenario(identityPoolResourceOwnerScenarioName).
		WhenScenarioStateIs(scenarioStateIdentityPoolHasBeenDeleted).
		WillReturn(
			string(readDeletedResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	// ...then create the v2 pool with a different resource_owner value.
	createStubV2 := wiremock.Post(wiremock.URLPathEqualTo(createPath)).
		WithQueryParam("assigned_resource_owner", wiremock.EqualTo(identityPoolResourceOwnerV2)).
		InScenario(identityPoolResourceOwnerScenarioName).
		WhenScenarioStateIs(scenarioStateIdentityPoolHasBeenDeleted).
		WillSetStateTo(scenarioStateIdentityPoolOwnerV2IsCreated).
		WillReturn(
			string(createResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createStubV2)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readPath)).
		InScenario(identityPoolResourceOwnerScenarioName).
		WhenScenarioStateIs(scenarioStateIdentityPoolOwnerV2IsCreated).
		WillReturn(
			string(readCreatedResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// Final teardown (CheckDestroy) deletes the v2 pool.
	deleteStubV2 := wiremock.Delete(wiremock.URLPathEqualTo(readPath)).
		InScenario(identityPoolResourceOwnerScenarioName).
		WhenScenarioStateIs(scenarioStateIdentityPoolOwnerV2IsCreated).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteStubV2)

	identityPoolResourceLabel := "test_identity_pool_owner_resource_label"
	fullIdentityPoolResourceLabel := fmt.Sprintf("confluent_identity_pool.%s", identityPoolResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckIdentityPoolConfigWithResourceOwner(mockServerUrl, identityPoolResourceLabel, identityPoolResourceOwnerV1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIdentityPoolExists(fullIdentityPoolResourceLabel),
					resource.TestCheckResourceAttr(fullIdentityPoolResourceLabel, paramResourceOwner, identityPoolResourceOwnerV1),
				),
			},
			{
				// Changing resource_owner alone must drive a real replace (destroy v1, create v2),
				// not an in-place update: the API has no update path for this attribute.
				Config: testAccCheckIdentityPoolConfigWithResourceOwner(mockServerUrl, identityPoolResourceLabel, identityPoolResourceOwnerV2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIdentityPoolExists(fullIdentityPoolResourceLabel),
					resource.TestCheckResourceAttr(fullIdentityPoolResourceLabel, paramResourceOwner, identityPoolResourceOwnerV2),
				),
			},
		},
	})

	checkStubCount(t, wiremockClient, createStubV1, fmt.Sprintf("POST %s (v1, owner=%s)", createPath, identityPoolResourceOwnerV1), expectedCountOne)
	checkStubCount(t, wiremockClient, createStubV2, fmt.Sprintf("POST %s (v2, owner=%s)", createPath, identityPoolResourceOwnerV2), expectedCountOne)
	// checkStubCount matches by request pattern; deleteStub and deleteStubV2 share the same
	// method+path (DELETE has no distinguishing query param), so they're counted together: one
	// delete for the forced replace, one more for this test's own final teardown.
	checkStubCount(t, wiremockClient, deleteStub, fmt.Sprintf("DELETE %s", readPath), expectedCountTwo)
}

func testAccCheckIdentityPoolConfigWithResourceOwner(mockServerUrl, identityPoolResourceLabel, resourceOwner string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_identity_pool" "%s" {
        identity_provider {
            id = "%s"
        }
		display_name    = "%s"
		description     = "%s"
		identity_claim  = "%s"
		filter          = %q
		resource_owner  = "%s"
	}
	`, mockServerUrl, identityPoolResourceLabel, identityProviderId, identityPoolDisplayName, identityPoolDescription, identityPoolIdentityClaim, identityPoolFilter, resourceOwner)
}
