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
	"reflect"
	"testing"
)

func testKafkaClusterBlockStateDataV0() map[string]interface{} {
	return map[string]interface{}{
		paramKafkaCluster: kafkaClusterId,
	}
}

func testKafkaClusterBlockStateDataV1() map[string]interface{} {
	v0 := testKafkaClusterBlockStateDataV0()
	kafkaClusterIdV0 := v0[paramKafkaCluster].(string)
	return map[string]interface{}{
		paramKafkaCluster: []interface{}{map[string]interface{}{
			paramId: kafkaClusterIdV0,
		}},
	}
}

func TestKafkaAclResourceStateUpgradeV0(t *testing.T) {
	expected := testKafkaClusterBlockStateDataV1()
	actual, err := kafkaClusterBlockStateUpgradeV0(context.Background(), testKafkaClusterBlockStateDataV0(), nil)
	if err != nil {
		t.Fatalf("error migrating state: %s", err)
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\n\nexpected:\n\n%#v\n\ngot:\n\n%#v\n\n", expected, actual)
	}
}

func TestExtractOrgIdFromResourceName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		err      error
	}{
		{
			input:    "crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa",
			expected: "1111aaaa-11aa-11aa-11aa-111111aaaaaa",
			err:      nil,
		},
		{
			input:    "crn://confluent.cloud/organization=1111aaaa/environment=env-abc123",
			expected: "1111aaaa",
			err:      nil,
		},
		{
			input:    "crn://confluent.cloud/environment=env-abc123",
			expected: "",
			err:      fmt.Errorf("could not find organization ID in resource_name: crn://confluent.cloud/environment=env-abc123"),
		},
		{
			input:    "crn://api.confluent.cloud/organization=foo/service-account=sa-12mgdv",
			expected: "foo",
			err:      nil,
		},
		{
			input:    "crn://api.confluent.cloud/organization=foo/environment=env-3732nw/flink-region=aws.us-east-1",
			expected: "foo",
			err:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := extractOrgIdFromResourceName(tt.input)
			if !reflect.DeepEqual(err, tt.err) {
				t.Fatalf("Unexpected error: expected %v, got %v", tt.err, err)
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Fatalf("Unexpected result: expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCanUpdateEntityName(t *testing.T) {
	tests := []struct {
		entityType    string
		oldEntityName string
		newEntityName string
		expected      bool
	}{
		{
			entityType:    schemaEntityType,
			oldEntityName: "lsrc-foobar:.:100002",
			newEntityName: "lsrc-foobar:.:100003",
			expected:      true,
		},
		{
			entityType:    schemaEntityType,
			oldEntityName: "lsrc-foobar:.:100002",
			newEntityName: "lsrc-foobar:.:100002",
			expected:      false,
		},
		{
			entityType:    schemaEntityType,
			oldEntityName: "lsrc-foobar:.:100003",
			newEntityName: "lsrc-foobar:.:100002",
			expected:      false,
		},
		{
			entityType:    recordEntityType,
			oldEntityName: "lsrc-foobar:.:100004:org.apache.flink.avro.generated.record",
			newEntityName: "lsrc-foobar:.:100005:org.apache.flink.avro.generated.record",
			expected:      true,
		},
		{
			entityType:    recordEntityType,
			oldEntityName: "lsrc-foobar:.:",
			newEntityName: "lsrc-foobar:.:",
			expected:      false,
		},
		{
			entityType:    fieldEntityType,
			oldEntityName: "lsrc-foobar:.:100006:org.apache.flink.avro.generated.record.random_value",
			newEntityName: "lsrc-foobar:.:100007:org.apache.flink.avro.generated.record.random_value",
			expected:      true,
		},
		{
			entityType:    fieldEntityType,
			oldEntityName: "flink.avro.generated.record.random_value",
			newEntityName: "flink.avro.generated.record.random_value",
			expected:      false,
		},
		{
			entityType:    "different_type",
			oldEntityName: "lsrc-foobar:.:100002",
			newEntityName: "lsrc-foobar:.:100003",
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.entityType+" "+tt.oldEntityName+" to "+tt.newEntityName, func(t *testing.T) {
			result := canUpdateEntityName(tt.entityType, tt.oldEntityName, tt.newEntityName)
			if result != tt.expected {
				t.Errorf("canUpdateEntityName(%s, %s, %s) = %v; want %v", tt.entityType, tt.oldEntityName, tt.newEntityName, result, tt.expected)
			}
		})
	}
}
