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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	connectv1 "github.com/confluentinc/ccloud-sdk-go-v2/connect/v1"
)

func TestWithoutUnsetKafkaAuthConfigs(t *testing.T) {
	config := map[string]string{
		"connector.class":                  "DatagenSource",
		"kafka.auth.mode":                  "KAFKA_API_KEY",
		"kafka.api.key":                    "key",
		"kafka.api.secret":                 "secret",
		"kafka.service.account.id":         "sa-123",
		"kafka.service.account.api.key":    "sa-key",
		"kafka.service.account.api.secret": "sa-secret",
		"kafka.topic":                      "topic",
	}

	t.Run("drops every Kafka auth config the user did not set", func(t *testing.T) {
		require.Equal(t, map[string]string{"connector.class": "DatagenSource", "kafka.topic": "topic"},
			withoutUnsetKafkaAuthConfigs(config, map[string]interface{}{"connector.class": "DatagenSource"}))
	})

	t.Run("keeps the Kafka auth configs the user set in config_nonsensitive", func(t *testing.T) {
		filtered := withoutUnsetKafkaAuthConfigs(config, map[string]interface{}{"kafka.auth.mode": "KAFKA_API_KEY", "kafka.service.account.id": "sa-123"})
		require.Equal(t, map[string]string{
			"connector.class":          "DatagenSource",
			"kafka.auth.mode":          "KAFKA_API_KEY",
			"kafka.service.account.id": "sa-123",
			"kafka.topic":              "topic",
		}, filtered)
	})

	t.Run("does not change the input", func(t *testing.T) {
		withoutUnsetKafkaAuthConfigs(config, nil)
		require.Len(t, config, 8)
	})
}

func TestNextConnectorsPageToken(t *testing.T) {
	respWithToken := func(token string) *http.Response {
		resp := &http.Response{Header: http.Header{}}
		if token != "" {
			resp.Header.Set(listConnectorsNextPageTokenHeader, token)
		}
		return resp
	}

	token, err := nextConnectorsPageToken(respWithToken("page-2"), "")
	require.NoError(t, err)
	require.Equal(t, "page-2", token)

	// The header is the only end-of-list signal, so its absence ends the loop.
	token, err = nextConnectorsPageToken(respWithToken(""), "page-2")
	require.NoError(t, err)
	require.Empty(t, token)

	// A server that keeps returning the same token would loop forever.
	_, err = nextConnectorsPageToken(respWithToken("page-2"), "page-2")
	require.Error(t, err)
}

// connectorsFake serves the Connect v1 routes the connector resource uses. Its connector list returns one connector
// per page, with the target connector on the last page, so a client that doesn't follow X-Next-Page-Token can't find it.
type connectorsFake struct {
	listCalls   atomic.Int64
	getCalls    atomic.Int64
	statusCalls atomic.Int64
	pageSizes   []string
	created     atomic.Bool
	config      map[string]string
	server      *httptest.Server
}

const (
	fakeEnvironmentId = "env-fake1"
	fakeClusterId     = "lkc-fake1"
	fakeConnectorId   = "lcc-fake1"
	fakeConnectorName = "page_two_connector"
)

