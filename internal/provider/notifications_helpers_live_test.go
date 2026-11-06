//go:build live_test && (all || core)

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
	"net/http"
	"os"
	"testing"

	notificationsv1 "github.com/confluentinc/ccloud-sdk-go-v2/notifications/v1"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	// notificationsLiveNotificationTypeId is "Kafka cluster expansion complete", whose resource
	// type is CLUSTER.
	notificationsLiveNotificationTypeId = "nt-XpNKG"
	notificationsLiveIntegration        = "confluent_notifications_integration.notifications_live"
)

type notificationsLiveEnv struct {
	endpoint           string
	apiKey             string
	apiSecret          string
	dedicatedClusterId string
	notificationTypeId string
	slackWebhookUrl    string
}

// loadNotificationsLiveEnv skips the test unless TF_ACC_PROD is set and fails it when a
// variable the caller asked for is missing.
func loadNotificationsLiveEnv(t *testing.T, needCluster bool) notificationsLiveEnv {
	t.Helper()
	if os.Getenv("TF_ACC_PROD") == "" {
		t.Skip("Skipping live test. Set TF_ACC_PROD=1 to run this test.")
	}

	env := notificationsLiveEnv{
		endpoint:           resolveNotificationsLiveEndpoint(),
		apiKey:             os.Getenv("CONFLUENT_CLOUD_API_KEY"),
		apiSecret:          os.Getenv("CONFLUENT_CLOUD_API_SECRET"),
		dedicatedClusterId: os.Getenv("KAFKA_DEDICATED_AWS_CLUSTER_ID"),
		notificationTypeId: notificationsLiveNotificationTypeId,
		slackWebhookUrl:    os.Getenv("CONFLUENT_CLOUD_SLACK_TARGET_WEBHOOK_URL"),
	}

	required := map[string]string{
		"CONFLUENT_CLOUD_API_KEY":    env.apiKey,
		"CONFLUENT_CLOUD_API_SECRET": env.apiSecret,
	}
	if needCluster {
		required["KAFKA_DEDICATED_AWS_CLUSTER_ID"] = env.dedicatedClusterId
	}
	if env.slackWebhookUrl == "" {
		env.slackWebhookUrl = "https://hooks.slack.com/services/T00000000/B00000000/tf_live_notifications"
	}
	for name, value := range required {
		if value == "" {
			t.Fatalf("%s must be set for notifications live tests", name)
		}
	}
	return env
}

// resolveNotificationsLiveEndpoint returns the Confluent Cloud API endpoint.
// Priority: CONFLUENT_CLOUD_ENDPOINT (full URL override) > TF_ACC_ENV (shorthand) > prod default.
func resolveNotificationsLiveEndpoint() string {
	if ep := os.Getenv("CONFLUENT_CLOUD_ENDPOINT"); ep != "" {
		return ep
	}
	switch os.Getenv("TF_ACC_ENV") {
	case "devel":
		return "https://api.devel.cpdev.cloud"
	case "stag":
		return "https://api.stag.cpdev.cloud"
	default:
		return "https://api.confluent.cloud"
	}
}

func (env notificationsLiveEnv) providerConfig() string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = %q
		cloud_api_key    = %q
		cloud_api_secret = %q
	}
	`, env.endpoint, env.apiKey, env.apiSecret)
}

// clusterCrn is the RBAC CRN of the long-lived dedicated cluster in the live-test environment,
// the form the notifications API resolves to a resource name and scope. It's built here rather
// than read with a confluent_kafka_cluster data source because ImportStateVerify fails on any
// data source in the config.
func (env notificationsLiveEnv) clusterCrn() string {
	return fmt.Sprintf("crn://confluent.cloud/organization=%s/environment=%s/cloud-cluster=%s",
		liveTestOrganizationId, liveTestEnvironmentId, env.dedicatedClusterId)
}

func (env notificationsLiveEnv) integrationConfig(randomSuffix int) string {
	return fmt.Sprintf(`
	resource "confluent_notifications_integration" "notifications_live" {
		display_name = "tf_live_notifications_%d"
		slack_target {
			webhook_url = %q
		}
	}
	`, randomSuffix, env.slackWebhookUrl)
}

// deleteOrgSubscriptions removes every org-level subscription to the test notification type.
// The API allows one per type, and the org can already hold one that no test created.
func (env notificationsLiveEnv) deleteOrgSubscriptions(t *testing.T) {
	t.Helper()
	cfg := notificationsv1.NewConfiguration()
	cfg.Servers[0].URL = env.endpoint
	api := notificationsv1.NewAPIClient(cfg).SubscriptionsNotificationsV1Api
	ctx := context.WithValue(context.Background(), notificationsv1.ContextBasicAuth, notificationsv1.BasicAuth{
		UserName: env.apiKey,
		Password: env.apiSecret,
	})

	var ids []string
	pageToken := ""
	for {
		req := api.ListNotificationsV1Subscriptions(ctx).PageSize(100)
		if pageToken != "" {
			req = req.PageToken(pageToken)
		}
		list, _, err := req.Execute()
		if err != nil {
			t.Fatalf("listing notification subscriptions: %s", createDescriptiveError(err))
		}
		for _, sub := range list.GetData() {
			if sub.GetNotificationType().Id == env.notificationTypeId {
				ids = append(ids, sub.GetId())
			}
		}
		meta := list.GetMetadata()
		next := meta.GetNext()
		if next == "" {
			break
		}
		if pageToken, err = extractPageToken(next); err != nil {
			t.Fatalf("paging notification subscriptions: %s", err)
		}
	}

	for _, id := range ids {
		resp, err := api.DeleteNotificationsV1Subscription(ctx, id).Execute()
		if err != nil && (resp == nil || resp.StatusCode != http.StatusNotFound) {
			t.Fatalf("deleting existing notification subscription %s: %s", id, createDescriptiveError(err, resp))
		}
	}
}

func testAccCheckNotificationsLiveExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("resource ID is not set")
		}
		return nil
	}
}
