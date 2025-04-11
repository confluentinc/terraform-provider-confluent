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
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

const (
	scenarioStateDnsRecordIsProvisioning   = "The new dns record is provisioning"
	scenarioStateDnsRecordHasBeenCreated   = "The new dns record has been just created"
	scenarioStateDnsRecordHasBeenUpdated   = "The new dns record has been updated"
	scenarioStateDnsRecordIsDeprovisioning = "The new dns record is deprovisioning"
	scenarioStateDnsRecordHasBeenDeleted   = "The new dms record's deletion has been just completed"
	dnsRecordScenarioName                  = "confluent_dns_record Resource Lifecycle"

	dnsRecordUrlPath       = "/networking/v1/dns-records"
	dnsRecordReadUrlPath   = "/networking/v1/dns-records/dnsrec-abc123"
	dnsRecordResourceLabel = "confluent_dns_record.main"
)

func TestAccDnsRecord(t *testing.T) {
	ctx := context.Background()

	time.Sleep(5 * time.Second)
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

	createDnsRecordResponse, _ := os.ReadFile("../testdata/network_dns_record/create_dnsrec.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(dnsRecordUrlPath)).
		InScenario(dnsRecordScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateDnsRecordIsProvisioning).
		WillReturn(
			string(createDnsRecordResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(dnsRecordReadUrlPath)).
		InScenario(dnsRecordScenarioName).
		WhenScenarioStateIs(scenarioStateDnsRecordIsProvisioning).
		WillSetStateTo(scenarioStateDnsRecordHasBeenCreated).
		WillReturn(
			string(createDnsRecordResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedDnsRecordResponse, _ := os.ReadFile("../testdata/network_dns_record/read_created_dnsrec.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(dnsRecordReadUrlPath)).
		InScenario(dnsRecordScenarioName).
		WhenScenarioStateIs(scenarioStateDnsRecordHasBeenCreated).
		WillReturn(
			string(readCreatedDnsRecordResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	updatedDnsRecordResponse, _ := os.ReadFile("../testdata/network_dns_record/updated_dnsrec.json")
	_ = wiremockClient.StubFor(wiremock.Patch(wiremock.URLPathEqualTo(dnsRecordReadUrlPath)).
		InScenario(dnsRecordScenarioName).
		WhenScenarioStateIs(scenarioStateDnsRecordHasBeenCreated).
		WillSetStateTo(scenarioStateDnsRecordHasBeenUpdated).
		WillReturn(
			string(updatedDnsRecordResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(dnsRecordReadUrlPath)).
		InScenario(dnsRecordScenarioName).
		WhenScenarioStateIs(scenarioStateDnsRecordHasBeenUpdated).
		WillReturn(
			string(updatedDnsRecordResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(dnsRecordReadUrlPath)).
		InScenario(dnsRecordScenarioName).
		WhenScenarioStateIs(scenarioStateDnsRecordHasBeenUpdated).
		WillSetStateTo(scenarioStateDnsRecordIsDeprovisioning).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	readDeprovisioningDnsRecResponse, _ := os.ReadFile("../testdata/network_dns_record/read_deprovisioning_dnsrec.json")
	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(dnsRecordReadUrlPath)).
		InScenario(dnsRecordScenarioName).
		WhenScenarioStateIs(scenarioStateDnsRecordIsDeprovisioning).
		WillSetStateTo(scenarioStateDnsRecordHasBeenDeleted).
		WillReturn(
			string(readDeprovisioningDnsRecResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedDnsRecordResponse, _ := os.ReadFile("../testdata/network_dns_record/read_deleted_dnsrec.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(dnsRecordReadUrlPath)).
		InScenario(dnsRecordScenarioName).
		WhenScenarioStateIs(scenarioStateDnsRecordHasBeenDeleted).
		WillReturn(
			string(readDeletedDnsRecordResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceDnsRecordWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dnsRecordResourceLabel, "id", "dnsrec-abc123"),
					resource.TestCheckResourceAttr(dnsRecordResourceLabel, "display_name", "prod-dnsrec-1"),
					resource.TestCheckResourceAttr(dnsRecordResourceLabel, "domain", "www.example.com"),
					resource.TestCheckResourceAttr(dnsRecordResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(dnsRecordResourceLabel, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(dnsRecordResourceLabel, "gateway.#", "1"),
					resource.TestCheckResourceAttr(dnsRecordResourceLabel, "gateway.0.id", "gw-abc123"),
					resource.TestCheckResourceAttr(dnsRecordResourceLabel, "private_link_access_point.#", "1"),
					resource.TestCheckResourceAttr(dnsRecordResourceLabel, "private_link_access_point.0.id", "ap-abc123"),
				),
			},
			{
				Config: testAccCheckResourceUpdateDnsRecordWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dnsRecordResourceLabel, "id", "dnsrec-abc123"),
					resource.TestCheckResourceAttr(dnsRecordResourceLabel, "display_name", "prod-dnsrec-2"),
					resource.TestCheckResourceAttr(dnsRecordResourceLabel, "domain", "www.example.com"),
					resource.TestCheckResourceAttr(dnsRecordResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(dnsRecordResourceLabel, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(dnsRecordResourceLabel, "gateway.#", "1"),
					resource.TestCheckResourceAttr(dnsRecordResourceLabel, "gateway.0.id", "gw-abc123"),
					resource.TestCheckResourceAttr(dnsRecordResourceLabel, "private_link_access_point.#", "1"),
					resource.TestCheckResourceAttr(dnsRecordResourceLabel, "private_link_access_point.0.id", "ap-def456"),
				),
			},
		},
	})
}

func testAccCheckResourceDnsRecordWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_dns_record" "main" {
		display_name = "prod-dnsrec-1"
		environment {
			id = "env-abc123"
		}
		domain = "www.example.com"
		gateway {
			id = "gw-abc123"
		}
		private_link_access_point {
			id = "ap-abc123"
		}
	}
	`, mockServerUrl)
}

func testAccCheckResourceUpdateDnsRecordWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_dns_record" "main" {
		display_name = "prod-dnsrec-2"
		environment {
			id = "env-abc123"
		}
		domain = "www.example.com"
		gateway {
			id = "gw-abc123"
		}
		private_link_access_point {
			id = "ap-def456"
		}
	}
	`, mockServerUrl)
}
