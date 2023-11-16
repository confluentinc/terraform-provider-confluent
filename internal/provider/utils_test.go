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

func TestExtractFlinkAttributes(t *testing.T) {
	invalidUrlHttpScheme := "http://flink.us-east-1.aws.confluent.cloud/sql/v1beta1/organizations/1111aaaa-11aa-11aa-11aa-111111aaaaaa/environments/env-abc123"
	invalidUrlEmptyEnvironment := "https://flink.us-east-1.aws.confluent.cloud/sql/v1beta1/organizations/1111aaaa-11aa-11aa-11aa-111111aaaaaa/environments/"
	invalidUrlNoEnvironmentPathParam := "https://flink.us-east-1.aws.confluent.cloud/sql/v1beta1/organizations/1111aaaa-11aa-11aa-11aa-111111aaaaaa"
	invalidUrlExtraValue := "https://flink.us-east-1.aws.confluent.cloud/sql/v1beta1/organizations/1111aaaa-11aa-11aa-11aa-111111aaaaaa/environments/env-abc123/foo"
	tests := []struct {
		input                  string
		expectedRestEndpoint   string
		expectedOrganizationId string
		expectedEnvironmentId  string
		err                    error
	}{
		{
			input:                  "http://localhost",
			expectedRestEndpoint:   "http://localhost",
			expectedOrganizationId: flinkOrganizationIdTest,
			expectedEnvironmentId:  flinkEnvironmentIdTest,
			err:                    nil,
		},
		{
			input:                  "https://flink.us-east-1.aws.confluent.cloud/sql/v1beta1/organizations/1111aaaa-11aa-11aa-11aa-111111aaaaaa/environments/env-abc123",
			expectedRestEndpoint:   "https://flink.us-east-1.aws.confluent.cloud",
			expectedOrganizationId: flinkOrganizationIdTest,
			expectedEnvironmentId:  flinkEnvironmentIdTest,
			err:                    nil,
		},
		{
			input:                  "",
			expectedRestEndpoint:   "",
			expectedOrganizationId: "",
			expectedEnvironmentId:  "",
			err:                    fmt.Errorf("failed to parse URL: URL is empty"),
		},
		{
			input:                  invalidUrlHttpScheme,
			expectedRestEndpoint:   "",
			expectedOrganizationId: "",
			expectedEnvironmentId:  "",
			err:                    fmt.Errorf("failed to parse URL=%s: scheme must be https, expected format for the URL is %s", invalidUrlHttpScheme, exampleFlinkRestEndpoint),
		},
		{
			input:                  invalidUrlEmptyEnvironment,
			expectedRestEndpoint:   "",
			expectedOrganizationId: "",
			expectedEnvironmentId:  "",
			err:                    fmt.Errorf("failed to parse URL=%s: expected format for the URL is %s", invalidUrlEmptyEnvironment, exampleFlinkRestEndpoint),
		},
		{
			input:                  invalidUrlNoEnvironmentPathParam,
			expectedRestEndpoint:   "",
			expectedOrganizationId: "",
			expectedEnvironmentId:  "",
			err:                    fmt.Errorf("failed to parse URL=%s: expected format for the URL is %s", invalidUrlNoEnvironmentPathParam, exampleFlinkRestEndpoint),
		},
		{
			input:                  invalidUrlExtraValue,
			expectedRestEndpoint:   "",
			expectedOrganizationId: "",
			expectedEnvironmentId:  "",
			err:                    fmt.Errorf("failed to parse URL=%s: expected format for the URL is %s", invalidUrlExtraValue, exampleFlinkRestEndpoint),
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			actualRestEndpoint, actualOrganizationId, actualEnvironmentId, err := extractFlinkAttributes(tt.input)
			if !reflect.DeepEqual(err, tt.err) {
				t.Fatalf("Unexpected error: expected %v, got %v", tt.err, err)
			}
			if !reflect.DeepEqual(actualRestEndpoint, tt.expectedRestEndpoint) {
				t.Fatalf("Unexpected REST endpoint: expected %v, got %v", tt.expectedRestEndpoint, actualRestEndpoint)
			}
			if !reflect.DeepEqual(actualOrganizationId, tt.expectedOrganizationId) {
				t.Fatalf("Unexpected Organization ID: expected %v, got %v", tt.expectedOrganizationId, actualOrganizationId)
			}
			if !reflect.DeepEqual(actualEnvironmentId, tt.expectedEnvironmentId) {
				t.Fatalf("Unexpected Environment ID: expected %v, got %v", tt.expectedEnvironmentId, actualEnvironmentId)
			}
		})
	}
}
