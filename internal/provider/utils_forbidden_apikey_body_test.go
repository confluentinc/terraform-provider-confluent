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
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestResponseHasStatusForbiddenDueToInvalidAPIKeyPreservesBody pins that the 403 classifier
// does not swallow the response body.
//
// ResponseHasStatusForbiddenDueToInvalidAPIKey reads resp.Body to look for the invalid-key
// marker. An http.Response body is a one-shot stream, so if the classifier does not restore it,
// the next reader on the same response (createDescriptiveError, when building the error returned
// to Terraform) gets an empty body and the raw API detail is silently dropped.
//
// This fails against the old helper (the body is drained after the call) and passes once the
// helper restores it.
func TestResponseHasStatusForbiddenDueToInvalidAPIKeyPreservesBody(t *testing.T) {
	const body = `{"errors":[{"detail":"403 forbidden: invalid API key for this operation"}]}`
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	// The classifier must still detect the invalid-key 403...
	require.True(t, ResponseHasStatusForbiddenDueToInvalidAPIKey(resp),
		"a 403 whose body contains the invalid-key marker must be classified as such")

	// ...and must leave resp.Body readable for the next consumer, rather than draining it.
	remaining, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, body, string(remaining),
		"ResponseHasStatusForbiddenDueToInvalidAPIKey must restore resp.Body after reading it; "+
			"otherwise the later createDescriptiveError call sees an empty body and the surfaced error loses the raw detail")
}
