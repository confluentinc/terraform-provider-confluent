// Copyright 2021 Confluent Inc. All Rights Reserved.
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
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	networkingApiVersion              = "networking/v1"
	ipAddressesDataSourceScenarioName = "confluent_ip_addresses Data Source Lifecycle"
	ipAddressKind                     = "IpAddress"
	ipAddressesResourceLabel          = "test_ip_address_label"
	ipAddressLastPagePageToken        = "dyJpZCI6InNhLTd5OXbybq"

	testIpAddressCloud          = "AWS"
	testIpAddressServiceConnect = "CONNECT"
	testIpAddressServiceKafka   = "KAFKA"
	testIpAddressAddressType    = "EGRESS"
)

func TestAccDataSourceIpAddresses(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	mockServerUrl := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	readIpAddressesPageOneResponse, _ := ioutil.ReadFile("../testdata/network_ip/read_ips_page_1.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/networking/v1/ip-addresses")).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listIPAddressesPageSize))).
		InScenario(ipAddressesDataSourceScenarioName).
		WillReturn(
			string(readIpAddressesPageOneResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readIpAddressesPageTwoResponse, _ := ioutil.ReadFile("../testdata/network_ip/read_ips_page_2.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/networking/v1/ip-addresses")).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listIPAddressesPageSize))).
		WithQueryParam("page_token", wiremock.EqualTo(ipAddressLastPagePageToken)).
		InScenario(ipAddressesDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readIpAddressesPageTwoResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	fullIpAddressesDataSourceLabel := fmt.Sprintf("data.confluent_ip_addresses.%s", ipAddressesResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceIpAddresses(mockServerUrl, ipAddressesResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceAccountExists(fullIpAddressesDataSourceLabel),
					resource.TestCheckResourceAttr(fullIpAddressesDataSourceLabel, "ip_addresses.#", "3"),

					resource.TestCheckResourceAttr(fullIpAddressesDataSourceLabel, "ip_addresses.0.api_version", networkingApiVersion),
					resource.TestCheckResourceAttr(fullIpAddressesDataSourceLabel, "ip_addresses.0.kind", ipAddressKind),
					resource.TestCheckResourceAttr(fullIpAddressesDataSourceLabel, "ip_addresses.0.ip_prefix", "10.200.0.0/28"),
					resource.TestCheckResourceAttr(fullIpAddressesDataSourceLabel, "ip_addresses.0.services.#", "1"),
					resource.TestCheckResourceAttr(fullIpAddressesDataSourceLabel, "ip_addresses.0.services.0", testIpAddressServiceConnect),
					resource.TestCheckResourceAttr(fullIpAddressesDataSourceLabel, "ip_addresses.0.cloud", testIpAddressCloud),
					resource.TestCheckResourceAttr(fullIpAddressesDataSourceLabel, "ip_addresses.0.region", "us-east-1"),
					resource.TestCheckResourceAttr(fullIpAddressesDataSourceLabel, "ip_addresses.0.address_type", "EGRESS"),

					resource.TestCheckResourceAttr(fullIpAddressesDataSourceLabel, "ip_addresses.1.api_version", networkingApiVersion),
					resource.TestCheckResourceAttr(fullIpAddressesDataSourceLabel, "ip_addresses.1.kind", ipAddressKind),
					resource.TestCheckResourceAttr(fullIpAddressesDataSourceLabel, "ip_addresses.1.ip_prefix", "10.200.0.0/24"),
					resource.TestCheckResourceAttr(fullIpAddressesDataSourceLabel, "ip_addresses.1.services.#", "1"),
					resource.TestCheckResourceAttr(fullIpAddressesDataSourceLabel, "ip_addresses.1.services.0", testIpAddressServiceConnect),
					resource.TestCheckResourceAttr(fullIpAddressesDataSourceLabel, "ip_addresses.1.cloud", testIpAddressCloud),
					resource.TestCheckResourceAttr(fullIpAddressesDataSourceLabel, "ip_addresses.1.region", "us-east-2"),
					resource.TestCheckResourceAttr(fullIpAddressesDataSourceLabel, "ip_addresses.1.address_type", testIpAddressAddressType),

					resource.TestCheckResourceAttr(fullIpAddressesDataSourceLabel, "ip_addresses.2.api_version", networkingApiVersion),
					resource.TestCheckResourceAttr(fullIpAddressesDataSourceLabel, "ip_addresses.2.kind", ipAddressKind),
					resource.TestCheckResourceAttr(fullIpAddressesDataSourceLabel, "ip_addresses.2.ip_prefix", "10.200.0.0/20"),
					resource.TestCheckResourceAttr(fullIpAddressesDataSourceLabel, "ip_addresses.2.services.#", "1"),
					resource.TestCheckResourceAttr(fullIpAddressesDataSourceLabel, "ip_addresses.2.services.0", testIpAddressServiceKafka),
					resource.TestCheckResourceAttr(fullIpAddressesDataSourceLabel, "ip_addresses.2.cloud", testIpAddressCloud),
					resource.TestCheckResourceAttr(fullIpAddressesDataSourceLabel, "ip_addresses.2.region", "us-east-2"),
					resource.TestCheckResourceAttr(fullIpAddressesDataSourceLabel, "ip_addresses.2.address_type", testIpAddressAddressType),
				),
			},
		},
	})
	err = wiremockContainer.Terminate(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func testAccCheckDataSourceIpAddresses(mockServerUrl, label string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_ip_addresses" "%s" {
		filter {
			clouds        = ["AWS"]
			regions       = ["us-east-1", "us-east-2"]
			services      = ["CONNECT", "KAFKA"]
			address_types = ["EGRESS"]
		}
	}
	`, mockServerUrl, label)
}
