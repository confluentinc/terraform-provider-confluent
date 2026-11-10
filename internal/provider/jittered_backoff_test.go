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

// checkUpTo20PercentJitter checks that backoff's waits fall within DefaultBackoff's wait plus 0-20%, and are spread out.
func checkUpTo20PercentJitter(t *testing.T, backoff retryablehttp.Backoff, attemptNum int, resp *http.Response) {
	t.Helper()
	base := retryablehttp.DefaultBackoff(testRetryWaitMin, testRetryWaitMax, attemptNum, resp)
	lowest, highest := time.Duration(math.MaxInt64), time.Duration(0)
	for i := 0; i < backoffSamples; i++ {
		wait := backoff(testRetryWaitMin, testRetryWaitMax, attemptNum, resp)
		lowest, highest = min(lowest, wait), max(highest, wait)
	}
	if lowest < base || highest > base+base/5 {
		t.Errorf("attempt %d: waits ranged %v-%v, want within %v-%v", attemptNum, lowest, highest, base, base+base/5)
	}
	if highest-lowest < base/10 {
		t.Errorf("attempt %d: waits ranged only %v-%v, expected jitter across %v-%v", attemptNum, lowest, highest, base, base+base/5)
	}
}

func TestJitteredBackoffAddsUpTo20PercentToExponentialWaits(t *testing.T) {
	for name, resp := range map[string]*http.Response{
		"no response (transport error)": nil,
		"500":                           responseWithRetryAfter(http.StatusInternalServerError, ""),
		"502 with Retry-After: 20":      responseWithRetryAfter(http.StatusBadGateway, "20"),
		"429 without Retry-After":       responseWithRetryAfter(http.StatusTooManyRequests, ""),
		"503 without Retry-After":       responseWithRetryAfter(http.StatusServiceUnavailable, ""),
	} {
		t.Run(name, func(t *testing.T) {
			for attemptNum := 0; attemptNum <= rateLimitedAPIMinMaxRetries; attemptNum++ {
				checkUpTo20PercentJitter(t, jitteredBackoff, attemptNum, resp)
			}
		})
	}
}

func TestJitteredBackoffWaitsExactlyRetryAfter(t *testing.T) {
	for name, resp := range map[string]*http.Response{
		"429 with Retry-After: 1":                 responseWithRetryAfter(http.StatusTooManyRequests, "1"),
		"503 with Retry-After: 5":                 responseWithRetryAfter(http.StatusServiceUnavailable, "5"),
		"429 with Retry-After above RetryWaitMax": responseWithRetryAfter(http.StatusTooManyRequests, "120"),
		"429 with Retry-After: 0":                 responseWithRetryAfter(http.StatusTooManyRequests, "0"),
		"429 with Retry-After in the past":        responseWithRetryAfter(http.StatusTooManyRequests, time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat)),
		"429 with invalid Retry-After":            responseWithRetryAfter(http.StatusTooManyRequests, "soon"),
	} {
		t.Run(name, func(t *testing.T) {
			for _, attemptNum := range []int{0, 3, rateLimitedAPIMinMaxRetries} {
				want := retryablehttp.DefaultBackoff(testRetryWaitMin, testRetryWaitMax, attemptNum, resp)
				for i := 0; i < 100; i++ {
					if got := jitteredBackoff(testRetryWaitMin, testRetryWaitMax, attemptNum, resp); got != want {
						t.Fatalf("attempt %d: got %v, want DefaultBackoff's %v", attemptNum, got, want)
					}
				}
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

func TestCreateRetryableClientJittersByDefault(t *testing.T) {
	for name, tc := range map[string]struct {
		opts []RetryableClientFactoryOption
		want retryablehttp.Backoff
	}{
		"default":              {nil, jitteredBackoff},
		"with max retries":     {[]RetryableClientFactoryOption{WithMaxRetries(20)}, jitteredBackoff},
		"WithRateLimitBackoff": {[]RetryableClientFactoryOption{WithRateLimitBackoff()}, rateLimitBackoff},
	} {
		client := retryClientOf(t, name, NewRetryableClientFactory(context.Background(), tc.opts...).CreateRetryableClient())
		if !isBackoff(client.Backoff, tc.want) {
			t.Errorf("%s: unexpected Backoff", name)
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
		if client := retryClientOf(t, name, httpClient); !isBackoff(client.Backoff, jitteredBackoff) {
			t.Errorf("%s client: expected jitteredBackoff", name)
		}
	}
}
