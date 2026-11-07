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
	"sync/atomic"

	connectv1 "github.com/confluentinc/ccloud-sdk-go-v2/connect/v1"
	"golang.org/x/sync/singleflight"
)

type connectorList = map[string]connectv1.ConnectV1ConnectorExpansion

type connectorListFetchFunc func() (connectorList, *http.Response, error)

type connectorListResult struct {
	connectors connectorList
	resp       *http.Response
}

// connectorListCalls lets concurrent refreshes of one cluster share an in-flight list call; nothing is kept after it returns.
type connectorListCalls struct {
	group singleflight.Group
}

func newConnectorListCalls() *connectorListCalls {
	return &connectorListCalls{}
}

func connectorListCallKey(environmentId, clusterId string) string {
	return fmt.Sprintf("%s/%s", environmentId, clusterId)
}

// do returns fetch's result, sharing one call among concurrent callers with the same key. A caller that joined a call
// that failed makes its own call (with its own retry budget), unless its context has ended. The caller that made the
// failed call returns its error as is. The returned list is shared, so treat it as read-only.
func (g *connectorListCalls) do(ctx context.Context, key string, fetch connectorListFetchFunc) (connectorList, *http.Response, error) {
	if g == nil {
		return fetch()
	}

	// singleflight runs fn only for the caller that starts the call, which tells that caller apart from those that joined.
	var ran atomic.Bool
	ch := g.group.DoChan(key, func() (interface{}, error) {
		ran.Store(true)
		connectors, resp, err := fetch()
		return connectorListResult{connectors, resp}, err
	})
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case r := <-ch:
		result := r.Val.(connectorListResult)
		if !ran.Load() && !isSuccessfulConnectorListResponse(result.resp, r.Err) && ctx.Err() == nil {
			return fetch()
		}
		return result.connectors, result.resp, r.Err
	}
}

func isSuccessfulConnectorListResponse(resp *http.Response, err error) bool {
	return err == nil && resp != nil && resp.StatusCode < http.StatusMultipleChoices
}
