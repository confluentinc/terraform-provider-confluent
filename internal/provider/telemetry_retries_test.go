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
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/confluentinc/terraform-provider-confluent/internal/provider/telemetry"
)

// retryTestServer answers each request with statuses[i] in turn, then 200 OK. A
// Retry-After of 0 makes every retry immediate.
func retryTestServer(t *testing.T, statuses ...int) (*httptest.Server, *int32) {
	t.Helper()
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := int(atomic.AddInt32(&requests, 1))
		if n <= len(statuses) {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(statuses[n-1])
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	return server, &requests
}

// sendWithRetries makes one GET to server's path with the recorder in ctx.
func sendWithRetries(t *testing.T, ctx context.Context, maxRetries int, server *httptest.Server, path string) {
	t.Helper()
	client := NewRetryableClientFactory(context.Background(), WithMaxRetries(maxRetries)).CreateRetryableClient()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}

func usageFor(r *apiRetries) telemetry.Usage {
	var u telemetry.Usage
	r.apply(&u)
	return u
}

func TestRetryTelemetryThroughRetryClient(t *testing.T) {
	tests := map[string]struct {
		maxRetries   int
		statuses     []int
		wantRequests int32
		want         telemetry.Usage
	}{
		"no retries": {
			wantRequests: 1,
			maxRetries:   4,
			want:         telemetry.Usage{},
		},
		"rate limit recovered": {
			wantRequests: 3,
			maxRetries:   4,
			statuses:     []int{429, 429},
			want:         telemetry.Usage{RetryCount: 2, RetryStatusCode: 429, RetryEndpoint: "GET connect/v1/environments"},
		},
		"server error recovered": {
			wantRequests: 2,
			maxRetries:   4,
			statuses:     []int{503},
			want:         telemetry.Usage{RetryCount: 1, RetryStatusCode: 503, RetryEndpoint: "GET connect/v1/environments"},
		},
		"rate limit exhausted": {
			wantRequests: 3,
			maxRetries:   2,
			statuses:     []int{429, 429, 429},
			want:         telemetry.Usage{RetryCount: 2, RetriesExhausted: true, RetryStatusCode: 429, RetryEndpoint: "GET connect/v1/environments"},
		},
		"mixed statuses exhausted": {
			wantRequests: 3,
			maxRetries:   2,
			statuses:     []int{503, 429, 429},
			want:         telemetry.Usage{RetryCount: 2, RetriesExhausted: true, RetryStatusCode: 429, RetryEndpoint: "GET connect/v1/environments"},
		},
		"zero retries configured": {
			wantRequests: 1,
			maxRetries:   0,
			statuses:     []int{429},
			want:         telemetry.Usage{RetriesExhausted: true, RetryStatusCode: 429, RetryEndpoint: "GET connect/v1/environments"},
		},
		"non-retryable status": {
			wantRequests: 1,
			maxRetries:   4,
			statuses:     []int{404},
			want:         telemetry.Usage{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			server, requests := retryTestServer(t, tc.statuses...)
			ctx, retries := withAPIRetries(context.Background())
			sendWithRetries(t, ctx, tc.maxRetries, server, "/connect/v1/environments/env-123/clusters/lkc-456/connectors/my-connector?expand=status")

			if got := usageFor(retries); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
			if got := atomic.LoadInt32(requests); got != tc.wantRequests {
				t.Errorf("server saw %d requests, want %d", got, tc.wantRequests)
			}
		})
	}
}

func TestRetryTelemetryContextEndedDuringBackoffIsNotExhausted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	ctx, retries := withAPIRetries(context.Background())
	ctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	sendWithRetries(t, ctx, 4, server, "/connect/v1/environments")

	if got := usageFor(retries); got.RetriesExhausted || got.RetryCount != 0 {
		t.Errorf("a request cut short mid-backoff must not count as exhausted, got %+v", got)
	}
}

func TestRetryTelemetryAggregatesRequestsAndReportsTheExhaustedOne(t *testing.T) {
	recovered, _ := retryTestServer(t, 503)
	exhausted, _ := retryTestServer(t, 429, 429)

	ctx, retries := withAPIRetries(context.Background())
	sendWithRetries(t, ctx, 4, recovered, "/iam/v2/service-accounts/sa-123")
	sendWithRetries(t, ctx, 1, exhausted, "/connect/v1/environments/env-123")

	want := telemetry.Usage{RetryCount: 2, RetriesExhausted: true, RetryStatusCode: 429, RetryEndpoint: "GET connect/v1/environments"}
	if got := usageFor(retries); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestRetryTelemetryIgnoresRequestsOutsideAnOperation(t *testing.T) {
	server, _ := retryTestServer(t, 429)
	var seen bool
	client := NewRetryableClientFactory(context.Background(), WithMaxRetries(2)).CreateRetryableClient()
	inner := client.Transport
	client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		_, seen = r.Context().Value(requestAttemptsKey{}).(*requestAttempts)
		return inner.RoundTrip(r)
	})
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if seen {
		t.Error("a request made outside a wrapped operation should not carry an attempt tracker")
	}
}

