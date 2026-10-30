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
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/confluentinc/terraform-provider-confluent/internal/provider/telemetry"
)

const (
	testCloudKafkaURL = "https://pkc-abc12.us-west-2.aws.confluent.cloud:443/kafka/v3/clusters"
	testGovKafkaURL   = "https://pkc-abc12.us-gov-west-1.aws.confluentgov.com:443/kafka/v3/clusters"
)

// roundTripFunc stubs an http.RoundTripper so tests make no network calls.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func okResponse(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("{}"))}, nil
}

// stubAPIClient is an API client built like the provider's: loggingTransport outermost.
func stubAPIClient() *http.Client {
	return &http.Client{Transport: &loggingTransport{transport: roundTripFunc(okResponse), ctx: context.Background()}}
}

// hostsFor returns a recorder that has seen the given request hosts.
func hostsFor(hosts ...string) *apiHosts {
	ctx, h := withAPIHosts(context.Background())
	for _, host := range hosts {
		recordAPIHost(ctx, host)
	}
	return h
}

func TestIsConfluentCloudHost(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{"api.confluent.cloud", true},
		{"pkc-abc12.us-west-2.aws.confluent.cloud", true},
		{"lkc-abc12-dom.us-east-1.aws.private.confluent.cloud", true},
		{"PKC-ABC12.US-WEST-2.AWS.CONFLUENT.CLOUD", true},
		{"psrc-abc12.us-east-2.aws.confluent.cloud.", true},
		{"confluent.cloud", true},
		{"api.confluentgov.com", false},
		{"pkc-abc12.us-gov-west-1.aws.confluentgov.com", false},
		{"confluentgov.com", false},
		{"evilconfluent.cloud", false},
		{"confluent.cloud.example.com", false},
		{"pkc.confluent.cloud.example.com", false},
		{"api.stag.cpdev.cloud", false},
		{"127.0.0.1", false},
		{"", false},
	}
	for _, tc := range tests {
		if got := isConfluentCloudHost(tc.host); got != tc.want {
			t.Errorf("isConfluentCloudHost(%q) = %v, want %v", tc.host, got, tc.want)
		}
	}
}

func TestAPIHostsRecorder(t *testing.T) {
	if hostsFor().onlyConfluentCloud() {
		t.Error("an operation with no requests must not count as Confluent Cloud only")
	}
	if !hostsFor("api.confluent.cloud", "pkc-abc12.us-west-2.aws.confluent.cloud").onlyConfluentCloud() {
		t.Error("requests only to Confluent Cloud hosts must count as Confluent Cloud only")
	}
	if hostsFor("api.confluent.cloud", "pkc-abc12.us-gov-west-1.aws.confluentgov.com").onlyConfluentCloud() {
		t.Error("one request outside confluent.cloud must disqualify the operation")
	}
	// A context without a recorder, and a nil recorder, are safe no-ops.
	recordAPIHost(context.Background(), "api.confluent.cloud")
	var nilHosts *apiHosts
	if nilHosts.onlyConfluentCloud() {
		t.Error("a nil recorder must not count as Confluent Cloud only")
	}
}

// TestLoggingTransportRecordsAPIHosts checks that the provider's shared API transport
// records each request's host on the recorder in the request context.
func TestLoggingTransportRecordsAPIHosts(t *testing.T) {
	get := func(client *http.Client, url string) *apiHosts {
		ctx, h := withAPIHosts(context.Background())
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request to %s failed: %v", url, err)
		}
		resp.Body.Close()
		return h
	}
	if !get(stubAPIClient(), testCloudKafkaURL).onlyConfluentCloud() {
		t.Error("a request to a Confluent Cloud host must be recorded as Confluent Cloud only")
	}
	if get(stubAPIClient(), testGovKafkaURL).onlyConfluentCloud() {
		t.Error("a request to a Government host must be recorded as outside Confluent Cloud")
	}

	// The real retryable API client records too; a loopback host is outside confluent.cloud.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer server.Close()
	h := get(NewRetryableClientFactory(context.Background()).CreateRetryableClient(), server.URL)
	h.mu.Lock()
	seen, other := h.seen, h.other
	h.mu.Unlock()
	if !seen || !other {
		t.Errorf("the retryable API client must record the loopback request host, got seen=%v other=%v", seen, other)
	}
}

