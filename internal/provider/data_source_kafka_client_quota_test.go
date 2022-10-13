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
	kafkaClientQuotaDataSourceScenarioName = "confluent_kafka_client_quota Data Source Lifecycle"
)

func TestAccDataSourceKafkaClientQuota(t *testing.T) {
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

	readCreatedKafkaClientQuotaResponse, _ := ioutil.ReadFile("../testdata/kafka_client_quota/read_created_kafka_client_quota.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(kafkaClientQuotaUrlPath)).
		InScenario(kafkaClientQuotaDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedKafkaClientQuotaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	fullKafkaClientQuotaDataSourceLabel := fmt.Sprintf("data.confluent_kafka_client_quota.%s", kafkaClientQuotaResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckKafkaClientQuotaDataSourceConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKafkaClientQuotaExists(fullKafkaClientQuotaDataSourceLabel),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaDataSourceLabel, paramId, "cq-e857e"),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaDataSourceLabel, paramDisplayName, kafkaClientQuotaDisplayName),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaDataSourceLabel, paramDescription, kafkaClientQuotaDescription),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaDataSourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaDataSourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), kafkaClientQuotaEnvrionmentId),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaDataSourceLabel, fmt.Sprintf("%s.#", paramKafkaCluster), "1"),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaDataSourceLabel, fmt.Sprintf("%s.0.%s", paramKafkaCluster, paramId), kafkaClientQuotaClusterId),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaDataSourceLabel, fmt.Sprintf("%s.#", paramThroughput), "1"),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaDataSourceLabel, "throughput.0.%", "2"),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaDataSourceLabel, fmt.Sprintf("%s.0.%s", paramThroughput, paramIngressByteRate), kafkaClientQuotaIngressByteRate),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaDataSourceLabel, fmt.Sprintf("%s.0.%s", paramThroughput, paramEgressByteRate), kafkaClientQuotaEgressByteRate),
					resource.TestCheckResourceAttr(fullKafkaClientQuotaDataSourceLabel, fmt.Sprintf("%s.#", paramPrincipals), strconv.Itoa(len(kafkaClientQuotaPrincipals))),
					resource.TestCheckTypeSetElemAttr(fullKafkaClientQuotaDataSourceLabel, fmt.Sprintf("%s.*", paramPrincipals), kafkaClientQuotaPrincipals[0]),
					resource.TestCheckTypeSetElemAttr(fullKafkaClientQuotaDataSourceLabel, fmt.Sprintf("%s.*", paramPrincipals), kafkaClientQuotaPrincipals[1]),
				),
			},
		},
	})
}

func testAccCheckKafkaClientQuotaDataSourceConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_kafka_client_quota" "%s" {
		id = "%s"
	}
	`, mockServerUrl, kafkaClientQuotaResourceLabel, kafkaClientQuotaId)
}
