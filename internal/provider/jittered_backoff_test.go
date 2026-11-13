package provider

import (
	"context"
	"math"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

// checkUpTo20PercentJitter checks that backoff's waits fall within DefaultBackoff's wait plus 0-20%, and are spread
// across at least 90% of that range.
func checkUpTo20PercentJitter(t *testing.T, backoff retryablehttp.Backoff, attemptNum int, resp *http.Response) {
	t.Helper()
	base := retryablehttp.DefaultBackoff(testRetryWaitMin, testRetryWaitMax, attemptNum, resp)
	extra := base / 5
	lowest, highest := time.Duration(math.MaxInt64), time.Duration(0)
	for i := 0; i < backoffSamples; i++ {
		wait := backoff(testRetryWaitMin, testRetryWaitMax, attemptNum, resp)
		lowest, highest = min(lowest, wait), max(highest, wait)
	}
	if lowest < base || highest > base+extra {
		t.Errorf("attempt %d: waits ranged %v-%v, want within %v-%v", attemptNum, lowest, highest, base, base+extra)
	}
	if highest-lowest < extra*9/10 {
		t.Errorf("attempt %d: waits ranged only %v-%v, expected jitter across %v-%v", attemptNum, lowest, highest, base, base+extra)
	}
}

func TestJitteredBackoffAddsUpTo20PercentToExponentialWaits(t *testing.T) {
	for name, resp := range map[string]*http.Response{
		"no response (transport error)": nil,
		"500":                           responseWithRetryAfter(http.StatusInternalServerError, ""),
		"502 with Retry-After: 20":      responseWithRetryAfter(http.StatusBadGateway, "20"),
	} {
		t.Run(name, func(t *testing.T) {
			for attemptNum := 0; attemptNum <= rateLimitedAPIMinMaxRetries; attemptNum++ {
				checkUpTo20PercentJitter(t, jitteredBackoff, attemptNum, resp)
			}
		})
	}
}

func TestWithJitterReturnsUnusableWaitsUnchanged(t *testing.T) {
	for _, wait := range []time.Duration{-time.Second, 0, 4 * time.Nanosecond, math.MaxInt64, math.MaxInt64 - 1} {
		if got := withJitter(wait); got != wait {
			t.Errorf("withJitter(%d) = %d, want it unchanged", wait, got)
		}
	}
}

func isBackoff(got, want retryablehttp.Backoff) bool {
	return reflect.ValueOf(got).Pointer() == reflect.ValueOf(want).Pointer()
}

func TestCreateRetryableClientUsesRateLimitBackoff(t *testing.T) {
	for name, opts := range map[string][]RetryableClientFactoryOption{
		"default":      nil,
		"more retries": {WithMaxRetries(20)},
	} {
		client := retryClientOf(t, name, NewRetryableClientFactory(context.Background(), opts...).CreateRetryableClient())
		if !isBackoff(client.Backoff, rateLimitBackoff) {
			t.Errorf("%s: expected rateLimitBackoff", name)
		}
	}
}

func TestDataPlaneClientsJitter(t *testing.T) {
	ctx := context.Background()
	maxRetries := 4
	const endpoint = "https://example.invalid"
	for name, httpClient := range map[string]*http.Client{
		"Kafka REST": KafkaRestClientFactory{ctx: ctx, maxRetries: &maxRetries}.
			CreateKafkaRestClient(endpoint, "lkc-1", "key", "secret", false, false, nil).apiClient.GetConfig().HTTPClient,
		"Schema Registry": SchemaRegistryRestClientFactory{ctx: ctx, maxRetries: &maxRetries}.
			CreateSchemaRegistryRestClient(endpoint, "lsrc-1", "key", "secret", false, nil).apiClient.GetConfig().HTTPClient,
		"Data Catalog": CatalogRestClientFactory{ctx: ctx, maxRetries: &maxRetries}.
			CreateCatalogRestClient(endpoint, "lsrc-1", "key", "secret", false, nil).apiClient.GetConfig().HTTPClient,
		"Flink": FlinkRestClientFactory{ctx: ctx, maxRetries: &maxRetries}.
			CreateFlinkRestClient(endpoint, "org", "env", "lfcp-1", "sa-1", "key", "secret", false, nil).apiClient.GetConfig().HTTPClient,
		"Tableflow": TableflowRestClientFactory{ctx: ctx, maxRetries: &maxRetries, endpoint: endpoint}.
			CreateTableflowRestClient("key", "secret", false, nil, nil).apiClient.GetConfig().HTTPClient,
	} {
		if client := retryClientOf(t, name, httpClient); !isBackoff(client.Backoff, rateLimitBackoff) {
			t.Errorf("%s client: expected rateLimitBackoff", name)
		}
	}
}
