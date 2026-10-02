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

func TestConnectorListCallsShareInFlightCall(t *testing.T) {
	calls := newConnectorListCalls()
	var fetches, joinedCount int32
	release := make(chan struct{})
	fetch := func() (connectorList, *http.Response, error) {
		atomic.AddInt32(&fetches, 1)
		<-release
		return testConnectorList("a"), &http.Response{StatusCode: http.StatusOK}, nil
	}

	const readers = 50
	var started, finished sync.WaitGroup
	started.Add(readers)
	finished.Add(readers)
	for i := 0; i < readers; i++ {
		go func() {
			defer finished.Done()
			started.Done()
			connectors, _, joined, err := calls.do(context.Background(), testConnectorListCallKey, fetch)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if _, ok := connectors["a"]; !ok {
				t.Errorf("expected connector %q in the shared list", "a")
			}
			if joined {
				atomic.AddInt32(&joinedCount, 1)
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
	if joinedCount != readers-1 {
		t.Fatalf("expected %d readers to report joining the in-flight call, got %d", readers-1, joinedCount)
	}
}

func TestConnectorListCallsKeepNothingAfterCallReturns(t *testing.T) {
	calls := newConnectorListCalls()
	var fetches int32
	fetch := func() (connectorList, *http.Response, error) {
		atomic.AddInt32(&fetches, 1)
		return testConnectorList("a"), &http.Response{StatusCode: http.StatusOK}, nil
	}

	for i := 0; i < 3; i++ {
		if _, _, joined, _ := calls.do(context.Background(), testConnectorListCallKey, fetch); joined {
			t.Fatalf("read %d: a sequential read must not reuse a completed call", i)
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
	fetch := func() (connectorList, *http.Response, error) {
		atomic.AddInt32(&fetches, 1)
		<-release
		return testConnectorList("a"), &http.Response{StatusCode: http.StatusOK}, nil
	}

	var wg sync.WaitGroup
	for _, key := range []string{connectorListCallKey("env-1", "lkc-1"), connectorListCallKey("env-1", "lkc-2")} {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			_, _, _, _ = calls.do(context.Background(), key, fetch)
		}(key)
	}
	time.Sleep(200 * time.Millisecond)
	close(release)
	wg.Wait()

	if fetches != 2 {
		t.Fatalf("expected reads of different clusters not to share a call (2 calls), got %d", fetches)
	}
}

func TestConnectorListCallsPassFailuresToJoinedCallers(t *testing.T) {
	calls := newConnectorListCalls()
	release := make(chan struct{})
	listErr := errors.New("429 Too Many Requests")
	fetch := func() (connectorList, *http.Response, error) {
		<-release
		return nil, &http.Response{StatusCode: http.StatusTooManyRequests}, listErr
	}

	type result struct {
		resp   *http.Response
		joined bool
		err    error
	}
	results := make(chan result, 2)
	for i := 0; i < 2; i++ {
		go func() {
			_, resp, joined, err := calls.do(context.Background(), testConnectorListCallKey, fetch)
			results <- result{resp, joined, err}
		}()
	}
	time.Sleep(200 * time.Millisecond)
	close(release)

	joinedSeen := false
	for i := 0; i < 2; i++ {
		r := <-results
		if r.err != listErr || r.resp.StatusCode != http.StatusTooManyRequests {
			t.Fatalf("expected the failure to be returned unchanged, got status=%d err=%v", r.resp.StatusCode, r.err)
		}
		joinedSeen = joinedSeen || r.joined
	}
	if !joinedSeen {
		t.Fatal("expected one caller to report joining the failed call, so it can retry on its own")
	}
}

func TestConnectorListCallsNilIsPassThrough(t *testing.T) {
	var calls *connectorListCalls
	var fetches int32
	fetch := func() (connectorList, *http.Response, error) {
		atomic.AddInt32(&fetches, 1)
		return testConnectorList("a"), &http.Response{StatusCode: http.StatusOK}, nil
	}

	_, _, joined, _ := calls.do(context.Background(), testConnectorListCallKey, fetch)
	if joined || fetches != 1 {
		t.Fatalf("expected a nil connectorListCalls to call fetch directly, got joined=%v fetches=%d", joined, fetches)
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

func TestConnectorListCallsJoinedCallerHonorsItsOwnContext(t *testing.T) {
	calls := newConnectorListCalls()
	release := make(chan struct{})
	defer close(release)
	fetch := func() (connectorList, *http.Response, error) {
		<-release
		return testConnectorList("a"), &http.Response{StatusCode: http.StatusOK}, nil
	}
	go func() { _, _, _, _ = calls.do(context.Background(), testConnectorListCallKey, fetch) }()
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan error, 1)
	go func() {
		_, _, joined, err := calls.do(ctx, testConnectorListCallKey, fetch)
		if !joined {
			err = errors.New("expected to join the in-flight call")
		}
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
}
