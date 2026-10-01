// Copyright 2022 Confluent Inc. All Rights Reserved.
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
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	connectv1 "github.com/confluentinc/ccloud-sdk-go-v2/connect/v1"
)

const testConnectorListCacheKey = "env-123/lkc-123"

func testConnectorList(name string) connectorList {
	return connectorList{name: *connectv1.NewConnectV1ConnectorExpansionWithDefaults()}
}

func countingFetch(calls *int32, list connectorList, status int, err error) connectorListFetchFunc {
	return func() (connectorList, *http.Response, error) {
		atomic.AddInt32(calls, 1)
		return list, &http.Response{StatusCode: status}, err
	}
}

func TestConnectorListCacheReusesFreshEntry(t *testing.T) {
	cache := newConnectorListCache(time.Minute)
	var calls int32
	fetch := countingFetch(&calls, testConnectorList("a"), http.StatusOK, nil)

	for i := 0; i < 5; i++ {
		connectors, _, err := cache.get(testConnectorListCacheKey, fetch)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := connectors["a"]; !ok {
			t.Fatalf("expected connector %q in cached list", "a")
		}
	}
	if calls != 1 {
		t.Fatalf("expected 1 list call, got %d", calls)
	}
}

func TestConnectorListCacheSharesInFlightFetch(t *testing.T) {
	cache := newConnectorListCache(time.Minute)
	var calls int32
	release := make(chan struct{})
	fetch := func() (connectorList, *http.Response, error) {
		atomic.AddInt32(&calls, 1)
		<-release
		return testConnectorList("a"), &http.Response{StatusCode: http.StatusOK}, nil
	}

	const readers = 50
	var started, finished sync.WaitGroup
	started.Add(readers)
	finished.Add(readers)
	for i := 0; i < readers; i++ {
		go func() {
			started.Done()
			defer finished.Done()
			if _, _, err := cache.get(testConnectorListCacheKey, fetch); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	started.Wait()
	time.Sleep(50 * time.Millisecond)
	close(release)
	finished.Wait()

	if calls != 1 {
		t.Fatalf("expected %d concurrent readers to share 1 list call, got %d", readers, calls)
	}
}

func TestConnectorListCacheExpires(t *testing.T) {
	cache := newConnectorListCache(time.Minute)
	now := time.Now()
	cache.now = func() time.Time { return now }
	var calls int32
	fetch := countingFetch(&calls, testConnectorList("a"), http.StatusOK, nil)

	_, _, _ = cache.get(testConnectorListCacheKey, fetch)
	now = now.Add(time.Minute + time.Second)
	_, _, _ = cache.get(testConnectorListCacheKey, fetch)

	if calls != 2 {
		t.Fatalf("expected an expired entry to be refetched (2 calls), got %d", calls)
	}
}

func TestConnectorListCacheInvalidate(t *testing.T) {
	cache := newConnectorListCache(time.Minute)
	var calls int32
	fetch := countingFetch(&calls, testConnectorList("a"), http.StatusOK, nil)

	_, _, _ = cache.get(testConnectorListCacheKey, fetch)
	cache.invalidate(testConnectorListCacheKey)
	_, _, _ = cache.get(testConnectorListCacheKey, fetch)

	if calls != 2 {
		t.Fatalf("expected a refetch after invalidate (2 calls), got %d", calls)
	}
}

func TestConnectorListCacheInvalidateDuringFetchDoesNotStoreStaleList(t *testing.T) {
	cache := newConnectorListCache(time.Minute)
	release := make(chan struct{})
	staleFetch := func() (connectorList, *http.Response, error) {
		<-release
		return testConnectorList("stale"), &http.Response{StatusCode: http.StatusOK}, nil
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _, _ = cache.get(testConnectorListCacheKey, staleFetch)
	}()
	time.Sleep(50 * time.Millisecond)
	cache.invalidate(testConnectorListCacheKey)

	var calls int32
	connectors, _, _ := cache.get(testConnectorListCacheKey, countingFetch(&calls, testConnectorList("fresh"), http.StatusOK, nil))
	close(release)
	<-done

	if _, ok := connectors["fresh"]; !ok || calls != 1 {
		t.Fatalf("expected a read after invalidate to start its own fetch, got %v (calls=%d)", connectors, calls)
	}
	connectors, _, _ = cache.get(testConnectorListCacheKey, countingFetch(&calls, testConnectorList("unused"), http.StatusOK, nil))
	if _, ok := connectors["stale"]; ok {
		t.Fatal("a fetch started before invalidate must not repopulate the cache")
	}
}

func TestConnectorListCacheDoesNotCacheFailures(t *testing.T) {
	for name, tc := range map[string]struct {
		status int
		err    error
	}{
		"error":                   {http.StatusTooManyRequests, errors.New("429 Too Many Requests")},
		"forbidden without error": {http.StatusForbidden, nil},
	} {
		t.Run(name, func(t *testing.T) {
			cache := newConnectorListCache(time.Minute)
			var calls int32
			fetch := countingFetch(&calls, nil, tc.status, tc.err)

			_, resp, err := cache.get(testConnectorListCacheKey, fetch)
			if err != tc.err || resp.StatusCode != tc.status {
				t.Fatalf("expected the fetch result to be returned unchanged, got status=%d err=%v", resp.StatusCode, err)
			}
			_, _, _ = cache.get(testConnectorListCacheKey, fetch)
			if calls != 2 {
				t.Fatalf("expected failures not to be cached (2 calls), got %d", calls)
			}
		})
	}
}

func TestConnectorListCacheNilIsPassThrough(t *testing.T) {
	var cache *connectorListCache
	var calls int32
	fetch := countingFetch(&calls, testConnectorList("a"), http.StatusOK, nil)

	_, _, _ = cache.get(testConnectorListCacheKey, fetch)
	_, _, _ = cache.get(testConnectorListCacheKey, fetch)
	cache.invalidate(testConnectorListCacheKey)

	if calls != 2 {
		t.Fatalf("expected a nil cache to call fetch every time, got %d", calls)
	}
}
