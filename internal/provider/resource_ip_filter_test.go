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
	ipFilterResourceScenarioName = "confluent_ip_filter Resource Lifecycle"

	scenarioStateIpFilterHasBeenCreated = "A new IP group has been created"

	createIpFilterUrlPath      = "/iam/v2/ip-filters"
	readCreatedIpFilterUrlPath = "/iam/v2/ip-filters/ipf-12345"
)

func TestAccResourceIpFilter(t *testing.T) {
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

	createIpFilterResponse, _ := ioutil.ReadFile("../testdata/create_ip_filter.json")

	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(createIpFilterUrlPath)).
		InScenario(ipFilterResourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateIpFilterHasBeenCreated).
		WillReturn(
			string(createIpFilterResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	readIpFilterResponse, _ := ioutil.ReadFile("../testdata/read_ip_filter.json")

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readCreatedIpFilterUrlPath)).
		InScenario(ipFilterResourceScenarioName).
		WhenScenarioStateIs(scenarioStateIpFilterHasBeenCreated).
		WillReturn(
			string(readIpFilterResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))
}

func testAccResourceIpFilterConfig(mockServerUrl, resourceLabel, filter_name, resource_group, resource_scope, operation_group, ip_group string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_ip_filter" "%s" {
		filter_name = "%s"
		resource_group = "%s"
		resource_scope = "%s"
		operation_groups = [ "%s" ]
		ip_groups = [ "%s" ]
	}
	`, mockServerUrl, resourceLabel, filter_name, resource_group, resource_scope, operation_group, ip_group)
}
