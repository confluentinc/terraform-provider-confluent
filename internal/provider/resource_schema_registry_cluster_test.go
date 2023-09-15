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
	scenarioStateSchemaRegistryClusterIsProvisioning = "The new Schema Registry Cluster is in provisioning state"
	scenarioStateSchemaRegistryClusterHasBeenCreated = "The new Schema Registry Cluster has been just created"
	scenarioStateSchemaRegistryClusterHasBeenDeleted = "The new Schema Registry Cluster has been deleted"
	schemaRegistryClusterScenarioName                = "confluent_schema_registry_cluster gcp Resource Lifecycle"
	schemaRegistryClusterHttpEndpoint                = "https://psrc-y1111.us-west-2.aws.confluent.cloud"
	schemaRegistryClusterRegionId                    = "sgreg-1"
	schemaRegistryClusterEnvironmentId               = "env-6w8d78"
	schemaRegistryClusterId                          = "lsrc-755ogo"
	schemaRegistryClusterResourceName                = "crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa/environment=env-6w8d78/schema-registry=lsrc-755ogo"
	schemaRegistryClusterApiVersion                  = "srcm/v2"
	schemaRegistryClusterKind                        = "Cluster"
	schemaRegistryClusterPackage                     = "ESSENTIALS"
	schemaRegistryClusterDisplayName                 = "Stream Governance Package"
)

var schemaRegistryClusterUrlPath = fmt.Sprintf("/srcm/v2/clusters/%s", schemaRegistryClusterId)

func TestAccSchemaRegistryCluster(t *testing.T) {
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
	createSchemaRegistryClusterResponse, _ := ioutil.ReadFile("../testdata/schema_registry_cluster/create_cluster.json")
	createSchemaRegistryClusterStub := wiremock.Post(wiremock.URLPathEqualTo("/srcm/v2/clusters")).
		InScenario(schemaRegistryClusterScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateSchemaRegistryClusterIsProvisioning).
		WillReturn(
			string(createSchemaRegistryClusterResponse),
			contentTypeJSONHeader,
			http.StatusAccepted,
		)
	_ = wiremockClient.StubFor(createSchemaRegistryClusterStub)

	readProvisioningSchemaRegistryClusterResponse, _ := ioutil.ReadFile("../testdata/schema_registry_cluster/read_provisioning_cluster.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(schemaRegistryClusterUrlPath)).
		InScenario(schemaRegistryClusterScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(schemaRegistryClusterEnvironmentId)).
		WhenScenarioStateIs(scenarioStateSchemaRegistryClusterIsProvisioning).
		WillSetStateTo(scenarioStateSchemaRegistryClusterHasBeenCreated).
		WillReturn(
			string(readProvisioningSchemaRegistryClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedSchemaRegistryClusterResponse, _ := ioutil.ReadFile("../testdata/schema_registry_cluster/read_provisioned_cluster.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(schemaRegistryClusterUrlPath)).
		InScenario(schemaRegistryClusterScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(schemaRegistryClusterEnvironmentId)).
		WhenScenarioStateIs(scenarioStateSchemaRegistryClusterHasBeenCreated).
		WillReturn(
			string(readCreatedSchemaRegistryClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteSchemaRegistryClusterStub := wiremock.Delete(wiremock.URLPathEqualTo(schemaRegistryClusterUrlPath)).
		InScenario(schemaRegistryClusterScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(schemaRegistryClusterEnvironmentId)).
		WhenScenarioStateIs(scenarioStateSchemaRegistryClusterHasBeenCreated).
		WillSetStateTo(scenarioStateSchemaRegistryClusterHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteSchemaRegistryClusterStub)

	readDeletedSchemaRegistryClusterResponse, _ := ioutil.ReadFile("../testdata/schema_registry_cluster/read_deleted_cluster.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(schemaRegistryClusterUrlPath)).
		InScenario(schemaRegistryClusterScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(schemaRegistryClusterEnvironmentId)).
		WhenScenarioStateIs(scenarioStateSchemaRegistryClusterHasBeenDeleted).
		WillReturn(
			string(readDeletedSchemaRegistryClusterResponse),
			contentTypeJSONHeader,
			http.StatusForbidden,
		))

	schemaRegistryClusterResourceLabel := "test"
	fullSchemaRegistryClusterResourceLabel := fmt.Sprintf("confluent_schema_registry_cluster.%s", schemaRegistryClusterResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSchemaRegistryClusterDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSchemaRegistryClusterConfig(mockServerUrl, schemaRegistryClusterResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaRegistryClusterExists(fullSchemaRegistryClusterResourceLabel),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterResourceLabel, paramId, schemaRegistryClusterId),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterResourceLabel, paramPackage, schemaRegistryClusterPackage),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterResourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), schemaRegistryClusterEnvironmentId),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterResourceLabel, fmt.Sprintf("%s.#", paramRegion), "1"),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterResourceLabel, fmt.Sprintf("%s.0.%s", paramRegion, paramId), schemaRegistryClusterRegionId),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterResourceLabel, paramDisplayName, schemaRegistryClusterDisplayName),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterResourceLabel, paramApiVersion, schemaRegistryClusterApiVersion),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterResourceLabel, paramKind, schemaRegistryClusterKind),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterResourceLabel, paramResourceName, schemaRegistryClusterResourceName),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterResourceLabel, paramRestEndpoint, schemaRegistryClusterHttpEndpoint),
				),
			},
			{
				ResourceName:      fullSchemaRegistryClusterResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					schemaRegistryClusterId := resources[fullSchemaRegistryClusterResourceLabel].Primary.ID
					environmentId := resources[fullSchemaRegistryClusterResourceLabel].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + schemaRegistryClusterId, nil
				},
			},
		},
	})

	checkStubCount(t, wiremockClient, createSchemaRegistryClusterStub, fmt.Sprintf("POST %s", schemaRegistryClusterUrlPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteSchemaRegistryClusterStub, fmt.Sprintf("DELETE %s?environment=%s", schemaRegistryClusterUrlPath, schemaRegistryClusterEnvironmentId), expectedCountOne)
}

func testAccCheckSchemaRegistryClusterDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each Schema Registry Cluster is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_schema_registry_cluster" {
			continue
		}
		deletedSchemaRegistryClusterId := rs.Primary.ID
		req := c.srcmClient.ClustersSrcmV2Api.GetSrcmV2Cluster(c.srcmApiContext(context.Background()), deletedSchemaRegistryClusterId).Environment(schemaRegistryClusterEnvironmentId)
		deletedSchemaRegistryCluster, response, err := req.Execute()
		if response != nil && response.StatusCode == http.StatusForbidden {
			return nil
		} else if err == nil && deletedSchemaRegistryCluster.Id != nil {
			// Otherwise return the error
			if *deletedSchemaRegistryCluster.Id == rs.Primary.ID {
				return fmt.Errorf("schema Registry Cluster (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckSchemaRegistryClusterConfig(mockServerUrl, resourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_schema_registry_cluster" "%s" {
        package          = "%s"
	    environment {
		  id = "%s"
	    }
	    region {
		  id = "%s"
	    }
	}
	`, mockServerUrl, resourceLabel, schemaRegistryClusterPackage, schemaRegistryClusterEnvironmentId, schemaRegistryClusterRegionId)
}

func testAccCheckSchemaRegistryClusterExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s Schema Registry Cluster has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s Schema Registry Cluster", n)
		}

		return nil
	}
}
