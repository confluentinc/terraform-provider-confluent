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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	connectv1 "github.com/confluentinc/ccloud-sdk-go-v2/connect/v1"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	listCacheTestEnvironmentId = "env-abc123"
	listCacheTestClusterId     = "lkc-abc123"
	listCacheTestConnectors    = 10
)

func listCacheTestConnectorName(i int) string {
	return fmt.Sprintf("connector_%d", i)
}

func newListCacheTestServer(t *testing.T, listCalls *int32) *httptest.Server {
	connectors := map[string]interface{}{}
	for i := 0; i < listCacheTestConnectors; i++ {
		name := listCacheTestConnectorName(i)
		connectors[name] = map[string]interface{}{
			"id":     map[string]string{"id": fmt.Sprintf("lcc-%d", i), "id_type": "ID"},
			"info":   map[string]interface{}{"name": name, "type": "source", "config": map[string]string{"name": name, "connector.class": "DatagenSource"}},
			"status": map[string]interface{}{"name": name, "type": "source", "connector": map[string]string{"state": "RUNNING", "worker_id": name}},
		}
	}
	body, err := json.Marshal(connectors)
	if err != nil {
		t.Fatal(err)
	}
	listPath := fmt.Sprintf("/connect/v1/environments/%s/clusters/%s/connectors", listCacheTestEnvironmentId, listCacheTestClusterId)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != listPath {
			http.NotFound(w, r)
			return
		}
		atomic.AddInt32(listCalls, 1)
		// Keep the call in flight long enough for concurrent reads to overlap it.
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
}

func newListCacheTestClient(serverUrl string) *Client {
	cfg := connectv1.NewConfiguration()
	cfg.Servers[0].URL = serverUrl
	return &Client{
		connectV1Client:    connectv1.NewAPIClient(cfg),
		connectorListCache: newConnectorListCache(connectorListCacheTTL),
		cloudApiKey:        "key",
		cloudApiSecret:     "secret",
	}
}

func newListCacheTestResourceData(t *testing.T, name string) *schema.ResourceData {
	d := schema.TestResourceDataRaw(t, connectorResource().Schema, map[string]interface{}{
		paramEnvironment:        []interface{}{map[string]interface{}{paramId: listCacheTestEnvironmentId}},
		paramKafkaCluster:       []interface{}{map[string]interface{}{paramId: listCacheTestClusterId}},
		paramNonSensitiveConfig: map[string]interface{}{connectorConfigAttributeName: name, connectorConfigAttributeClass: "DatagenSource"},
	})
	d.SetId("lcc-placeholder")
	return d
}

func TestConnectorRefreshSharesOneListCallPerCluster(t *testing.T) {
	var listCalls int32
	server := newListCacheTestServer(t, &listCalls)
	defer server.Close()
	client := newListCacheTestClient(server.URL)

	var wg sync.WaitGroup
	resourceData := make([]*schema.ResourceData, listCacheTestConnectors)
	for i := 0; i < listCacheTestConnectors; i++ {
		resourceData[i] = newListCacheTestResourceData(t, listCacheTestConnectorName(i))
		wg.Add(1)
		go func(d *schema.ResourceData) {
			defer wg.Done()
			if diags := connectorRead(context.Background(), d, client); diags.HasError() {
				t.Errorf("unexpected read error: %v", diags)
			}
		}(resourceData[i])
	}
	wg.Wait()

	for i, d := range resourceData {
		if got, want := d.Id(), fmt.Sprintf("lcc-%d", i); got != want {
			t.Errorf("connector %d: expected id %q, got %q", i, want, got)
		}
		if got := d.Get(paramStatus).(string); got != "RUNNING" {
			t.Errorf("connector %d: expected status RUNNING, got %q", i, got)
		}
	}
	if listCalls != 1 {
		t.Fatalf("expected %d concurrent refreshes to share 1 list call, got %d", listCacheTestConnectors, listCalls)
	}

	if diags := connectorRead(context.Background(), newListCacheTestResourceData(t, listCacheTestConnectorName(0)), client); diags.HasError() {
		t.Fatalf("unexpected read error: %v", diags)
	}
	if listCalls != 1 {
		t.Fatalf("expected a later refresh to reuse the cached list, got %d list calls", listCalls)
	}
}

func TestConnectorReadOfNewResourceSkipsCache(t *testing.T) {
	var listCalls int32
	server := newListCacheTestServer(t, &listCalls)
	defer server.Close()
	client := newListCacheTestClient(server.URL)

	for i := 0; i < 2; i++ {
		d := newListCacheTestResourceData(t, listCacheTestConnectorName(0))
		d.MarkNewResource()
		if _, err := readConnectorAndSetAttributes(context.Background(), d, client, listCacheTestConnectorName(0), listCacheTestEnvironmentId, listCacheTestClusterId); err != nil {
			t.Fatalf("unexpected read error: %v", err)
		}
	}
	if listCalls != 2 {
		t.Fatalf("expected create/import reads to always fetch a fresh list (2 calls), got %d", listCalls)
	}
}
