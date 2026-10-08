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
	"time"

	cmkv2 "github.com/confluentinc/ccloud-sdk-go-v2/cmk/v2"
	ksqlv2 "github.com/confluentinc/ccloud-sdk-go-v2/ksql/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Standard clusters allow 500 partition creations and deletions per 5 minutes, so the cleanup deletes at most half
// that per test run to leave room for the tests' own topics. Later runs delete whatever leaked topics remain.
const maxLeakedPartitionsDeletedPerRun = 250

// Tests wait for the leaked topic cleanups, so they give up after this long.
const leakedLiveTestTopicsCleanupTimeout = 15 * time.Minute

var (
	cleanupLeakedLiveTestTopicsOnce    sync.Once
	cleanupLeakedLiveTestTopicsSummary string
)

// cleanupLeakedLiveTestTopics deletes the topics earlier runs left on the shared Standard cluster (KAFKA_STANDARD_AWS_*),
// whose 2,500-partition limit they would otherwise fill up. It runs once per test binary and concurrent callers wait
// for it, so tests that create topics there start with free partitions. Every caller logs the outcome; errors don't fail the test.
func cleanupLeakedLiveTestTopics(t *testing.T) {
	cleanupLeakedLiveTestTopicsOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), leakedLiveTestTopicsCleanupTimeout)
		defer cancel()
		cleanupLeakedLiveTestTopicsSummary = deleteLeakedLiveTestTopics(ctx)
	})
	t.Log(cleanupLeakedLiveTestTopicsSummary)
}

// deleteLeakedLiveTestTopics deletes leaked topics from the shared Standard cluster and returns a summary of what it did.
func deleteLeakedLiveTestTopics(ctx context.Context) string {
	kafkaRestClient := newLiveStandardKafkaRestClient(ctx)
	if kafkaRestClient == nil {
		return "Skipped leaked topic cleanup: KAFKA_STANDARD_AWS_* environment variables are not all set"
	}
	c := newLiveCloudClient(ctx)

	// Only ksqlDB clusters in liveTestEnvironmentId are checked below, so only clean up a Kafka cluster in that environment.
	kafkaCluster, resp, err := c.cmkV2Client.ClustersCmkV2Api.GetCmkV2Cluster(c.cmkV2ApiContext(ctx), kafkaRestClient.clusterId).Environment(liveTestEnvironmentId).Execute()
	if err != nil {
		return fmt.Sprintf("Skipped leaked topic cleanup: error reading Kafka Cluster %q in environment %q: %s", kafkaRestClient.clusterId, liveTestEnvironmentId, createDescriptiveError(err, resp))
	}
	if environmentId := kafkaCluster.GetSpec().Environment.GetId(); environmentId != liveTestEnvironmentId {
		return fmt.Sprintf("Skipped leaked topic cleanup: Kafka Cluster %q is in environment %q, not %q", kafkaRestClient.clusterId, environmentId, liveTestEnvironmentId)
	}

	// List topics before ksqlDB clusters, so any processing log topic listed here whose cluster still exists
	// shows up in the ksqlDB cluster list too.
	topics, resp, err := kafkaRestClient.apiClient.TopicV3Api.ListKafkaTopics(kafkaRestClient.apiContext(ctx), kafkaRestClient.clusterId).Execute()
	if err != nil {
		return fmt.Sprintf("Skipped leaked topic cleanup: error listing Kafka Topics: %s", createDescriptiveError(err, resp))
	}
	activeKsqlTopicPrefixes, err := listLiveKsqlTopicPrefixes(ctx, c)
	if err != nil {
		return fmt.Sprintf("Skipped leaked topic cleanup: %s", err)
	}

	deletedTopics, deletedPartitions, failedTopics := 0, int32(0), 0
	attemptedPartitions := int32(0)
	var firstErr error
	reachedLimit := false
	for _, topic := range topics.GetData() {
		if ctx.Err() != nil {
			break
		}
		if !isLeakedLiveTestTopic(topic.GetTopicName(), topic.GetPartitionsCount(), activeKsqlTopicPrefixes) {
			continue
		}
		// Count every topic as at least 1 partition, and count failed deletes too, so the limit always bounds the deletes.
		partitions := max(topic.GetPartitionsCount(), 1)
		if partitions > maxLeakedPartitionsDeletedPerRun-attemptedPartitions {
			reachedLimit = true
			continue
		}
		attemptedPartitions += partitions
		if err := deleteLiveStandardKafkaTopic(ctx, kafkaRestClient, topic.GetTopicName()); err != nil {
			failedTopics++
			if firstErr == nil {
				firstErr = fmt.Errorf("error deleting Kafka Topic %q: %s", topic.GetTopicName(), err)
			}
			continue
		}
		deletedTopics++
		deletedPartitions += partitions
	}

	summary := fmt.Sprintf("Deleted %d leaked Kafka Topics (%d partitions) from Kafka Cluster %q", deletedTopics, deletedPartitions, kafkaRestClient.clusterId)
	if failedTopics > 0 {
		summary += fmt.Sprintf("; failed to delete %d (first error: %s)", failedTopics, firstErr)
	}
	if ctx.Err() != nil {
		summary += fmt.Sprintf("; stopped at the %s time limit, later runs will delete the rest", leakedLiveTestTopicsCleanupTimeout)
	} else if reachedLimit {
		summary += fmt.Sprintf("; stopped at the %d-partition limit per run, later runs will delete the rest", maxLeakedPartitionsDeletedPerRun)
	}
	return summary
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

// deleteKsqlProcessingLogTopicLive deletes the processing log topic that a destroyed ksqlDB cluster leaves on the shared
// Standard cluster. It does nothing for a blank topicPrefix (nothing was captured, e.g. the cluster was never created).
// If the test failed, its ksqlDB cluster may not have been destroyed, so the topic is left to a later run's cleanup.
func deleteKsqlProcessingLogTopicLive(t *testing.T, topicPrefix string) {
	if topicPrefix == "" {
		return
	}
	topicName := topicPrefix + ksqlProcessingLogTopicSuffix
	if t.Failed() {
		t.Logf("Skipping deletion of ksqlDB processing log Kafka Topic %q: the test failed, so its ksqlDB cluster may still exist", topicName)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), leakedLiveTestTopicsCleanupTimeout)
	defer cancel()
	kafkaRestClient := newLiveStandardKafkaRestClient(ctx)
	if kafkaRestClient == nil {
		t.Log("Skipping ksqlDB processing log topic cleanup: KAFKA_STANDARD_AWS_* environment variables are not all set")
		return
	}
	if err := deleteLiveStandardKafkaTopic(ctx, kafkaRestClient, topicName); err != nil {
		t.Logf("Error deleting ksqlDB processing log Kafka Topic %q: %s", topicName, err)
	}
}

