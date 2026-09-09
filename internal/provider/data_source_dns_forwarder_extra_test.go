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
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

// TestAccDataSourceDnsForwarderForwardViaIpFromResourceFixture and its GCP counterpart below read
// the data source against ../testdata/network_dns_forwarder, the same fixtures
// TestAccDnsForwarder/TestAccDnsForwarderGcp (resource_dns_forwarder_test.go) already use, rather
// than the generated data_source_dns_forwarder_test.go's own dedicated fixtures under
// ../testdata/dns_forwarder. That mirrors how every other resource+data-source pair in this
// package is tested (e.g. TestAccByokKeyAws / TestAccDataSourceByokKeyAws both read
// ../testdata/byok/aws_key.json) and ties the data source's read/flatten path
// (setDnsForwarderAttributes, shared with the resource) to the one fixture already trusted for
// the resource, instead of a second, parallel set of values that could drift out of sync with it.
func TestAccDataSourceDnsForwarderForwardViaIpFromResourceFixture(t *testing.T) {
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

	readCreatedDnsForwarderResponse, _ := os.ReadFile("../testdata/network_dns_forwarder/read_created_dnsf.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(dnsForwarderReadUrlPath)).
		InScenario(dnsForwarderDataSourceScenarioName).
		WithQueryParam("environment", wiremock.EqualTo("env-xxx")).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedDnsForwarderResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	dnsForwarderDataSourceLabel := "test_forward_via_ip_from_resource_fixture"
	fullDnsForwarderDataSourceLabel := fmt.Sprintf("data.confluent_dns_forwarder.%s", dnsForwarderDataSourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceDnsForwarderConfigWithId(mockServerUrl, dnsForwarderDataSourceLabel, "dnsf-xxx", "env-xxx"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fullDnsForwarderDataSourceLabel, "id", "dnsf-xxx"),
					resource.TestCheckResourceAttr(fullDnsForwarderDataSourceLabel, "display_name", "dns1"),
					resource.TestCheckResourceAttr(fullDnsForwarderDataSourceLabel, "environment.0.id", "env-xxx"),
					resource.TestCheckResourceAttr(fullDnsForwarderDataSourceLabel, "gateway.0.id", "gw-xxx"),
					resource.TestCheckResourceAttr(fullDnsForwarderDataSourceLabel, "domains.#", "2"),
					resource.TestCheckTypeSetElemAttr(fullDnsForwarderDataSourceLabel, "domains.*", "example.com"),
					resource.TestCheckTypeSetElemAttr(fullDnsForwarderDataSourceLabel, "domains.*", "domainname.com"),
					resource.TestCheckResourceAttr(fullDnsForwarderDataSourceLabel, "forward_via_ip.0.dns_server_ips.#", "2"),
					resource.TestCheckTypeSetElemAttr(fullDnsForwarderDataSourceLabel, "forward_via_ip.0.dns_server_ips.*", "10.200.0.0"),
					resource.TestCheckTypeSetElemAttr(fullDnsForwarderDataSourceLabel, "forward_via_ip.0.dns_server_ips.*", "10.200.0.1"),
				),
			},
		},
	})
}

func TestAccDataSourceDnsForwarderForwardViaGcpDnsZonesFromResourceFixture(t *testing.T) {
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

	readCreatedDnsForwarderGcpResponse, _ := os.ReadFile("../testdata/network_dns_forwarder/read_created_dnsf_gcp.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(dnsForwarderReadUrlPathGcp)).
		InScenario(dnsForwarderDataSourceScenarioName).
		WithQueryParam("environment", wiremock.EqualTo("env-xxxx")).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedDnsForwarderGcpResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	dnsForwarderDataSourceLabel := "test_forward_via_gcp_dns_zones_from_resource_fixture"
	fullDnsForwarderDataSourceLabel := fmt.Sprintf("data.confluent_dns_forwarder.%s", dnsForwarderDataSourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceDnsForwarderConfigWithId(mockServerUrl, dnsForwarderDataSourceLabel, "dnsf-gcp", "env-xxxx"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fullDnsForwarderDataSourceLabel, "id", "dnsf-gcp"),
					resource.TestCheckResourceAttr(fullDnsForwarderDataSourceLabel, "display_name", "dns2"),
					resource.TestCheckResourceAttr(fullDnsForwarderDataSourceLabel, "environment.0.id", "env-xxxx"),
					resource.TestCheckResourceAttr(fullDnsForwarderDataSourceLabel, "gateway.0.id", "gw-xxx"),
					resource.TestCheckResourceAttr(fullDnsForwarderDataSourceLabel, "domains.#", "2"),
					resource.TestCheckTypeSetElemAttr(fullDnsForwarderDataSourceLabel, "domains.*", "example.com"),
					resource.TestCheckTypeSetElemAttr(fullDnsForwarderDataSourceLabel, "domains.*", "test.com"),
					resource.TestCheckResourceAttr(fullDnsForwarderDataSourceLabel, "forward_via_gcp_dns_zones.0.domain_mappings.test.com", "zone-1,project-123"),
					resource.TestCheckResourceAttr(fullDnsForwarderDataSourceLabel, "forward_via_gcp_dns_zones.0.domain_mappings.example.com", "zone-2,project-456"),
				),
			},
		},
	})
}

func testAccCheckDataSourceDnsForwarderConfigWithId(mockServerUrl, dnsForwarderDataSourceLabel, id, environmentId string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_dns_forwarder" "%s" {
		id = %q
		environment {
			id = %q
		}
	}
	`, mockServerUrl, dnsForwarderDataSourceLabel, id, environmentId)
}
