package provider

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
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
		"503 without Retry-After": responseWithRetryAfter(http.StatusServiceUnavailable, ""),
		"429 with Retry-After: 0": responseWithRetryAfter(http.StatusTooManyRequests, "0"),
		"429 with Retry-After in the past": responseWithRetryAfter(http.StatusTooManyRequests,
			time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat)),
	} {
		t.Run(name, func(t *testing.T) {
			checkJitteredSchedule(t, resp)
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
		{"longer than RetryWaitMax gets up to 20% extra", "120", 120 * time.Second, 144 * time.Second},
		{"exactly RetryWaitMax gets up to 20% extra", "30", testRetryWaitMax, 36 * time.Second},
		{"HTTP date", time.Now().Add(10 * time.Second).UTC().Format(http.TimeFormat), 8 * time.Second, 20 * time.Second},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lowest, highest := sampleBackoff(0, responseWithRetryAfter(http.StatusTooManyRequests, tc.retryAfter))
			if lowest < tc.lower || highest > tc.upper {
				t.Errorf("waits ranged %v-%v, want within %v-%v", lowest, highest, tc.lower, tc.upper)
			}
			if highest-lowest < (tc.upper-tc.lower)/2 {
				t.Errorf("waits ranged only %v-%v, expected jitter across %v-%v", lowest, highest, tc.lower, tc.upper)
			}
		})
	}
}

func TestRateLimitBackoffIgnoresInvalidRetryAfter(t *testing.T) {
	for _, retryAfter := range []string{"soon", "-5", "NaN", "Inf", "1e300"} {
		t.Run(retryAfter, func(t *testing.T) {
			checkJitteredSchedule(t, responseWithRetryAfter(http.StatusTooManyRequests, retryAfter))
		})
	}
}

// checkJitteredSchedule checks every attempt's waits: within 1-2s, 1.6-3.2s, 2.56-5.12s, 4.1-8.2s, 6.6-13.1s,
// 10.5-21s, then 15-30s, and spread out.
func checkJitteredSchedule(t *testing.T, resp *http.Response) {
	t.Helper()
	expected := []struct{ lower, upper time.Duration }{
		{1 * time.Second, 2 * time.Second},
		{1600 * time.Millisecond, 3200 * time.Millisecond},
		{2560 * time.Millisecond, 5120 * time.Millisecond},
		{4096 * time.Millisecond, 8192 * time.Millisecond},
		{6553600 * time.Microsecond, 13107200 * time.Microsecond},
		{10485760 * time.Microsecond, 20971520 * time.Microsecond},
		{15 * time.Second, 30 * time.Second},
		{15 * time.Second, 30 * time.Second},
		{15 * time.Second, 30 * time.Second},
		{15 * time.Second, 30 * time.Second},
		{15 * time.Second, 30 * time.Second},
		{15 * time.Second, 30 * time.Second},
	}
	for attemptNum, want := range expected {
		lowest, highest := sampleBackoff(attemptNum, resp)
		if lowest < want.lower || highest > want.upper {
			t.Errorf("attempt %d: waits ranged %v-%v, want within %v-%v", attemptNum, lowest, highest, want.lower, want.upper)
		}
		if highest-lowest < (want.upper-want.lower)*9/10 {
			t.Errorf("attempt %d: waits ranged only %v-%v, expected jitter across %v-%v", attemptNum, lowest, highest, want.lower, want.upper)
		}
	}
}

func TestRateLimitBackoffJittersOtherResponses(t *testing.T) {
	for name, resp := range map[string]*http.Response{
		"500":                           responseWithRetryAfter(http.StatusInternalServerError, ""),
		"502 with Retry-After: 20":      responseWithRetryAfter(http.StatusBadGateway, "20"),
		"no response (transport error)": nil,
	} {
		t.Run(name, func(t *testing.T) {
			for attemptNum := 0; attemptNum < 7; attemptNum++ {
				checkUpTo20PercentJitter(t, rateLimitBackoff, attemptNum, resp)
			}
		})
	}
}

