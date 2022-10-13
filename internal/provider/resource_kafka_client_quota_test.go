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
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	scenarioStateKafkaClientQuotaHasBeenCreated             = "The new Kafka Client Quota has been just created"
	scenarioStateKafkaClientQuotaDescriptionHaveBeenUpdated = "The new Kafka Client Quota has been just updated"
	scenarioStateKafkaClientQuotaHasBeenDeleted             = "The new Kafka Client Quota has been deleted"
	kafkaClientQuotaScenarioName                            = "confluent_kafka_client_quota Resource Lifecycle"

	kafkaClientQuotaId      = "cq-e857e"
	kafkaClientQuotaUrlPath = "/kafka-quotas/v1/client-quotas/cq-e857e"

	kafkaClientQuotaClusterId     = "lkc-03roj2"
	kafkaClientQuotaEnvrionmentId = "env-nyyz3d"

	kafkaClientQuotaResourceLabel   = "test_kafka_client_quota_resource_label"
	kafkaClientQuotaDisplayName     = "QuotaForSA1"
	kafkaClientQuotaDescription     = "test"
	kafkaClientQuotaIngressByteRate = "12288"
	kafkaClientQuotaEgressByteRate  = "12289"
)

var kafkaClientQuotaPrincipals = []string{"sa-rv1vo7", "sa-jzgzgq"}

func TestAccKafkaClientQuota(t *testing.T) {
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
	createKafkaClientQuotaResponse, _ := ioutil.ReadFile("../testdata/kafka_client_quota/create_kafka_client_quota.json")
	createKafkaClientQuotaStub := wiremock.Post(wiremock.URLPathEqualTo("/kafka-quotas/v1/client-quotas")).
		InScenario(kafkaClientQuotaScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateKafkaClientQuotaHasBeenCreated).
		WillReturn(
			string(createKafkaClientQuotaResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createKafkaClientQuotaStub)

	readCreatedKafkaClientQuotaResponse, _ := ioutil.ReadFile("../testdata/kafka_client_quota/read_created_kafka_client_quota.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(kafkaClientQuotaUrlPath)).
		InScenario(kafkaClientQuotaScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaClientQuotaHasBeenCreated).
		WillReturn(
			string(readCreatedKafkaClientQuotaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedKafkaClientQuotaResponse, _ := ioutil.ReadFile("../testdata/kafka_client_quota/read_updated_kafka_client_quota.json")
	patchKafkaClientQuotaStub := wiremock.Patch(wiremock.URLPathEqualTo(kafkaClientQuotaUrlPath)).
		InScenario(kafkaClientQuotaScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaClientQuotaHasBeenCreated).
		WillSetStateTo(scenarioStateKafkaClientQuotaDescriptionHaveBeenUpdated).
		WillReturn(
			string(readUpdatedKafkaClientQuotaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(patchKafkaClientQuotaStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(kafkaClientQuotaUrlPath)).
		InScenario(kafkaClientQuotaScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaClientQuotaDescriptionHaveBeenUpdated).
		WillReturn(
			string(readUpdatedKafkaClientQuotaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedKafkaClientQuotaResponse, _ := ioutil.ReadFile("../testdata/kafka_client_quota/read_deleted_kafka_client_quota.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/kafka-quotas/v1/client-quotas/cq-e857e")).
		InScenario(kafkaClientQuotaScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaClientQuotaHasBeenDeleted).
		WillReturn(
			string(readDeletedKafkaClientQuotaResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	deleteKafkaClientQuotaStub := wiremock.Delete(wiremock.URLPathEqualTo(kafkaClientQuotaUrlPath)).
		InScenario(kafkaClientQuotaScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaClientQuotaDescriptionHaveBeenUpdated).
		WillSetStateTo(scenarioStateKafkaClientQuotaHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteKafkaClientQuotaStub)

	// in order to test tf update (step #3)
	kafkaClientQuotaUpdatedDisplayName := "QuotaForSA1-updated"
	kafkaClientQuotaUpdatedDescription := "test-updated"
	kafkaClientQuotaUpdatedPrincipals := []string{"sa-rv1vo7"}
	kafkaClientQuotaUpdatedIngressByteRate := "12280"
	kafkaClientQuotaUpdatedEgressByteRate := "12281"
	fullKafkaClientQuotaResourceLabel := fmt.Sprintf("confluent_kafka_client_quota.%s", kafkaClientQuotaResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKafkaClientQuotaDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckKafkaClientQuotaConfig(mockServerUrl, kafkaClientQuotaResourceLabel, kafkaClientQuotaDisplayName, kafkaClientQuotaDescription, kafkaClientQuotaPrincipals, kafkaClientQuotaIngressByteRate, kafkaClientQuotaEgressByteRate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKafkaClientQuotaExists(fullKafkaClientQuotaResourceLabel),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaResourceLabel, paramId, "cq-e857e"),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaResourceLabel, paramDisplayName, kafkaClientQuotaDisplayName),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaResourceLabel, paramDescription, kafkaClientQuotaDescription),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaResourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), kafkaClientQuotaEnvrionmentId),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaResourceLabel, fmt.Sprintf("%s.#", paramKafkaCluster), "1"),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaResourceLabel, fmt.Sprintf("%s.0.%s", paramKafkaCluster, paramId), kafkaClientQuotaClusterId),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaResourceLabel, fmt.Sprintf("%s.#", paramThroughput), "1"),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaResourceLabel, "throughput.0.%", "2"),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaResourceLabel, fmt.Sprintf("%s.0.%s", paramThroughput, paramIngressByteRate), kafkaClientQuotaIngressByteRate),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaResourceLabel, fmt.Sprintf("%s.0.%s", paramThroughput, paramEgressByteRate), kafkaClientQuotaEgressByteRate),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaResourceLabel, fmt.Sprintf("%s.#", paramPrincipals), strconv.Itoa(len(kafkaClientQuotaPrincipals))),
					resource.TestCheckTypeSetElemAttr(fullKafkaClientQuotaResourceLabel, fmt.Sprintf("%s.*", paramPrincipals), kafkaClientQuotaPrincipals[0]),
					resource.TestCheckTypeSetElemAttr(fullKafkaClientQuotaResourceLabel, fmt.Sprintf("%s.*", paramPrincipals), kafkaClientQuotaPrincipals[1]),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullKafkaClientQuotaResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCheckKafkaClientQuotaConfig(mockServerUrl, kafkaClientQuotaResourceLabel, kafkaClientQuotaUpdatedDisplayName, kafkaClientQuotaUpdatedDescription, kafkaClientQuotaUpdatedPrincipals, kafkaClientQuotaUpdatedIngressByteRate, kafkaClientQuotaUpdatedEgressByteRate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKafkaClientQuotaExists(fullKafkaClientQuotaResourceLabel),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaResourceLabel, paramId, "cq-e857e"),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaResourceLabel, paramDisplayName, kafkaClientQuotaUpdatedDisplayName),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaResourceLabel, paramDescription, kafkaClientQuotaUpdatedDescription),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaResourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), kafkaClientQuotaEnvrionmentId),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaResourceLabel, fmt.Sprintf("%s.#", paramKafkaCluster), "1"),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaResourceLabel, fmt.Sprintf("%s.0.%s", paramKafkaCluster, paramId), kafkaClientQuotaClusterId),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaResourceLabel, fmt.Sprintf("%s.#", paramThroughput), "1"),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaResourceLabel, "throughput.0.%", "2"),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaResourceLabel, fmt.Sprintf("%s.0.%s", paramThroughput, paramIngressByteRate), kafkaClientQuotaUpdatedIngressByteRate),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaResourceLabel, fmt.Sprintf("%s.0.%s", paramThroughput, paramEgressByteRate), kafkaClientQuotaUpdatedEgressByteRate),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaResourceLabel, fmt.Sprintf("%s.#", paramPrincipals), strconv.Itoa(len(kafkaClientQuotaUpdatedPrincipals))),
					resource.TestCheckTypeSetElemAttr(fullKafkaClientQuotaResourceLabel, fmt.Sprintf("%s.*", paramPrincipals), kafkaClientQuotaUpdatedPrincipals[0]),
				),
			},
			{
				ResourceName:      fullKafkaClientQuotaResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, createKafkaClientQuotaStub, "POST /kafka-quotas/v1/client-quotas", expectedCountOne)
	checkStubCount(t, wiremockClient, patchKafkaClientQuotaStub, "PATCH /kafka-quotas/v1/client-quotas/cq-e857e", expectedCountOne)
	checkStubCount(t, wiremockClient, deleteKafkaClientQuotaStub, "DELETE /kafka-quotas/v1/client-quotas/cq-e857e", expectedCountOne)
}

func testAccCheckKafkaClientQuotaDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each Kafka Client Quota is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_kafka_client_quota" {
			continue
		}
		deletedKafkaClientQuotaId := rs.Primary.ID
		req := c.quotasClient.ClientQuotasKafkaQuotasV1Api.GetKafkaQuotasV1ClientQuota(c.quotasApiContext(context.Background()), deletedKafkaClientQuotaId)
		deletedKafkaClientQuota, response, err := req.Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			// v2/service-accounts/{nonExistentKafkaClientQuotaId/deletedKafkaClientQuotaID} returns http.StatusForbidden instead of http.StatusNotFound
			// If the error is equivalent to http.StatusNotFound, the Kafka Client Quota is destroyed.
			return nil
		} else if err == nil && deletedKafkaClientQuota.Id != nil {
			// Otherwise return the error
			if *deletedKafkaClientQuota.Id == rs.Primary.ID {
				return fmt.Errorf("kafka Client Quota (%q) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckKafkaClientQuotaConfig(mockServerUrl, kafkaClientQuotaResourceLabel, kafkaClientQuotaDisplayName, kafkaClientQuotaDescription string, kafkaClientQuotaPrincipals []string, kafkaClientQuotaIngressByteRate, kafkaClientQuotaEgressByteRate string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_kafka_client_quota" "%s" {
		display_name = "%s"
		description = "%s"
	    kafka_cluster {
		  id = "%s"
	    }
	    environment {
		  id = "%s"
	    }
		throughput {
		  ingress_byte_rate = "%s"
		  egress_byte_rate = "%s"
		}
		principals = %s
	}
	`, mockServerUrl, kafkaClientQuotaResourceLabel, kafkaClientQuotaDisplayName, kafkaClientQuotaDescription,
		kafkaClientQuotaClusterId, kafkaClientQuotaEnvrionmentId,
		kafkaClientQuotaIngressByteRate, kafkaClientQuotaEgressByteRate, formatListOfStringsForHcl(kafkaClientQuotaPrincipals))
}

func testAccCheckKafkaClientQuotaExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s Kafka Client Quota has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s Kafka Client Quota", n)
		}

		return nil
	}
}

// Converts ["foo", "bar"] into a string: "[\"foo\", \"bar\"]"
func formatListOfStringsForHcl(items []string) string {
	itemsHcl := make([]string, len(items))
	for i, _ := range items {
		itemsHcl[i] = fmt.Sprintf("%q", items[i])
	}
	return fmt.Sprintf("[%s]", strings.Join(itemsHcl[:], ","))
}
