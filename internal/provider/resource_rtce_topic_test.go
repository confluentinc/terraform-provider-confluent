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
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
)

const (
	scenarioStateRtceTopicHasBeenCreated = "The new rtce_topic has been just created"
	scenarioStateRtceTopicHasBeenUpdated = "The rtce_topic has been just updated"
	scenarioStateRtceTopicHasBeenDeleted = "The rtce_topic has been deleted"
	rtce_topicScenarioName               = "confluent_rtce_topic Resource Lifecycle"
	rtce_topicTopicName                  = "orders_topic"
)

func TestAccRtceRtceTopic(t *testing.T) {
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

	createRtceTopicResponse, _ := ioutil.ReadFile("../testdata/rtce_rtce_topic/create_rtce_topic.json")
	createRtceTopicStub := wiremock.Post(wiremock.URLPathEqualTo("/rtce/v1/rtce-topics")).
		InScenario(rtce_topicScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateRtceTopicHasBeenCreated).
		WillReturn(
			string(createRtceTopicResponse),
			contentTypeJSONHeader,
			http.StatusAccepted,
		)
	_ = wiremockClient.StubFor(createRtceTopicStub)

	readCreatedRtceTopicResponse, _ := ioutil.ReadFile("../testdata/rtce_rtce_topic/read_created_rtce_topic.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/rtce/v1/rtce-topics/%s", rtce_topicTopicName))).
		InScenario(rtce_topicScenarioName).
		WhenScenarioStateIs(scenarioStateRtceTopicHasBeenCreated).
		WillReturn(
			string(readCreatedRtceTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedRtceTopicResponse, _ := ioutil.ReadFile("../testdata/rtce_rtce_topic/read_updated_rtce_topic.json")
	patchRtceTopicStub := wiremock.Patch(wiremock.URLPathEqualTo(fmt.Sprintf("/rtce/v1/rtce-topics/%s", rtce_topicTopicName))).
		InScenario(rtce_topicScenarioName).
		WhenScenarioStateIs(scenarioStateRtceTopicHasBeenCreated).
		WillSetStateTo(scenarioStateRtceTopicHasBeenUpdated).
		WillReturn(
			string(readUpdatedRtceTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(patchRtceTopicStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/rtce/v1/rtce-topics/%s", rtce_topicTopicName))).
		InScenario(rtce_topicScenarioName).
		WhenScenarioStateIs(scenarioStateRtceTopicHasBeenUpdated).
		WillReturn(
			string(readUpdatedRtceTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedRtceTopicResponse, _ := ioutil.ReadFile("../testdata/rtce_rtce_topic/read_deleted_rtce_topic.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/rtce/v1/rtce-topics/%s", rtce_topicTopicName))).
		InScenario(rtce_topicScenarioName).
		WhenScenarioStateIs(scenarioStateRtceTopicHasBeenDeleted).
		WillReturn(
			string(readDeletedRtceTopicResponse),
			contentTypeJSONHeader,
			http.StatusForbidden,
		))

	deleteRtceTopicStub := wiremock.Delete(wiremock.URLPathEqualTo(fmt.Sprintf("/rtce/v1/rtce-topics/%s", rtce_topicTopicName))).
		InScenario(rtce_topicScenarioName).
		WhenScenarioStateIs(scenarioStateRtceTopicHasBeenUpdated).
		WillSetStateTo(scenarioStateRtceTopicHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteRtceTopicStub)
	cloud := "AWS"
	description := "Customer orders table for real-time analytics"
	environment := "env-00000"
	kafkaCluster := "lkc-00000"
	region := "us-west-2"
	topicName := "orders_topic"
	// in order to test tf update (step #3)
	descriptionUpdated := "Customer orders table for real-time analytics_updated"
	rtce_topicResourceLabel := "test_rtce_topic_resource_label"
	fullRtceTopicResourceLabel := fmt.Sprintf("confluent_rtce_topic.%s", rtce_topicResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRtceRtceTopicDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckRtceRtceTopicConfig(mockServerUrl, rtce_topicResourceLabel, cloud, description, environment, kafkaCluster, region, topicName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRtceRtceTopicExists(fullRtceTopicResourceLabel),
					resource.TestCheckResourceAttr(fullRtceTopicResourceLabel, "id", fmt.Sprintf("%s/%s/%s", environment, kafkaCluster, topicName)),
					resource.TestCheckResourceAttr(fullRtceTopicResourceLabel, "cloud", cloud),
					resource.TestCheckResourceAttr(fullRtceTopicResourceLabel, "description", description),
					resource.TestCheckResourceAttr(fullRtceTopicResourceLabel, "environment.0.id", environment),
					resource.TestCheckResourceAttr(fullRtceTopicResourceLabel, "kafka_cluster.0.id", kafkaCluster),
					resource.TestCheckResourceAttr(fullRtceTopicResourceLabel, "region", region),
					resource.TestCheckResourceAttr(fullRtceTopicResourceLabel, "topic_name", topicName),
					resource.TestCheckResourceAttr(fullRtceTopicResourceLabel, "api_version", "rtce/v1"),
					resource.TestCheckResourceAttr(fullRtceTopicResourceLabel, "kind", "RtceTopic"),
					resource.TestCheckResourceAttr(fullRtceTopicResourceLabel, "resource_name", "crn://confluent.cloud/organization=9bb441c4-edef-46ac-8a41-c49e44a3fd9a/environment=env-abc123/cloud-cluster=lkc-12345/topic=tt-12345"),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullRtceTopicResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCheckRtceRtceTopicConfig(mockServerUrl, rtce_topicResourceLabel, cloud, descriptionUpdated, environment, kafkaCluster, region, topicName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRtceRtceTopicExists(fullRtceTopicResourceLabel),
					resource.TestCheckResourceAttr(fullRtceTopicResourceLabel, "id", fmt.Sprintf("%s/%s/%s", environment, kafkaCluster, topicName)),
					resource.TestCheckResourceAttr(fullRtceTopicResourceLabel, "cloud", cloud),
					resource.TestCheckResourceAttr(fullRtceTopicResourceLabel, "description", descriptionUpdated),
					resource.TestCheckResourceAttr(fullRtceTopicResourceLabel, "environment.0.id", environment),
					resource.TestCheckResourceAttr(fullRtceTopicResourceLabel, "kafka_cluster.0.id", kafkaCluster),
					resource.TestCheckResourceAttr(fullRtceTopicResourceLabel, "region", region),
					resource.TestCheckResourceAttr(fullRtceTopicResourceLabel, "topic_name", topicName),
					resource.TestCheckResourceAttr(fullRtceTopicResourceLabel, "api_version", "rtce/v1"),
					resource.TestCheckResourceAttr(fullRtceTopicResourceLabel, "kind", "RtceTopic"),
					resource.TestCheckResourceAttr(fullRtceTopicResourceLabel, "resource_name", "crn://confluent.cloud/organization=9bb441c4-edef-46ac-8a41-c49e44a3fd9a/environment=env-abc123/cloud-cluster=lkc-12345/topic=tt-12345"),
				),
			},
			{
				ResourceName:      fullRtceTopicResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, createRtceTopicStub, "POST /rtce/v1/rtce-topics", expectedCountOne)
	checkStubCount(t, wiremockClient, patchRtceTopicStub, fmt.Sprintf("PATCH /rtce/v1/rtce-topics/%s", rtce_topicTopicName), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteRtceTopicStub, fmt.Sprintf("DELETE /rtce/v1/rtce-topics/%s", rtce_topicTopicName), expectedCountOne)
}

func testAccCheckRtceRtceTopicDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each rtce_topic is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_rtce_topic" {
			continue
		}
		topicName := rs.Primary.Attributes[paramTopicName]
		req := c.rtceV1Client.RtceTopicsRtceV1Api.GetRtceV1RtceTopic(c.rtceV1ApiContext(context.Background()), topicName).Environment(rs.Primary.Attributes[paramEnvironment+".0.id"]).SpecKafkaCluster(rs.Primary.Attributes[paramKafkaCluster+".0.id"])
		deletedRtceTopic, response, err := req.Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			return nil
		} else if err == nil && deletedRtceTopic.Spec != nil {
			deletedSpec := deletedRtceTopic.GetSpec()
			if deletedSpec.GetTopicName() == topicName {
				return fmt.Errorf("rtce_topic (%q) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckRtceRtceTopicConfig(mockServerUrl, rtce_topicResourceLabel string, cloud string, description string, environment string, kafkaCluster string, region string, topicName string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_rtce_topic" "%s" {
		cloud = "%s"
		description = "%s"
		environment {
			id = "%s"
		}
		kafka_cluster {
			id = "%s"
		}
		region = "%s"
		topic_name = "%s"
	}
	`, mockServerUrl, rtce_topicResourceLabel, cloud, description, environment, kafkaCluster, region, topicName)
}

func testAccCheckRtceRtceTopicExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s rtce_topic has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s rtce_topic", n)
		}

		return nil
	}
}