// timeTwoRateLimitedRetries returns how long a client takes to get through two "429, Retry-After: 1" responses.
func timeTwoRateLimitedRetries(t *testing.T) time.Duration {
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

	client := NewRetryableClientFactory(context.Background()).CreateRetryableClient()
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

func TestCreateRetryableClientBacksOffExponentiallyDespiteRetryAfter(t *testing.T) {
	// Honoring "Retry-After: 1" verbatim would wait 1s + 1s; the exponential floor waits at least 1s + 1.6s.
	if elapsed := timeTwoRateLimitedRetries(t); elapsed < 2600*time.Millisecond {
		t.Fatalf("expected at least 2.6s of backoff across two 429s, waited %v", elapsed)
	}
}

func TestRetryBackoffGrowsByFactorWithoutRetryAfter(t *testing.T) {
	// 1s × 1.6^attempt, capped at 30s.
	want := []time.Duration{
		1 * time.Second,
		1600 * time.Millisecond,
		2560 * time.Millisecond,
		4096 * time.Millisecond,
		6553600 * time.Microsecond,
		10485760 * time.Microsecond,
		16777216 * time.Microsecond,
		26843545600 * time.Nanosecond,
		30 * time.Second,
		30 * time.Second,
		30 * time.Second,
		30 * time.Second,
	}
	for name, resp := range map[string]*http.Response{
		"500":                           responseWithRetryAfter(http.StatusInternalServerError, ""),
		"429 without Retry-After":       responseWithRetryAfter(http.StatusTooManyRequests, ""),
		"no response (transport error)": nil,
	} {
		t.Run(name, func(t *testing.T) {
			for attemptNum, w := range want {
				if got := retryBackoff(testRetryWaitMin, testRetryWaitMax, attemptNum, resp); got != w {
					t.Errorf("attempt %d: got %v, want %v", attemptNum, got, w)
				}
			}
		})
	}
}

func TestRetryBackoffHonorsRetryAfter(t *testing.T) {
	cases := []struct {
		name       string
		resp       *http.Response
		attemptNum int
		want       time.Duration
	}{
		{"429 with Retry-After: 1", responseWithRetryAfter(http.StatusTooManyRequests, "1"), 5, 1 * time.Second},
		{"429 with Retry-After: 0", responseWithRetryAfter(http.StatusTooManyRequests, "0"), 5, 0},
		{"503 with Retry-After: 1", responseWithRetryAfter(http.StatusServiceUnavailable, "1"), 5, 1 * time.Second},
		{"longer than RetryWaitMax is honored as-is", responseWithRetryAfter(http.StatusTooManyRequests, "120"), 0, 120 * time.Second},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := retryBackoff(testRetryWaitMin, testRetryWaitMax, tc.attemptNum, tc.resp); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}

	t.Run("invalid Retry-After falls back to the exponential wait", func(t *testing.T) {
		resp := responseWithRetryAfter(http.StatusTooManyRequests, "soon")
		if got := retryBackoff(testRetryWaitMin, testRetryWaitMax, 3, resp); got != 4096*time.Millisecond {
			t.Errorf("got %v, want %v", got, 4096*time.Millisecond)
		}
	})
}

func TestCreateRetryableClientBacksOffByFactorOnServerErrors(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&requests, 1) <= 3 {
			// A 500 takes jitteredBackoff: retryBackoff's exponential wait plus up to 20%.
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewRetryableClientFactory(context.Background()).CreateRetryableClient()
	start := time.Now()
	resp, err := client.Get(server.URL)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || requests != 4 {
		t.Fatalf("expected success on the 4th request, got status %d after %d requests", resp.StatusCode, requests)
	}
	// 1s + 1.6s + 2.56s = 5.16s with the 1.6x factor, plus up to 20% (6.19s); DefaultBackoff's 2x would take at least 7s.
	if elapsed < 5100*time.Millisecond || elapsed >= 6900*time.Millisecond {
		t.Errorf("expected 5.16-6.19s of backoff across three 500s, waited %v", elapsed)
	}
}
