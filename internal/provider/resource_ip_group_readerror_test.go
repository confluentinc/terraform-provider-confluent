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

// TestAccIPGroupReadErrorSurfacesResponseBody pins the behavior the regenerated read path
// fixes; see TestAccServiceAccountReadErrorSurfacesResponseBody for the full explanation.
//
// Create succeeds, then the post-create read returns a 501 whose body cannot be parsed as a
// structured API error. ExpectError requires that raw body to reach the surfaced error, so
// this fails on the old read path (body comes back empty) and passes on the current one.
func TestAccIPGroupReadErrorSurfacesResponseBody(t *testing.T) {
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

	const readErrorScenario = "IPGroupReadError"
	const stateCreated = "ipg-created"
	const stateReadFailed = "ipg-read-failed"
	const readErrorMarker = "IP_GROUP_READ_501_MARKER_quota_backend_unavailable"

	itemUrlPath := fmt.Sprintf("/iam/v2/ip-groups/%s", testIPGroupID)

	// Create succeeds and sets the id (ipg-wyorq, from the fixture).
	createIPGroupResponse, _ := ioutil.ReadFile("../testdata/ip_group/create_ip_group.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo("/iam/v2/ip-groups")).
		InScenario(readErrorScenario).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(stateCreated).
		WillReturn(string(createIPGroupResponse), contentTypeJSONHeader, http.StatusCreated))

	// The post-create read returns an unparseable error carrying a distinctive marker. 501 is
	// used because it is not one of the statuses the provider's retryable HTTP client retries,
	// so the read fails on a single deterministic response.
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(itemUrlPath)).
		InScenario(readErrorScenario).
		WhenScenarioStateIs(stateCreated).
		WillSetStateTo(stateReadFailed).
		WillReturn(fmt.Sprintf(`{"unexpected":%q}`, readErrorMarker), contentTypeJSONHeader, http.StatusNotImplemented))

	// The failed apply leaves a tainted resource; teardown deletes it.
	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(itemUrlPath)).
		InScenario(readErrorScenario).
		WhenScenarioStateIs(stateReadFailed).
		WillReturn("", contentTypeJSONHeader, http.StatusNoContent))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccCheckIPGroupConfig(mockServerUrl, testIPGroupResourceLabel, testIPGroupName, testIPGroupCidrBlocks),
				ExpectError: regexp.MustCompile(readErrorMarker),
			},
		},
	})
}
