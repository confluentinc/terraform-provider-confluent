//go:build live_test

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
	"sync"
	"testing"

	ksqlv2 "github.com/confluentinc/ccloud-sdk-go-v2/ksql/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var cleanupLeakedLiveTestTopicsOnce sync.Once

// cleanupLeakedLiveTestTopics deletes the topics earlier runs left on the shared Standard cluster (KAFKA_STANDARD_AWS_*),
// whose 2,500-partition limit they would otherwise fill up. It runs once per test binary and concurrent callers wait
// for it, so tests that create topics there start with free partitions. Errors are logged, not failed on.
func cleanupLeakedLiveTestTopics(t *testing.T) {
	cleanupLeakedLiveTestTopicsOnce.Do(func() {
		ctx := context.Background()
		kafkaRestClient := newLiveStandardKafkaRestClient(ctx)
		if kafkaRestClient == nil {
			t.Log("Skipping leaked topic cleanup: KAFKA_STANDARD_AWS_* environment variables are not set")
			return
		}

		// List topics before ksqlDB clusters, so any processing-log topic listed here whose cluster still exists
		// shows up in the ksqlDB cluster list too.
		topics, resp, err := kafkaRestClient.apiClient.TopicV3Api.ListKafkaTopics(kafkaRestClient.apiContext(ctx), kafkaRestClient.clusterId).Execute()
		if err != nil {
			t.Logf("Skipping leaked topic cleanup: error listing Kafka Topics: %s", createDescriptiveError(err, resp))
			return
		}
		activeKsqlTopicPrefixes, err := listLiveKsqlTopicPrefixes(ctx)
		if err != nil {
			t.Logf("Skipping leaked topic cleanup: %s", err)
			return
		}

		deletedCount := 0
		for _, topic := range topics.GetData() {
			if !isLeakedLiveTestTopic(topic.GetTopicName(), topic.GetPartitionsCount(), activeKsqlTopicPrefixes) {
				continue
			}
			if err := deleteLiveStandardKafkaTopic(ctx, kafkaRestClient, topic.GetTopicName()); err != nil {
				t.Logf("Error deleting leaked Kafka Topic %q: %s", topic.GetTopicName(), err)
				continue
			}
			deletedCount++
		}
		t.Logf("Deleted %d leaked Kafka Topics from Kafka Cluster %q", deletedCount, kafkaRestClient.clusterId)
	})
}

// testAccCaptureKsqlTopicPrefixLive stores the ksqlDB cluster's topic_prefix in topicPrefix for deleteKsqlProcessingLogTopicLive.
func testAccCaptureKsqlTopicPrefixLive(resourceName string, topicPrefix *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}
		*topicPrefix = rs.Primary.Attributes[paramTopicPrefix]
		return nil
	}
}

// deleteKsqlProcessingLogTopicLive deletes the processing-log topic that a destroyed ksqlDB cluster leaves on the shared
// Standard cluster. A blank topicPrefix (the cluster was never created) is a no-op.
func deleteKsqlProcessingLogTopicLive(t *testing.T, topicPrefix string) {
	if topicPrefix == "" {
		return
	}
	ctx := context.Background()
	kafkaRestClient := newLiveStandardKafkaRestClient(ctx)
	if kafkaRestClient == nil {
		t.Log("Skipping ksqlDB processing log topic cleanup: KAFKA_STANDARD_AWS_* environment variables are not set")
		return
	}
	topicName := topicPrefix + ksqlProcessingLogTopicSuffix
	if err := deleteLiveStandardKafkaTopic(ctx, kafkaRestClient, topicName); err != nil {
		t.Logf("Error deleting ksqlDB processing log Kafka Topic %q: %s", topicName, err)
	}
}

// newLiveStandardKafkaRestClient returns a Kafka REST client for the shared Standard cluster, or nil if its environment variables are not set.
func newLiveStandardKafkaRestClient(ctx context.Context) *KafkaRestClient {
	clusterId := os.Getenv("KAFKA_STANDARD_AWS_CLUSTER_ID")
	apiKey := os.Getenv("KAFKA_STANDARD_AWS_API_KEY")
	apiSecret := os.Getenv("KAFKA_STANDARD_AWS_API_SECRET")
	restEndpoint := os.Getenv("KAFKA_STANDARD_AWS_REST_ENDPOINT")
	if clusterId == "" || apiKey == "" || apiSecret == "" || restEndpoint == "" {
		return nil
	}
	return KafkaRestClientFactory{ctx: ctx}.CreateKafkaRestClient(restEndpoint, clusterId, apiKey, apiSecret, false, false, nil)
}

func deleteLiveStandardKafkaTopic(ctx context.Context, c *KafkaRestClient, topicName string) error {
	resp, err := c.apiClient.TopicV3Api.DeleteKafkaTopic(c.apiContext(ctx), c.clusterId, topicName).Execute()
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if err != nil {
		return createDescriptiveError(err, resp)
	}
	return nil
}

// listLiveKsqlTopicPrefixes returns the topic_prefix of every ksqlDB cluster in the live test environment.
func listLiveKsqlTopicPrefixes(ctx context.Context) (map[string]bool, error) {
	endpoint := os.Getenv("CONFLUENT_CLOUD_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.confluent.cloud"
	}
	cfg := ksqlv2.NewConfiguration()
	cfg.Servers[0].URL = endpoint
	cfg.HTTPClient = NewRetryableClientFactory(ctx).CreateRetryableClient()
	c := &Client{
		ksqlV2Client:   ksqlv2.NewAPIClient(cfg),
		cloudApiKey:    os.Getenv("CONFLUENT_CLOUD_API_KEY"),
		cloudApiSecret: os.Getenv("CONFLUENT_CLOUD_API_SECRET"),
	}
	ksqlClusters, err := loadKsqlClusters(ctx, c, liveTestEnvironmentId)
	if err != nil {
		return nil, err
	}
	topicPrefixes := make(map[string]bool, len(ksqlClusters))
	for _, ksqlCluster := range ksqlClusters {
		// Without its prefix, a cluster's processing-log topic can't be told apart from a leaked one.
		topicPrefix := ksqlCluster.Status.GetTopicPrefix()
		if topicPrefix == "" {
			return nil, fmt.Errorf("ksqlDB Cluster %q has no topic prefix yet", ksqlCluster.GetId())
		}
		topicPrefixes[topicPrefix] = true
	}
	return topicPrefixes, nil
}