// newLiveStandardKafkaRestClient returns a Kafka REST client for the shared Standard cluster, or nil if its environment variables are not all set.
func newLiveStandardKafkaRestClient(ctx context.Context) *KafkaRestClient {
	clusterId := os.Getenv("KAFKA_STANDARD_AWS_CLUSTER_ID")
	apiKey := os.Getenv("KAFKA_STANDARD_AWS_API_KEY")
	apiSecret := os.Getenv("KAFKA_STANDARD_AWS_API_SECRET")
	restEndpoint := os.Getenv("KAFKA_STANDARD_AWS_REST_ENDPOINT")
	if clusterId == "" || apiKey == "" || apiSecret == "" || restEndpoint == "" {
		return nil
	}
	return KafkaRestClientFactory{ctx: ctx, userAgent: "terraform-provider-confluent-live-test"}.CreateKafkaRestClient(restEndpoint, clusterId, apiKey, apiSecret, false, false, nil)
}

// deleteLiveStandardKafkaTopic deletes topicName, treating a topic that's already gone as deleted.
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

// newLiveCloudClient returns a Client with just the Kafka Cluster and ksqlDB Cluster APIs, authenticated with CONFLUENT_CLOUD_API_KEY.
func newLiveCloudClient(ctx context.Context) *Client {
	endpoint := os.Getenv("CONFLUENT_CLOUD_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.confluent.cloud"
	}
	cmkV2Cfg := cmkv2.NewConfiguration()
	cmkV2Cfg.Servers[0].URL = endpoint
	cmkV2Cfg.HTTPClient = NewRetryableClientFactory(ctx).CreateRetryableClient()
	ksqlV2Cfg := ksqlv2.NewConfiguration()
	ksqlV2Cfg.Servers[0].URL = endpoint
	ksqlV2Cfg.HTTPClient = NewRetryableClientFactory(ctx).CreateRetryableClient()
	return &Client{
		cmkV2Client:    cmkv2.NewAPIClient(cmkV2Cfg),
		ksqlV2Client:   ksqlv2.NewAPIClient(ksqlV2Cfg),
		cloudApiKey:    os.Getenv("CONFLUENT_CLOUD_API_KEY"),
		cloudApiSecret: os.Getenv("CONFLUENT_CLOUD_API_SECRET"),
	}
}

// listLiveKsqlTopicPrefixes returns the topic_prefix of every ksqlDB cluster in the live test environment.
func listLiveKsqlTopicPrefixes(ctx context.Context, c *Client) (map[string]bool, error) {
	ksqlClusters, err := loadKsqlClusters(ctx, c, liveTestEnvironmentId)
	if err != nil {
		return nil, err
	}
	topicPrefixes := make(map[string]bool, len(ksqlClusters))
	for _, ksqlCluster := range ksqlClusters {
		// Without its prefix, a cluster's processing log topic can't be told apart from a leaked one.
		topicPrefix := ksqlCluster.Status.GetTopicPrefix()
		if topicPrefix == "" {
			return nil, fmt.Errorf("ksqlDB Cluster %q has no topic prefix yet", ksqlCluster.GetId())
		}
		topicPrefixes[topicPrefix] = true
	}
	return topicPrefixes, nil
}
