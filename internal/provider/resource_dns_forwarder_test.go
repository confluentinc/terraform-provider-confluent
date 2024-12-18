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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"testing"
)

const (
	scenarioStateDnsForwarderIsProvisioning = "The new dns forwarder is provisioning"
	scenarioStateDnsForwarderHasBeenCreated = "The new dns forwarder has been created"
	dnsForwarderScenarioName                = "confluent_dns_forwarder Resource Lifecycle"

	dnsForwarderUrlPath        = "/networking/v1/dns-forwarders"
	dnsForwarderReadUrlPath    = "/networking/v1/dns-forwarders/dnsf-xxx"
	dnsForwarderResourceLabel  = "confluent_dns_forwarder.main"
	dnsForwarderReadUrlPathGcp = "/networking/v1/dns-forwarders/dnsf-gcp"
)

func TestAccDnsForwarder(t *testing.T) {
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

	createDnsForwarderResponse, _ := ioutil.ReadFile("../testdata/network_dns_forwarder/create_dnsf.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(dnsForwarderUrlPath)).
		InScenario(dnsForwarderScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateDnsForwarderIsProvisioning).
		WillReturn(
			string(createDnsForwarderResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(dnsForwarderReadUrlPath)).
		InScenario(dnsForwarderScenarioName).
		WithQueryParam("environment", wiremock.EqualTo("env-xxx")).
		WhenScenarioStateIs(scenarioStateDnsForwarderIsProvisioning).
		WillSetStateTo(scenarioStateDnsForwarderHasBeenCreated).
		WillReturn(
			string(createDnsForwarderResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedDnsForwarderResponse, _ := ioutil.ReadFile("../testdata/network_dns_forwarder/read_created_dnsf.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(dnsForwarderReadUrlPath)).
		InScenario(dnsForwarderScenarioName).
		WithQueryParam("environment", wiremock.EqualTo("env-xxx")).
		WhenScenarioStateIs(scenarioStateDnsForwarderHasBeenCreated).
		WillReturn(
			string(readCreatedDnsForwarderResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(dnsForwarderReadUrlPath)).
		InScenario(dnsForwarderScenarioName).
		WithQueryParam("environment", wiremock.EqualTo("env-xxx")).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceDnsForwarderWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dnsForwarderResourceLabel, "id", "dnsf-xxx"),
					resource.TestCheckResourceAttr(dnsForwarderResourceLabel, "display_name", "dns1"),
					resource.TestCheckResourceAttr(dnsForwarderResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(dnsForwarderResourceLabel, "environment.0.id", "env-xxx"),
					resource.TestCheckResourceAttr(dnsForwarderResourceLabel, "gateway.#", "1"),
					resource.TestCheckResourceAttr(dnsForwarderResourceLabel, "gateway.0.id", "gw-xxx"),
					resource.TestCheckResourceAttr(dnsForwarderResourceLabel, "domains.#", "2"),
					resource.TestCheckResourceAttr(dnsForwarderResourceLabel, "domains.0", "domainname.com"),
					resource.TestCheckResourceAttr(dnsForwarderResourceLabel, "domains.1", "example.com"),
					resource.TestCheckResourceAttr(dnsForwarderResourceLabel, "forward_via_ip.0.dns_server_ips.#", "2"),
					resource.TestCheckResourceAttr(dnsForwarderResourceLabel, "forward_via_ip.0.dns_server_ips.0", "10.200.0.0"),
					resource.TestCheckResourceAttr(dnsForwarderResourceLabel, "forward_via_ip.0.dns_server_ips.1", "10.200.0.1")),
			},
		},
	})
}

func TestAccDnsForwarderGcp(t *testing.T) {
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

	createDnsForwarderResponse, _ := ioutil.ReadFile("../testdata/network_dns_forwarder/create_dnsf_gcp.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(dnsForwarderUrlPath)).
		InScenario(dnsForwarderScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateDnsForwarderIsProvisioning).
		WillReturn(
			string(createDnsForwarderResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(dnsForwarderReadUrlPathGcp)).
		InScenario(dnsForwarderScenarioName).
		WithQueryParam("environment", wiremock.EqualTo("env-xxxx")).
		WhenScenarioStateIs(scenarioStateDnsForwarderIsProvisioning).
		WillSetStateTo(scenarioStateDnsForwarderHasBeenCreated).
		WillReturn(
			string(createDnsForwarderResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedDnsForwarderResponse, _ := ioutil.ReadFile("../testdata/network_dns_forwarder/read_created_dnsf_gcp.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(dnsForwarderReadUrlPathGcp)).
		InScenario(dnsForwarderScenarioName).
		WithQueryParam("environment", wiremock.EqualTo("env-xxxx")).
		WhenScenarioStateIs(scenarioStateDnsForwarderHasBeenCreated).
		WillReturn(
			string(readCreatedDnsForwarderResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(dnsForwarderReadUrlPathGcp)).
		InScenario(dnsForwarderScenarioName).
		WithQueryParam("environment", wiremock.EqualTo("env-xxxx")).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceDnsForwarderGcpWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dnsForwarderResourceLabel, "id", "dnsf-gcp"),
					resource.TestCheckResourceAttr(dnsForwarderResourceLabel, "display_name", "dns2"),
					resource.TestCheckResourceAttr(dnsForwarderResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(dnsForwarderResourceLabel, "environment.0.id", "env-xxxx"),
					resource.TestCheckResourceAttr(dnsForwarderResourceLabel, "gateway.#", "1"),
					resource.TestCheckResourceAttr(dnsForwarderResourceLabel, "gateway.0.id", "gw-xxx"),
					resource.TestCheckResourceAttr(dnsForwarderResourceLabel, "domains.#", "2"),
					resource.TestCheckResourceAttr(dnsForwarderResourceLabel, "domains.0", "domainname.com"),
					resource.TestCheckResourceAttr(dnsForwarderResourceLabel, "domains.1", "example.com"),
					resource.TestCheckResourceAttr(dnsForwarderResourceLabel, "forward_via_gcp_dns_zones.0.domain_mappings.example.com", "zone-1,project-123"),
					resource.TestCheckResourceAttr(dnsForwarderResourceLabel, "forward_via_gcp_dns_zones.0.domain_mappings.test.com", "zone-2,project-456")),
			},
		},
	})
}

func testAccCheckResourceDnsForwarderWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_dns_forwarder" "main" {
		display_name = "dns1"
		environment {
			id = "env-xxx"
		}
		domains = ["example.com", "domainname.com"]
		gateway {
			id = "gw-xxx"
		}
		forward_via_ip {
			dns_server_ips = ["10.200.0.0", "10.200.0.1"]
		}
	}
	`, mockServerUrl)
}

func testAccCheckResourceDnsForwarderGcpWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_dns_forwarder" "main" {
		display_name = "dns2"
		environment {
			id = "env-xxxx"
		}
		domains = ["example.com", "domainname.com"]
		gateway {
			id = "gw-xxx"
		}
		forward_via_gcp_dns_zones {
			domain_mappings = {
					"example.com" = "zone-1,project-123"
 					"test.com"    = "zone-2,project-456"
			}
		}
	}
	`, mockServerUrl)
}
