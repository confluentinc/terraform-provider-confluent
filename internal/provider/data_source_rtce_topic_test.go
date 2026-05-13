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
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

const (
	rtce_topicDataSourceScenarioName = "confluent_rtce_topic Data Source Lifecycle"
)

func TestAccDataSourceRtceRtceTopic(t *testing.T) {
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

	readRtceTopicResponse, _ := ioutil.ReadFile("../testdata/rtce_rtce_topic/read_created_rtce_topic.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/rtce/v1/rtce-topics/%s", rtce_topicTopicName))).
		InScenario(rtce_topicDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readRtceTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))
	cloud := "AWS"
	description := "Customer orders table for real-time analytics"
	environment := "env-00000"
	kafkaCluster := "lkc-00000"
	region := "us-west-2"
	topicName := "orders_topic"
	rtce_topicDataSourceLabel := "test_rtce_topic_data_source_label"
	fullRtceTopicDataSourceLabel := fmt.Sprintf("data.confluent_rtce_topic.%s", rtce_topicDataSourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceRtceRtceTopicConfig(mockServerUrl, rtce_topicDataSourceLabel, environment, kafkaCluster),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fullRtceTopicDataSourceLabel, "id", fmt.Sprintf("%s/%s/%s", environment, kafkaCluster, topicName)),
					resource.TestCheckResourceAttr(fullRtceTopicDataSourceLabel, "cloud", cloud),
					resource.TestCheckResourceAttr(fullRtceTopicDataSourceLabel, "description", description),
					resource.TestCheckResourceAttr(fullRtceTopicDataSourceLabel, "environment.0.id", environment),
					resource.TestCheckResourceAttr(fullRtceTopicDataSourceLabel, "kafka_cluster.0.id", kafkaCluster),
					resource.TestCheckResourceAttr(fullRtceTopicDataSourceLabel, "region", region),
					resource.TestCheckResourceAttr(fullRtceTopicDataSourceLabel, "topic_name", topicName),
					resource.TestCheckResourceAttr(fullRtceTopicDataSourceLabel, "api_version", "rtce/v1"),
					resource.TestCheckResourceAttr(fullRtceTopicDataSourceLabel, "kind", "RtceTopic"),
					resource.TestCheckResourceAttr(fullRtceTopicDataSourceLabel, "resource_name", "crn://confluent.cloud/organization=9bb441c4-edef-46ac-8a41-c49e44a3fd9a/environment=env-abc123/cloud-cluster=lkc-12345/topic=tt-12345"),
				),
			},
		},
	})
}

func testAccCheckDataSourceRtceRtceTopicConfig(mockServerUrl, rtce_topicDataSourceLabel string, environmentId string, kafkaClusterId string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_rtce_topic" "%s" {
		topic_name = "%s"
		environment {
			id = "%s"
		}
		kafka_cluster {
			id = "%s"
		}
	}
	`, mockServerUrl, rtce_topicDataSourceLabel, rtce_topicTopicName, environmentId, kafkaClusterId)
}
