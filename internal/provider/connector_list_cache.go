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
	"fmt"
	"net/http"
	"sync"
	"time"

	connectv1 "github.com/confluentinc/ccloud-sdk-go-v2/connect/v1"
)

type connectorList = map[string]connectv1.ConnectV1ConnectorExpansion

type connectorListFetchFunc func() (connectorList, *http.Response, error)

type connectorListCacheEntry struct {
	connectors connectorList
	expiresAt  time.Time
}

type connectorListCall struct {
	done       chan struct{}
	connectors connectorList
	resp       *http.Response
	err        error
}

// connectorListCache shares one list-connectors response per Kafka cluster across all
// confluent_connector reads in a single Terraform run, so refreshing N connectors costs one
// list call instead of N.
type connectorListCache struct {
	mu       sync.Mutex
	ttl      time.Duration
	now      func() time.Time
	entries  map[string]connectorListCacheEntry
	inflight map[string]*connectorListCall
}

func newConnectorListCache(ttl time.Duration) *connectorListCache {
	return &connectorListCache{
		ttl:      ttl,
		now:      time.Now,
		entries:  make(map[string]connectorListCacheEntry),
		inflight: make(map[string]*connectorListCall),
	}
}

func connectorListCacheKey(environmentId, clusterId string) string {
	return fmt.Sprintf("%s/%s", environmentId, clusterId)
}

// get returns a fresh cached list, joins an in-flight fetch for the same key, or runs fetch.
// Only successful responses are cached; errors are returned to every caller that shared the fetch.
func (lc *connectorListCache) get(key string, fetch connectorListFetchFunc) (connectorList, *http.Response, error) {
	if lc == nil {
		return fetch()
	}

	lc.mu.Lock()
	if entry, ok := lc.entries[key]; ok && lc.now().Before(entry.expiresAt) {
		lc.mu.Unlock()
		return entry.connectors, &http.Response{StatusCode: http.StatusOK}, nil
	}
	if call, ok := lc.inflight[key]; ok {
		lc.mu.Unlock()
		<-call.done
		return call.connectors, call.resp, call.err
	}
	call := &connectorListCall{done: make(chan struct{})}
	lc.inflight[key] = call
	lc.mu.Unlock()

	call.connectors, call.resp, call.err = fetch()

	lc.mu.Lock()
	// invalidate() detaches the call, so a fetch that started before a write never repopulates the cache.
	if lc.inflight[key] == call {
		delete(lc.inflight, key)
		if call.err == nil && call.resp != nil && call.resp.StatusCode < http.StatusMultipleChoices {
			lc.entries[key] = connectorListCacheEntry{connectors: call.connectors, expiresAt: lc.now().Add(lc.ttl)}
		}
	}
	lc.mu.Unlock()
	close(call.done)

	return call.connectors, call.resp, call.err
}

func (lc *connectorListCache) invalidate(key string) {
	if lc == nil {
		return
	}
	lc.mu.Lock()
	delete(lc.entries, key)
	delete(lc.inflight, key)
	lc.mu.Unlock()
}
