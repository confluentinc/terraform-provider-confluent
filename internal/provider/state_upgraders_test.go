// Copyright 2022 Confluent Inc. All Rights Reserved.
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
	"reflect"
	"testing"
)

const testEndpoint = "https://pkc-012345.us-central1.gcp.confluent.cloud:443"

func testKafkaStateDataV010() map[string]interface{} {
	return map[string]interface{}{
		paramHttpEndpoint: testEndpoint,
	}
}

func testKafkaStateDataV011() map[string]interface{} {
	return map[string]interface{}{
		paramRestEndpoint: testEndpoint,
	}
}

func testKafkaStateDataV1() map[string]interface{} {
	return map[string]interface{}{
		paramRestEndpoint: testEndpoint,
	}
}

func testKafkaStateDataV010Empty() map[string]interface{} {
	return map[string]interface{}{
		paramHttpEndpoint: "",
	}
}

func testKafkaStateDataV1Empty() map[string]interface{} {
	return map[string]interface{}{
		paramRestEndpoint: "",
	}
}

func TestResourceExampleInstanceStateUpgradeV010(t *testing.T) {
	expected := testKafkaStateDataV1()
	actual, err := kafkaStateUpgradeV0(context.Background(), testKafkaStateDataV010(), nil)
	if err != nil {
		t.Fatalf("error migrating state: %s", err)
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\n\nexpected:\n\n%#v\n\ngot:\n\n%#v\n\n", expected, actual)
	}
}

func TestResourceExampleInstanceStateUpgradeV011(t *testing.T) {
	expected := testKafkaStateDataV1()
	actual, err := kafkaStateUpgradeV0(context.Background(), testKafkaStateDataV011(), nil)
	if err != nil {
		t.Fatalf("error migrating state: %s", err)
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\n\nexpected:\n\n%#v\n\ngot:\n\n%#v\n\n", expected, actual)
	}
}

func TestResourceExampleInstanceStateUpgradeV010Empty(t *testing.T) {
	expected := testKafkaStateDataV1Empty()
	actual, err := kafkaStateUpgradeV0(context.Background(), testKafkaStateDataV010Empty(), nil)
	if err != nil {
		t.Fatalf("error migrating state: %s", err)
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\n\nexpected:\n\n%#v\n\ngot:\n\n%#v\n\n", expected, actual)
	}
}

// TestKafkaStateUpgradeV0BothEndpoints verifies that when the state has BOTH
// http_endpoint and rest_endpoint, the http_endpoint value overwrites rest_endpoint
// and http_endpoint is removed.
func TestKafkaStateUpgradeV0BothEndpoints(t *testing.T) {
	httpValue := "https://pkc-http.us-central1.gcp.confluent.cloud:443"
	restValue := "https://pkc-rest.us-central1.gcp.confluent.cloud:443"

	rawState := map[string]interface{}{
		paramHttpEndpoint: httpValue,
		paramRestEndpoint: restValue,
	}

	expected := map[string]interface{}{
		paramRestEndpoint: httpValue,
	}

	actual, err := kafkaStateUpgradeV0(context.Background(), rawState, nil)
	if err != nil {
		t.Fatalf("error migrating state: %s", err)
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\n\nexpected:\n\n%#v\n\ngot:\n\n%#v\n\n", expected, actual)
	}

	if _, found := actual[paramHttpEndpoint]; found {
		t.Fatalf("expected http_endpoint to be removed from state, but it was still present")
	}
}

// TestKafkaStateUpgradeV0NeitherEndpoint verifies that when the state has NEITHER
// http_endpoint nor rest_endpoint, the upgrade succeeds without error and returns
// the state unchanged.
func TestKafkaStateUpgradeV0NeitherEndpoint(t *testing.T) {
	rawState := map[string]interface{}{
		paramDisplayName: "my-cluster",
	}

	expected := map[string]interface{}{
		paramDisplayName: "my-cluster",
	}

	actual, err := kafkaStateUpgradeV0(context.Background(), rawState, nil)
	if err != nil {
		t.Fatalf("error migrating state: %s", err)
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\n\nexpected:\n\n%#v\n\ngot:\n\n%#v\n\n", expected, actual)
	}

	if _, found := actual[paramRestEndpoint]; found {
		t.Fatalf("expected rest_endpoint to be absent when neither endpoint was in the original state")
	}
}

// TestKafkaStateUpgradeV0PreservesAdditionalFields verifies that all fields besides
// http_endpoint are preserved through the migration.
func TestKafkaStateUpgradeV0PreservesAdditionalFields(t *testing.T) {
	rawState := map[string]interface{}{
		paramHttpEndpoint:      testEndpoint,
		paramDisplayName:       "my-cluster",
		paramCloud:             "GCP",
		paramRegion:            "us-central1",
		paramAvailability:      "SINGLE_ZONE",
		paramBootStrapEndpoint: "SASL_SSL://pkc-012345.us-central1.gcp.confluent.cloud:9092",
		paramRbacCrn:           "crn://confluent.cloud/kafka=lkc-012345",
	}

	expected := map[string]interface{}{
		paramRestEndpoint:      testEndpoint,
		paramDisplayName:       "my-cluster",
		paramCloud:             "GCP",
		paramRegion:            "us-central1",
		paramAvailability:      "SINGLE_ZONE",
		paramBootStrapEndpoint: "SASL_SSL://pkc-012345.us-central1.gcp.confluent.cloud:9092",
		paramRbacCrn:           "crn://confluent.cloud/kafka=lkc-012345",
	}

	actual, err := kafkaStateUpgradeV0(context.Background(), rawState, nil)
	if err != nil {
		t.Fatalf("error migrating state: %s", err)
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\n\nexpected:\n\n%#v\n\ngot:\n\n%#v\n\n", expected, actual)
	}

	if _, found := actual[paramHttpEndpoint]; found {
		t.Fatalf("expected http_endpoint to be removed from state, but it was still present")
	}

	// Verify each additional field individually for clearer failure messages
	for _, key := range []string{paramDisplayName, paramCloud, paramRegion, paramAvailability, paramBootStrapEndpoint, paramRbacCrn} {
		if actual[key] != expected[key] {
			t.Fatalf("field %q was not preserved: expected %q, got %q", key, expected[key], actual[key])
		}
	}
}

// TestKafkaStateUpgradeV0ModifiesInPlace verifies that kafkaStateUpgradeV0 modifies
// the original rawState map in place rather than returning a copy.
func TestKafkaStateUpgradeV0ModifiesInPlace(t *testing.T) {
	rawState := map[string]interface{}{
		paramHttpEndpoint: testEndpoint,
		paramDisplayName:  "my-cluster",
	}

	actual, err := kafkaStateUpgradeV0(context.Background(), rawState, nil)
	if err != nil {
		t.Fatalf("error migrating state: %s", err)
	}

	// Verify the returned map is the same reference as the input map by checking
	// that mutations to rawState are visible through actual and vice versa.
	rawState["sentinel_key"] = "sentinel_value"
	if actual["sentinel_key"] != "sentinel_value" {
		t.Fatalf("expected returned map to be the same reference as the input map (modified in place), but it appears to be a copy")
	}

	// Also verify the original rawState was modified by the upgrade function
	if _, found := rawState[paramHttpEndpoint]; found {
		t.Fatalf("expected http_endpoint to be removed from the original rawState map")
	}
	if rawState[paramRestEndpoint] != testEndpoint {
		t.Fatalf("expected rest_endpoint to be set on the original rawState map, got %q", rawState[paramRestEndpoint])
	}
}