// TestDataPlaneClientsUseLoggingTransport pins that every data-plane REST client sends
// through loggingTransport, which the host check relies on to see their requests.
func TestDataPlaneClientsUseLoggingTransport(t *testing.T) {
	ctx := context.Background()
	clients := map[string]*http.Client{
		"Kafka REST":      KafkaRestClientFactory{ctx: ctx}.CreateKafkaRestClient("https://pkc.example", "lkc-1", "k", "s", false, false, nil).apiClient.GetConfig().HTTPClient,
		"Schema Registry": SchemaRegistryRestClientFactory{ctx: ctx}.CreateSchemaRegistryRestClient("https://psrc.example", "lsrc-1", "k", "s", false, nil).apiClient.GetConfig().HTTPClient,
		"Catalog":         CatalogRestClientFactory{ctx: ctx}.CreateCatalogRestClient("https://psrc.example", "lsrc-1", "k", "s", false, nil).apiClient.GetConfig().HTTPClient,
		"Flink":           FlinkRestClientFactory{ctx: ctx}.CreateFlinkRestClient("https://flink.example", "org-1", "env-1", "lfcp-1", "sa-1", "k", "s", false, nil).apiClient.GetConfig().HTTPClient,
		"Tableflow":       TableflowRestClientFactory{ctx: ctx, endpoint: defaultCloudEndpoint}.CreateTableflowRestClient("k", "s", false, nil, nil).apiClient.GetConfig().HTTPClient,
	}
	for name, c := range clients {
		if _, ok := c.Transport.(*loggingTransport); !ok {
			t.Errorf("%s client transport = %T, want *loggingTransport", name, c.Transport)
		}
	}
}

// TestKafkaRestClientRecordsAPIHost drives a real Kafka REST call the way resources do
// (apiContext(ctx)) and checks the operation's recorder sees its host.
func TestKafkaRestClientRecordsAPIHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) }))
	defer server.Close()
	c := KafkaRestClientFactory{ctx: context.Background()}.CreateKafkaRestClient(server.URL, "lkc-1", "k", "s", false, false, nil)

	ctx, h := withAPIHosts(context.Background())
	_, _, _ = c.apiClient.TopicV3Api.GetKafkaTopic(c.apiContext(ctx), c.clusterId, "orders").Execute()
	h.mu.Lock()
	seen, other := h.seen, h.other
	h.mu.Unlock()
	if !seen || !other {
		t.Errorf("the Kafka REST request host was not recorded: seen=%v other=%v", seen, other)
	}
}

// TestCamV1ApiContextKeepsValuesWithoutDeadline checks that connect artifact requests
// carry the operation's recorder but not its deadline, which would cut provisioning
// waits short.
func TestCamV1ApiContextKeepsValuesWithoutDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	ctx, h := withAPIHosts(ctx)
	camCtx := (&Client{cloudApiKey: "k", cloudApiSecret: "s"}).camV1ApiContext(ctx)
	if _, ok := camCtx.Deadline(); ok {
		t.Error("camV1ApiContext must not inherit the operation's deadline")
	}
	if got, _ := camCtx.Value(apiHostsKey{}).(*apiHosts); got != h {
		t.Error("camV1ApiContext must keep the operation's API host recorder")
	}
}

