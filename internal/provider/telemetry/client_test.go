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

package telemetry

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	terraformusagev1 "github.com/confluentinc/ccloud-sdk-go-v2/terraform-usage/v1"
)

// TestSDKPoster_PostsMappedUsage asserts the poster hits the contract path with
// the correct method, applies auth, and maps the Usage onto the wire body —
// including a non-null empty changed_attributes.
func TestSDKPoster_PostsMappedUsage(t *testing.T) {
	type captured struct {
		method string
		path   string
		auth   string
		body   map[string]interface{}
	}
	got := make(chan captured, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]interface{}
		_ = json.Unmarshal(raw, &body)
		got <- captured{method: r.Method, path: r.URL.Path, auth: r.Header.Get("Authorization"), body: body}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	authFunc := func(ctx context.Context) context.Context {
		return context.WithValue(ctx, terraformusagev1.ContextBasicAuth, terraformusagev1.BasicAuth{
			UserName: "key",
			Password: "secret",
		})
	}
	p := NewSDKPoster(srv.URL, srv.Client(), "terraform-provider-confluent/test", authFunc)

	u := Usage{
		RunID:             "run-123",
		Sequence:          7,
		StartedAt:         time.Unix(1700000000, 0).UTC(),
		DurationMs:        42,
		OS:                "linux",
		Arch:              "amd64",
		ProviderVersion:   "2.0.0",
		TerraformVersion:  "1.8.0",
		ResourceType:      "confluent_kafka_topic",
		Operation:         OperationCreate,
		ChangedAttributes: []string{}, // empty but must serialize as []
	}
	if err := p.Post(context.Background(), u); err != nil {
		t.Fatalf("Post returned error: %v", err)
	}

	c := <-got
	if c.method != http.MethodPost {
		t.Errorf("method = %s, want POST", c.method)
	}
	if c.path != "/terraform-usage/v1/usages" {
		t.Errorf("path = %s, want /terraform-usage/v1/usages", c.path)
	}
	if !strings.HasPrefix(c.auth, "Basic ") {
		t.Errorf("Authorization = %q, want a Basic auth header", c.auth)
	}
	if c.body["run_id"] != "run-123" {
		t.Errorf("run_id = %v, want run-123", c.body["run_id"])
	}
	if c.body["operation"] != "CREATE" {
		t.Errorf("operation = %v, want CREATE", c.body["operation"])
	}
	if c.body["resource_type"] != "confluent_kafka_topic" {
		t.Errorf("resource_type = %v, want confluent_kafka_topic", c.body["resource_type"])
	}
	// sequence/duration_ms are numbers on the wire (int32 in the contract).
	if seq, ok := c.body["sequence"].(float64); !ok || seq != 7 {
		t.Errorf("sequence = %v, want 7", c.body["sequence"])
	}
	// changed_attributes must be present and an empty array — not null, not absent.
	ca, present := c.body["changed_attributes"]
	if !present {
		t.Errorf("changed_attributes missing; must serialize as []")
	} else if arr, ok := ca.([]interface{}); !ok || len(arr) != 0 {
		t.Errorf("changed_attributes = %v, want []", ca)
	}
	// The remaining scalar fields must all be mapped onto the wire body.
	if c.body["os"] != "linux" {
		t.Errorf("os = %v, want linux", c.body["os"])
	}
	if c.body["arch"] != "amd64" {
		t.Errorf("arch = %v, want amd64", c.body["arch"])
	}
	if c.body["provider_version"] != "2.0.0" {
		t.Errorf("provider_version = %v, want 2.0.0", c.body["provider_version"])
	}
	if c.body["terraform_version"] != "1.8.0" {
		t.Errorf("terraform_version = %v, want 1.8.0", c.body["terraform_version"])
	}
	if dur, ok := c.body["duration_ms"].(float64); !ok || dur != 42 {
		t.Errorf("duration_ms = %v, want 42", c.body["duration_ms"])
	}
	// error is a required field and must be present even when false.
	if e, present := c.body["error"]; !present || e != false {
		t.Errorf("error = %v (present=%v), want false and present", c.body["error"], present)
	}
	// started_at must be present and round-trip to the same instant.
	if sa, ok := c.body["started_at"].(string); !ok {
		t.Errorf("started_at = %v, want an RFC3339 string", c.body["started_at"])
	} else if ts, err := time.Parse(time.RFC3339, sa); err != nil || !ts.Equal(u.StartedAt) {
		t.Errorf("started_at = %q, want %s", sa, u.StartedAt.Format(time.RFC3339))
	}
}

// TestSDKPoster_Non2xxIsError asserts a non-2xx response surfaces as an error so
// the transport logs and drops it.
func TestSDKPoster_Non2xxIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := NewSDKPoster(srv.URL, srv.Client(), "ua", nil)
	if err := p.Post(context.Background(), sampleUsage()); err == nil {
		t.Fatal("expected an error for a 500 response, got nil")
	}
}

// TestSDKPoster_HonorsContextDeadline asserts the poster respects a cancelled
// context (the transport's per-report timeout), so a hung endpoint can't wedge a
// worker.
func TestSDKPoster_HonorsContextDeadline(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-block // hang until the test releases us
	}))
	// Defer ordering matters (LIFO): unblock the handler FIRST, then Close, so
	// srv.Close() isn't left waiting on an in-flight handler.
	defer srv.Close()
	defer close(block)

	p := NewSDKPoster(srv.URL, srv.Client(), "ua", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := p.Post(ctx, sampleUsage())
	if err == nil {
		t.Fatal("expected a deadline error, got nil")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("Post did not honor the context deadline (took %s)", elapsed)
	}
}

// TestSDKPoster_NilChangedAttributesSerializesAsEmpty asserts the poster coerces a
// nil ChangedAttributes to [] on the wire. changed_attributes is contract-required
// and non-nullable, so sending null (what SetChangedAttributes(nil) would emit)
// would be rejected. The wrapper always supplies a non-nil slice, but the poster
// hardens the last step before the wire rather than trusting that.
func TestSDKPoster_NilChangedAttributesSerializesAsEmpty(t *testing.T) {
	got := make(chan []byte, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		got <- raw
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := NewSDKPoster(srv.URL, srv.Client(), "ua", nil)
	u := sampleUsage()
	u.ChangedAttributes = nil // the required, non-nullable field must still be []
	if err := p.Post(context.Background(), u); err != nil {
		t.Fatalf("Post returned error: %v", err)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(<-got, &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	ca, present := body["changed_attributes"]
	if !present {
		t.Fatalf("changed_attributes missing; must serialize as []")
	}
	if arr, ok := ca.([]interface{}); !ok || len(arr) != 0 {
		t.Errorf("changed_attributes = %v (type %T), want [] (not null)", ca, ca)
	}
}
