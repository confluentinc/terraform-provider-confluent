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
	"github.com/walkerus/go-wiremock"
)

// TestAccRoleBindingReadErrorSurfacesResponseBody pins the behavior the regenerated read path
// fixes; see TestAccServiceAccountReadErrorSurfacesResponseBody for the full explanation.
//
// Create succeeds, then the post-create read returns a 501 whose body cannot be parsed as a
// structured API error. ExpectError requires that raw body to reach the surfaced error, so
// this fails on the old read path (body comes back empty) and passes on the current one.
func TestAccRoleBindingReadErrorSurfacesResponseBody(t *testing.T) {
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

	const readErrorScenario = "RoleBindingReadError"
	const stateCreated = "rb-created"
	const stateReadFailed = "rb-read-failed"
	const readErrorMarker = "ROLE_BINDING_READ_501_MARKER_quota_backend_unavailable"

	// Create succeeds and sets the id (rb-OOXL7, from the fixture).
	createRolebindingResponse, _ := ioutil.ReadFile("../testdata/role_binding/create_role_binding.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo("/iam/v2/role-bindings")).
		InScenario(readErrorScenario).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(stateCreated).
		WillReturn(string(createRolebindingResponse), contentTypeJSONHeader, http.StatusCreated))

	// The post-create read returns an unparseable error carrying a distinctive marker. 501 is
	// used because it is not one of the statuses the provider's retryable HTTP client retries,
	// so the read fails on a single deterministic response.
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(roleBindingUrlPath)).
		InScenario(readErrorScenario).
		WhenScenarioStateIs(stateCreated).
		WillSetStateTo(stateReadFailed).
		WillReturn(fmt.Sprintf(`{"unexpected":%q}`, readErrorMarker), contentTypeJSONHeader, http.StatusNotImplemented))

	// The failed apply leaves a tainted resource; teardown deletes it.
	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(roleBindingUrlPath)).
		InScenario(readErrorScenario).
		WhenScenarioStateIs(stateReadFailed).
		WillReturn("", contentTypeJSONHeader, http.StatusNoContent))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccCheckRoleBindingConfig(mockServerUrl, rbResourceLabel, rbPrincipal, rbRolename, rbCrn, testShouldDisableBefore),
				ExpectError: regexp.MustCompile(readErrorMarker),
			},
		},
	})
}
