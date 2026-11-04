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
	"regexp"
	"strings"
	"sync"

	"github.com/hashicorp/go-retryablehttp"

	"github.com/confluentinc/terraform-provider-confluent/internal/provider/telemetry"
)

// apiRetries records how an operation's API requests were retried, so its usage
// event can say whether the retries recovered or ran out.
type apiRetries struct {
	mu               sync.Mutex
	retries          int    // attempts beyond the first, across all requests
	rateLimitRetries int    // retries made after an HTTP 429
	exhausted        bool   // at least one request gave up with no retries left
	status           int    // status that last triggered a retry of the reported request; 0 for a transport error
	endpoint         string // sanitized endpoint of the reported request (see retryEndpoint)
}

type apiRetriesKey struct{}

// withAPIRetries returns ctx carrying a new recorder for the requests made with it.
func withAPIRetries(ctx context.Context) (context.Context, *apiRetries) {
	r := &apiRetries{}
	return context.WithValue(ctx, apiRetriesKey{}, r), r
}

// record folds one finished request into the operation's totals. The reported
// endpoint and status are the first exhausted request's, or failing that the first
// retried request's.
func (r *apiRetries) record(req *http.Request, a *requestAttempts, exhausted bool) {
	if a.attempts <= 1 && !exhausted {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.retries += a.attempts - 1
	r.rateLimitRetries += a.rateLimitRetries
	if (exhausted && !r.exhausted) || r.endpoint == "" {
		r.status = a.retryStatus
		r.endpoint = retryEndpoint(req)
	}
	r.exhausted = r.exhausted || exhausted
}

// apply copies the totals onto u.
func (r *apiRetries) apply(u *telemetry.Usage) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	u.RetryCount = r.retries
	u.RateLimitRetryCount = r.rateLimitRetries
	u.RetriesExhausted = r.exhausted
	u.RetryStatusCode = r.status
	u.RetryEndpoint = r.endpoint
}

// requestAttempts follows one request through the retry loop. loggingTransport
// installs it and recordingRetryPolicy updates it after every attempt. The loop runs
// its attempts one after another, so it needs no lock.
type requestAttempts struct {
	attempts         int  // attempts completed so far
	rateLimitRetries int  // attempts made after an HTTP 429
	lastStatus       int  // status of the latest attempt; 0 for a transport error
	retryStatus      int  // status of the latest attempt the policy wanted to retry
	wantedRetry      bool // the retry policy asked to retry the latest attempt
}

type requestAttemptsKey struct{}

// withRequestAttempts returns req carrying a new attempt tracker, but only when the
// operation records retries, so requests made outside a wrapped operation are untouched.
func withRequestAttempts(req *http.Request) (*http.Request, *requestAttempts) {
	if _, ok := req.Context().Value(apiRetriesKey{}).(*apiRetries); !ok {
		return req, nil
	}
	a := &requestAttempts{}
	return req.WithContext(context.WithValue(req.Context(), requestAttemptsKey{}, a)), a
}

// finishRequest reports req's attempts to the operation's recorder. The retry loop
// stops while still wanting a retry either because it ran out (attempts past
// retryMax) or because the context ended during a backoff wait (fewer attempts), so
// only the first counts as exhausted.
func finishRequest(req *http.Request, a *requestAttempts, retryMax int) {
	if a == nil {
		return
	}
	r, ok := req.Context().Value(apiRetriesKey{}).(*apiRetries)
	if !ok || r == nil {
		return
	}
	r.record(req, a, a.wantedRetry && a.attempts > retryMax)
}

// recordingRetryPolicy wraps policy to note each attempt's outcome on the request's
// tracker. The client sets no response handler, so the policy runs once per attempt.
func recordingRetryPolicy(policy retryablehttp.CheckRetry) retryablehttp.CheckRetry {
	return func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		retry, checkErr := policy(ctx, resp, err)
		if a, ok := ctx.Value(requestAttemptsKey{}).(*requestAttempts); ok && a != nil {
			// An attempt after the first is a retry of the one before it.
			if a.attempts > 0 && a.lastStatus == http.StatusTooManyRequests {
				a.rateLimitRetries++
			}
			a.attempts++
			a.lastStatus = 0
			if resp != nil {
				a.lastStatus = resp.StatusCode
			}
			a.wantedRetry = retry
			if retry {
				a.retryStatus = a.lastStatus
			}
		}
		return retry, checkErr
	}
}

// apiGroups are the Confluent Cloud API groups the provider calls, taken from the paths of the
// ccloud-sdk-go-v2 clients it uses. Each starts paths of the form /<group>/<version>/<collection>, with a
// fixed collection name in third place. A path whose first segment is any other value, such as Schema
// Registry's /subjects/<subject>, may hold a user's name there, so it is reported as "other". A group added
// later is also reported as "other" until it's listed here.
var apiGroups = map[string]bool{
	"artifact": true, "byok": true, "cam": true, "catalog": true, "ccpm": true, "cmk": true, "connect": true,
	"dek-registry": true, "endpoint": true, "fcpm": true, "iam": true, "kafka": true, "kafka-quotas": true,
	"ksqldbcm": true, "networking": true, "notifications": true, "org": true, "pim": true, "rtce": true,
	"sql": true, "srcm": true, "sts": true, "switchover": true, "tableflow": true,
}

var (
	apiVersionSegment = regexp.MustCompile(`^v[0-9]+((alpha|beta)[0-9]+)?$`)
	collectionSegment = regexp.MustCompile(`^[a-z]+(-[a-z]+)*$`)
)

// retryEndpoint names req's endpoint without any value from its URL: the method, then the API group,
// version and top-level collection, as in "GET connect/v1/environments". IDs and names only appear after
// the collection, and the query string is dropped. A path outside the known API groups (see apiGroups) is
// reported as "other" rather than risk reporting a name.
func retryEndpoint(req *http.Request) string {
	method := strings.ToUpper(req.Method)
	if req.URL == nil {
		return method + " other"
	}
	segs := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
	if len(segs) < 2 || !apiGroups[segs[0]] || !apiVersionSegment.MatchString(segs[1]) {
		return method + " other"
	}
	endpoint := segs[0] + "/" + segs[1]
	if len(segs) > 2 && collectionSegment.MatchString(segs[2]) {
		endpoint += "/" + segs[2]
	}
	return method + " " + endpoint
}
