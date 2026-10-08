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
	"strings"
	"testing"
)

// A ksqlDB cluster writes its processing log to "<topic_prefix>-processing-log", where topic_prefix is "pksqlc-<id>".
const ksqlProcessingLogTopicSuffix = "-processing-log"

// isLeakedLiveTestTopic reports whether a topic on the shared live-test Kafka cluster was left behind by an earlier run:
//   - "<topic_prefix>-processing-log", which a deleted ksqlDB cluster leaves behind, once no ksqlDB cluster uses that prefix.
//   - A 1-partition "tf-live-connector-*" topic. The connector live tests create theirs with 6 partitions, so a 1-partition
//     one was re-created after the test deleted the original, most likely by its Datagen connector while shutting down.
func isLeakedLiveTestTopic(topicName string, partitionsCount int32, activeKsqlTopicPrefixes map[string]bool) bool {
	if strings.HasPrefix(topicName, "pksqlc-") && strings.HasSuffix(topicName, ksqlProcessingLogTopicSuffix) {
		return !activeKsqlTopicPrefixes[strings.TrimSuffix(topicName, ksqlProcessingLogTopicSuffix)]
	}
	return strings.HasPrefix(topicName, "tf-live-connector-") && partitionsCount == 1
}

func TestIsLeakedLiveTestTopic(t *testing.T) {
	activeKsqlTopicPrefixes := map[string]bool{"pksqlc-active": true}
	tests := []struct {
		topicName       string
		partitionsCount int32
		want            bool
	}{
		{"pksqlc-deleted-processing-log", 8, true},
		{"pksqlc-active-processing-log", 8, false},
		{"pksqlc-activex-processing-log", 8, true},
		{"pksqlc-act-processing-log", 8, true},
		{"pksqlc-deleted", 8, false},
		{"pksqlc-deleted-processing-log-copy", 8, false},
		{"my-pksqlc-deleted-processing-log", 8, false},
		{"tf-live-connector-123-topic", 1, true},
		{"tf-live-connector-update-123-topic", 1, true},
		{"tf-live-connector-123-topic", 6, false},
		{"tf-live-topic-123", 1, false},
		{"orders", 1, false},
	}
	for _, tt := range tests {
		if got := isLeakedLiveTestTopic(tt.topicName, tt.partitionsCount, activeKsqlTopicPrefixes); got != tt.want {
			t.Errorf("isLeakedLiveTestTopic(%q, %d) = %t, want %t", tt.topicName, tt.partitionsCount, got, tt.want)
		}
	}
}
