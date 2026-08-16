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

// TestAccGroupMappingReadErrorSurfacesResponseBody pins the behavior the regenerated read path
// fixes; see TestAccServiceAccountReadErrorSurfacesResponseBody for the full explanation.
//
// Create succeeds, then the post-create read returns a 501 whose body cannot be parsed as a
// structured API error. ExpectError requires that raw body to reach the surfaced error, so
// this fails on the old read path (body comes back empty) and passes on the current one.
func TestAccGroupMappingReadErrorSurfacesResponseBody(t *testing.T) {
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

	const readErrorScenario = "GroupMappingReadError"
	const stateCreated = "gm-created"
	const stateReadFailed = "gm-read-failed"
	const readErrorMarker = "GROUP_MAPPING_READ_501_MARKER_quota_backend_unavailable"

	// Create succeeds and sets the id (group-w4vP, from the fixture).
	createGroupMappingResponse, _ := ioutil.ReadFile("../testdata/group_mapping/create_group_mapping.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo("/iam/v2/sso/group-mappings")).
		InScenario(readErrorScenario).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(stateCreated).
		WillReturn(string(createGroupMappingResponse), contentTypeJSONHeader, http.StatusCreated))

	// The post-create read returns an unparseable error carrying a distinctive marker. 501 is
	// used because it is not one of the statuses the provider's retryable HTTP client retries,
	// so the read fails on a single deterministic response.
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/sso/group-mappings/group-w4vP")).
		InScenario(readErrorScenario).
		WhenScenarioStateIs(stateCreated).
		WillSetStateTo(stateReadFailed).
		WillReturn(fmt.Sprintf(`{"unexpected":%q}`, readErrorMarker), contentTypeJSONHeader, http.StatusNotImplemented))

	// The failed apply leaves a tainted resource; teardown deletes it.
	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo("/iam/v2/sso/group-mappings/group-w4vP")).
		InScenario(readErrorScenario).
		WhenScenarioStateIs(stateReadFailed).
		WillReturn("", contentTypeJSONHeader, http.StatusNoContent))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccCheckGroupMappingConfig(mockServerUrl, groupMappingResourceLabel, groupMappingDisplayName, groupMappingFilter, groupMappingDescription),
				ExpectError: regexp.MustCompile(readErrorMarker),
			},
		},
	})
}
