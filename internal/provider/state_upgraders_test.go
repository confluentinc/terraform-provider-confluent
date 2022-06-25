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
