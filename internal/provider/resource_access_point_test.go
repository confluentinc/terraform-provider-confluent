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
	scenarioStateAccessPointIsProvisioning = "The new access point is provisioning"
	scenarioStateAccessPointHasBeenCreated = "The new access point has been just created"
	scenarioStateAccessPointHasBeenUpdated = "The new access point has been updated"
	accessPointScenarioName                = "confluent_access_point Resource Lifecycle"

	accessPointUrlPath       = "/networking/v1/access-points"
	accessPointReadUrlPath   = "/networking/v1/access-points/ap-abc123"
	accessPointResourceLabel = "confluent_access_point.main"
)

func TestAccAccessPoint(t *testing.T) {
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

	createAccessPointResponse, _ := os.ReadFile("../testdata/network_access_point/create_ap.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(accessPointUrlPath)).
		InScenario(accessPointScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateAccessPointIsProvisioning).
		WillReturn(
			string(createAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(accessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointIsProvisioning).
		WillSetStateTo(scenarioStateAccessPointHasBeenCreated).
		WillReturn(
			string(createAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedAccessPointResponse, _ := os.ReadFile("../testdata/network_access_point/read_created_ap.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(accessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointHasBeenCreated).
		WillReturn(
			string(readCreatedAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	updatedAccessPointResponse, _ := os.ReadFile("../testdata/network_access_point/update_ap.json")
	_ = wiremockClient.StubFor(wiremock.Patch(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(accessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointHasBeenCreated).
		WillSetStateTo(scenarioStateAccessPointHasBeenUpdated).
		WillReturn(
			string(updatedAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(accessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointHasBeenUpdated).
		WillReturn(
			string(updatedAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(accessPointScenarioName).
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
				Config: testAccCheckResourceAccessPointWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(accessPointResourceLabel, "id", "ap-abc123"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "display_name", "prod-ap-1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gateway.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gateway.0.id", "gw-abc123"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_egress_private_link_endpoint.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_egress_private_link_endpoint.0.vpc_endpoint_service_name", "com.amazonaws.vpce.us-west-2.vpce-svc-00000000000000000"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_egress_private_link_endpoint.0.vpc_endpoint_id", "vpce-00000000000000000"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_egress_private_link_endpoint.0.vpc_endpoint_dns_name", "*.vpce-00000000000000000-abcd1234.s3.us-west-2.vpce.amazonaws.com"),
				),
			},
			{
				Config: testAccCheckResourceUpdateAccessPointWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(accessPointResourceLabel, "id", "ap-abc123"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "display_name", "prod-ap-2"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gateway.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gateway.0.id", "gw-abc123"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_egress_private_link_endpoint.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_egress_private_link_endpoint.0.vpc_endpoint_service_name", "com.amazonaws.vpce.us-west-2.vpce-svc-00000000000000000"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_egress_private_link_endpoint.0.vpc_endpoint_id", "vpce-00000000000000000"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_egress_private_link_endpoint.0.vpc_endpoint_dns_name", "*.vpce-00000000000000000-abcd1234.s3.us-west-2.vpce.amazonaws.com"),
				),
			},
		},
	})
}

func testAccCheckResourceAccessPointWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_access_point" "main" {
		display_name = "prod-ap-1"
		environment {
			id = "env-abc123"
		}
		gateway {
			id = "gw-abc123"
		}
		aws_egress_private_link_endpoint {
			vpc_endpoint_service_name = "com.amazonaws.vpce.us-west-2.vpce-svc-00000000000000000"
		}
	}
	`, mockServerUrl)
}

func testAccCheckResourceUpdateAccessPointWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_access_point" "main" {
		display_name = "prod-ap-2"
		environment {
			id = "env-abc123"
		}
		gateway {
			id = "gw-abc123"
		}
		aws_egress_private_link_endpoint {
			vpc_endpoint_service_name = "com.amazonaws.vpce.us-west-2.vpce-svc-00000000000000000"
		}
	}
	`, mockServerUrl)
}
