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
