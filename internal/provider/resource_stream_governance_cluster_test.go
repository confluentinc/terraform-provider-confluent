// Copyright 2022 Confluent Inc. All Rights Reserved.
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
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	scenarioStateStreamGovernanceClusterIsProvisioning = "The new Stream Governance Cluster is in provisioning state"
	scenarioStateStreamGovernanceClusterHasBeenCreated = "The new Stream Governance Cluster has been just created"
	scenarioStateStreamGovernanceClusterHasBeenDeleted = "The new Stream Governance Cluster has been deleted"
	streamGovernanceClusterScenarioName                = "confluent_stream_governance_cluster gcp Resource Lifecycle"
	streamGovernanceClusterHttpEndpoint                = "https://psrc-y1111.us-west-2.aws.confluent.cloud"
	streamGovernanceClusterRegionId                    = "sgreg-1"
	streamGovernanceClusterEnvironmentId               = "env-6w8d78"
	streamGovernanceClusterId                          = "lsrc-755ogo"
	streamGovernanceClusterResourceName                = "crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa/environment=env-6w8d78/schema-registry=lsrc-755ogo"
	streamGovernanceClusterApiVersion                  = "stream-governance/v2"
	streamGovernanceClusterKind                        = "Cluster"
	streamGovernanceClusterPackage                     = "ESSENTIALS"
	streamGovernanceClusterDisplayName                 = "Stream Governance Package"
)

var streamGovernanceClusterUrlPath = fmt.Sprintf("/stream-governance/v2/clusters/%s", streamGovernanceClusterId)

