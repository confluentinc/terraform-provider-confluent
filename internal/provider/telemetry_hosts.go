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
	"strings"
	"sync"
)

// apiHosts records which hosts an operation's API requests went to, so reporting can
// be limited to operations that talked only to Confluent Cloud.
type apiHosts struct {
	mu    sync.Mutex
	seen  bool // at least one request was made
	other bool // at least one request went to a host outside confluent.cloud
}

type apiHostsKey struct{}

// withAPIHosts returns ctx carrying a new recorder for the requests made with it.
func withAPIHosts(ctx context.Context) (context.Context, *apiHosts) {
	h := &apiHosts{}
	return context.WithValue(ctx, apiHostsKey{}, h), h
}

// recordAPIHost notes host on the recorder carried by ctx, if any.
func recordAPIHost(ctx context.Context, host string) {
	h, ok := ctx.Value(apiHostsKey{}).(*apiHosts)
	if !ok || h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.seen = true
	if !isConfluentCloudHost(host) {
		h.other = true
	}
}

// onlyConfluentCloud reports whether at least one request was made and every request
// went to a Confluent Cloud host.
func (h *apiHosts) onlyConfluentCloud() bool {
	if h == nil {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.seen && !h.other
}

// isConfluentCloudHost reports whether host is confluent.cloud or one of its subdomains.
func isConfluentCloudHost(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	return host == "confluent.cloud" || strings.HasSuffix(host, ".confluent.cloud")
}
