package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

func TestMaxRetriesWithFloor(t *testing.T) {
	for _, tc := range []struct{ configured, floor, want int }{
		{4, connectMaxRetriesFloor, 12},
		{12, connectMaxRetriesFloor, 12},
		{20, connectMaxRetriesFloor, 20},
		{6, iamAndRBACMaxRetriesFloor, 14},
		{15, iamAndRBACMaxRetriesFloor, 15},
		{6, apiKeysMaxRetriesFloor, 20},
		{25, apiKeysMaxRetriesFloor, 25},
	} {
		if got := maxRetriesWithFloor(tc.configured, tc.floor); got != tc.want {
			t.Errorf("maxRetriesWithFloor(%d, %d) = %d, want %d", tc.configured, tc.floor, got, tc.want)
		}
	}
}

func TestRateLimitedAPIRetryBudgetOutlastsRepeated429s(t *testing.T) {
	// More 429s in a row than the default 6 retries outlast, but fewer than the lowest floor (Connect's 12).
	const rejections = 10
	tests := map[string]struct {
		maxRetries   int
		wantSuccess  bool
		wantRequests int32
	}{
		"default budget runs out":           {maxRetries: 6, wantSuccess: false, wantRequests: 7},
		"rate-limited API budget is enough": {maxRetries: maxRetriesWithFloor(6, connectMaxRetriesFloor), wantSuccess: true, wantRequests: rejections + 1},
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
			// Millisecond waits keep the test fast; it checks the retry count, not the wait.
			retryClient := retryClientOf(t, name, client)
			retryClient.RetryWaitMin, retryClient.RetryWaitMax = time.Millisecond, 30*time.Millisecond
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

func TestProviderConfigureGivesRateLimitedAPIClientsTheLongerRetryBudgetAndBackoff(t *testing.T) {
	// Retries per client for each max_retries: a lower max_retries doesn't lower the floors, a higher one raises all.
	// Every client uses rateLimitBackoff.
	for _, want := range []struct {
		name, maxRetriesEnv                  string
		apiKeys, iamAndRBAC, connect, others int
	}{
		{"max_retries not set (6)", "", 20, 14, 12, 6},
		{"max_retries = 4, below the floors", "4", 20, 14, 12, 4},
		{"max_retries = 25, above the floors", "25", 25, 25, 25, 25},
	} {
		t.Run(want.name, func(t *testing.T) {
			_, c := configureWithMaxRetriesEnv(t, want.maxRetriesEnv)
			for name, tc := range map[string]struct {
				httpClient *http.Client
				want       int
			}{
				"API keys":                {c.apiKeysV2Client.GetConfig().HTTPClient, want.apiKeys},
				"Connect":                 {c.connectV1Client.GetConfig().HTTPClient, want.connect},
				"IAM":                     {c.iamV2Client.GetConfig().HTTPClient, want.iamAndRBAC},
				"RBAC":                    {c.mdsV2Client.GetConfig().HTTPClient, want.iamAndRBAC},
				"Org (keeps max_retries)": {c.orgV2Client.GetConfig().HTTPClient, want.others},
			} {
				client := retryClientOf(t, name, tc.httpClient)
				if client.RetryMax != tc.want {
					t.Errorf("%s client: RetryMax = %d, want %d", name, client.RetryMax, tc.want)
				}
				if !isBackoff(client.Backoff, rateLimitBackoff) {
					t.Errorf("%s client: expected rateLimitBackoff", name)
				}
			}
		})
	}
}

// retryClientOf returns the retryablehttp.Client behind an http.Client built by CreateRetryableClient.
func retryClientOf(t *testing.T, name string, httpClient *http.Client) *retryablehttp.Client {
	t.Helper()
	logging, ok := httpClient.Transport.(*loggingTransport)
	if !ok {
		t.Fatalf("%s client: expected a *loggingTransport, got %T", name, httpClient.Transport)
	}
	roundTripper, ok := logging.transport.(*retryablehttp.RoundTripper)
	if !ok {
		t.Fatalf("%s client: expected a *retryablehttp.RoundTripper, got %T", name, logging.transport)
	}
	return roundTripper.Client
}
