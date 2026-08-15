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
	"net/http"
	"net/http/httptest"
	"testing"

	iamv2 "github.com/confluentinc/ccloud-sdk-go-v2/iam/v2"
	"github.com/stretchr/testify/require"
)

// TestServiceAccountReadSurfacesResponseBodyOnError pins the behavior the regenerated read
// path fixes.
//
// createDescriptiveError (utils.go) enriches an otherwise-opaque error by reading the HTTP
// response body. An http.Response body is a one-shot stream, and the SDK hands it back as a
// single re-readable buffer. The previous generated code called createDescriptiveError twice
// on the same response, once to log and once to return, so the second call read an already
// drained body and the error surfaced to Terraform silently dropped the detail. The current
// code enriches once and reuses that value for both, so the raw body reaches the user.
//
// This fails against the old read function (the returned error is missing the body) and
// passes against the current one.
func TestServiceAccountReadSurfacesResponseBodyOnError(t *testing.T) {
	// A non-JSON body so it cannot be parsed as a structured API error, which is exactly the
	// case where createDescriptiveError falls back to reading the raw response body.
	const detail = "boom: service account backend is on fire"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(detail))
	}))
	defer server.Close()

	cfg := iamv2.NewConfiguration()
	cfg.Servers[0].URL = server.URL
	c := &Client{
		iamV2Client:    iamv2.NewAPIClient(cfg),
		cloudApiKey:    "test-key",
		cloudApiSecret: "test-secret",
	}

	d := serviceAccountResource().TestResourceData()
	d.SetId("sa-123456")

	diags := serviceAccountRead(context.Background(), d, c)

	require.True(t, diags.HasError(), "a 500 on read must return an error, not remove the resource from state")
	require.Contains(t, diags[0].Summary, detail,
		"the error returned to Terraform must include the raw response body; if it does not, the read path is reading the response body twice")
}
