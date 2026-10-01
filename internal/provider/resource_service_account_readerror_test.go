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

// TestAccServiceAccountReadErrorSurfacesResponseBody pins the behavior the regenerated read
// path fixes.
//
// createDescriptiveError (utils.go) enriches an otherwise-opaque error by reading the HTTP
// response body. The body is a one-shot stream the SDK hands back as a single re-readable
// buffer. The previous generated read path called createDescriptiveError twice on the same
// response, once to log and once to return, so the second call read an already-drained body
// and the error surfaced to Terraform silently dropped the detail. The current code enriches
// once and reuses it.
//
// Create succeeds, then the post-create read returns a 501 whose body cannot be parsed as a
// structured API error, which is exactly the case that falls back to reading the raw body.
// (501, like the ksql error test, because the provider's retryable client does not retry it,
// so the read fails on a single deterministic response.) ExpectError requires that raw body
// to reach the surfaced error, so this fails on the old read path (body comes back empty) and
// passes on the current one.
func TestAccServiceAccountReadErrorSurfacesResponseBody(t *testing.T) {
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

	const readErrorScenario = "ServiceAccountReadError"
	const stateCreated = "sa-created"
	const stateReadFailed = "sa-read-failed"
	const readErrorMarker = "SA_READ_500_MARKER_quota_backend_unavailable"

	// Create succeeds and sets the id (sa-1jjv26, from the fixture).
	createSaResponse, _ := ioutil.ReadFile("../testdata/service_account/create_sa.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo("/iam/v2/service-accounts")).
		InScenario(readErrorScenario).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(stateCreated).
		WillReturn(string(createSaResponse), contentTypeJSONHeader, http.StatusCreated))

	// The post-create read returns an unparseable error carrying a distinctive marker. 501 is
	// used (like the ksql error test) because it is not one of the statuses the provider's
	// retryable HTTP client retries, so the read fails on a single deterministic response.
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/service-accounts/sa-1jjv26")).
		InScenario(readErrorScenario).
		WhenScenarioStateIs(stateCreated).
		WillSetStateTo(stateReadFailed).
		WillReturn(fmt.Sprintf(`{"unexpected":%q}`, readErrorMarker), contentTypeJSONHeader, http.StatusNotImplemented))

	// The failed apply leaves a tainted resource; teardown deletes it.
	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo("/iam/v2/service-accounts/sa-1jjv26")).
		InScenario(readErrorScenario).
		WhenScenarioStateIs(stateReadFailed).
		WillReturn("", contentTypeJSONHeader, http.StatusNoContent))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccCheckServiceAccountConfig(mockServerUrl, "sa_read_error", "test_sa_read_error", "desc"),
				ExpectError: regexp.MustCompile(readErrorMarker),
			},
		},
	})
}
