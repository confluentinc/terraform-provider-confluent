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
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	connectv1 "github.com/confluentinc/ccloud-sdk-go-v2/connect/v1"
)

const testConnectorListCallKey = "env-123/lkc-123"

func testConnectorList(name string) connectorList {
	return connectorList{name: *connectv1.NewConnectV1ConnectorExpansionWithDefaults()}
}

func okConnectorListFetch(fetches *int32, release <-chan struct{}) connectorListFetchFunc {
	return func() (connectorList, *http.Response, error) {
		atomic.AddInt32(fetches, 1)
		if release != nil {
			<-release
		}
		return testConnectorList("a"), &http.Response{StatusCode: http.StatusOK}, nil
	}
}

func TestConnectorListCallsShareInFlightCall(t *testing.T) {
	calls := newConnectorListCalls()
	var fetches int32
	release := make(chan struct{})
	fetch := okConnectorListFetch(&fetches, release)

	const readers = 50
	var started, finished sync.WaitGroup
	started.Add(readers)
	finished.Add(readers)
	for i := 0; i < readers; i++ {
		go func() {
			defer finished.Done()
			started.Done()
			connectors, _, err := calls.do(context.Background(), testConnectorListCallKey, fetch)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if _, ok := connectors["a"]; !ok {
				t.Errorf("expected connector %q in the shared list", "a")
			}
		}()
	}
	started.Wait()
	time.Sleep(200 * time.Millisecond)
	close(release)
	finished.Wait()

	if fetches != 1 {
		t.Fatalf("expected %d concurrent readers to share 1 list call, got %d", readers, fetches)
	}
}

func TestConnectorListCallsKeepNothingAfterCallReturns(t *testing.T) {
	calls := newConnectorListCalls()
	var fetches int32
	fetch := okConnectorListFetch(&fetches, nil)

	for i := 0; i < 3; i++ {
		if _, _, err := calls.do(context.Background(), testConnectorListCallKey, fetch); err != nil {
			t.Fatalf("read %d: unexpected error: %v", i, err)
		}
	}
	if fetches != 3 {
		t.Fatalf("expected each sequential read to make its own list call (3), got %d", fetches)
	}
}

func TestConnectorListCallsKeyByCluster(t *testing.T) {
	calls := newConnectorListCalls()
	var fetches int32
	release := make(chan struct{})
	fetch := okConnectorListFetch(&fetches, release)

	var wg sync.WaitGroup
	for _, key := range []string{connectorListCallKey("env-1", "lkc-1"), connectorListCallKey("env-1", "lkc-2")} {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			_, _, _ = calls.do(context.Background(), key, fetch)
		}(key)
	}
	time.Sleep(200 * time.Millisecond)
	close(release)
	wg.Wait()

	if fetches != 2 {
		t.Fatalf("expected reads of different clusters not to share a call (2 calls), got %d", fetches)
	}
}

func TestConnectorListCallsJoinedCallerRetriesWhenSharedCallFails(t *testing.T) {
	calls := newConnectorListCalls()
	var fetches int32
	release := make(chan struct{})
	listErr := errors.New("429 Too Many Requests")
	fetch := func() (connectorList, *http.Response, error) {
		if atomic.AddInt32(&fetches, 1) == 1 {
			<-release
			return nil, &http.Response{StatusCode: http.StatusTooManyRequests}, listErr
		}
		return testConnectorList("a"), &http.Response{StatusCode: http.StatusOK}, nil
	}

	leaderErr := make(chan error, 1)
	go func() {
		_, _, err := calls.do(context.Background(), testConnectorListCallKey, fetch)
		leaderErr <- err
	}()
	time.Sleep(100 * time.Millisecond)
	joinerErr := make(chan error, 1)
	go func() {
		_, _, err := calls.do(context.Background(), testConnectorListCallKey, fetch)
		joinerErr <- err
	}()
	time.Sleep(100 * time.Millisecond)
	close(release)

	if err := <-leaderErr; err != listErr {
		t.Fatalf("expected the caller that made the failed call to get its error unchanged, got %v", err)
	}
	if err := <-joinerErr; err != nil {
		t.Fatalf("expected the joined caller to succeed on its own call, got %v", err)
	}
	if fetches != 2 {
		t.Fatalf("expected the failed call plus one call by the joined caller (2), got %d", fetches)
	}
}

func TestConnectorListCallsJoinedCallerHonorsItsOwnContext(t *testing.T) {
	calls := newConnectorListCalls()
	var fetches int32
	release := make(chan struct{})
	defer close(release)
	fetch := okConnectorListFetch(&fetches, release)
	go func() { _, _, _ = calls.do(context.Background(), testConnectorListCallKey, fetch) }()
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan error, 1)
	go func() {
		_, _, err := calls.do(ctx, testConnectorListCallKey, fetch)
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("a joined caller whose context is canceled must not wait for the in-flight call")
	}
	if n := atomic.LoadInt32(&fetches); n != 1 {
		t.Fatalf("expected a canceled joined caller not to make its own call (1 call), got %d", n)
	}
}

func TestConnectorListCallsNilIsPassThrough(t *testing.T) {
	var calls *connectorListCalls
	var fetches int32

	if _, _, err := calls.do(context.Background(), testConnectorListCallKey, okConnectorListFetch(&fetches, nil)); err != nil || fetches != 1 {
		t.Fatalf("expected a nil connectorListCalls to call fetch directly, got err=%v fetches=%d", err, fetches)
	}
}

func TestIsSuccessfulConnectorListResponse(t *testing.T) {
	cases := []struct {
		resp *http.Response
		err  error
		want bool
	}{
		{&http.Response{StatusCode: http.StatusOK}, nil, true},
		{&http.Response{StatusCode: http.StatusForbidden}, nil, false},
		{&http.Response{StatusCode: http.StatusTooManyRequests}, errors.New("429"), false},
		{nil, errors.New("connection reset"), false},
		{nil, nil, false},
	}
	for i, tc := range cases {
		if got := isSuccessfulConnectorListResponse(tc.resp, tc.err); got != tc.want {
			t.Errorf("case %d: got %v, want %v", i, got, tc.want)
		}
	}
}
