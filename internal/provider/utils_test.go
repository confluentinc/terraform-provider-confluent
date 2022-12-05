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
	"github.com/stretchr/testify/assert"
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

func TestCompareDifferentProtos(t *testing.T) {
	requestProto := "syntax = \"proto3\";\npackage io.confluent.developer.proto;\n\noption java_outer_classname = \"PurchaseProto\";\n\nmessage Purchase {\n  string item = 1;\n  double amount = 2;\n  string customer_id = 3;\n}\n"
	responseProto := "syntax = \"proto3\";\npackage io.confluent.developer.proto;\n\noption java_outer_classname = \"PurchaseProto\";\n\nmessage Purchase2 {\n  string item_2 = 1;\n  double amount = 2;\n  string customer_id = 3;\n}\n"

	assert.False(t, compareTwoProtos(requestProto, responseProto), "The two protos should be different:", requestProto, responseProto)
}

func TestCompareSameProtos(t *testing.T) {
	requestProto := "syntax = \"proto3\";\npackage io.confluent.developer.proto;\n\noption java_outer_classname = \"PurchaseProto\";\n\nmessage Purchase {\n  string item = 1;\n  double amount = 2;\n  string customer_id = 3;\n}\n"
	responseProto := "syntax = \"proto3\";\npackage io.confluent.developer.proto;\n\noption java_outer_classname = \"PurchaseProto\";\n\nmessage Purchase {\n  string item = 1;\n  double amount = 2;\n  string customer_id = 3;\n}\n"

	assert.True(t, compareTwoProtos(requestProto, responseProto), "The two protos should be the same:", requestProto, responseProto)
}

func TestCompareProtosWithExtraNewLinesAndWhitespaces(t *testing.T) {
	requestProto := "syntax = \"proto3\";\n\n\npackage io.confluent.developer.proto;\n\noption java_outer_classname = \"PurchaseProto\";\n\nmessage Purchase {\n  string  item      = 1;\n  double      amount =  2        ;\n  string  customer_id = 3;\n}\n"
	responseProto := "syntax = \"proto3\";\npackage io.confluent.developer.proto;\n\noption java_outer_classname = \"PurchaseProto\";\n\nmessage Purchase {\n  string item = 1;\n  double amount = 2;\n  string customer_id = 3;\n}\n"

	assert.True(t, compareTwoProtos(requestProto, responseProto), "The two protos should be the same:", requestProto, responseProto)
}

func TestCompareProtosWithExtraNewLine(t *testing.T) {
	requestProto := "syntax = \"proto3\";\n\npackage io.confluent.developer.proto;\noption java_outer_classname = \"PageViewProto\";\n\nmessage PageView {\n  string url = 1;\n  bool is_special = 2;\n  string id = 3;\n}\n"
	responseProto := "syntax = \"proto3\";\npackage io.confluent.developer.proto;\n\noption java_outer_classname = \"PageViewProto\";\n\nmessage PageView {\n  string url = 1;\n  bool is_special = 2;\n  string id = 3;\n}\n"

	assert.True(t, compareTwoProtos(requestProto, responseProto), "The two protos should be the same:", requestProto, responseProto)
}

func TestCompareProtosWithReorderedAttributes(t *testing.T) {
	requestProto := "syntax = \"proto3\";\n\npackage io.confluent.developer.proto;\n\nimport \"purchase.proto\";\nimport \"page_view.proto\";\n\noption java_outer_classname = \"CustomerEventProto\";\n\nmessage CustomerEvent {\n\n  oneof action {\n    Purchase purchase = 1;\n    PageView page_view = 2;\n  }\n  string id = 3;\n}\n"
	responseProto := "syntax = \"proto3\";\npackage io.confluent.developer.proto;\n\nimport \"purchase.proto\";\nimport \"page_view.proto\";\n\noption java_outer_classname = \"CustomerEventProto\";\n\nmessage CustomerEvent {\n  string id = 3;\n\n  oneof action {\n    Purchase purchase = 1;\n    PageView page_view = 2;\n  }\n}\n"

	assert.True(t, compareTwoProtos(requestProto, responseProto), "The two protos should be the same:", requestProto, responseProto)
}

func TestCompareProtosWithRemovedNewLines(t *testing.T) {
	requestProto := "syntax = \"proto3\"; package io.confluent.developer.proto;\noption java_outer_classname = \"PurchaseProto\";\nmessage Purchase {string item = 1;double amount = 2;string customer_id          = 3;}\n"
	responseProto := "syntax = \"proto3\";\npackage io.confluent.developer.proto;\n\noption java_outer_classname = \"PurchaseProto\";\n\nmessage Purchase {\n  string item = 1;\n  double amount = 2;\n  string customer_id = 3;\n}\n"

	assert.True(t, compareTwoProtos(requestProto, responseProto), "The two protos should be the same:", requestProto, responseProto)
}