func TestAccStreamGovernanceCluster(t *testing.T) {
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
	createStreamGovernanceClusterResponse, _ := ioutil.ReadFile("../testdata/stream_governance_cluster/create_cluster.json")
	createStreamGovernanceClusterStub := wiremock.Post(wiremock.URLPathEqualTo("/stream-governance/v2/clusters")).
		InScenario(streamGovernanceClusterScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateStreamGovernanceClusterIsProvisioning).
		WillReturn(
			string(createStreamGovernanceClusterResponse),
			contentTypeJSONHeader,
			http.StatusAccepted,
		)
	_ = wiremockClient.StubFor(createStreamGovernanceClusterStub)

	readProvisioningStreamGovernanceClusterResponse, _ := ioutil.ReadFile("../testdata/stream_governance_cluster/read_provisioning_cluster.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(streamGovernanceClusterUrlPath)).
		InScenario(streamGovernanceClusterScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(streamGovernanceClusterEnvironmentId)).
		WhenScenarioStateIs(scenarioStateStreamGovernanceClusterIsProvisioning).
		WillSetStateTo(scenarioStateStreamGovernanceClusterHasBeenCreated).
		WillReturn(
			string(readProvisioningStreamGovernanceClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedStreamGovernanceClusterResponse, _ := ioutil.ReadFile("../testdata/stream_governance_cluster/read_provisioned_cluster.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(streamGovernanceClusterUrlPath)).
		InScenario(streamGovernanceClusterScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(streamGovernanceClusterEnvironmentId)).
		WhenScenarioStateIs(scenarioStateStreamGovernanceClusterHasBeenCreated).
		WillReturn(
			string(readCreatedStreamGovernanceClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteStreamGovernanceClusterStub := wiremock.Delete(wiremock.URLPathEqualTo(streamGovernanceClusterUrlPath)).
		InScenario(streamGovernanceClusterScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(streamGovernanceClusterEnvironmentId)).
		WhenScenarioStateIs(scenarioStateStreamGovernanceClusterHasBeenCreated).
		WillSetStateTo(scenarioStateStreamGovernanceClusterHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteStreamGovernanceClusterStub)

	readDeletedStreamGovernanceClusterResponse, _ := ioutil.ReadFile("../testdata/stream_governance_cluster/read_deleted_cluster.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(streamGovernanceClusterUrlPath)).
		InScenario(streamGovernanceClusterScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(streamGovernanceClusterEnvironmentId)).
		WhenScenarioStateIs(scenarioStateStreamGovernanceClusterHasBeenDeleted).
		WillReturn(
			string(readDeletedStreamGovernanceClusterResponse),
			contentTypeJSONHeader,
			http.StatusForbidden,
		))

	streamGovernanceClusterResourceLabel := "test"
	fullStreamGovernanceClusterResourceLabel := fmt.Sprintf("confluent_stream_governance_cluster.%s", streamGovernanceClusterResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckStreamGovernanceClusterDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckStreamGovernanceClusterConfig(mockServerUrl, streamGovernanceClusterResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStreamGovernanceClusterExists(fullStreamGovernanceClusterResourceLabel),
					resource.TestCheckResourceAttr(fullStreamGovernanceClusterResourceLabel, paramId, streamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullStreamGovernanceClusterResourceLabel, paramPackage, streamGovernanceClusterPackage),
					resource.TestCheckResourceAttr(fullStreamGovernanceClusterResourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullStreamGovernanceClusterResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), streamGovernanceClusterEnvironmentId),
					resource.TestCheckResourceAttr(fullStreamGovernanceClusterResourceLabel, fmt.Sprintf("%s.#", paramRegion), "1"),
					resource.TestCheckResourceAttr(fullStreamGovernanceClusterResourceLabel, fmt.Sprintf("%s.0.%s", paramRegion, paramId), streamGovernanceClusterRegionId),
					resource.TestCheckResourceAttr(fullStreamGovernanceClusterResourceLabel, paramDisplayName, streamGovernanceClusterDisplayName),
					resource.TestCheckResourceAttr(fullStreamGovernanceClusterResourceLabel, paramApiVersion, streamGovernanceClusterApiVersion),
					resource.TestCheckResourceAttr(fullStreamGovernanceClusterResourceLabel, paramKind, streamGovernanceClusterKind),
					resource.TestCheckResourceAttr(fullStreamGovernanceClusterResourceLabel, paramResourceName, streamGovernanceClusterResourceName),
					resource.TestCheckResourceAttr(fullStreamGovernanceClusterResourceLabel, paramHttpEndpoint, streamGovernanceClusterHttpEndpoint),
				),
			},
			{
				ResourceName:      fullStreamGovernanceClusterResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					streamGovernanceClusterId := resources[fullStreamGovernanceClusterResourceLabel].Primary.ID
					environmentId := resources[fullStreamGovernanceClusterResourceLabel].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + streamGovernanceClusterId, nil
				},
			},
		},
	})

	checkStubCount(t, wiremockClient, createStreamGovernanceClusterStub, fmt.Sprintf("POST %s", streamGovernanceClusterUrlPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteStreamGovernanceClusterStub, fmt.Sprintf("DELETE %s?environment=%s", streamGovernanceClusterUrlPath, streamGovernanceClusterEnvironmentId), expectedCountOne)
}

func testAccCheckStreamGovernanceClusterDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each Stream Governance Cluster is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_stream_governance_cluster" {
			continue
		}
		deletedStreamGovernanceClusterId := rs.Primary.ID
		req := c.sgClient.ClustersStreamGovernanceV2Api.GetStreamGovernanceV2Cluster(c.sgApiContext(context.Background()), deletedStreamGovernanceClusterId).Environment(streamGovernanceClusterEnvironmentId)
		deletedStreamGovernanceCluster, response, err := req.Execute()
		if response != nil && response.StatusCode == http.StatusForbidden {
			return nil
		} else if err == nil && deletedStreamGovernanceCluster.Id != nil {
			// Otherwise return the error
			if *deletedStreamGovernanceCluster.Id == rs.Primary.ID {
				return fmt.Errorf("stream Governance Cluster (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckStreamGovernanceClusterConfig(mockServerUrl, resourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_stream_governance_cluster" "%s" {
        package          = "%s"
	    environment {
		  id = "%s"
	    }
	    region {
		  id = "%s"
	    }
	}
	`, mockServerUrl, resourceLabel, streamGovernanceClusterPackage, streamGovernanceClusterEnvironmentId, streamGovernanceClusterRegionId)
}

func testAccCheckStreamGovernanceClusterExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s Stream Governance Cluster has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s Stream Governance Cluster", n)
		}

		return nil
	}
}
