//go:build live_test && (all || core)

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
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccIPFilterLive(t *testing.T) {
	// Enable parallel execution for I/O bound operations
	t.Parallel()

	// Skip this test unless explicitly enabled
	if os.Getenv("TF_ACC_PROD") == "" {
		t.Skip("Skipping live test. Set TF_ACC_PROD=1 to run this test.")
	}

	// Read credentials and configuration from environment variables (populated by Vault)
	apiKey := os.Getenv("CONFLUENT_CLOUD_API_KEY")
	apiSecret := os.Getenv("CONFLUENT_CLOUD_API_SECRET")
	endpoint := os.Getenv("CONFLUENT_CLOUD_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.confluent.cloud" // Use default endpoint if not set
	}

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	// Get current public IP to avoid lockout
	currentIP := getCurrentPublicIP(t)
	if currentIP == "" {
		t.Fatal("Could not determine current public IP address for IP filter testing")
	}
	t.Logf("Current public IP: %s", currentIP)

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	ipGroupName1 := fmt.Sprintf("tf-live-ip-group-filter-1-%d", randomSuffix)
	ipGroupName2 := fmt.Sprintf("tf-live-ip-group-filter-2-%d", randomSuffix)
	ipFilterName := fmt.Sprintf("tf-live-ip-filter-%d", randomSuffix)
	ipFilterUpdatedName := fmt.Sprintf("tf-live-ip-filter-updated-%d", randomSuffix)
	ipGroupResourceLabel1 := "test_live_ip_group_for_filter_1"
	ipGroupResourceLabel2 := "test_live_ip_group_for_filter_2"
	ipFilterResourceLabel := "test_live_ip_filter"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckIPFilterLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckIPFilterUpdateInitialLiveConfig(endpoint, ipGroupResourceLabel1, ipGroupResourceLabel2, ipFilterResourceLabel, ipGroupName1, ipGroupName2, ipFilterName, currentIP, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIPFilterLiveExists(fmt.Sprintf("confluent_ip_filter.%s", ipFilterResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_ip_filter.%s", ipFilterResourceLabel), "filter_name", ipFilterName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_ip_filter.%s", ipFilterResourceLabel), "resource_group", "multiple"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_ip_filter.%s", ipFilterResourceLabel), "operation_groups.#", "1"),
					resource.TestCheckTypeSetElemAttr(fmt.Sprintf("confluent_ip_filter.%s", ipFilterResourceLabel), "operation_groups.*", "MANAGEMENT"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_ip_filter.%s", ipFilterResourceLabel), "ip_groups.#", "1"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_ip_filter.%s", ipFilterResourceLabel), "id"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_ip_filter.%s", ipFilterResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCheckIPFilterUpdateFinalLiveConfig(endpoint, ipGroupResourceLabel1, ipGroupResourceLabel2, ipFilterResourceLabel, ipGroupName1, ipGroupName2, ipFilterUpdatedName, currentIP, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIPFilterLiveExists(fmt.Sprintf("confluent_ip_filter.%s", ipFilterResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_ip_filter.%s", ipFilterResourceLabel), "filter_name", ipFilterUpdatedName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_ip_filter.%s", ipFilterResourceLabel), "resource_group", "multiple"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_ip_filter.%s", ipFilterResourceLabel), "operation_groups.#", "2"),
					resource.TestCheckTypeSetElemAttr(fmt.Sprintf("confluent_ip_filter.%s", ipFilterResourceLabel), "operation_groups.*", "MANAGEMENT"),
					resource.TestCheckTypeSetElemAttr(fmt.Sprintf("confluent_ip_filter.%s", ipFilterResourceLabel), "operation_groups.*", "FLINK"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_ip_filter.%s", ipFilterResourceLabel), "ip_groups.#", "2"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_ip_filter.%s", ipFilterResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckIPFilterLiveDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each IP Filter is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_ip_filter" {
			continue
		}
		deletedIPFilterId := rs.Primary.ID
		req := c.iamIPClient.IPFiltersIamV2Api.GetIamV2IpFilter(c.iamIPApiContext(context.Background()), deletedIPFilterId)
		deletedIPFilter, response, err := req.Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			// If the error is equivalent to http.StatusNotFound, the IP Filter is destroyed.
			return nil
		} else if err == nil && deletedIPFilter.Id != nil {
			// Otherwise return the error
			if *deletedIPFilter.Id == rs.Primary.ID {
				return fmt.Errorf("IP Filter (%q) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckIPFilterLiveConfig(endpoint, ipGroupResourceLabel, ipFilterResourceLabel, ipGroupName, ipFilterName, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
		cloud_api_key = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_ip_group" "%s" {
		group_name = "%s"
		cidr_blocks = [
			"10.200.0.0/16"
		]
	}

	resource "confluent_ip_filter" "%s" {
		filter_name = "%s"
		resource_group = "MANAGEMENT"
		operation_groups = [
			"MANAGEMENT",
			"FLINK"
		]
		ip_groups = [
			confluent_ip_group.%s.id
		]
	}
	`, endpoint, apiKey, apiSecret, ipGroupResourceLabel, ipGroupName, ipFilterResourceLabel, ipFilterName, ipGroupResourceLabel)
}

func getCurrentPublicIP(t *testing.T) string {
	// Try multiple IP detection services for reliability
	services := []string{
		"https://api.ipify.org",
		"https://ifconfig.me/ip",
		"https://icanhazip.com",
	}

	for _, service := range services {
		resp, err := http.Get(service)
		if err != nil {
			t.Logf("Failed to get IP from %s: %v", service, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Logf("Failed to read response from %s: %v", service, err)
				continue
			}
			ip := strings.TrimSpace(string(body))
			if ip != "" {
				return ip
			}
		}
	}

	return ""
}

func testAccCheckIPFilterUpdateInitialLiveConfig(endpoint, ipGroupResourceLabel1, ipGroupResourceLabel2, ipFilterResourceLabel, ipGroupName1, ipGroupName2, ipFilterName, currentIP, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
		cloud_api_key = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_ip_group" "%s" {
		group_name = "%s"
		cidr_blocks = [
			"%s/24"
		]
	}

	resource "confluent_ip_group" "%s" {
		group_name = "%s"
		cidr_blocks = [
			"172.20.0.0/16"
		]
	}

	resource "confluent_ip_filter" "%s" {
		filter_name = "%s"
		resource_group = "multiple"
		operation_groups = [
			"MANAGEMENT"
		]
		ip_groups = [
			confluent_ip_group.%s.id
		]
	}
	`, endpoint, apiKey, apiSecret, ipGroupResourceLabel1, ipGroupName1, currentIP, ipGroupResourceLabel2, ipGroupName2, ipFilterResourceLabel, ipFilterName, ipGroupResourceLabel1)
}

func testAccCheckIPFilterUpdateFinalLiveConfig(endpoint, ipGroupResourceLabel1, ipGroupResourceLabel2, ipFilterResourceLabel, ipGroupName1, ipGroupName2, ipFilterName, currentIP, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
		cloud_api_key = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_ip_group" "%s" {
		group_name = "%s"
		cidr_blocks = [
			"%s/32"
		]
	}

	resource "confluent_ip_group" "%s" {
		group_name = "%s"
		cidr_blocks = [
			"172.20.0.0/16"
		]
	}

	resource "confluent_ip_filter" "%s" {
		filter_name = "%s"
		resource_group = "multiple"
		operation_groups = [
			"MANAGEMENT",
			"FLINK"
		]
		ip_groups = [
			confluent_ip_group.%s.id,
			confluent_ip_group.%s.id
		]
	}
	`, endpoint, apiKey, apiSecret, ipGroupResourceLabel1, ipGroupName1, currentIP, ipGroupResourceLabel2, ipGroupName2, ipFilterResourceLabel, ipFilterName, ipGroupResourceLabel1, ipGroupResourceLabel2)
}

func testAccCheckIPFilterLiveExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s IP Filter has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s IP Filter", n)
		}

		return nil
	}
}

