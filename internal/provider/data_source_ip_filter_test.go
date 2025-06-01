// Copyright 2023 Confluent Inc. All Rights Reserved.
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
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/walkerus/go-wiremock"
)

const (
	ipFilterDataSourceScenarioName = "confluent_ip_filter Data Source Lifecycle"

	ipFilterUrlPath = "/iam/v2/ip-filters/ipf-12345"
	ipFilterId      = "ipf-12345"
)

func TestAccDataSourceIpFilter(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockServerUrl := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	readIpFilterResponse, _ := ioutil.ReadFile("../testdata/read_ip_filter.json")

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(ipFilterUrlPath)).
		InScenario(ipFilterDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readIpFilterResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))
}

func testAccDataSourceIpFilterConfig(mockServerUrl, resourceLabel, id string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_ip_filter" "%s" {
		id = "%s"
	}
	`, mockServerUrl, resourceLabel, id)
}
