// Copyright 2024 Confluent Inc. All Rights Reserved.
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
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// Verifies that creating a confluent_schema_registry_dek
// does not crash with a SIGSEGV when the API returns a nil HTTP response.
func TestAccDekCreate_NilResponse(t *testing.T) {
	// Create a server that immediately closes the connection.
	// This causes the HTTP client to get a connection error and Execute() to return (nil, nil, error).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("webserver doesn't support hijacking")
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			t.Fatal(err)
		}
		conn.Close()
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      dekNilResponseConfig(server.URL),
				ExpectError: regexp.MustCompile("error creating Schema Registry DEK"),
			},
		},
	})
}

func dekNilResponseConfig(serverUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  schema_registry_id             = "lsrc-test"
	  schema_registry_rest_endpoint  = "%s"
	  schema_registry_api_key        = "test-key"
	  schema_registry_api_secret     = "test-secret"
	}
	resource "confluent_schema_registry_dek" "test" {
	  kek_name     = "test-kek"
	  subject_name = "test-subject"
	  hard_delete  = true
	}
	`, serverUrl)
}