func newConnectorsFake(t *testing.T) *connectorsFake {
	f := &connectorsFake{}
	base := fmt.Sprintf("/connect/v1/environments/%s/clusters/%s", fakeEnvironmentId, fakeClusterId)
	mux := http.NewServeMux()
	mux.HandleFunc("PUT "+base+"/connector-plugins/{class}/config/validate", func(w http.ResponseWriter, r *http.Request) {
		writeFakeJSON(w, http.StatusOK, map[string]any{"name": r.PathValue("class"), "error_count": 0, "groups": []string{}, "configs": []any{}})
	})
	mux.HandleFunc("POST "+base+"/connectors", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Config map[string]string `json:"config"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		f.config = req.Config
		f.created.Store(true)
		writeFakeJSON(w, http.StatusCreated, map[string]any{"name": fakeConnectorName, "type": "source", "config": req.Config})
	})
	mux.HandleFunc("GET "+base+"/connectors", func(w http.ResponseWriter, r *http.Request) {
		f.listCalls.Add(1)
		f.pageSizes = append(f.pageSizes, r.URL.Query().Get("page_size"))
		if r.URL.Query().Get("page_token") == "" {
			w.Header().Set(listConnectorsNextPageTokenHeader, "page-2")
			writeFakeJSON(w, http.StatusOK, map[string]any{"decoy": f.expansion("decoy", "lcc-decoy")})
			return
		}
		writeFakeJSON(w, http.StatusOK, map[string]any{fakeConnectorName: f.expansion(fakeConnectorName, fakeConnectorId)})
	})
	mux.HandleFunc("GET "+base+"/connectors/{name}", func(w http.ResponseWriter, r *http.Request) {
		f.getCalls.Add(1)
		if !f.created.Load() || r.PathValue("name") != fakeConnectorName {
			writeFakeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
			return
		}
		// The single-connector GET returns the Kafka auth configs unmasked, unlike the list.
		config := map[string]string{}
		for k, v := range f.config {
			config[k] = v
		}
		config["kafka.auth.mode"] = "KAFKA_API_KEY"
		config["kafka.api.key"] = "unmasked-key"
		config["kafka.api.secret"] = "unmasked-secret"
		writeFakeJSON(w, http.StatusOK, map[string]any{"name": fakeConnectorName, "type": "source", "config": config})
	})
	mux.HandleFunc("GET "+base+"/connectors/{name}/status", func(w http.ResponseWriter, r *http.Request) {
		f.statusCalls.Add(1)
		writeFakeJSON(w, http.StatusOK, map[string]any{"name": fakeConnectorName, "type": "source", "connector": map[string]string{"state": "RUNNING"}})
	})
	mux.HandleFunc("DELETE "+base+"/connectors/{name}", func(w http.ResponseWriter, r *http.Request) {
		f.created.Store(false)
		w.WriteHeader(http.StatusNoContent)
	})
	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)
	return f
}

// expansion returns a list entry; like the real list it masks the Kafka auth configs.
func (f *connectorsFake) expansion(name, id string) map[string]any {
	config := map[string]string{}
	for k, v := range f.config {
		config[k] = v
	}
	config["kafka.api.key"] = "****************"
	config["kafka.api.secret"] = "****************"
	return map[string]any{
		"id":     map[string]string{"id": id, "id_type": "ID"},
		"info":   map[string]any{"name": name, "type": "source", "config": config},
		"status": map[string]any{"name": name, "connector": map[string]string{"state": "RUNNING"}},
	}
}

func writeFakeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func newFakeConnectClient(f *connectorsFake) *Client {
	cfg := connectv1.NewConfiguration()
	cfg.Servers = connectv1.ServerConfigurations{{URL: f.server.URL}}
	return &Client{connectV1Client: connectv1.NewAPIClient(cfg)}
}

func TestExecuteConnectorReadFollowsNextPageToken(t *testing.T) {
	f := newConnectorsFake(t)

	connector, _, err := executeConnectorRead(context.Background(), newFakeConnectClient(f), fakeConnectorName, fakeEnvironmentId, fakeClusterId)

	require.NoError(t, err)
	require.Equal(t, fakeConnectorId, connector.Id.GetId())
	require.Equal(t, int64(2), f.listCalls.Load())
	require.Equal(t, []string{"100", "100"}, f.pageSizes)
}

func TestExecuteConnectorReadNotFoundAfterLastPage(t *testing.T) {
	f := newConnectorsFake(t)

	_, resp, err := executeConnectorRead(context.Background(), newFakeConnectClient(f), "missing", fakeEnvironmentId, fakeClusterId)

	require.Error(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	require.Equal(t, int64(2), f.listCalls.Load())
}

// A refresh of an existing connector must read it by name; listing the cluster once per resource is what
// rate limits large Terraform configurations. Create and import still list, because they don't know the ID yet.
func TestAccConnectorRefreshReadsSingleConnector(t *testing.T) {
	f := newConnectorsFake(t)
	resourceLabel := "confluent_connector.test"
	var listCallsAfterCreate, getCallsAfterCreate int64

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fakeConnectorConfig(f.server.URL),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceLabel, paramId, fakeConnectorId),
					resource.TestCheckResourceAttr(resourceLabel, paramStatus, "RUNNING"),
					// The unmasked Kafka auth configs from the GET stay out of the state, as they did with the list.
					resource.TestCheckResourceAttr(resourceLabel, fmt.Sprintf("%s.%%", paramNonSensitiveConfig), "3"),
					resource.TestCheckNoResourceAttr(resourceLabel, fmt.Sprintf("%s.kafka.api.key", paramNonSensitiveConfig)),
					resource.TestCheckNoResourceAttr(resourceLabel, fmt.Sprintf("%s.kafka.auth.mode", paramNonSensitiveConfig)),
					func(*terraform.State) error {
						listCallsAfterCreate, getCallsAfterCreate = f.listCalls.Load(), f.getCalls.Load()
						require.Equal(t, int64(2), listCallsAfterCreate, "create looks up the ID in the list, following the next page token")
						return nil
					},
				),
			},
			{
				// Terraform refreshes the resource and plans: no diff, and no list call.
				Config: fakeConnectorConfig(f.server.URL),
				Check: func(*terraform.State) error {
					require.Equal(t, listCallsAfterCreate, f.listCalls.Load(), "refresh must not list the connectors")
					require.Greater(t, f.getCalls.Load(), getCallsAfterCreate, "refresh must read the connector by name")
					return nil
				},
			},
			{
				ResourceName:            resourceLabel,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{paramSensitiveConfig, paramOffsetsConfig},
				ImportStateId:           fmt.Sprintf("%s/%s/%s", fakeEnvironmentId, fakeClusterId, fakeConnectorName),
			},
		},
	})
}

func fakeConnectorConfig(serverUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}
	resource "confluent_connector" "test" {
	  environment {
	    id = "%s"
	  }
	  kafka_cluster {
	    id = "%s"
	  }
	  config_sensitive = {
	    "kafka.api.secret" = "secret"
	  }
	  config_nonsensitive = {
	    "name"            = "%s"
	    "connector.class" = "DatagenSource"
	    "kafka.topic"     = "test_topic"
	  }
	}
	`, serverUrl, fakeEnvironmentId, fakeClusterId, fakeConnectorName)
}
