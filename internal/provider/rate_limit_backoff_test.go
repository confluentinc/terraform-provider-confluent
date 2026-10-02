package provider

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

const (
	testRetryWaitMin = 1 * time.Second
	testRetryWaitMax = 30 * time.Second
	backoffSamples   = 2000
)

func responseWithRetryAfter(statusCode int, retryAfter string) *http.Response {
	resp := &http.Response{StatusCode: statusCode, Header: http.Header{}}
	if retryAfter != "" {
		resp.Header.Set("Retry-After", retryAfter)
	}
	return resp
}

func sampleBackoff(attemptNum int, resp *http.Response) (time.Duration, time.Duration) {
	lowest, highest := time.Duration(math.MaxInt64), time.Duration(0)
	for i := 0; i < backoffSamples; i++ {
		wait := rateLimitBackoff(testRetryWaitMin, testRetryWaitMax, attemptNum, resp)
		lowest, highest = min(lowest, wait), max(highest, wait)
	}
	return lowest, highest
}

func TestRateLimitBackoffJittersAboveExponentialSchedule(t *testing.T) {
	for name, resp := range map[string]*http.Response{
		"429 with Retry-After: 1": responseWithRetryAfter(http.StatusTooManyRequests, "1"),
		"429 without Retry-After": responseWithRetryAfter(http.StatusTooManyRequests, ""),
		"503 with Retry-After: 1": responseWithRetryAfter(http.StatusServiceUnavailable, "1"),
	} {
		t.Run(name, func(t *testing.T) {
			expected := []struct{ lower, upper time.Duration }{
				{1 * time.Second, 2 * time.Second},
				{2 * time.Second, 4 * time.Second},
				{4 * time.Second, 8 * time.Second},
				{8 * time.Second, 16 * time.Second},
				{15 * time.Second, 30 * time.Second},
				{15 * time.Second, 30 * time.Second},
				{15 * time.Second, 30 * time.Second},
			}
			for attemptNum, want := range expected {
				lowest, highest := sampleBackoff(attemptNum, resp)
				if lowest < want.lower || highest > want.upper {
					t.Errorf("attempt %d: waits ranged %v-%v, want within %v-%v", attemptNum, lowest, highest, want.lower, want.upper)
				}
				if highest-lowest < (want.upper-want.lower)/2 {
					t.Errorf("attempt %d: waits ranged only %v-%v, expected jitter across %v-%v", attemptNum, lowest, highest, want.lower, want.upper)
				}
			}
		})
	}
}

func TestRateLimitBackoffTreatsRetryAfterAsFloor(t *testing.T) {
	cases := []struct {
		name         string
		retryAfter   string
		lower, upper time.Duration
	}{
		{"integer seconds above the exponential wait", "20", 20 * time.Second, 30 * time.Second},
		{"fractional seconds", "2.5", 2500 * time.Millisecond, 5 * time.Second},
		{"longer than RetryWaitMax is honored as-is", "120", 120 * time.Second, 120 * time.Second},
		{"exactly RetryWaitMax", "30", testRetryWaitMax, testRetryWaitMax},
		{"HTTP date", time.Now().Add(10 * time.Second).UTC().Format(http.TimeFormat), 8 * time.Second, 20 * time.Second},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lowest, highest := sampleBackoff(0, responseWithRetryAfter(http.StatusTooManyRequests, tc.retryAfter))
			if lowest < tc.lower || highest > tc.upper {
				t.Errorf("waits ranged %v-%v, want within %v-%v", lowest, highest, tc.lower, tc.upper)
			}
		})
	}
}

func TestRateLimitBackoffIgnoresInvalidRetryAfter(t *testing.T) {
	for _, retryAfter := range []string{"soon", "-5", "NaN", "Inf", "1e300"} {
		lowest, highest := sampleBackoff(2, responseWithRetryAfter(http.StatusTooManyRequests, retryAfter))
		if lowest < 4*time.Second || highest > 8*time.Second {
			t.Errorf("Retry-After %q: waits ranged %v-%v, want the exponential 4s-8s", retryAfter, lowest, highest)
		}
	}
}

func TestRateLimitBackoffKeepsDefaultBackoffForOtherResponses(t *testing.T) {
	for name, resp := range map[string]*http.Response{
		"500":                           responseWithRetryAfter(http.StatusInternalServerError, ""),
		"502 with Retry-After: 20":      responseWithRetryAfter(http.StatusBadGateway, "20"),
		"no response (transport error)": nil,
	} {
		t.Run(name, func(t *testing.T) {
			for attemptNum := 0; attemptNum < 7; attemptNum++ {
				want := retryablehttp.DefaultBackoff(testRetryWaitMin, testRetryWaitMax, attemptNum, resp)
				if got := rateLimitBackoff(testRetryWaitMin, testRetryWaitMax, attemptNum, resp); got != want {
					t.Errorf("attempt %d: got %v, want DefaultBackoff's %v", attemptNum, got, want)
				}
			}
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	cases := map[string]struct {
		want time.Duration
		ok   bool
	}{
		"1":                             {time.Second, true},
		" 3 ":                           {3 * time.Second, true},
		"0.5":                           {500 * time.Millisecond, true},
		"0":                             {0, true},
		"1e300":                         {0, false},
		"Fri, 31 Dec 1999 23:59:59 GMT": {0, true},
		"":                              {0, false},
		"-1":                            {0, false},
		"tomorrow":                      {0, false},
	}
	for value, tc := range cases {
		got, ok := parseRetryAfter(value)
		if got != tc.want || ok != tc.ok {
			t.Errorf("parseRetryAfter(%q) = %v, %v; want %v, %v", value, got, ok, tc.want, tc.ok)
		}
	}
}

// timeTwoRateLimitedRetries returns how long a client takes to get through two "429, Retry-After: 1" responses.
func timeTwoRateLimitedRetries(t *testing.T, opts ...RetryableClientFactoryOption) time.Duration {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&requests, 1) <= 2 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewRetryableClientFactory(context.Background(), opts...).CreateRetryableClient()
	start := time.Now()
	resp, err := client.Get(server.URL)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || requests != 3 {
		t.Fatalf("expected success on the 3rd request, got status %d after %d requests", resp.StatusCode, requests)
	}
	return elapsed
}

func TestWithRateLimitBackoffBacksOffExponentiallyDespiteRetryAfter(t *testing.T) {
	// Honoring "Retry-After: 1" verbatim would wait 1s + 1s; the exponential floor waits at least 1s + 2s.
	if elapsed := timeTwoRateLimitedRetries(t, WithRateLimitBackoff()); elapsed < 3*time.Second {
		t.Fatalf("expected at least 3s of backoff across two 429s, waited %v", elapsed)
	}
}

func TestCreateRetryableClientKeepsDefaultBackoffWithoutOption(t *testing.T) {
	if elapsed := timeTwoRateLimitedRetries(t); elapsed >= 3*time.Second {
		t.Fatalf("expected clients without WithRateLimitBackoff to keep honoring Retry-After: 1 (about 2s), waited %v", elapsed)
	}
}

func TestConnectAPIMaxRetries(t *testing.T) {
	for configured, want := range map[int]int{4: 8, 6: 8, 8: 8, 12: 12} {
		if got := connectAPIMaxRetries(configured); got != want {
			t.Errorf("connectAPIMaxRetries(%d) = %d, want %d", configured, got, want)
		}
	}
}