func TestRetryTelemetryTransportErrorExhausted(t *testing.T) {
	// The retry loop reports a transport error with no response.
	ctx, retries := withAPIRetries(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.confluent.cloud/iam/v2/api-keys", nil)
	req, attempts := withRequestAttempts(req)
	policy := recordingRetryPolicy(func(context.Context, *http.Response, error) (bool, error) { return true, nil })
	for i := 0; i < 3; i++ {
		_, _ = policy(req.Context(), nil, &url.Error{Op: "Post", Err: context.DeadlineExceeded})
	}
	finishRequest(req, attempts, 2)

	want := telemetry.Usage{RetryCount: 2, RetriesExhausted: true, RetryEndpoint: "POST iam/v2/api-keys"}
	if got := usageFor(retries); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestRetryEndpoint(t *testing.T) {
	tests := map[string]string{
		"/connect/v1/environments/env-1/clusters/lkc-1/connectors/orders-sink/status": "connect/v1/environments",
		"/kafka/v3/clusters/lkc-1/topics/orders/configs":                              "kafka/v3/clusters",
		"/iam/v2/api-keys/ABCDEF123":                                                  "iam/v2/api-keys",
		"/iam/v2/service-accounts?page_size=100":                                      "iam/v2/service-accounts",
		"/cmk/v2/clusters/":                                                           "cmk/v2/clusters",
		"/dek-registry/v1/keks/my-kek":                                                "dek-registry/v1/keks",
		"/sql/v1/organizations/1111aaaa-11aa/environments":                            "sql/v1/organizations",
		"/fcpm/v2":                      "fcpm/v2",
		"/sql/v1beta1/organizations":    "sql/v1beta1/organizations",
		"/catalog/v1/entity/type/kafka": "catalog/v1/entity",
		"/connect/v1/env-123":           "connect/v1",
		"/connect/v1/my_topic":          "connect/v1",
		// Schema Registry paths hold the subject name in the first segments, even when it looks like a version.
		"/subjects/orders-value/versions/latest":    "other",
		"/subjects/v1payroll/versions/latest":       "other",
		"/subjects/v1/versions":                     "other",
		"/config/v1payroll":                         "other",
		"/config/orders-value":                      "other",
		"/api/metadata/security/v2alpha1/authorize": "other",
		"/":                          "other",
		"/Orders/v1/environments":    "other",
		"/connect/1/environments":    "other",
		"/connect/v1payroll/things":  "other",
		"/unknown-group/v1/things":   "other",
		"/terraform-usage/v1/usages": "other",
	}
	for path, want := range tests {
		req := httptest.NewRequest(http.MethodGet, "https://api.confluent.cloud"+path, nil)
		if got := retryEndpoint(req); got != "GET "+want {
			t.Errorf("retryEndpoint(%q) = %q, want %q", path, got, "GET "+want)
		}
	}
}

func TestWrapperReportsRetryTelemetry(t *testing.T) {
	server, _ := retryTestServer(t, 429, 429, 429)
	client := NewRetryableClientFactory(context.Background(), WithMaxRetries(2)).CreateRetryableClient()

	rec := &recordingReporter{}
	r := newTestResource()
	r.ReadContext = func(ctx context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/connect/v1/environments/env-1/clusters/lkc-1/connectors/c", nil)
		resp, err := client.Do(req)
		if err != nil {
			return diag.FromErr(err)
		}
		resp.Body.Close()
		return nil
	}
	wrapResourcesMapForTelemetry(map[string]*schema.Resource{"confluent_connector": r}, testWrapConfig(rec))

	if diags := r.ReadContext(context.Background(), nil, nil); !diags.HasError() {
		t.Fatal("expected the read to fail once retries ran out")
	}
	if rec.count() != 1 {
		t.Fatalf("expected 1 event, got %d", rec.count())
	}
	u := rec.snapshot()[0]
	if !u.Error || !u.RetriesExhausted || u.RetryCount != 2 || u.RetryStatusCode != 429 || u.RetryEndpoint != "GET connect/v1/environments" {
		t.Errorf("unexpected retry telemetry on the event: %+v", u)
	}
}
