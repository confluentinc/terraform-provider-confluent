package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestRateLimitedAPIMaxRetries(t *testing.T) {
	for configured, want := range map[int]int{4: 12, 8: 12, 12: 12, 20: 20} {
		if got := rateLimitedAPIMaxRetries(configured); got != want {
			t.Errorf("rateLimitedAPIMaxRetries(%d) = %d, want %d", configured, got, want)
		}
	}
}

func TestRateLimitedAPIRetryBudgetOutlastsRepeated429s(t *testing.T) {
	// Ten "429, Retry-After: 0" responses in a row: more than the default 4 retries get through, fewer than the floor.
	const rejections = 10
	tests := map[string]struct {
		maxRetries   int
		wantSuccess  bool
		wantRequests int32
	}{
		"default budget runs out":           {maxRetries: 4, wantSuccess: false, wantRequests: 5},
		"rate-limited API budget is enough": {maxRetries: rateLimitedAPIMaxRetries(4), wantSuccess: true, wantRequests: rejections + 1},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var requests int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if atomic.AddInt32(&requests, 1) <= rejections {
					w.Header().Set("Retry-After", "0")
					w.WriteHeader(http.StatusTooManyRequests)
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			client := NewRetryableClientFactory(context.Background(), WithMaxRetries(tc.maxRetries)).CreateRetryableClient()
			resp, err := client.Get(server.URL)
			if resp != nil {
				defer resp.Body.Close()
			}
			if succeeded := err == nil && resp.StatusCode == http.StatusOK; succeeded != tc.wantSuccess {
				t.Fatalf("expected success=%t, got error %v", tc.wantSuccess, err)
			}
			if got := atomic.LoadInt32(&requests); got != tc.wantRequests {
				t.Errorf("expected %d requests, got %d", tc.wantRequests, got)
			}
		})
	}
}

func TestProviderConfigureGivesRateLimitedAPIClientsTheLongerRetryBudget(t *testing.T) {
	p := New("test", "")()
	d := schema.TestResourceDataRaw(t, p.Schema, map[string]interface{}{
		"cloud_api_key":    "test-key",
		"cloud_api_secret": "test-secret",
	})
	meta, diags := providerConfigure(context.Background(), d, p, "test", "")
	if diags.HasError() {
		t.Fatalf("providerConfigure failed: %v", diags)
	}
	c := meta.(*Client)
	configured := d.Get("max_retries").(int)

	retryMax := func(name string, httpClient *http.Client) int {
		logging, ok := httpClient.Transport.(*loggingTransport)
		if !ok {
			t.Fatalf("%s client: expected a *loggingTransport, got %T", name, httpClient.Transport)
		}
		roundTripper, ok := logging.transport.(*retryablehttp.RoundTripper)
		if !ok {
			t.Fatalf("%s client: expected a *retryablehttp.RoundTripper, got %T", name, logging.transport)
		}
		return roundTripper.Client.RetryMax
	}
	for name, tc := range map[string]struct {
		httpClient *http.Client
		want       int
	}{
		"API keys":                {c.apiKeysV2Client.GetConfig().HTTPClient, rateLimitedAPIMaxRetries(configured)},
		"Connect":                 {c.connectV1Client.GetConfig().HTTPClient, rateLimitedAPIMaxRetries(configured)},
		"IAM":                     {c.iamV2Client.GetConfig().HTTPClient, rateLimitedAPIMaxRetries(configured)},
		"RBAC":                    {c.mdsV2Client.GetConfig().HTTPClient, rateLimitedAPIMaxRetries(configured)},
		"Org (keeps max_retries)": {c.orgV2Client.GetConfig().HTTPClient, configured},
	} {
		if got := retryMax(name, tc.httpClient); got != tc.want {
			t.Errorf("%s client: RetryMax = %d, want %d", name, got, tc.want)
		}
	}
}
