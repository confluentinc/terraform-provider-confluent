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
	"fmt"
	"net/http"
	"sync"

	connectv1 "github.com/confluentinc/ccloud-sdk-go-v2/connect/v1"
)

type connectorList = map[string]connectv1.ConnectV1ConnectorExpansion

type connectorListFetchFunc func() (connectorList, *http.Response, error)

type connectorListCall struct {
	done       chan struct{}
	connectors connectorList
	resp       *http.Response
	err        error
}

// connectorListCalls lets concurrent refreshes of one cluster share an in-flight list call; nothing is kept after it returns.
type connectorListCalls struct {
	mu       sync.Mutex
	inflight map[string]*connectorListCall
}

func newConnectorListCalls() *connectorListCalls {
	return &connectorListCalls{inflight: make(map[string]*connectorListCall)}
}

func connectorListCallKey(environmentId, clusterId string) string {
	return fmt.Sprintf("%s/%s", environmentId, clusterId)
}

// do runs fetch or joins the one in flight for key (joined); the returned list is shared, so treat it as read-only.
func (g *connectorListCalls) do(ctx context.Context, key string, fetch connectorListFetchFunc) (connectors connectorList, resp *http.Response, joined bool, err error) {
	if g == nil {
		connectors, resp, err = fetch()
		return connectors, resp, false, err
	}

	g.mu.Lock()
	if call, ok := g.inflight[key]; ok {
		g.mu.Unlock()
		select {
		case <-call.done:
			return call.connectors, call.resp, true, call.err
		case <-ctx.Done():
			return nil, nil, true, ctx.Err()
		}
	}
	call := &connectorListCall{done: make(chan struct{})}
	g.inflight[key] = call
	g.mu.Unlock()

	call.connectors, call.resp, call.err = fetch()

	g.mu.Lock()
	delete(g.inflight, key)
	g.mu.Unlock()
	close(call.done)

	return call.connectors, call.resp, false, call.err
}

func isSuccessfulConnectorListResponse(resp *http.Response, err error) bool {
	return err == nil && resp != nil && resp.StatusCode < http.StatusMultipleChoices
}
