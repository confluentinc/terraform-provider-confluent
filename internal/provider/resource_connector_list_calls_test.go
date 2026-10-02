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
	listCallsTestEnvironmentId = "env-abc123"
	listCallsTestClusterId     = "lkc-abc123"
	listCallsTestConnectors    = 10
)

func listCallsTestConnectorName(i int) string {
	return fmt.Sprintf("connector_%d", i)
}

// listCallsTestServer serves the list-connectors endpoint, holding each response for delay so that
// concurrent reads overlap. If failFirst is set, the first request gets a 429.
type listCallsTestServer struct {
	*httptest.Server
	calls int32
}

func newListCallsTestServer(t *testing.T, delay time.Duration, failFirst bool) *listCallsTestServer {
	connectors := map[string]interface{}{}
	for i := 0; i < listCallsTestConnectors; i++ {
		name := listCallsTestConnectorName(i)
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
	listPath := fmt.Sprintf("/connect/v1/environments/%s/clusters/%s/connectors", listCallsTestEnvironmentId, listCallsTestClusterId)
	s := &listCallsTestServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != listPath {
			http.NotFound(w, r)
			return
		}
		n := atomic.AddInt32(&s.calls, 1)
		time.Sleep(delay)
		w.Header().Set("Content-Type", "application/json")
		if failFirst && n == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"errors":[{"status":"429","detail":"Exceeded rate limit"}]}`))
			return
		}
		_, _ = w.Write(body)
	}))
	return s
}

func newListCallsTestClient(serverUrl string) *Client {
	cfg := connectv1.NewConfiguration()
	cfg.Servers[0].URL = serverUrl
	return &Client{
		connectV1Client:    connectv1.NewAPIClient(cfg),
		connectorListCalls: newConnectorListCalls(),
		cloudApiKey:        "key",
		cloudApiSecret:     "secret",
	}
}

func newListCallsTestResourceData(t *testing.T, name string) *schema.ResourceData {
	d := schema.TestResourceDataRaw(t, connectorResource().Schema, map[string]interface{}{
		paramEnvironment:        []interface{}{map[string]interface{}{paramId: listCallsTestEnvironmentId}},
		paramKafkaCluster:       []interface{}{map[string]interface{}{paramId: listCallsTestClusterId}},
		paramNonSensitiveConfig: map[string]interface{}{connectorConfigAttributeName: name, connectorConfigAttributeClass: "DatagenSource"},
	})
	d.SetId("lcc-placeholder")
	return d
}

func TestConnectorRefreshesShareOneInFlightListCall(t *testing.T) {
	server := newListCallsTestServer(t, 500*time.Millisecond, false)
	defer server.Close()
	client := newListCallsTestClient(server.URL)

	var wg sync.WaitGroup
	resourceData := make([]*schema.ResourceData, listCallsTestConnectors)
	for i := range resourceData {
		resourceData[i] = newListCallsTestResourceData(t, listCallsTestConnectorName(i))
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
	if server.calls != 1 {
		t.Fatalf("expected %d concurrent refreshes to share 1 list call, got %d", listCallsTestConnectors, server.calls)
	}
}

func TestConnectorSequentialRefreshesEachMakeTheirOwnListCall(t *testing.T) {
	server := newListCallsTestServer(t, 0, false)
	defer server.Close()
	client := newListCallsTestClient(server.URL)

	for i := 0; i < 3; i++ {
		if diags := connectorRead(context.Background(), newListCallsTestResourceData(t, listCallsTestConnectorName(i)), client); diags.HasError() {
			t.Fatalf("unexpected read error: %v", diags)
		}
	}
	if server.calls != 3 {
		t.Fatalf("expected nothing to be kept between calls (3 list calls), got %d", server.calls)
	}
}

func TestConnectorReadAfterWriteDoesNotJoinInFlightListCall(t *testing.T) {
	server := newListCallsTestServer(t, 500*time.Millisecond, false)
	defer server.Close()
	client := newListCallsTestClient(server.URL)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_ = connectorRead(context.Background(), newListCallsTestResourceData(t, listCallsTestConnectorName(0)), client)
	}()
	time.Sleep(100 * time.Millisecond)
	go func() {
		defer wg.Done()
		// The read at the end of connectorCreate/connectorUpdate.
		if diags := readConnector(context.Background(), newListCallsTestResourceData(t, listCallsTestConnectorName(1)), client, false); diags.HasError() {
			t.Errorf("unexpected read error: %v", diags)
		}
	}()
	wg.Wait()

	if server.calls != 2 {
		t.Fatalf("expected the read after a write to make its own list call (2 calls), got %d", server.calls)
	}
}

func TestConnectorRefreshRetriesOnItsOwnWhenSharedListCallFails(t *testing.T) {
	server := newListCallsTestServer(t, 500*time.Millisecond, true)
	defer server.Close()
	client := newListCallsTestClient(server.URL)

	leaderDone := make(chan bool, 1)
	go func() {
		leaderDone <- connectorRead(context.Background(), newListCallsTestResourceData(t, listCallsTestConnectorName(0)), client).HasError()
	}()
	time.Sleep(100 * time.Millisecond)

	const joiners = 4
	var wg sync.WaitGroup
	for i := 1; i <= joiners; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			d := newListCallsTestResourceData(t, listCallsTestConnectorName(i))
			if diags := connectorRead(context.Background(), d, client); diags.HasError() {
				t.Errorf("connector %d: a read that joined a failed call should succeed on its own call, got %v", i, diags)
			}
		}(i)
	}
	wg.Wait()

	if leaderFailed := <-leaderDone; !leaderFailed {
		t.Fatal("expected the read that made the 429'd call to report the error, as it does today")
	}
	if server.calls < 2 || server.calls > joiners+1 {
		t.Fatalf("expected the failed call plus the joiners' own retry calls (2-%d), got %d", joiners+1, server.calls)
	}
}