// TestWrapperReportsOnlyOperationsTheHostPolicyAllows checks that the wrapper drops an
// operation's event when the host policy rejects the hosts its API requests went to,
// whether the operation succeeds, fails, or panics.
func TestWrapperReportsOnlyOperationsTheHostPolicyAllows(t *testing.T) {
	// Requests to a /fail path fail in the transport, after the host is recorded.
	client := &http.Client{Transport: &loggingTransport{ctx: context.Background(), transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/fail" {
			return nil, errors.New("connection refused")
		}
		return okResponse(r)
	})}}
	run := func(ctx context.Context, urls []string, after string) error {
		var firstErr error
		for _, url := range urls {
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
			if err != nil {
				return err
			}
			resp, err := client.Do(req)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			resp.Body.Close()
		}
		switch after {
		case "error":
			return errors.New("operation failed")
		case "panic":
			panic("operation panicked")
		}
		return firstErr
	}
	const govFailURL = "https://pkc-abc12.us-gov-west-1.aws.confluentgov.com/fail"
	onlyCloud := (*apiHosts).onlyConfluentCloud
	tests := []struct {
		name   string
		urls   []string
		after  string // "", "error" or "panic"
		policy func(*apiHosts) bool
		want   int
	}{
		{"Confluent Cloud host is reported", []string{testCloudKafkaURL}, "", onlyCloud, 1},
		{"Government host is not reported", []string{testGovKafkaURL}, "", onlyCloud, 0},
		{"operation without a request is not reported", nil, "", onlyCloud, 0},
		{"failing Confluent Cloud operation is reported", []string{testCloudKafkaURL}, "error", onlyCloud, 1},
		{"failing Government operation is not reported", []string{testGovKafkaURL}, "error", onlyCloud, 0},
		{"panicking Confluent Cloud operation is reported", []string{testCloudKafkaURL}, "panic", onlyCloud, 1},
		{"panicking Government operation is not reported", []string{testGovKafkaURL}, "panic", onlyCloud, 0},
		{"a failed Government request still counts", []string{testCloudKafkaURL, govFailURL}, "", onlyCloud, 0},
		{"no policy reports every operation", []string{testGovKafkaURL}, "", nil, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := &recordingReporter{}
			r := newTestResource()
			r.CreateContext = func(ctx context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
				return diag.FromErr(run(ctx, tc.urls, tc.after))
			}
			r.Importer.StateContext = func(ctx context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
				return []*schema.ResourceData{d}, run(ctx, tc.urls, tc.after)
			}
			wrapResourcesMapForTelemetry(map[string]*schema.Resource{"confluent_thing": r}, telemetryWrapConfig{reporter: rec, reportHosts: tc.policy})

			_ = r.CreateContext(context.Background(), nil, nil)
			if got := rec.count(); got != tc.want {
				t.Errorf("create: reported %d events, want %d", got, tc.want)
			}
			_, _ = r.Importer.StateContext(context.Background(), nil, nil)
			if got := rec.count(); got != 2*tc.want {
				t.Errorf("import: reported %d events in total, want %d", got, 2*tc.want)
			}
		})
	}
}

// TestPublishedTelemetryHostPolicy checks that only a runtime published for the default
// endpoint limits reporting to operations that talked only to Confluent Cloud hosts.
func TestPublishedTelemetryHostPolicy(t *testing.T) {
	restorePublishedTelemetry(t)
	t.Setenv(disableProviderAnalyticsEnvVar, "")
	cloud := hostsFor("pkc-abc12.us-west-2.aws.confluent.cloud")
	gov := hostsFor("pkc-abc12.us-gov-west-1.aws.confluentgov.com")
	none := hostsFor()

	publishedTelemetry.Store(nil)
	if !publishedTelemetryHostPolicy(gov) {
		t.Error("with nothing published the policy must not filter")
	}

	tests := []struct {
		name       string
		endpoint   string
		restricted bool
	}{
		{"default endpoint restricts to Confluent Cloud hosts", defaultCloudEndpoint, true},
		{"staging endpoint does not restrict", stagingCloudEndpoint, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			restorePublishedTelemetry(t)
			publishTelemetryRuntime(context.Background(), tc.endpoint, "ua", "cloud-key", "cloud-secret", nil, nil, false)
			rt := publishedTelemetry.Load()
			if rt == nil {
				t.Fatal("expected a published runtime")
			}
			if c, ok := rt.reporter.(interface{ Close() }); ok {
				c.Close()
			}
			if rt.confluentCloudHostsOnly != tc.restricted {
				t.Fatalf("confluentCloudHostsOnly = %v, want %v", rt.confluentCloudHostsOnly, tc.restricted)
			}
			if !publishedTelemetryHostPolicy(cloud) {
				t.Error("an operation that talked only to Confluent Cloud must be reportable")
			}
			if got := publishedTelemetryHostPolicy(gov); got == tc.restricted {
				t.Errorf("Government operation reportable = %v, want %v", got, !tc.restricted)
			}
			if got := publishedTelemetryHostPolicy(none); got == tc.restricted {
				t.Errorf("operation without a request reportable = %v, want %v", got, !tc.restricted)
			}
		})
	}
}

// TestNewWiresTelemetryHostPolicy checks that New() installs the host policy: on a
// runtime restricted to Confluent Cloud hosts, an operation that made no API request
// is not reported.
func TestNewWiresTelemetryHostPolicy(t *testing.T) {
	restorePublishedTelemetry(t)
	r := New(testVersion, "")().ResourcesMap["confluent_environment"]
	for _, restricted := range []bool{true, false} {
		rec := &recordingReporter{}
		publishedTelemetry.Store(&telemetryRuntime{config: telemetry.NewConfig(false), reporter: rec, confluentCloudHostsOnly: restricted})
		// nil arguments fail before any API request is made.
		_ = r.ReadContext(context.Background(), nil, nil)
		want := 1
		if restricted {
			want = 0
		}
		if got := rec.count(); got != want {
			t.Errorf("restricted=%v: reported %d events, want %d", restricted, got, want)
		}
	}
}
