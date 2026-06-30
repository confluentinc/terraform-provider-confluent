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
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	apikeysv2 "github.com/confluentinc/ccloud-sdk-go-v2/apikeys/v2"
	ccpmv1 "github.com/confluentinc/ccloud-sdk-go-v2/ccpm/v1"
	kafkarestv3 "github.com/confluentinc/ccloud-sdk-go-v2/kafkarest/v3"
	networkingdnsforwarderv1 "github.com/confluentinc/ccloud-sdk-go-v2/networking-dnsforwarder/v1"
	schemaregistryv1 "github.com/confluentinc/ccloud-sdk-go-v2/schema-registry/v1"
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

func TestIsSchemaRegistryApiKey(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   apikeysv2.IamV2ApiKey
		expected bool
	}{
		{
			name: "SR API Key with api_version=srcm/v3",
			apiKey: apikeysv2.IamV2ApiKey{
				Spec: &apikeysv2.IamV2ApiKeySpec{
					Resource: &apikeysv2.ObjectReference{
						Kind:       apikeysv2.PtrString(schemaRegistryKind),
						ApiVersion: apikeysv2.PtrString(srcmV3ApiVersion),
					},
				},
			},
			expected: true,
		},
		{
			name: "SR API Key with api_version=srcm/v2",
			apiKey: apikeysv2.IamV2ApiKey{
				Spec: &apikeysv2.IamV2ApiKeySpec{
					Resource: &apikeysv2.ObjectReference{
						Kind:       apikeysv2.PtrString(schemaRegistryKind),
						ApiVersion: apikeysv2.PtrString(srcmV2ApiVersion),
					},
				},
			},
			expected: true,
		},
		{
			name: "Kafka API Key",
			apiKey: apikeysv2.IamV2ApiKey{
				Spec: &apikeysv2.IamV2ApiKeySpec{
					Resource: &apikeysv2.ObjectReference{
						Kind:       apikeysv2.PtrString(schemaRegistryKind),
						ApiVersion: apikeysv2.PtrString(cmkApiVersion),
					},
				},
			},
			expected: false,
		},
		{
			name: "Cloud API Key",
			apiKey: apikeysv2.IamV2ApiKey{
				Spec: &apikeysv2.IamV2ApiKeySpec{
					Resource: &apikeysv2.ObjectReference{
						Kind:       apikeysv2.PtrString("Cloud"),
						ApiVersion: apikeysv2.PtrString(iamApiVersion),
					},
				},
			},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSchemaRegistryApiKey(tt.apiKey)
			if result != tt.expected {
				t.Errorf("%s: isSchemaRegistryApiKey() = %v; want %v", tt.name, result, tt.expected)
			}
		})
	}
}

func TestConvertToStringObjectMap(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		map1 := map[string]string{
			"example": "zone1,project1",
		}
		map1Expected := map[string]networkingdnsforwarderv1.NetworkingV1ForwardViaGcpDnsZonesDomainMappings{
			"example": {Zone: networkingdnsforwarderv1.PtrString("zone1"), Project: networkingdnsforwarderv1.PtrString("project1")},
		}
		actual, _ := convertToStringObjectMap(map1)

		if !reflect.DeepEqual(actual, map1Expected) {
			t.Fatalf("Unexpected error: expected %v, got %v", map1Expected, actual)
		}
	})

	t.Run("success, extra spaces", func(t *testing.T) {
		map1 := map[string]string{
			"example": " zone1,  project1",
		}
		map1Expected := map[string]networkingdnsforwarderv1.NetworkingV1ForwardViaGcpDnsZonesDomainMappings{
			"example": {Zone: networkingdnsforwarderv1.PtrString("zone1"), Project: networkingdnsforwarderv1.PtrString("project1")},
		}
		actual, _ := convertToStringObjectMap(map1)

		if !reflect.DeepEqual(actual, map1Expected) {
			t.Fatalf("Unexpected error: expected %v, got %v", map1Expected, actual)
		}
	})

	t.Run("fail", func(t *testing.T) {
		map1 := map[string]string{
			"example": "zone1,project1xyz",
		}
		map1Expected := map[string]networkingdnsforwarderv1.NetworkingV1ForwardViaGcpDnsZonesDomainMappings{
			"example": {Zone: networkingdnsforwarderv1.PtrString("zone1"), Project: networkingdnsforwarderv1.PtrString("project1")},
		}
		actual, _ := convertToStringObjectMap(map1)

		if reflect.DeepEqual(actual, map1Expected) {
			t.Fatalf("Unexpected error: expected %v, got %v", map1Expected, actual)
		}
	})

	t.Run("fail, missing comma", func(t *testing.T) {
		map1 := map[string]string{
			"example": "zone1 project1",
		}
		map1Expected := map[string]networkingdnsforwarderv1.NetworkingV1ForwardViaGcpDnsZonesDomainMappings{
			"example": {Zone: networkingdnsforwarderv1.PtrString("zone1"), Project: networkingdnsforwarderv1.PtrString("project1")},
		}
		actual, _ := convertToStringObjectMap(map1)

		if reflect.DeepEqual(actual, map1Expected) {
			t.Fatalf("Unexpected error: expected %v, got %v", map1Expected, actual)
		}
	})
}

func TestBuildTfRules(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		domainRules := []schemaregistryv1.Rule{
			{
				Name:     schemaregistryv1.PtrString("ABC"),
				Disabled: schemaregistryv1.PtrBool(false),
				Doc:      schemaregistryv1.PtrString("Doc"),
				Expr:     schemaregistryv1.PtrString("EXPR"),
				Kind:     schemaregistryv1.PtrString("TRANSFORM"),
				Mode:     schemaregistryv1.PtrString("WRITEREAD"),
				Type:     schemaregistryv1.PtrString("ENCRYPT"),
				Tags: &[]string{
					"PII",
				},
				Params: &map[string]string{
					"encrypt.kek.name": "testkek2",
				},
				OnSuccess: schemaregistryv1.PtrString("NONE,NONE"),
				OnFailure: schemaregistryv1.PtrString("ERROR,ERROR"),
			},
		}
		migrationRules := []schemaregistryv1.Rule{
			{
				Name:     schemaregistryv1.PtrString("ABC"),
				Disabled: schemaregistryv1.PtrBool(false),
				Doc:      schemaregistryv1.PtrString("Doc"),
				Expr:     schemaregistryv1.PtrString("EXPR"),
				Kind:     schemaregistryv1.PtrString("TRANSFORM"),
				Mode:     schemaregistryv1.PtrString("WRITEREAD"),
				Type:     schemaregistryv1.PtrString("ENCRYPT"),
				Tags: &[]string{
					"PIIM",
				},
				Params: &map[string]string{
					"encrypt.kek.name": "testkekM",
				},
				OnSuccess: schemaregistryv1.PtrString("NONE,NONE"),
				OnFailure: schemaregistryv1.PtrString("ERROR,ERROR"),
			},
		}
		tfDomainMigrationRules := make(map[string]interface{})
		tfRulesDomain := make([]map[string]interface{}, 1)
		tfRuleDomain := make(map[string]interface{})
		tfRuleDomain[paramName] = "ABC"
		tfRuleDomain[paramDoc] = "Doc"
		tfRuleDomain[paramKind] = "TRANSFORM"
		tfRuleDomain[paramMode] = "WRITEREAD"
		tfRuleDomain[paramType] = "ENCRYPT"
		tfRuleDomain[paramExpr] = "EXPR"
		tfRuleDomain[paramOnSuccess] = "NONE,NONE"
		tfRuleDomain[paramOnFailure] = "ERROR,ERROR"
		tfRuleDomain[paramDisabled] = false
		tfRuleDomain[paramTags] = []string{
			"PII",
		}
		tfRuleDomain[paramParams] = map[string]string{
			"encrypt.kek.name": "testkek2",
		}

		tfRulesMigration := make([]map[string]interface{}, 1)
		tfRuleMigration := make(map[string]interface{})
		tfRuleMigration[paramName] = "ABC"
		tfRuleMigration[paramDoc] = "Doc"
		tfRuleMigration[paramKind] = "TRANSFORM"
		tfRuleMigration[paramMode] = "WRITEREAD"
		tfRuleMigration[paramType] = "ENCRYPT"
		tfRuleMigration[paramExpr] = "EXPR"
		tfRuleMigration[paramOnSuccess] = "NONE,NONE"
		tfRuleMigration[paramOnFailure] = "ERROR,ERROR"
		tfRuleMigration[paramDisabled] = false
		tfRuleMigration[paramTags] = []string{
			"PIIM",
		}
		tfRuleMigration[paramParams] = map[string]string{
			"encrypt.kek.name": "testkekM",
		}
		tfRulesDomain[0] = tfRuleDomain
		tfRulesMigration[0] = tfRuleMigration
		tfDomainMigrationRules[paramDomainRules] = tfRulesDomain
		tfDomainMigrationRules[paramMigrationRules] = tfRulesMigration
		tfRuleSet := make([]map[string]interface{}, 1)
		tfRuleSet[0] = tfDomainMigrationRules
		actual := buildTfRules(domainRules, migrationRules, []schemaregistryv1.Rule{})

		if !reflect.DeepEqual(*actual, tfRuleSet) {
			t.Fatalf("Unexpected error: expected %v, got %v", tfRuleSet, *actual)
		}
	})

	t.Run("success, incomplete set", func(t *testing.T) {
		domainRules := []schemaregistryv1.Rule{
			{
				Name:     schemaregistryv1.PtrString("ABC"),
				Disabled: schemaregistryv1.PtrBool(false),
				Expr:     schemaregistryv1.PtrString("EXPR"),
				Kind:     schemaregistryv1.PtrString("TRANSFORM"),
				Mode:     schemaregistryv1.PtrString("WRITEREAD"),
				Type:     schemaregistryv1.PtrString("ENCRYPT"),
				Tags: &[]string{
					"PII",
				},
				Params: &map[string]string{
					"encrypt.kek.name": "testkek2",
				},
				OnSuccess: schemaregistryv1.PtrString("NONE,NONE"),
				OnFailure: schemaregistryv1.PtrString("ERROR,ERROR"),
			},
		}
		migrationRules := []schemaregistryv1.Rule{
			{
				Name:     schemaregistryv1.PtrString("ABC"),
				Disabled: schemaregistryv1.PtrBool(false),
				Expr:     schemaregistryv1.PtrString("EXPR"),
				Kind:     schemaregistryv1.PtrString("TRANSFORM"),
				Mode:     schemaregistryv1.PtrString("WRITEREAD"),
				Type:     schemaregistryv1.PtrString("ENCRYPT"),
				Tags: &[]string{
					"PIIM",
				},
				Params: &map[string]string{
					"encrypt.kek.name": "testkekM",
				},
				OnSuccess: schemaregistryv1.PtrString("NONE,NONE"),
				OnFailure: schemaregistryv1.PtrString("ERROR,ERROR"),
			},
		}
		tfDomainMigrationRules := make(map[string]interface{})
		tfRulesDomain := make([]map[string]interface{}, 1)
		tfRuleDomain := make(map[string]interface{})
		tfRuleDomain[paramName] = "ABC"
		tfRuleDomain[paramKind] = "TRANSFORM"
		tfRuleDomain[paramDoc] = ""
		tfRuleDomain[paramMode] = "WRITEREAD"
		tfRuleDomain[paramType] = "ENCRYPT"
		tfRuleDomain[paramExpr] = "EXPR"
		tfRuleDomain[paramOnSuccess] = "NONE,NONE"
		tfRuleDomain[paramOnFailure] = "ERROR,ERROR"
		tfRuleDomain[paramDisabled] = false
		tfRuleDomain[paramTags] = []string{
			"PII",
		}
		tfRuleDomain[paramParams] = map[string]string{
			"encrypt.kek.name": "testkek2",
		}

		tfRulesMigration := make([]map[string]interface{}, 1)
		tfRuleMigration := make(map[string]interface{})
		tfRuleMigration[paramName] = "ABC"
		tfRuleMigration[paramKind] = "TRANSFORM"
		tfRuleMigration[paramDoc] = ""
		tfRuleMigration[paramMode] = "WRITEREAD"
		tfRuleMigration[paramType] = "ENCRYPT"
		tfRuleMigration[paramExpr] = "EXPR"
		tfRuleMigration[paramOnSuccess] = "NONE,NONE"
		tfRuleMigration[paramOnFailure] = "ERROR,ERROR"
		tfRuleMigration[paramDisabled] = false
		tfRuleMigration[paramTags] = []string{
			"PIIM",
		}
		tfRuleMigration[paramParams] = map[string]string{
			"encrypt.kek.name": "testkekM",
		}
		tfRulesDomain[0] = tfRuleDomain
		tfRulesMigration[0] = tfRuleMigration
		tfDomainMigrationRules[paramDomainRules] = tfRulesDomain
		tfDomainMigrationRules[paramMigrationRules] = tfRulesMigration
		tfRuleSet := make([]map[string]interface{}, 1)
		tfRuleSet[0] = tfDomainMigrationRules
		actual := buildTfRules(domainRules, migrationRules, []schemaregistryv1.Rule{})

		if !reflect.DeepEqual(*actual, tfRuleSet) {
			t.Fatalf("Unexpected error: expected %v, got %v", tfRuleSet, *actual)
		}
	})

	t.Run("success, without migration rules", func(t *testing.T) {
		domainRules := []schemaregistryv1.Rule{
			{
				Name:     schemaregistryv1.PtrString("ABC"),
				Disabled: schemaregistryv1.PtrBool(false),
				Expr:     schemaregistryv1.PtrString("EXPR"),
				Kind:     schemaregistryv1.PtrString("TRANSFORM"),
				Mode:     schemaregistryv1.PtrString("WRITEREAD"),
				Type:     schemaregistryv1.PtrString("ENCRYPT"),
				Tags: &[]string{
					"PII",
				},
				Params: &map[string]string{
					"encrypt.kek.name": "testkek2",
				},
				OnSuccess: schemaregistryv1.PtrString("NONE,NONE"),
				OnFailure: schemaregistryv1.PtrString("ERROR,ERROR"),
			},
		}
		tfDomainMigrationRules := make(map[string]interface{})
		tfRulesDomain := make([]map[string]interface{}, 1)
		tfRuleDomain := make(map[string]interface{})
		tfRuleDomain[paramName] = "ABC"
		tfRuleDomain[paramKind] = "TRANSFORM"
		tfRuleDomain[paramDoc] = ""
		tfRuleDomain[paramMode] = "WRITEREAD"
		tfRuleDomain[paramType] = "ENCRYPT"
		tfRuleDomain[paramExpr] = "EXPR"
		tfRuleDomain[paramOnSuccess] = "NONE,NONE"
		tfRuleDomain[paramOnFailure] = "ERROR,ERROR"
		tfRuleDomain[paramDisabled] = false
		tfRuleDomain[paramTags] = []string{
			"PII",
		}
		tfRuleDomain[paramParams] = map[string]string{
			"encrypt.kek.name": "testkek2",
		}

		tfRulesDomain[0] = tfRuleDomain
		tfDomainMigrationRules[paramDomainRules] = tfRulesDomain
		tfRuleSet := make([]map[string]interface{}, 1)
		tfRuleSet[0] = tfDomainMigrationRules
		actual := buildTfRules(domainRules, []schemaregistryv1.Rule{}, []schemaregistryv1.Rule{})

		if !reflect.DeepEqual(*actual, tfRuleSet) {
			t.Fatalf("Unexpected error: expected %v, got %v", tfRuleSet, *actual)
		}
	})

	t.Run("success, without domain rules", func(t *testing.T) {
		migrationRules := []schemaregistryv1.Rule{
			{
				Name:     schemaregistryv1.PtrString("ABC"),
				Disabled: schemaregistryv1.PtrBool(false),
				Expr:     schemaregistryv1.PtrString("EXPR"),
				Kind:     schemaregistryv1.PtrString("TRANSFORM"),
				Mode:     schemaregistryv1.PtrString("WRITEREAD"),
				Type:     schemaregistryv1.PtrString("ENCRYPT"),
				Tags: &[]string{
					"PII",
				},
				Params: &map[string]string{
					"encrypt.kek.name": "testkek2",
				},
				OnSuccess: schemaregistryv1.PtrString("NONE,NONE"),
				OnFailure: schemaregistryv1.PtrString("ERROR,ERROR"),
			},
		}
		tfDomainMigrationRules := make(map[string]interface{})
		tfRulesMigration := make([]map[string]interface{}, 1)
		tfRuleMigration := make(map[string]interface{})
		tfRuleMigration[paramName] = "ABC"
		tfRuleMigration[paramKind] = "TRANSFORM"
		tfRuleMigration[paramDoc] = ""
		tfRuleMigration[paramMode] = "WRITEREAD"
		tfRuleMigration[paramType] = "ENCRYPT"
		tfRuleMigration[paramExpr] = "EXPR"
		tfRuleMigration[paramOnSuccess] = "NONE,NONE"
		tfRuleMigration[paramOnFailure] = "ERROR,ERROR"
		tfRuleMigration[paramDisabled] = false
		tfRuleMigration[paramTags] = []string{
			"PII",
		}
		tfRuleMigration[paramParams] = map[string]string{
			"encrypt.kek.name": "testkek2",
		}

		tfRulesMigration[0] = tfRuleMigration
		tfDomainMigrationRules[paramMigrationRules] = tfRulesMigration
		tfRuleSet := make([]map[string]interface{}, 1)
		tfRuleSet[0] = tfDomainMigrationRules
		actual := buildTfRules([]schemaregistryv1.Rule{}, migrationRules, []schemaregistryv1.Rule{})

		if !reflect.DeepEqual(*actual, tfRuleSet) {
			t.Fatalf("Unexpected error: expected %v, got %v", tfRuleSet, *actual)
		}
	})

	t.Run("fail, inconsistent domain rules", func(t *testing.T) {
		migrationRules := []schemaregistryv1.Rule{
			{
				Name:     schemaregistryv1.PtrString("ABC"),
				Disabled: schemaregistryv1.PtrBool(false),
				Expr:     schemaregistryv1.PtrString("EXPR"),
				Kind:     schemaregistryv1.PtrString("TRANSFORM"),
				Mode:     schemaregistryv1.PtrString("WRITEREAD"),
				Type:     schemaregistryv1.PtrString("ENCRYPT"),
				Tags: &[]string{
					"PII",
				},
				Params: &map[string]string{
					"encrypt.kek.name": "testkek2",
				},
				OnSuccess: schemaregistryv1.PtrString("NONE,NONE"),
				OnFailure: schemaregistryv1.PtrString("ERROR,ERROR"),
			},
		}
		tfDomainMigrationRules := make(map[string]interface{})
		tfRulesMigration := make([]map[string]interface{}, 1)
		tfRuleMigration := make(map[string]interface{})
		tfRuleMigration[paramName] = "ABC"
		tfRuleMigration[paramKind] = "TRANSFORM"
		tfRuleMigration[paramDoc] = ""
		tfRuleMigration[paramMode] = "WRITEREAD"
		tfRuleMigration[paramType] = "ENCRYPT"
		tfRuleMigration[paramExpr] = "EXPR"
		tfRuleMigration[paramOnSuccess] = "NONE,NONE"
		tfRuleMigration[paramOnFailure] = "ERROR,ERROR"
		tfRuleMigration[paramDisabled] = false
		tfRuleMigration[paramTags] = []string{
			"PII",
		}
		tfRuleMigration[paramParams] = map[string]string{
			"encrypt.kek.name": "testkek2",
		}

		tfRulesMigration[0] = tfRuleMigration
		tfDomainMigrationRules[paramMigrationRules] = tfRulesMigration
		tfRuleSet := make([]map[string]interface{}, 1)
		tfRuleSet[0] = tfDomainMigrationRules
		actual := buildTfRules(migrationRules, migrationRules, []schemaregistryv1.Rule{})

		if reflect.DeepEqual(*actual, tfRuleSet) {
			t.Fatalf("Unexpected error: expected %v, got %v", tfRuleSet, *actual)
		}
	})

	t.Run("With encoding rules (CSPE)", func(t *testing.T) {
		encodingRules := make([]schemaregistryv1.Rule, 1)
		var encodingRule schemaregistryv1.Rule
		encodingRule.SetName("encryptCSPE")
		encodingRule.SetKind("TRANSFORM")
		encodingRule.SetMode("WRITEREAD")
		encodingRule.SetType("ENCRYPT")
		encodingRule.SetDoc("DOC")
		encodingRule.SetExpr("EXPR")
		encodingRule.SetOnSuccess("NONE,NONE")
		encodingRule.SetOnFailure("ERROR,ERROR")
		encodingRule.SetDisabled(false)
		encodingRule.SetTags([]string{"CSPE"})
		params := make(map[string]string)
		params["encrypt.kek.name"] = "cspe-kek"
		encodingRule.SetParams(params)
		encodingRules[0] = encodingRule

		tfDomainMigrationEncodingRules := make(map[string]interface{})
		tfRulesEncoding := make([]map[string]interface{}, 1)
		tfRuleEncoding := make(map[string]interface{})
		tfRuleEncoding[paramName] = "encryptCSPE"
		tfRuleEncoding[paramKind] = "TRANSFORM"
		tfRuleEncoding[paramDoc] = "DOC"
		tfRuleEncoding[paramMode] = "WRITEREAD"
		tfRuleEncoding[paramType] = "ENCRYPT"
		tfRuleEncoding[paramExpr] = "EXPR"
		tfRuleEncoding[paramOnSuccess] = "NONE,NONE"
		tfRuleEncoding[paramOnFailure] = "ERROR,ERROR"
		tfRuleEncoding[paramDisabled] = false
		tfRuleEncoding[paramTags] = []string{
			"CSPE",
		}
		tfRuleEncoding[paramParams] = map[string]string{
			"encrypt.kek.name": "cspe-kek",
		}

		tfRulesEncoding[0] = tfRuleEncoding
		tfDomainMigrationEncodingRules[paramEncodingRules] = tfRulesEncoding
		tfRuleSet := make([]map[string]interface{}, 1)
		tfRuleSet[0] = tfDomainMigrationEncodingRules
		actual := buildTfRules([]schemaregistryv1.Rule{}, []schemaregistryv1.Rule{}, encodingRules)

		if !reflect.DeepEqual(*actual, tfRuleSet) {
			t.Fatalf("Unexpected error: expected %v, got %v", tfRuleSet, *actual)
		}
	})

	t.Run("With all three rule types", func(t *testing.T) {
		domainRules := make([]schemaregistryv1.Rule, 1)
		var domainRule schemaregistryv1.Rule
		domainRule.SetName("encryptPII")
		domainRule.SetKind("TRANSFORM")
		domainRule.SetMode("WRITEREAD")
		domainRule.SetType("ENCRYPT")
		domainRule.SetDoc("")
		domainRule.SetExpr("")
		domainRule.SetOnSuccess("NONE,NONE")
		domainRule.SetOnFailure("ERROR,ERROR")
		domainRule.SetDisabled(false)
		domainRule.SetTags([]string{"PII"})
		domainParams := make(map[string]string)
		domainParams["encrypt.kek.name"] = "testkek2"
		domainRule.SetParams(domainParams)
		domainRules[0] = domainRule

		migrationRules := make([]schemaregistryv1.Rule, 1)
		var migrationRule schemaregistryv1.Rule
		migrationRule.SetName("migrateEncrypt")
		migrationRule.SetKind("TRANSFORM")
		migrationRule.SetMode("WRITEREAD")
		migrationRule.SetType("ENCRYPT")
		migrationRule.SetDoc("")
		migrationRule.SetExpr("")
		migrationRule.SetOnSuccess("NONE,NONE")
		migrationRule.SetOnFailure("ERROR,ERROR")
		migrationRule.SetDisabled(false)
		migrationRule.SetTags([]string{"MIGRATION"})
		migrationParams := make(map[string]string)
		migrationParams["encrypt.kek.name"] = "migration-kek"
		migrationRule.SetParams(migrationParams)
		migrationRules[0] = migrationRule

		encodingRules := make([]schemaregistryv1.Rule, 1)
		var encodingRule schemaregistryv1.Rule
		encodingRule.SetName("encryptCSPE")
		encodingRule.SetKind("TRANSFORM")
		encodingRule.SetMode("WRITEREAD")
		encodingRule.SetType("ENCRYPT")
		encodingRule.SetDoc("")
		encodingRule.SetExpr("")
		encodingRule.SetOnSuccess("NONE,NONE")
		encodingRule.SetOnFailure("ERROR,ERROR")
		encodingRule.SetDisabled(false)
		encodingRule.SetTags([]string{"CSPE"})
		encodingParams := make(map[string]string)
		encodingParams["encrypt.kek.name"] = "cspe-kek"
		encodingRule.SetParams(encodingParams)
		encodingRules[0] = encodingRule

		tfAllRules := make(map[string]interface{})

		// Domain rules
		tfRulesDomain := make([]map[string]interface{}, 1)
		tfRuleDomain := make(map[string]interface{})
		tfRuleDomain[paramName] = "encryptPII"
		tfRuleDomain[paramKind] = "TRANSFORM"
		tfRuleDomain[paramDoc] = ""
		tfRuleDomain[paramMode] = "WRITEREAD"
		tfRuleDomain[paramType] = "ENCRYPT"
		tfRuleDomain[paramExpr] = ""
		tfRuleDomain[paramOnSuccess] = "NONE,NONE"
		tfRuleDomain[paramOnFailure] = "ERROR,ERROR"
		tfRuleDomain[paramDisabled] = false
		tfRuleDomain[paramTags] = []string{"PII"}
		tfRuleDomain[paramParams] = map[string]string{"encrypt.kek.name": "testkek2"}
		tfRulesDomain[0] = tfRuleDomain
		tfAllRules[paramDomainRules] = tfRulesDomain

		tfRulesMigration := make([]map[string]interface{}, 1)
		tfRuleMigration := make(map[string]interface{})
		tfRuleMigration[paramName] = "migrateEncrypt"
		tfRuleMigration[paramKind] = "TRANSFORM"
		tfRuleMigration[paramDoc] = ""
		tfRuleMigration[paramMode] = "WRITEREAD"
		tfRuleMigration[paramType] = "ENCRYPT"
		tfRuleMigration[paramExpr] = ""
		tfRuleMigration[paramOnSuccess] = "NONE,NONE"
		tfRuleMigration[paramOnFailure] = "ERROR,ERROR"
		tfRuleMigration[paramDisabled] = false
		tfRuleMigration[paramTags] = []string{"MIGRATION"}
		tfRuleMigration[paramParams] = map[string]string{"encrypt.kek.name": "migration-kek"}
		tfRulesMigration[0] = tfRuleMigration
		tfAllRules[paramMigrationRules] = tfRulesMigration

		// Encoding rules
		tfRulesEncoding := make([]map[string]interface{}, 1)
		tfRuleEncoding := make(map[string]interface{})
		tfRuleEncoding[paramName] = "encryptCSPE"
		tfRuleEncoding[paramKind] = "TRANSFORM"
		tfRuleEncoding[paramDoc] = ""
		tfRuleEncoding[paramMode] = "WRITEREAD"
		tfRuleEncoding[paramType] = "ENCRYPT"
		tfRuleEncoding[paramExpr] = ""
		tfRuleEncoding[paramOnSuccess] = "NONE,NONE"
		tfRuleEncoding[paramOnFailure] = "ERROR,ERROR"
		tfRuleEncoding[paramDisabled] = false
		tfRuleEncoding[paramTags] = []string{"CSPE"}
		tfRuleEncoding[paramParams] = map[string]string{"encrypt.kek.name": "cspe-kek"}
		tfRulesEncoding[0] = tfRuleEncoding
		tfAllRules[paramEncodingRules] = tfRulesEncoding

		tfRuleSet := make([]map[string]interface{}, 1)
		tfRuleSet[0] = tfAllRules
		actual := buildTfRules(domainRules, migrationRules, encodingRules)

		if !reflect.DeepEqual(*actual, tfRuleSet) {
			t.Fatalf("Unexpected error: expected %v, got %v", tfRuleSet, *actual)
		}
	})
}

func TestBuildTfConnectorClasses(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		classes := []ccpmv1.CcpmV1ConnectorClass{
			{
				ClassName: "Class1",
				Type:      "SOURCE",
			},
			{
				ClassName: "Class2",
				Type:      "SOURCE",
			},
		}
		actual := buildTfConnectorClasses(classes)
		connectorClasses := make([]map[string]interface{}, 2)
		tfConnectorClasses := make(map[string]interface{})
		tfConnectorClasses[paramConnectorClassName] = "Class1"
		tfConnectorClasses[paramConnectorType] = "SOURCE"
		tfConnectorClasses2 := make(map[string]interface{})
		tfConnectorClasses2[paramConnectorClassName] = "Class2"
		tfConnectorClasses2[paramConnectorType] = "SOURCE"
		connectorClasses[0] = tfConnectorClasses
		connectorClasses[1] = tfConnectorClasses2
		if !reflect.DeepEqual(*actual, connectorClasses) {
			t.Fatalf("Unexpected error: expected %v, got %v", connectorClasses, *actual)
		}
	})

	t.Run("success empty", func(t *testing.T) {
		classes := []ccpmv1.CcpmV1ConnectorClass{
			{
				ClassName: "",
				Type:      "SOURCE",
			},
		}
		actual := buildTfConnectorClasses(classes)
		connectorClasses := make([]map[string]interface{}, 1)
		tfConnectorClasses := make(map[string]interface{})
		tfConnectorClasses[paramConnectorClassName] = ""
		tfConnectorClasses[paramConnectorType] = "SOURCE"
		connectorClasses[0] = tfConnectorClasses
		if !reflect.DeepEqual(*actual, connectorClasses) {
			t.Fatalf("Unexpected error: expected %v, got %v", connectorClasses, *actual)
		}
	})

	t.Run("fail wrong value", func(t *testing.T) {
		classes := []ccpmv1.CcpmV1ConnectorClass{
			{
				ClassName: "name1",
				Type:      "SOURCE",
			},
		}
		actual := buildTfConnectorClasses(classes)
		connectorClasses := make(map[string]interface{})
		connectorClasses[paramConnectorClassName] = ""
		connectorClasses[paramConnectorType] = "SOURCE"
		if reflect.DeepEqual(*actual, connectorClasses) {
			t.Fatalf("Unexpected error: expected %v, got %v", connectorClasses, *actual)
		}
	})
}

func TestBuildConnectorClass(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		connectorClasses := make(map[string]interface{})
		connectorClasses[paramConnectorClassName] = "Class1"
		connectorClasses[paramConnectorType] = "SOURCE"

		connectorClasses2 := make(map[string]interface{})
		connectorClasses2[paramConnectorClassName] = "Class2"
		connectorClasses2[paramConnectorType] = "SOURCE"

		connectorClass := []interface{}{connectorClasses, connectorClasses2}
		actual := buildConnectorClass(connectorClass)
		classes := make([]ccpmv1.CcpmV1ConnectorClass, 2)

		class := ccpmv1.NewCcpmV1ConnectorClassWithDefaults()
		class.SetClassName("Class1")
		class.SetType("SOURCE")

		class2 := ccpmv1.NewCcpmV1ConnectorClassWithDefaults()
		class2.SetClassName("Class2")
		class2.SetType("SOURCE")

		classes[0] = *class
		classes[1] = *class2

		if !reflect.DeepEqual(actual, classes) {
			t.Fatalf("Unexpected error: expected %v, got %v", classes, actual)
		}
	})

	t.Run("success - empty", func(t *testing.T) {
		connectorClasses := make(map[string]interface{})
		connectorClasses[paramConnectorClassName] = "Class1"
		connectorClasses[paramConnectorType] = "SOURCE"

		connectorClasses2 := make(map[string]interface{})
		connectorClasses2[paramConnectorClassName] = ""
		connectorClasses2[paramConnectorType] = ""

		connectorClass := []interface{}{connectorClasses, connectorClasses2}
		actual := buildConnectorClass(connectorClass)
		classes := make([]ccpmv1.CcpmV1ConnectorClass, 2)

		class := ccpmv1.NewCcpmV1ConnectorClassWithDefaults()
		class.SetClassName("Class1")
		class.SetType("SOURCE")

		class2 := ccpmv1.NewCcpmV1ConnectorClassWithDefaults()
		class2.SetClassName("")
		class2.SetType("")

		classes[0] = *class
		classes[1] = *class2

		if !reflect.DeepEqual(actual, classes) {
			t.Fatalf("Unexpected error: expected %v, got %v", classes, actual)
		}
	})

	t.Run("fail - wrong value", func(t *testing.T) {
		connectorClasses := make(map[string]interface{})
		connectorClasses[paramConnectorClassName] = "Class1"
		connectorClasses[paramConnectorType] = "SOURCE"

		connectorClasses2 := make(map[string]interface{})
		connectorClasses2[paramConnectorClassName] = "Class3"
		connectorClasses2[paramConnectorType] = "SOURCE"

		connectorClass := []interface{}{connectorClasses, connectorClasses2}
		actual := buildConnectorClass(connectorClass)
		classes := make([]ccpmv1.CcpmV1ConnectorClass, 2)

		class := ccpmv1.NewCcpmV1ConnectorClassWithDefaults()
		class.SetClassName("Class1")
		class.SetType("SOURCE")

		class2 := ccpmv1.NewCcpmV1ConnectorClassWithDefaults()
		class2.SetClassName("Class2")
		class2.SetType("")

		classes[0] = *class
		classes[1] = *class2

		if reflect.DeepEqual(actual, classes) {
			t.Fatalf("Unexpected error: expected %v, got %v", classes, actual)
		}
	})
}

func TestNormalizeCrn(t *testing.T) {
	tests := []struct {
		name  string
		a     string
		b     string
		equal bool
	}{
		{
			name:  "Identical CRN 1",
			a:     "crn://confluent.cloud/organization=org-123/environment=env-abc",
			b:     "crn://confluent.cloud/organization=org-123/environment=env-abc",
			equal: true,
		},
		{
			name:  "Identical CRN 2",
			a:     "crn://confluent.cloud/organization=org-123/environment=env-abc/cloud-cluster=lkc-123/kafka=lkc-123/topic=my.topic",
			b:     "crn://confluent.cloud/organization=org-123/environment=env-abc/cloud-cluster=lkc-123/kafka=lkc-123/topic=my.topic",
			equal: true,
		},
		{
			name:  "Logically equivalent CRN",
			a:     "crn://confluent.cloud/organization=org-123/environment=env-abc/schema-registry=lsrc-123/subject=:.context:subject.v1",
			b:     "crn://confluent.cloud/organization=org-123/environment=env-abc/schema-registry=lsrc-123/subject=%3A.context%3Asubject.v1",
			equal: true,
		},
		{
			name:  "CRN with different SRs",
			a:     "crn://confluent.cloud/organization=org-123/environment=env-abc/schema-registry=lsrc-123/subject=:.context:subject.v1",
			b:     "crn://confluent.cloud/organization=org-123/environment=env-abc/schema-registry=lsrc-456/subject=:.context:subject.v1",
			equal: false,
		},
		{
			name:  "CRN with different subjects",
			a:     "crn://confluent.cloud/organization=org-123/environment=env-abc/schema-registry=lsrc-123/subject=:.context:subject.v1",
			b:     "crn://confluent.cloud/organization=org-123/environment=env-abc/schema-registry=lsrc-123/subject=subject.v1",
			equal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeCrn(tt.a) == normalizeCrn(tt.b)
			if got != tt.equal {
				t.Fatalf("Unexpected error: %v expected %v, got %v", tt.name, tt.equal, got)
			}
		})
	}
}

func TestConvertConfigDataToAlterConfigBatchRequestData(t *testing.T) {
	tests := []struct {
		name     string
		input    []kafkarestv3.ConfigData
		expected []kafkarestv3.AlterConfigBatchRequestDataData
	}{
		{
			name:     "empty configs",
			input:    []kafkarestv3.ConfigData{},
			expected: []kafkarestv3.AlterConfigBatchRequestDataData{},
		},
		{
			name: "single config with PLAIN mechanism",
			input: []kafkarestv3.ConfigData{
				{
					Name:  "sasl.mechanism",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("PLAIN")),
				},
			},
			expected: []kafkarestv3.AlterConfigBatchRequestDataData{
				{
					Name:      "sasl.mechanism",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("PLAIN")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
			},
		},
		{
			name: "multiple credential configs",
			input: []kafkarestv3.ConfigData{
				{
					Name:  "sasl.mechanism",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("PLAIN")),
				},
				{
					Name:  "sasl.jaas.config",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("org.apache.kafka.common.security.plain.PlainLoginModule required username=\"test-key\" password=\"test-secret\";")),
				},
				{
					Name:  "security.protocol",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SASL_SSL")),
				},
			},
			expected: []kafkarestv3.AlterConfigBatchRequestDataData{
				{
					Name:      "sasl.mechanism",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("PLAIN")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
				{
					Name:      "sasl.jaas.config",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("org.apache.kafka.common.security.plain.PlainLoginModule required username=\"test-key\" password=\"test-secret\";")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
				{
					Name:      "security.protocol",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SASL_SSL")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
			},
		},
		{
			name: "OAuth OAUTHBEARER configs",
			input: []kafkarestv3.ConfigData{
				{
					Name:  "sasl.mechanism",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("OAUTHBEARER")),
				},
				{
					Name:  "sasl.oauthbearer.token.endpoint.url",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("https://example.com/oauth/token")),
				},
				{
					Name:  "sasl.login.callback.handler.class",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("org.apache.kafka.common.security.oauthbearer.OAuthBearerLoginCallbackHandler")),
				},
				{
					Name:  "sasl.jaas.config",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("org.apache.kafka.common.security.oauthbearer.OAuthBearerLoginModule required clientId='test-client' scope='test-scope' clientSecret='test-secret' extension_logicalCluster='lkc-123' extension_identityPoolId='pool-123';")),
				},
			},
			expected: []kafkarestv3.AlterConfigBatchRequestDataData{
				{
					Name:      "sasl.mechanism",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("OAUTHBEARER")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
				{
					Name:      "sasl.oauthbearer.token.endpoint.url",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("https://example.com/oauth/token")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
				{
					Name:      "sasl.login.callback.handler.class",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("org.apache.kafka.common.security.oauthbearer.OAuthBearerLoginCallbackHandler")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
				{
					Name:      "sasl.jaas.config",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("org.apache.kafka.common.security.oauthbearer.OAuthBearerLoginModule required clientId='test-client' scope='test-scope' clientSecret='test-secret' extension_logicalCluster='lkc-123' extension_identityPoolId='pool-123';")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
			},
		},
		{
			name: "local cluster configs for bidirectional link",
			input: []kafkarestv3.ConfigData{
				{
					Name:  "local.security.protocol",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SASL_SSL")),
				},
				{
					Name:  "local.sasl.mechanism",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("PLAIN")),
				},
				{
					Name:  "local.sasl.jaas.config",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("org.apache.kafka.common.security.plain.PlainLoginModule required username=\"local-key\" password=\"local-secret\";")),
				},
			},
			expected: []kafkarestv3.AlterConfigBatchRequestDataData{
				{
					Name:      "local.security.protocol",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SASL_SSL")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
				{
					Name:      "local.sasl.mechanism",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("PLAIN")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
				{
					Name:      "local.sasl.jaas.config",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("org.apache.kafka.common.security.plain.PlainLoginModule required username=\"local-key\" password=\"local-secret\";")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
			},
		},
		{
			name: "bootstrap servers config",
			input: []kafkarestv3.ConfigData{
				{
					Name:  "bootstrap.servers",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("pkc-12345.us-east-1.aws.confluent.cloud:9092")),
				},
			},
			expected: []kafkarestv3.AlterConfigBatchRequestDataData{
				{
					Name:      "bootstrap.servers",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("pkc-12345.us-east-1.aws.confluent.cloud:9092")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
			},
		},
		{
			name: "config with empty string value",
			input: []kafkarestv3.ConfigData{
				{
					Name:  "test.config",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("")),
				},
			},
			expected: []kafkarestv3.AlterConfigBatchRequestDataData{
				{
					Name:      "test.config",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
			},
		},
		{
			name: "cluster link mode and connection mode configs",
			input: []kafkarestv3.ConfigData{
				{
					Name:  "link.mode",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("BIDIRECTIONAL")),
				},
				{
					Name:  "connection.mode",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("OUTBOUND")),
				},
			},
			expected: []kafkarestv3.AlterConfigBatchRequestDataData{
				{
					Name:      "link.mode",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("BIDIRECTIONAL")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
				{
					Name:      "connection.mode",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("OUTBOUND")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertConfigDataToAlterConfigBatchRequestData(tt.input)

			// Verify length
			if len(result) != len(tt.expected) {
				t.Fatalf("Unexpected result length: expected %d, got %d", len(tt.expected), len(result))
			}

			// Verify each config entry
			for i := range result {
				// Check Name
				if result[i].Name != tt.expected[i].Name {
					t.Errorf("Config at index %d: unexpected Name: expected %q, got %q", i, tt.expected[i].Name, result[i].Name)
				}

				// Check Value - carefully handle NullableString
				resultValue := result[i].Value.Get()
				expectedValue := tt.expected[i].Value.Get()

				if (resultValue == nil) != (expectedValue == nil) {
					t.Errorf("Config at index %d (%s): Value nullability mismatch: expected nil=%v, got nil=%v",
						i, result[i].Name, expectedValue == nil, resultValue == nil)
				}

				if resultValue != nil && expectedValue != nil {
					if *resultValue != *expectedValue {
						t.Errorf("Config at index %d (%s): unexpected Value: expected %q, got %q",
							i, result[i].Name, *expectedValue, *resultValue)
					}
				}

				// Check Operation - must always be "SET"
				resultOperation := result[i].Operation.Get()
				if resultOperation == nil {
					t.Errorf("Config at index %d (%s): Operation is nil, expected 'SET'", i, result[i].Name)
				} else if *resultOperation != "SET" {
					t.Errorf("Config at index %d (%s): unexpected Operation: expected 'SET', got %q",
						i, result[i].Name, *resultOperation)
				}

				// Verify Operation matches expected
				expectedOperation := tt.expected[i].Operation.Get()
				if expectedOperation != nil && resultOperation != nil {
					if *resultOperation != *expectedOperation {
						t.Errorf("Config at index %d (%s): Operation mismatch: expected %q, got %q",
							i, result[i].Name, *expectedOperation, *resultOperation)
					}
				}
			}

			// Additional deep equality check for paranoia
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Deep equality check failed for test case %q", tt.name)
				t.Logf("Expected: %+v", tt.expected)
				t.Logf("Got:      %+v", result)
			}
		})
	}
}

func TestExtractCredentialConfigs(t *testing.T) {
	tests := []struct {
		name     string
		input    []kafkarestv3.ConfigData
		expected []kafkarestv3.AlterConfigBatchRequestDataData
	}{
		{
			name:     "empty configs",
			input:    []kafkarestv3.ConfigData{},
			expected: []kafkarestv3.AlterConfigBatchRequestDataData{},
		},
		{
			name: "only credential configs - PLAIN mechanism",
			input: []kafkarestv3.ConfigData{
				{
					Name:  "sasl.mechanism",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("PLAIN")),
				},
				{
					Name:  "sasl.jaas.config",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("org.apache.kafka.common.security.plain.PlainLoginModule required username=\"test-key\" password=\"test-secret\";")),
				},
			},
			expected: []kafkarestv3.AlterConfigBatchRequestDataData{
				{
					Name:      "sasl.mechanism",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("PLAIN")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
				{
					Name:      "sasl.jaas.config",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("org.apache.kafka.common.security.plain.PlainLoginModule required username=\"test-key\" password=\"test-secret\";")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
			},
		},
		{
			name: "mixed configs - filters out non-credential configs",
			input: []kafkarestv3.ConfigData{
				{
					Name:  "sasl.mechanism",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("PLAIN")),
				},
				{
					Name:  "bootstrap.servers",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("pkc-12345.us-east-1.aws.confluent.cloud:9092")),
				},
				{
					Name:  "sasl.jaas.config",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("org.apache.kafka.common.security.plain.PlainLoginModule required username=\"key\" password=\"secret\";")),
				},
				{
					Name:  "security.protocol",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SASL_SSL")),
				},
				{
					Name:  "link.mode",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("DESTINATION")),
				},
			},
			expected: []kafkarestv3.AlterConfigBatchRequestDataData{
				{
					Name:      "sasl.mechanism",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("PLAIN")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
				{
					Name:      "sasl.jaas.config",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("org.apache.kafka.common.security.plain.PlainLoginModule required username=\"key\" password=\"secret\";")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
			},
		},
		{
			name: "local cluster credential configs for bidirectional link",
			input: []kafkarestv3.ConfigData{
				{
					Name:  "local.sasl.mechanism",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("PLAIN")),
				},
				{
					Name:  "local.sasl.jaas.config",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("org.apache.kafka.common.security.plain.PlainLoginModule required username=\"local-key\" password=\"local-secret\";")),
				},
				{
					Name:  "local.security.protocol",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SASL_SSL")),
				},
			},
			expected: []kafkarestv3.AlterConfigBatchRequestDataData{
				{
					Name:      "local.sasl.mechanism",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("PLAIN")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
				{
					Name:      "local.sasl.jaas.config",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("org.apache.kafka.common.security.plain.PlainLoginModule required username=\"local-key\" password=\"local-secret\";")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
			},
		},
		{
			name: "both local and remote credential configs",
			input: []kafkarestv3.ConfigData{
				{
					Name:  "sasl.mechanism",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("PLAIN")),
				},
				{
					Name:  "sasl.jaas.config",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("org.apache.kafka.common.security.plain.PlainLoginModule required username=\"remote-key\" password=\"remote-secret\";")),
				},
				{
					Name:  "local.sasl.mechanism",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("PLAIN")),
				},
				{
					Name:  "local.sasl.jaas.config",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("org.apache.kafka.common.security.plain.PlainLoginModule required username=\"local-key\" password=\"local-secret\";")),
				},
				{
					Name:  "bootstrap.servers",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("pkc-99999.us-west-2.aws.confluent.cloud:9092")),
				},
			},
			expected: []kafkarestv3.AlterConfigBatchRequestDataData{
				{
					Name:      "sasl.mechanism",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("PLAIN")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
				{
					Name:      "sasl.jaas.config",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("org.apache.kafka.common.security.plain.PlainLoginModule required username=\"remote-key\" password=\"remote-secret\";")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
				{
					Name:      "local.sasl.mechanism",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("PLAIN")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
				{
					Name:      "local.sasl.jaas.config",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("org.apache.kafka.common.security.plain.PlainLoginModule required username=\"local-key\" password=\"local-secret\";")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
			},
		},
		{
			name: "no credential configs - all filtered out",
			input: []kafkarestv3.ConfigData{
				{
					Name:  "bootstrap.servers",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("pkc-12345.us-east-1.aws.confluent.cloud:9092")),
				},
				{
					Name:  "security.protocol",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SASL_SSL")),
				},
				{
					Name:  "link.mode",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("DESTINATION")),
				},
				{
					Name:  "connection.mode",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("OUTBOUND")),
				},
			},
			expected: []kafkarestv3.AlterConfigBatchRequestDataData{},
		},
		{
			name: "only sasl.mechanism without jaas.config",
			input: []kafkarestv3.ConfigData{
				{
					Name:  "sasl.mechanism",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("OAUTHBEARER")),
				},
				{
					Name:  "sasl.oauthbearer.token.endpoint.url",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("https://example.com/oauth/token")),
				},
			},
			expected: []kafkarestv3.AlterConfigBatchRequestDataData{
				{
					Name:      "sasl.mechanism",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("OAUTHBEARER")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
			},
		},
		{
			name: "only jaas.config without mechanism",
			input: []kafkarestv3.ConfigData{
				{
					Name:  "sasl.jaas.config",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("org.apache.kafka.common.security.plain.PlainLoginModule required username=\"key\" password=\"secret\";")),
				},
				{
					Name:  "bootstrap.servers",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("pkc-12345.us-east-1.aws.confluent.cloud:9092")),
				},
			},
			expected: []kafkarestv3.AlterConfigBatchRequestDataData{
				{
					Name:      "sasl.jaas.config",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("org.apache.kafka.common.security.plain.PlainLoginModule required username=\"key\" password=\"secret\";")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
			},
		},
		{
			name: "credential config with empty value",
			input: []kafkarestv3.ConfigData{
				{
					Name:  "sasl.mechanism",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("")),
				},
			},
			expected: []kafkarestv3.AlterConfigBatchRequestDataData{
				{
					Name:      "sasl.mechanism",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
			},
		},
		{
			name: "case sensitivity - non-matching similar keys",
			input: []kafkarestv3.ConfigData{
				{
					Name:  "SASL.MECHANISM", // Wrong case
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("PLAIN")),
				},
				{
					Name:  "sasl.mechanism", // Correct case
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("PLAIN")),
				},
				{
					Name:  "Sasl.Jaas.Config", // Wrong case
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("config")),
				},
			},
			expected: []kafkarestv3.AlterConfigBatchRequestDataData{
				{
					Name:      "sasl.mechanism",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("PLAIN")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
			},
		},
		{
			name: "all four credential config keys",
			input: []kafkarestv3.ConfigData{
				{
					Name:  "sasl.mechanism",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("PLAIN")),
				},
				{
					Name:  "sasl.jaas.config",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("org.apache.kafka.common.security.plain.PlainLoginModule required username=\"remote-key\" password=\"remote-secret\";")),
				},
				{
					Name:  "local.sasl.mechanism",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("PLAIN")),
				},
				{
					Name:  "local.sasl.jaas.config",
					Value: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("org.apache.kafka.common.security.plain.PlainLoginModule required username=\"local-key\" password=\"local-secret\";")),
				},
			},
			expected: []kafkarestv3.AlterConfigBatchRequestDataData{
				{
					Name:      "sasl.mechanism",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("PLAIN")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
				{
					Name:      "sasl.jaas.config",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("org.apache.kafka.common.security.plain.PlainLoginModule required username=\"remote-key\" password=\"remote-secret\";")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
				{
					Name:      "local.sasl.mechanism",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("PLAIN")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
				{
					Name:      "local.sasl.jaas.config",
					Value:     *kafkarestv3.NewNullableString(kafkarestv3.PtrString("org.apache.kafka.common.security.plain.PlainLoginModule required username=\"local-key\" password=\"local-secret\";")),
					Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractCredentialConfigs(tt.input)

			// Verify length
			if len(result) != len(tt.expected) {
				t.Fatalf("Unexpected result length: expected %d, got %d\nInput configs: %+v\nExpected: %+v\nGot: %+v",
					len(tt.expected), len(result), tt.input, tt.expected, result)
			}

			// Verify each filtered and converted config entry
			for i := range result {
				// Check Name
				if result[i].Name != tt.expected[i].Name {
					t.Errorf("Config at index %d: unexpected Name: expected %q, got %q", i, tt.expected[i].Name, result[i].Name)
				}

				// Check Value - carefully handle NullableString
				resultValue := result[i].Value.Get()
				expectedValue := tt.expected[i].Value.Get()

				if (resultValue == nil) != (expectedValue == nil) {
					t.Errorf("Config at index %d (%s): Value nullability mismatch: expected nil=%v, got nil=%v",
						i, result[i].Name, expectedValue == nil, resultValue == nil)
				}

				if resultValue != nil && expectedValue != nil {
					if *resultValue != *expectedValue {
						t.Errorf("Config at index %d (%s): unexpected Value: expected %q, got %q",
							i, result[i].Name, *expectedValue, *resultValue)
					}
				}

				// Check Operation - must always be "SET"
				resultOperation := result[i].Operation.Get()
				if resultOperation == nil {
					t.Errorf("Config at index %d (%s): Operation is nil, expected 'SET'", i, result[i].Name)
				} else if *resultOperation != "SET" {
					t.Errorf("Config at index %d (%s): unexpected Operation: expected 'SET', got %q",
						i, result[i].Name, *resultOperation)
				}

				// Verify Operation matches expected
				expectedOperation := tt.expected[i].Operation.Get()
				if expectedOperation != nil && resultOperation != nil {
					if *resultOperation != *expectedOperation {
						t.Errorf("Config at index %d (%s): Operation mismatch: expected %q, got %q",
							i, result[i].Name, *expectedOperation, *resultOperation)
					}
				}
			}

			// Verify no non-credential configs leaked through
			for _, resultConfig := range result {
				isCredentialConfig := false
				credentialKeys := []string{
					"sasl.jaas.config",
					"local.sasl.jaas.config",
					"sasl.mechanism",
					"local.sasl.mechanism",
				}
				for _, key := range credentialKeys {
					if resultConfig.Name == key {
						isCredentialConfig = true
						break
					}
				}
				if !isCredentialConfig {
					t.Errorf("Non-credential config leaked through filter: %q", resultConfig.Name)
				}
			}

			// Additional deep equality check for paranoia
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Deep equality check failed for test case %q", tt.name)
				t.Logf("Expected: %+v", tt.expected)
				t.Logf("Got:      %+v", result)
			}
		})
	}
}

func TestValidateAllOrNoneAttributesSetForResources(t *testing.T) {
	tests := []struct {
		name string

		kafkaApiKey, kafkaApiSecret, kafkaID, kafkaRestEndpoint                                                                       string
		schemaRegistryApiKey, schemaRegistryApiSecret, schemaRegistryClusterId, schemaRegistryRestEndpoint, catalogRestEndpoint       string
		flinkApiKey, flinkApiSecret, flinkOrganizationId, flinkEnvironmentId, flinkComputePoolId, flinkRestEndpoint, flinkPrincipalId string
		tableflowApiKey, tableflowApiSecret                                                                                           string

		shouldErr      bool
		expectedErrMsg string
		expectedFlags  ResourceMetadataSetFlags
	}{
		{
			name: "all attributes unset - valid",
			expectedFlags: ResourceMetadataSetFlags{
				isKafkaMetadataSet:          false,
				isSchemaRegistryMetadataSet: false,
				isCatalogMetadataSet:        false,
				isFlinkMetadataSet:          false,
				isTableflowMetadataSet:      false,
			},
		},
		{
			name:              "all attributes set correctly - valid",
			kafkaApiKey:       "kafka-key",
			kafkaApiSecret:    "kafka-secret",
			kafkaID:           "lkc-abc123",
			kafkaRestEndpoint: "https://lkc-123.us-east-1.aws.confluent.cloud",

			schemaRegistryApiKey:       "sr-key",
			schemaRegistryApiSecret:    "sr-secret",
			schemaRegistryClusterId:    "lsrc-abc123",
			schemaRegistryRestEndpoint: "https://lsrc-123.us-east-1.aws.confluent.cloud",

			flinkApiKey:         "flink-key",
			flinkApiSecret:      "flink-secret",
			flinkOrganizationId: "org-123",
			flinkEnvironmentId:  "env-456",
			flinkComputePoolId:  "pool-789",
			flinkRestEndpoint:   "https://flink.us-east-1.aws.confluent.cloud",
			flinkPrincipalId:    "u-123456",

			tableflowApiKey:    "tf-key",
			tableflowApiSecret: "tf-secret",

			expectedFlags: ResourceMetadataSetFlags{
				isKafkaMetadataSet:          true,
				isSchemaRegistryMetadataSet: true,
				isCatalogMetadataSet:        true,
				isFlinkMetadataSet:          true,
				isTableflowMetadataSet:      true,
			},
		},
		{
			name:           "Kafka partially set - missing rest endpoint",
			kafkaApiKey:    "kafka-key",
			kafkaApiSecret: "kafka-secret",
			shouldErr:      true,
			expectedErrMsg: "(kafka_api_key, kafka_api_secret, kafka_rest_endpoint)",
			expectedFlags:  ResourceMetadataSetFlags{},
		},
		{
			name:                    "Schema Registry partially set - missing endpoint",
			schemaRegistryApiKey:    "sr-key",
			schemaRegistryApiSecret: "sr-secret",
			schemaRegistryClusterId: "lsrc-abc123",
			shouldErr:               true,
			expectedErrMsg:          "All 4 schema_registry_api_key",
			expectedFlags:           ResourceMetadataSetFlags{isKafkaMetadataSet: false},
		},
		{
			name:                    "Schema Registry valid via catalog endpoint",
			schemaRegistryApiKey:    "sr-key",
			schemaRegistryApiSecret: "sr-secret",
			schemaRegistryClusterId: "lsrc-abc123",
			catalogRestEndpoint:     "https://catalog.us-east-1.aws.confluent.cloud",
			expectedFlags: ResourceMetadataSetFlags{
				isKafkaMetadataSet:          false,
				isSchemaRegistryMetadataSet: true,
				isCatalogMetadataSet:        true,
				isFlinkMetadataSet:          false,
				isTableflowMetadataSet:      false,
			},
		},
		{
			name:                "Flink partially set - missing principal ID",
			flinkApiKey:         "flink-key",
			flinkApiSecret:      "flink-secret",
			flinkOrganizationId: "org-123",
			flinkEnvironmentId:  "env-456",
			flinkComputePoolId:  "pool-789",
			flinkRestEndpoint:   "https://flink.us-east-1.aws.confluent.cloud",
			shouldErr:           true,
			expectedErrMsg:      "All 7 flink_api_key, flink_api_secret",
			expectedFlags: ResourceMetadataSetFlags{
				isKafkaMetadataSet:          false,
				isSchemaRegistryMetadataSet: false,
				isCatalogMetadataSet:        false,
				isFlinkMetadataSet:          false,
				isTableflowMetadataSet:      false,
			},
		},
		{
			name:            "Tableflow partially set - missing secret",
			tableflowApiKey: "tf-key",
			shouldErr:       true,
			expectedErrMsg:  "Both tableflow_api_key and tableflow_api_secret",
			expectedFlags: ResourceMetadataSetFlags{
				isKafkaMetadataSet:          false,
				isSchemaRegistryMetadataSet: false,
				isCatalogMetadataSet:        false,
				isFlinkMetadataSet:          false,
				isTableflowMetadataSet:      false,
			},
		},
		{
			name:              "Kafka and Schema Registry valid, Flink and Tableflow unset - valid",
			kafkaApiKey:       "kafka-key",
			kafkaApiSecret:    "kafka-secret",
			kafkaID:           "lkc-abc123",
			kafkaRestEndpoint: "https://lkc-123.us-east-1.aws.confluent.cloud",

			schemaRegistryApiKey:       "sr-key",
			schemaRegistryApiSecret:    "sr-secret",
			schemaRegistryClusterId:    "lsrc-abc123",
			schemaRegistryRestEndpoint: "https://lsrc-123.us-east-1.aws.confluent.cloud",

			expectedFlags: ResourceMetadataSetFlags{
				isKafkaMetadataSet:          true,
				isSchemaRegistryMetadataSet: true,
				isCatalogMetadataSet:        true,
				isFlinkMetadataSet:          false,
				isTableflowMetadataSet:      false,
			},
		},
		{
			name: "Kafka valid but Schema Registry partially set - invalid",
			// Kafka valid
			kafkaApiKey:       "kafka-key",
			kafkaApiSecret:    "kafka-secret",
			kafkaID:           "lkc-abc123",
			kafkaRestEndpoint: "https://lkc-123.us-east-1.aws.confluent.cloud",
			// Schema Registry invalid (missing endpoint)
			schemaRegistryApiKey:    "sr-key",
			schemaRegistryApiSecret: "sr-secret",
			schemaRegistryClusterId: "lsrc-abc123",
			shouldErr:               true,
			expectedErrMsg:          "All 4 schema_registry_api_key",
			expectedFlags: ResourceMetadataSetFlags{
				isKafkaMetadataSet:          true,  // Kafka should still be marked as set
				isSchemaRegistryMetadataSet: false, // Validation fails here
				isCatalogMetadataSet:        false,
				isFlinkMetadataSet:          false,
				isTableflowMetadataSet:      false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags, diags := validateAllOrNoneAttributesSetForResources(
				tt.kafkaApiKey, tt.kafkaApiSecret, tt.kafkaID, tt.kafkaRestEndpoint,
				tt.schemaRegistryApiKey, tt.schemaRegistryApiSecret, tt.schemaRegistryClusterId, tt.schemaRegistryRestEndpoint, tt.catalogRestEndpoint,
				tt.flinkApiKey, tt.flinkApiSecret, tt.flinkOrganizationId, tt.flinkEnvironmentId, tt.flinkComputePoolId, tt.flinkRestEndpoint, tt.flinkPrincipalId,
				tt.tableflowApiKey, tt.tableflowApiSecret,
			)

			if tt.shouldErr {
				if len(diags) == 0 || !diags.HasError() {
					t.Fatalf("expected error, got none")
				}
				got := diags[0].Summary
				if !strings.Contains(got, tt.expectedErrMsg) {
					t.Fatalf("expected diagnostic to contain %q, got %q", tt.expectedErrMsg, got)
				}
				if !reflect.DeepEqual(flags, tt.expectedFlags) {
					t.Fatalf("expected flags %+v, got %+v", tt.expectedFlags, flags)
				}
				return
			}

			if diags.HasError() {
				t.Fatalf("expected no error, got: %+v", diags)
			}
			if !reflect.DeepEqual(flags, tt.expectedFlags) {
				t.Fatalf("expected flags %+v, got %+v", tt.expectedFlags, flags)
			}
		})
	}
}

func TestValidateAllOrNoneAttributesSetForResourcesWithOAuth(t *testing.T) {
	tests := []struct {
		name string

		kafkaID, kafkaRestEndpoint                                               string
		srID, srRestEndpoint, catalogRestEndpoint                                string
		flinkOrgID, flinkEnvID, flinkPoolID, flinkRestEndpoint, flinkPrincipalID string

		shouldErr      bool
		expectedErrMsg string
		expectedFlags  ResourceMetadataSetFlags
	}{
		{
			name: "all attributes unset - valid",
			expectedFlags: ResourceMetadataSetFlags{
				isKafkaMetadataSet:          false,
				isSchemaRegistryMetadataSet: false,
				isCatalogMetadataSet:        false,
				isFlinkMetadataSet:          false,
			},
		},
		{
			name:              "all attributes set correctly - valid",
			kafkaID:           "lkc-abc123",
			kafkaRestEndpoint: "https://lkc-123.us-east-1.aws.confluent.cloud",
			srID:              "lsrc-abc123",
			srRestEndpoint:    "https://lsrc-123.us-east-1.aws.confluent.cloud",
			flinkOrgID:        "org-123",
			flinkEnvID:        "env-456",
			flinkPoolID:       "pool-789",
			flinkRestEndpoint: "https://flink.us-east-1.aws.confluent.cloud",
			flinkPrincipalID:  "u-123456",
			expectedFlags: ResourceMetadataSetFlags{
				isKafkaMetadataSet:          true,
				isSchemaRegistryMetadataSet: true,
				isCatalogMetadataSet:        true,
				isFlinkMetadataSet:          true,
			},
		},
		{
			name:           "Kafka partially set - invalid (missing rest endpoint)",
			kafkaID:        "lkc-abc123",
			shouldErr:      true,
			expectedErrMsg: "(kafka_rest_endpoint, kafka_id) attributes should both be set",
			expectedFlags:  ResourceMetadataSetFlags{},
		},
		{
			name:              "Kafka partially set - invalid (missing ID)",
			kafkaRestEndpoint: "https://lkc-123.us-east-1.aws.confluent.cloud",
			shouldErr:         true,
			expectedErrMsg:    "(kafka_rest_endpoint, kafka_id) attributes should both be set",
			expectedFlags:     ResourceMetadataSetFlags{},
		},
		{
			name:           "Schema Registry partially set (only ID) - invalid",
			srID:           "lsrc-abc123",
			shouldErr:      true,
			expectedErrMsg: "(either schema_registry_rest_endpoint or catalog_rest_endpoint)",
			expectedFlags:  ResourceMetadataSetFlags{isKafkaMetadataSet: false},
		},
		{
			name:           "Schema Registry partially set (only endpoint) - invalid",
			srRestEndpoint: "https://lsrc-123.us-east-1.aws.confluent.cloud",
			shouldErr:      true,
			expectedErrMsg: "(either schema_registry_rest_endpoint or catalog_rest_endpoint)",
			expectedFlags:  ResourceMetadataSetFlags{isKafkaMetadataSet: false},
		},
		{
			name:                "Schema Registry valid via catalog endpoint",
			srID:                "lsrc-abc123",
			catalogRestEndpoint: "https://catalog.us-east-1.aws.confluent.cloud",
			expectedFlags: ResourceMetadataSetFlags{
				isKafkaMetadataSet:          false,
				isSchemaRegistryMetadataSet: true,
				isCatalogMetadataSet:        true,
				isFlinkMetadataSet:          false,
			},
		},
		{
			name:              "Flink partially set - missing principal",
			flinkOrgID:        "org-123",
			flinkEnvID:        "env-456",
			flinkPoolID:       "pool-789",
			flinkRestEndpoint: "https://flink.us-east-1.aws.confluent.cloud",
			shouldErr:         true,
			expectedErrMsg:    "All 5 (flink_rest_endpoint, organization_id, environment_id, flink_compute_pool_id, flink_principal_id)",
			expectedFlags: ResourceMetadataSetFlags{
				isKafkaMetadataSet:          false,
				isSchemaRegistryMetadataSet: false,
				isCatalogMetadataSet:        false,
				isFlinkMetadataSet:          false,
			},
		},
		{
			name:              "Kafka and Schema Registry valid, Flink unset - valid",
			kafkaID:           "lkc-abc123",
			kafkaRestEndpoint: "https://lkc-123.us-east-1.aws.confluent.cloud",
			srID:              "lsrc-abc123",
			srRestEndpoint:    "https://lsrc-123.us-east-1.aws.confluent.cloud",
			expectedFlags: ResourceMetadataSetFlags{
				isKafkaMetadataSet:          true,
				isSchemaRegistryMetadataSet: true,
				isCatalogMetadataSet:        true,
				isFlinkMetadataSet:          false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags, diags := validateAllOrNoneAttributesSetForResourcesWithOAuth(
				tt.kafkaID, tt.kafkaRestEndpoint,
				tt.srID, tt.srRestEndpoint, tt.catalogRestEndpoint,
				tt.flinkOrgID, tt.flinkEnvID, tt.flinkPoolID, tt.flinkRestEndpoint, tt.flinkPrincipalID,
			)

			if tt.shouldErr {
				if len(diags) == 0 || !diags.HasError() {
					t.Fatalf("expected an error but got none")
				}
				got := diags[0].Summary
				if !strings.Contains(got, tt.expectedErrMsg) {
					t.Fatalf("expected diagnostic to contain %q, got %q", tt.expectedErrMsg, got)
				}
				if !reflect.DeepEqual(flags, tt.expectedFlags) {
					t.Fatalf("expected flags %+v, got %+v", tt.expectedFlags, flags)
				}
				return
			}

			if diags.HasError() {
				t.Fatalf("expected no error, got: %+v", diags)
			}
			if !reflect.DeepEqual(flags, tt.expectedFlags) {
				t.Fatalf("expected flags %+v, got %+v", tt.expectedFlags, flags)
			}
		})
	}
}

func TestGetTimeoutFor(t *testing.T) {
	tests := []struct {
		name        string
		clusterType string
		expected    time.Duration
	}{
		{
			name:        "dedicated cluster returns 72 hours",
			clusterType: kafkaClusterTypeDedicated,
			expected:    72 * time.Hour,
		},
		{
			name:        "basic cluster returns 1 hour",
			clusterType: "Basic",
			expected:    1 * time.Hour,
		},
		{
			name:        "standard cluster returns 1 hour",
			clusterType: "Standard",
			expected:    1 * time.Hour,
		},
		{
			name:        "empty string returns 1 hour",
			clusterType: "",
			expected:    1 * time.Hour,
		},
		{
			name:        "arbitrary string returns 1 hour",
			clusterType: "SomeOtherType",
			expected:    1 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTimeoutFor(tt.clusterType)
			if got != tt.expected {
				t.Fatalf("getTimeoutFor(%q) = %v, want %v", tt.clusterType, got, tt.expected)
			}
		})
	}
}

func TestStringToAclResourceType(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    kafkarestv3.AclResourceType
		expectError bool
	}{
		{name: "UNKNOWN", input: "UNKNOWN", expected: kafkarestv3.UNKNOWN},
		{name: "ANY", input: "ANY", expected: kafkarestv3.ANY},
		{name: "TOPIC", input: "TOPIC", expected: kafkarestv3.TOPIC},
		{name: "GROUP", input: "GROUP", expected: kafkarestv3.GROUP},
		{name: "CLUSTER", input: "CLUSTER", expected: kafkarestv3.CLUSTER},
		{name: "TRANSACTIONAL_ID", input: "TRANSACTIONAL_ID", expected: kafkarestv3.TRANSACTIONAL_ID},
		{name: "DELEGATION_TOKEN", input: "DELEGATION_TOKEN", expected: kafkarestv3.DELEGATION_TOKEN},
		{name: "invalid type returns error", input: "INVALID", expectError: true},
		{name: "empty string returns error", input: "", expectError: true},
		{name: "lowercase topic returns error", input: "topic", expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := stringToAclResourceType(tt.input)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error for input %q, got nil", tt.input)
				}
				if !strings.Contains(err.Error(), "unknown ACL resource type") {
					t.Fatalf("expected 'unknown ACL resource type' in error, got: %s", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for input %q: %s", tt.input, err)
			}
			if got != tt.expected {
				t.Fatalf("stringToAclResourceType(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGetEnv(t *testing.T) {

	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		setEnv       bool
		expected     string
	}{
		{
			name:         "returns env var when set",
			key:          testEnvKey,
			defaultValue: "default",
			envValue:     "from-env",
			setEnv:       true,
			expected:     "from-env",
		},
		{
			name:         "returns default when env var not set",
			key:          testEnvKey,
			defaultValue: "default-val",
			setEnv:       false,
			expected:     "default-val",
		},
		{
			name:         "returns default when env var is empty string",
			key:          testEnvKey,
			defaultValue: "fallback",
			envValue:     "",
			setEnv:       true,
			expected:     "fallback",
		},
		{
			name:         "returns env var with empty default",
			key:          testEnvKey,
			defaultValue: "",
			envValue:     "value",
			setEnv:       true,
			expected:     "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(testEnvKey)
			if tt.setEnv {
				os.Setenv(testEnvKey, tt.envValue)
				defer os.Unsetenv(testEnvKey)
			}
			got := getEnv(tt.key, tt.defaultValue)
			if got != tt.expected {
				t.Fatalf("getEnv(%q, %q) = %q, want %q", tt.key, tt.defaultValue, got, tt.expected)
			}
		})
	}
}

func TestClusterCrnToRbacClusterCrn(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name:     "valid CRN with kafka suffix",
			input:    "crn://confluent.cloud/organization=org1/environment=env1/cloud-cluster=lkc-198rjz/kafka=lkc-198rjz",
			expected: "crn://confluent.cloud/organization=org1/environment=env1/cloud-cluster=lkc-198rjz",
		},
		{
			name:        "CRN without kafka suffix returns error",
			input:       "crn://confluent.cloud/organization=org1/environment=env1/cloud-cluster=lkc-198rjz",
			expectError: true,
		},
		{
			name:        "empty string returns error",
			input:       "",
			expectError: true,
		},
		{
			name:     "CRN with multiple kafka suffixes strips last one",
			input:    "crn://confluent.cloud/kafka=first/kafka=second",
			expected: "crn://confluent.cloud/kafka=first",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := clusterCrnToRbacClusterCrn(tt.input)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error for input %q, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for input %q: %s", tt.input, err)
			}
			if got != tt.expected {
				t.Fatalf("clusterCrnToRbacClusterCrn(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestConvertToStringStringMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]string
	}{
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: map[string]string{},
		},
		{
			name:     "single entry",
			input:    map[string]interface{}{"key1": "value1"},
			expected: map[string]string{"key1": "value1"},
		},
		{
			name:     "multiple entries",
			input:    map[string]interface{}{"a": "1", "b": "2", "c": "3"},
			expected: map[string]string{"a": "1", "b": "2", "c": "3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertToStringStringMap(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Fatalf("convertToStringStringMap() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConvertToStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    []interface{}
		expected []string
	}{
		{
			name:     "empty slice",
			input:    []interface{}{},
			expected: []string{},
		},
		{
			name:     "string items",
			input:    []interface{}{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "mixed types use fmt.Sprint",
			input:    []interface{}{"hello", 42, true},
			expected: []string{"hello", "42", "true"},
		},
		{
			name:     "single item",
			input:    []interface{}{"only"},
			expected: []string{"only"},
		},
		{
			name:     "nil items inside slice",
			input:    []interface{}{nil, "a", nil},
			expected: []string{"<nil>", "a", "<nil>"},
		},
		{
			name:     "numeric items",
			input:    []interface{}{0, 1, -1},
			expected: []string{"0", "1", "-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertToStringSlice(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Fatalf("convertToStringSlice() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestExtractPageToken(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name:     "valid URL with page_token",
			input:    "https://api.confluent.cloud/iam/v2/service-accounts?page_token=abc123",
			expected: "abc123",
		},
		{
			name:     "URL with multiple query params",
			input:    "https://api.confluent.cloud/iam/v2/service-accounts?page_size=10&page_token=xyz789&other=val",
			expected: "xyz789",
		},
		{
			name:        "URL without page_token",
			input:       "https://api.confluent.cloud/iam/v2/service-accounts?page_size=10",
			expectError: true,
		},
		{
			name:        "invalid URL",
			input:       "://not-a-valid-url",
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expectError: true,
		},
		{
			name:        "page_token with empty value",
			input:       "https://api.confluent.cloud/iam/v2/service-accounts?page_token=",
			expectError: true,
		},
		{
			name:     "page_token with encoded characters",
			input:    "https://api.confluent.cloud/v2/accounts?page_token=abc%3D%3D123",
			expected: "abc==123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractPageToken(tt.input)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error for input %q, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for input %q: %s", tt.input, err)
			}
			if got != tt.expected {
				t.Fatalf("extractPageToken(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestVerifyListValues(t *testing.T) {
	tests := []struct {
		name           string
		values         []string
		acceptedValues []string
		ignoreCase     bool
		expectError    bool
	}{
		{
			name:           "all values accepted",
			values:         []string{"a", "b"},
			acceptedValues: []string{"a", "b", "c"},
			ignoreCase:     false,
		},
		{
			name:           "value not in accepted list",
			values:         []string{"a", "d"},
			acceptedValues: []string{"a", "b", "c"},
			ignoreCase:     false,
			expectError:    true,
		},
		{
			name:           "case mismatch without ignoreCase",
			values:         []string{"A"},
			acceptedValues: []string{"a", "b"},
			ignoreCase:     false,
			expectError:    true,
		},
		{
			name:           "case mismatch with ignoreCase",
			values:         []string{"A"},
			acceptedValues: []string{"a", "b"},
			ignoreCase:     true,
		},
		{
			name:           "empty values list",
			values:         []string{},
			acceptedValues: []string{"a"},
			ignoreCase:     false,
		},
		{
			name:           "empty accepted list",
			values:         []string{"a"},
			acceptedValues: []string{},
			ignoreCase:     false,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := verifyListValues(tt.values, tt.acceptedValues, tt.ignoreCase)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
		})
	}
}

func TestStringInSlice(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		slice      []string
		ignoreCase bool
		expected   bool
	}{
		{
			name:     "found exact match",
			target:   "b",
			slice:    []string{"a", "b", "c"},
			expected: true,
		},
		{
			name:     "not found",
			target:   "d",
			slice:    []string{"a", "b", "c"},
			expected: false,
		},
		{
			name:       "case insensitive match",
			target:     "ABC",
			slice:      []string{"abc", "def"},
			ignoreCase: true,
			expected:   true,
		},
		{
			name:       "case sensitive no match",
			target:     "ABC",
			slice:      []string{"abc", "def"},
			ignoreCase: false,
			expected:   false,
		},
		{
			name:     "empty slice",
			target:   "a",
			slice:    []string{},
			expected: false,
		},
		{
			name:     "empty target in slice",
			target:   "",
			slice:    []string{"", "a"},
			expected: true,
		},
		{
			name:     "empty target not in slice",
			target:   "",
			slice:    []string{"a", "b"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stringInSlice(tt.target, tt.slice, tt.ignoreCase)
			if got != tt.expected {
				t.Fatalf("stringInSlice(%q, %v, %v) = %v, want %v", tt.target, tt.slice, tt.ignoreCase, got, tt.expected)
			}
		})
	}
}

func TestExtractCloudAndRegionName(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedCloud  string
		expectedRegion string
		expectError    bool
	}{
		{
			name:           "two-part format (new API)",
			input:          "aws.us-east-1",
			expectedCloud:  "aws",
			expectedRegion: "us-east-1",
		},
		{
			name:           "three-part format (old API)",
			input:          "env-abc.gcp.us-central1",
			expectedCloud:  "gcp",
			expectedRegion: "us-central1",
		},
		{
			name:        "single part is invalid",
			input:       "aws",
			expectError: true,
		},
		{
			name:        "four parts is invalid",
			input:       "a.b.c.d",
			expectError: true,
		},
		{
			name:        "empty string is invalid",
			input:       "",
			expectError: true,
		},
		{
			name:           "cloud with empty region (trailing dot)",
			input:          "aws.",
			expectedCloud:  "aws",
			expectedRegion: "",
		},
		{
			name:           "empty cloud with region (leading dot)",
			input:          ".us-east-1",
			expectedCloud:  "",
			expectedRegion: "us-east-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cloud, region, err := extractCloudAndRegionName(tt.input)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error for input %q, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for input %q: %s", tt.input, err)
			}
			if cloud != tt.expectedCloud {
				t.Fatalf("cloud: got %q, want %q", cloud, tt.expectedCloud)
			}
			if region != tt.expectedRegion {
				t.Fatalf("region: got %q, want %q", region, tt.expectedRegion)
			}
		})
	}
}

func TestParseStatementName(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name:     "valid three-part ID",
			input:    "env-abc/pool-xyz/my-statement",
			expected: "my-statement",
		},
		{
			name:     "valid with complex statement name",
			input:    "env-123/pool-456/tf-2024-01-01-120000-uuid-here",
			expected: "tf-2024-01-01-120000-uuid-here",
		},
		{
			name:        "two parts is invalid",
			input:       "env-abc/pool-xyz",
			expectError: true,
		},
		{
			name:        "one part is invalid",
			input:       "env-abc",
			expectError: true,
		},
		{
			name:        "four parts is invalid",
			input:       "env-abc/pool-xyz/stmt/extra",
			expectError: true,
		},
		{
			name:        "empty string is invalid",
			input:       "",
			expectError: true,
		},
		{
			name:     "parts with empty middle segment",
			input:    "env-abc//stmt-name",
			expected: "stmt-name",
		},
		{
			name:     "parts with empty first segment",
			input:    "/pool-xyz/stmt-name",
			expected: "stmt-name",
		},
		{
			name:     "empty statement name at end",
			input:    "env-abc/pool-xyz/",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseStatementName(tt.input)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error for input %q, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for input %q: %s", tt.input, err)
			}
			if got != tt.expected {
				t.Fatalf("parseStatementName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCanUpdateEntityNameBusinessMetadata(t *testing.T) {
	tests := []struct {
		name          string
		entityType    string
		oldEntityName string
		newEntityName string
		expected      bool
	}{
		{
			name:          "same entity name returns true for schema type",
			entityType:    schemaEntityType,
			oldEntityName: "lsrc-abc:subject1:100",
			newEntityName: "lsrc-abc:subject1:100",
			expected:      true,
		},
		{
			name:          "same entity name returns true for field type",
			entityType:    fieldEntityType,
			oldEntityName: "lsrc-abc:subject1:100:field1",
			newEntityName: "lsrc-abc:subject1:100:field1",
			expected:      true,
		},
		{
			name:          "schema type with newer schema ID",
			entityType:    schemaEntityType,
			oldEntityName: "lsrc-abc:subject1:100",
			newEntityName: "lsrc-abc:subject1:101",
			expected:      true,
		},
		{
			name:          "schema type with older schema ID",
			entityType:    schemaEntityType,
			oldEntityName: "lsrc-abc:subject1:101",
			newEntityName: "lsrc-abc:subject1:100",
			expected:      false,
		},
		{
			name:          "schema type with different subject",
			entityType:    schemaEntityType,
			oldEntityName: "lsrc-abc:subject1:100",
			newEntityName: "lsrc-abc:subject2:101",
			expected:      false,
		},
		{
			name:          "field type not supported for business metadata (different names)",
			entityType:    fieldEntityType,
			oldEntityName: "lsrc-abc:subject1:100:field1",
			newEntityName: "lsrc-abc:subject1:101:field1",
			expected:      false,
		},
		{
			name:          "record type not supported for business metadata (different names)",
			entityType:    recordEntityType,
			oldEntityName: "lsrc-abc:subject1:100:record1",
			newEntityName: "lsrc-abc:subject1:101:record1",
			expected:      false,
		},
		{
			name:          "unknown entity type with different names returns false",
			entityType:    "unknown_type",
			oldEntityName: "something",
			newEntityName: "something_else",
			expected:      false,
		},
		{
			name:          "unknown entity type with same names returns true",
			entityType:    "unknown_type",
			oldEntityName: "same",
			newEntityName: "same",
			expected:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := canUpdateEntityNameBusinessMetadata(tt.entityType, tt.oldEntityName, tt.newEntityName)
			if got != tt.expected {
				t.Fatalf("canUpdateEntityNameBusinessMetadata(%q, %q, %q) = %v, want %v",
					tt.entityType, tt.oldEntityName, tt.newEntityName, got, tt.expected)
			}
		})
	}
}

func TestResponseHasExpectedStatusCode(t *testing.T) {
	tests := []struct {
		name               string
		response           *http.Response
		expectedStatusCode int
		expected           bool
	}{
		{
			name:               "nil response returns false",
			response:           nil,
			expectedStatusCode: http.StatusOK,
			expected:           false,
		},
		{
			name:               "matching status code",
			response:           &http.Response{StatusCode: http.StatusOK},
			expectedStatusCode: http.StatusOK,
			expected:           true,
		},
		{
			name:               "non-matching status code",
			response:           &http.Response{StatusCode: http.StatusNotFound},
			expectedStatusCode: http.StatusOK,
			expected:           false,
		},
		{
			name:               "forbidden status code",
			response:           &http.Response{StatusCode: http.StatusForbidden},
			expectedStatusCode: http.StatusForbidden,
			expected:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResponseHasExpectedStatusCode(tt.response, tt.expectedStatusCode)
			if got != tt.expected {
				t.Fatalf("ResponseHasExpectedStatusCode() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPtr(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "non-empty string", input: "hello"},
		{name: "empty string", input: ""},
		{name: "string with spaces", input: "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ptr(tt.input)
			if got == nil {
				t.Fatalf("ptr(%q) returned nil", tt.input)
			}
			if *got != tt.input {
				t.Fatalf("*ptr(%q) = %q, want %q", tt.input, *got, tt.input)
			}
		})
	}
}

func TestIsNewSchemaIdGreaterThanOld(t *testing.T) {
	tests := []struct {
		name     string
		oldParts []string
		newParts []string
		expected bool
	}{
		{
			name:     "new schema ID greater",
			oldParts: []string{"lsrc-abc", "subject1", "100"},
			newParts: []string{"lsrc-abc", "subject1", "101"},
			expected: true,
		},
		{
			name:     "new schema ID equal",
			oldParts: []string{"lsrc-abc", "subject1", "100"},
			newParts: []string{"lsrc-abc", "subject1", "100"},
			expected: false,
		},
		{
			name:     "new schema ID less",
			oldParts: []string{"lsrc-abc", "subject1", "101"},
			newParts: []string{"lsrc-abc", "subject1", "100"},
			expected: false,
		},
		{
			name:     "old schema ID not a number",
			oldParts: []string{"lsrc-abc", "subject1", "abc"},
			newParts: []string{"lsrc-abc", "subject1", "100"},
			expected: false,
		},
		{
			name:     "new schema ID not a number",
			oldParts: []string{"lsrc-abc", "subject1", "100"},
			newParts: []string{"lsrc-abc", "subject1", "abc"},
			expected: false,
		},
		{
			name:     "both not numbers",
			oldParts: []string{"lsrc-abc", "subject1", "abc"},
			newParts: []string{"lsrc-abc", "subject1", "def"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNewSchemaIdGreaterThanOld(tt.oldParts, tt.newParts)
			if got != tt.expected {
				t.Fatalf("isNewSchemaIdGreaterThanOld(%v, %v) = %v, want %v",
					tt.oldParts, tt.newParts, got, tt.expected)
			}
		})
	}
}

func TestGenerateFlinkStatementName(t *testing.T) {
	name := generateFlinkStatementName()

	// Expected format: tf-YYYY-MM-DD-HHMMSS-<uuid>
	pattern := `^tf-\d{4}-\d{2}-\d{2}-\d{6}-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`
	matched, err := regexp.MatchString(pattern, name)
	if err != nil {
		t.Fatalf("regex error: %s", err)
	}
	if !matched {
		t.Fatalf("generateFlinkStatementName() = %q, does not match expected pattern %q", name, pattern)
	}

	if !strings.HasPrefix(name, "tf-") {
		t.Fatalf("expected prefix 'tf-', got %q", name)
	}

	// Generate two names and verify they are different (due to UUID)
	name2 := generateFlinkStatementName()
	if name == name2 {
		t.Fatalf("two generated names should be different, but both are %q", name)
	}
}

func TestCreateDescriptiveError(t *testing.T) {
	t.Run("nil error returns nil", func(t *testing.T) {
		got := createDescriptiveError(nil)
		if got != nil {
			t.Fatalf("createDescriptiveError(nil) = %v, want nil", got)
		}
	})

	t.Run("regular error returns same message", func(t *testing.T) {
		err := fmt.Errorf("something went wrong")
		got := createDescriptiveError(err)
		if got == nil {
			t.Fatal("expected non-nil error")
		}
		if got.Error() != "something went wrong" {
			t.Fatalf("got %q, want %q", got.Error(), "something went wrong")
		}
	})

	t.Run("error with response body appends body", func(t *testing.T) {
		err := fmt.Errorf("400 Bad Request")
		body := io.NopCloser(bytes.NewBufferString(`{"error":"invalid input"}`))
		resp := &http.Response{
			StatusCode: 400,
			Body:       body,
		}
		got := createDescriptiveError(err, resp)
		if got == nil {
			t.Fatal("expected non-nil error")
		}
		if !strings.Contains(got.Error(), "400 Bad Request") {
			t.Fatalf("expected error to contain '400 Bad Request', got %q", got.Error())
		}
		if !strings.Contains(got.Error(), "invalid input") {
			t.Fatalf("expected error to contain response body content, got %q", got.Error())
		}
	})

	t.Run("nil error with response still returns nil", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString("ok")),
		}
		got := createDescriptiveError(nil, resp)
		if got != nil {
			t.Fatalf("createDescriptiveError(nil, resp) = %v, want nil", got)
		}
	})

	t.Run("error with gzip compressed response body", func(t *testing.T) {
		var buf bytes.Buffer
		gzWriter := gzip.NewWriter(&buf)
		gzWriter.Write([]byte(`{"error":"gzip compressed error detail"}`))
		gzWriter.Close()

		err := fmt.Errorf("500 Internal Server Error")
		resp := &http.Response{
			StatusCode: 500,
			Body:       io.NopCloser(&buf),
		}
		got := createDescriptiveError(err, resp)
		if got == nil {
			t.Fatal("expected non-nil error")
		}
		if !strings.Contains(got.Error(), "gzip compressed error detail") {
			t.Fatalf("expected error to contain decompressed body, got %q", got.Error())
		}
	})

	t.Run("error with nil response body", func(t *testing.T) {
		err := fmt.Errorf("503 Service Unavailable")
		resp := &http.Response{
			StatusCode: 503,
			Body:       nil,
		}
		got := createDescriptiveError(err, resp)
		if got == nil {
			t.Fatal("expected non-nil error")
		}
		if got.Error() != "503 Service Unavailable" {
			t.Fatalf("expected original error message, got %q", got.Error())
		}
	})

	t.Run("error with empty response body", func(t *testing.T) {
		err := fmt.Errorf("502 Bad Gateway")
		resp := &http.Response{
			StatusCode: 502,
			Body:       io.NopCloser(bytes.NewBufferString("")),
		}
		got := createDescriptiveError(err, resp)
		if got == nil {
			t.Fatal("expected non-nil error")
		}
		if !strings.Contains(got.Error(), "502 Bad Gateway") {
			t.Fatalf("expected error to contain original message, got %q", got.Error())
		}
	})

	t.Run("error with no response provided", func(t *testing.T) {
		err := fmt.Errorf("connection timeout")
		got := createDescriptiveError(err)
		if got == nil {
			t.Fatal("expected non-nil error")
		}
		if got.Error() != "connection timeout" {
			t.Fatalf("expected 'connection timeout', got %q", got.Error())
		}
	})
}

func TestIsNonKafkaRestApiResourceNotFound(t *testing.T) {
	tests := []struct {
		name     string
		response *http.Response
		expected bool
	}{
		{
			name:     "nil response returns false",
			response: nil,
			expected: false,
		},
		{
			name:     "404 returns true",
			response: &http.Response{StatusCode: http.StatusNotFound},
			expected: true,
		},
		{
			name: "403 without invalid API key message returns true",
			response: &http.Response{
				StatusCode: http.StatusForbidden,
				Body:       io.NopCloser(strings.NewReader(`{"error":"not authorized"}`)),
			},
			expected: true,
		},
		{
			name: "403 with invalid API key message returns false",
			response: &http.Response{
				StatusCode: http.StatusForbidden,
				Body:       io.NopCloser(strings.NewReader(`{"error":"invalid API key"}`)),
			},
			expected: false,
		},
		{
			name:     "200 returns false",
			response: &http.Response{StatusCode: http.StatusOK},
			expected: false,
		},
		{
			name:     "500 returns false",
			response: &http.Response{StatusCode: http.StatusInternalServerError},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNonKafkaRestApiResourceNotFound(tt.response)
			if got != tt.expected {
				t.Fatalf("isNonKafkaRestApiResourceNotFound() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsKafkaApiKey(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   apikeysv2.IamV2ApiKey
		expected bool
	}{
		{
			name: "Kafka API Key with Cluster kind and cmk/v2",
			apiKey: apikeysv2.IamV2ApiKey{
				Spec: &apikeysv2.IamV2ApiKeySpec{
					Resource: &apikeysv2.ObjectReference{
						Kind:       apikeysv2.PtrString(clusterKind),
						ApiVersion: apikeysv2.PtrString(cmkApiVersion),
					},
				},
			},
			expected: true,
		},
		{
			name: "Not Kafka - SR kind with srcm/v3",
			apiKey: apikeysv2.IamV2ApiKey{
				Spec: &apikeysv2.IamV2ApiKeySpec{
					Resource: &apikeysv2.ObjectReference{
						Kind:       apikeysv2.PtrString(schemaRegistryKind),
						ApiVersion: apikeysv2.PtrString(srcmV3ApiVersion),
					},
				},
			},
			expected: false,
		},
		{
			name: "Not Kafka - Cluster kind but wrong api version",
			apiKey: apikeysv2.IamV2ApiKey{
				Spec: &apikeysv2.IamV2ApiKeySpec{
					Resource: &apikeysv2.ObjectReference{
						Kind:       apikeysv2.PtrString(clusterKind),
						ApiVersion: apikeysv2.PtrString(srcmV2ApiVersion),
					},
				},
			},
			expected: false,
		},
		{
			name: "Not Kafka - Region kind with fcpm/v2",
			apiKey: apikeysv2.IamV2ApiKey{
				Spec: &apikeysv2.IamV2ApiKeySpec{
					Resource: &apikeysv2.ObjectReference{
						Kind:       apikeysv2.PtrString(regionKind),
						ApiVersion: apikeysv2.PtrString(fcpmApiVersion),
					},
				},
			},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isKafkaApiKey(tt.apiKey)
			if result != tt.expected {
				t.Errorf("%s: isKafkaApiKey() = %v; want %v", tt.name, result, tt.expected)
			}
		})
	}
}

func TestIsFlinkApiKey(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   apikeysv2.IamV2ApiKey
		expected bool
	}{
		{
			name: "Flink API Key with Region kind and fcpm/v2",
			apiKey: apikeysv2.IamV2ApiKey{
				Spec: &apikeysv2.IamV2ApiKeySpec{
					Resource: &apikeysv2.ObjectReference{
						Kind:       apikeysv2.PtrString(regionKind),
						ApiVersion: apikeysv2.PtrString(fcpmApiVersion),
					},
				},
			},
			expected: true,
		},
		{
			name: "Not Flink - Cluster kind with cmk/v2",
			apiKey: apikeysv2.IamV2ApiKey{
				Spec: &apikeysv2.IamV2ApiKeySpec{
					Resource: &apikeysv2.ObjectReference{
						Kind:       apikeysv2.PtrString(clusterKind),
						ApiVersion: apikeysv2.PtrString(cmkApiVersion),
					},
				},
			},
			expected: false,
		},
		{
			name: "Not Flink - Region kind but wrong api version",
			apiKey: apikeysv2.IamV2ApiKey{
				Spec: &apikeysv2.IamV2ApiKeySpec{
					Resource: &apikeysv2.ObjectReference{
						Kind:       apikeysv2.PtrString(regionKind),
						ApiVersion: apikeysv2.PtrString(cmkApiVersion),
					},
				},
			},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isFlinkApiKey(tt.apiKey)
			if result != tt.expected {
				t.Errorf("%s: isFlinkApiKey() = %v; want %v", tt.name, result, tt.expected)
			}
		})
	}
}

func TestIsKsqlDbClusterApiKey(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   apikeysv2.IamV2ApiKey
		expected bool
	}{
		{
			name: "ksqlDB API Key with ksqlDB kind and ksqldbcm/v2",
			apiKey: apikeysv2.IamV2ApiKey{
				Spec: &apikeysv2.IamV2ApiKeySpec{
					Resource: &apikeysv2.ObjectReference{
						Kind:       apikeysv2.PtrString(ksqlDbKind),
						ApiVersion: apikeysv2.PtrString(ksqldbcmApiVersion),
					},
				},
			},
			expected: true,
		},
		{
			name: "ksqlDB API Key with Cluster kind and ksqldbcm/v2",
			apiKey: apikeysv2.IamV2ApiKey{
				Spec: &apikeysv2.IamV2ApiKeySpec{
					Resource: &apikeysv2.ObjectReference{
						Kind:       apikeysv2.PtrString(clusterKind),
						ApiVersion: apikeysv2.PtrString(ksqldbcmApiVersion),
					},
				},
			},
			expected: true,
		},
		{
			name: "Not ksqlDB - ksqlDB kind but wrong api version",
			apiKey: apikeysv2.IamV2ApiKey{
				Spec: &apikeysv2.IamV2ApiKeySpec{
					Resource: &apikeysv2.ObjectReference{
						Kind:       apikeysv2.PtrString(ksqlDbKind),
						ApiVersion: apikeysv2.PtrString(cmkApiVersion),
					},
				},
			},
			expected: false,
		},
		{
			name: "Not ksqlDB - Region kind with ksqldbcm/v2",
			apiKey: apikeysv2.IamV2ApiKey{
				Spec: &apikeysv2.IamV2ApiKeySpec{
					Resource: &apikeysv2.ObjectReference{
						Kind:       apikeysv2.PtrString(regionKind),
						ApiVersion: apikeysv2.PtrString(ksqldbcmApiVersion),
					},
				},
			},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isKsqlDbClusterApiKey(tt.apiKey)
			if result != tt.expected {
				t.Errorf("%s: isKsqlDbClusterApiKey() = %v; want %v", tt.name, result, tt.expected)
			}
		})
	}
}

func TestIsTableflowApiKey(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   apikeysv2.IamV2ApiKey
		expected bool
	}{
		{
			name: "Tableflow API Key",
			apiKey: apikeysv2.IamV2ApiKey{
				Spec: &apikeysv2.IamV2ApiKeySpec{
					Resource: &apikeysv2.ObjectReference{
						Kind: apikeysv2.PtrString(tableflowKind),
						Id:   tableflowKindInLowercase,
					},
				},
			},
			expected: true,
		},
		{
			name: "Not Tableflow - wrong kind",
			apiKey: apikeysv2.IamV2ApiKey{
				Spec: &apikeysv2.IamV2ApiKeySpec{
					Resource: &apikeysv2.ObjectReference{
						Kind: apikeysv2.PtrString(clusterKind),
						Id:   tableflowKindInLowercase,
					},
				},
			},
			expected: false,
		},
		{
			name: "Not Tableflow - wrong id",
			apiKey: apikeysv2.IamV2ApiKey{
				Spec: &apikeysv2.IamV2ApiKeySpec{
					Resource: &apikeysv2.ObjectReference{
						Kind: apikeysv2.PtrString(tableflowKind),
						Id:   "wrong-id",
					},
				},
			},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTableflowApiKey(tt.apiKey)
			if result != tt.expected {
				t.Errorf("%s: isTableflowApiKey() = %v; want %v", tt.name, result, tt.expected)
			}
		})
	}
}

func TestValidateApiKey(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    apikeysv2.IamV2ApiKey
		expectErr bool
	}{
		{
			name: "valid API key with both ID and secret",
			apiKey: apikeysv2.IamV2ApiKey{
				Id: apikeysv2.PtrString("ABCDEFGHIJK123"),
				Spec: &apikeysv2.IamV2ApiKeySpec{
					Secret: apikeysv2.PtrString("supersecret"),
				},
			},
			expectErr: false,
		},
		{
			name: "missing secret",
			apiKey: apikeysv2.IamV2ApiKey{
				Id:   apikeysv2.PtrString("ABCDEFGHIJK123"),
				Spec: &apikeysv2.IamV2ApiKeySpec{},
			},
			expectErr: true,
		},
		{
			name: "missing ID",
			apiKey: apikeysv2.IamV2ApiKey{
				Spec: &apikeysv2.IamV2ApiKeySpec{
					Secret: apikeysv2.PtrString("supersecret"),
				},
			},
			expectErr: true,
		},
		{
			name: "empty string ID returns error",
			apiKey: apikeysv2.IamV2ApiKey{
				Id: apikeysv2.PtrString(""),
				Spec: &apikeysv2.IamV2ApiKeySpec{
					Secret: apikeysv2.PtrString("supersecret"),
				},
			},
			expectErr: true,
		},
		{
			name: "empty string secret returns error",
			apiKey: apikeysv2.IamV2ApiKey{
				Id: apikeysv2.PtrString("ABCDEFGHIJK123"),
				Spec: &apikeysv2.IamV2ApiKeySpec{
					Secret: apikeysv2.PtrString(""),
				},
			},
			expectErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateApiKey(tt.apiKey)
			if (err != nil) != tt.expectErr {
				t.Errorf("validateApiKey() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestCkuCheck(t *testing.T) {
	tests := []struct {
		name         string
		cku          int32
		availability string
		expectErr    bool
	}{
		{
			name:         "single-zone with 1 CKU is valid",
			cku:          1,
			availability: singleZone,
			expectErr:    false,
		},
		{
			name:         "single-zone with 0 CKU is invalid",
			cku:          0,
			availability: singleZone,
			expectErr:    true,
		},
		{
			name:         "multi-zone with 2 CKUs is valid",
			cku:          2,
			availability: multiZone,
			expectErr:    false,
		},
		{
			name:         "multi-zone with 1 CKU is invalid",
			cku:          1,
			availability: multiZone,
			expectErr:    true,
		},
		{
			name:         "multi-zone with 0 CKU is invalid",
			cku:          0,
			availability: multiZone,
			expectErr:    true,
		},
		{
			name:         "single-zone with 5 CKUs is valid",
			cku:          5,
			availability: singleZone,
			expectErr:    false,
		},
		{
			name:         "multi-zone with 10 CKUs is valid",
			cku:          10,
			availability: multiZone,
			expectErr:    false,
		},
		{
			name:         "negative CKU single-zone is invalid",
			cku:          -1,
			availability: singleZone,
			expectErr:    true,
		},
		{
			name:         "negative CKU multi-zone is invalid",
			cku:          -5,
			availability: multiZone,
			expectErr:    true,
		},
		{
			name:         "unknown availability with CKU 0 passes (no check for unknown)",
			cku:          0,
			availability: "UNKNOWN",
			expectErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ckuCheck(tt.cku, tt.availability)
			if (err != nil) != tt.expectErr {
				t.Errorf("ckuCheck(%d, %q) error = %v, expectErr %v", tt.cku, tt.availability, err, tt.expectErr)
			}
		})
	}
}

func TestCreateSchemaId(t *testing.T) {
	tests := []struct {
		name                   string
		clusterId              string
		subjectName            string
		identifier             int32
		shouldRecreateOnUpdate bool
		expected               string
	}{
		{
			name:                   "recreate on update uses numeric identifier",
			clusterId:              "lsrc-abc123",
			subjectName:            "my-subject",
			identifier:             100042,
			shouldRecreateOnUpdate: true,
			expected:               "lsrc-abc123/my-subject/100042",
		},
		{
			name:                   "no recreate on update uses latest",
			clusterId:              "lsrc-abc123",
			subjectName:            "my-subject",
			identifier:             100042,
			shouldRecreateOnUpdate: false,
			expected:               "lsrc-abc123/my-subject/latest",
		},
		{
			name:                   "subject with slashes and recreate",
			clusterId:              "lsrc-xyz",
			subjectName:            "com.example/my-topic-value",
			identifier:             1,
			shouldRecreateOnUpdate: true,
			expected:               "lsrc-xyz/com.example/my-topic-value/1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createSchemaId(tt.clusterId, tt.subjectName, tt.identifier, tt.shouldRecreateOnUpdate)
			if result != tt.expected {
				t.Errorf("createSchemaId() = %q; want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractSubjectInfoFromTfId(t *testing.T) {
	tests := []struct {
		name               string
		terraformId        string
		expectedClusterId  string
		expectedSubject    string
		expectedIdentifier string
		expectErr          bool
	}{
		{
			name:               "standard 3-part ID",
			terraformId:        "lsrc-abc123/my-subject/100042",
			expectedClusterId:  "lsrc-abc123",
			expectedSubject:    "my-subject",
			expectedIdentifier: "100042",
		},
		{
			name:               "subject with slashes",
			terraformId:        "lsrc-abc123/com.example/my-topic-value/latest",
			expectedClusterId:  "lsrc-abc123",
			expectedSubject:    "com.example/my-topic-value",
			expectedIdentifier: "latest",
		},
		{
			name:               "latest identifier",
			terraformId:        "lsrc-abc123/my-subject/latest",
			expectedClusterId:  "lsrc-abc123",
			expectedSubject:    "my-subject",
			expectedIdentifier: "latest",
		},
		{
			name:        "too few parts",
			terraformId: "lsrc-abc123/my-subject",
			expectErr:   true,
		},
		{
			name:        "single part",
			terraformId: "lsrc-abc123",
			expectErr:   true,
		},
		{
			name:        "empty string",
			terraformId: "",
			expectErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clusterId, subject, identifier, err := extractSubjectInfoFromTfId(tt.terraformId)
			if (err != nil) != tt.expectErr {
				t.Fatalf("extractSubjectInfoFromTfId(%q) error = %v, expectErr %v", tt.terraformId, err, tt.expectErr)
			}
			if err != nil {
				return
			}
			if clusterId != tt.expectedClusterId {
				t.Errorf("clusterId = %q; want %q", clusterId, tt.expectedClusterId)
			}
			if subject != tt.expectedSubject {
				t.Errorf("subject = %q; want %q", subject, tt.expectedSubject)
			}
			if identifier != tt.expectedIdentifier {
				t.Errorf("identifier = %q; want %q", identifier, tt.expectedIdentifier)
			}
		})
	}
}

func TestExtractSchemaIdentifierFromTfId(t *testing.T) {
	tests := []struct {
		name        string
		terraformId string
		expected    string
		expectErr   bool
	}{
		{
			name:        "numeric identifier",
			terraformId: "lsrc-abc123/my-subject/100042",
			expected:    "100042",
		},
		{
			name:        "latest identifier",
			terraformId: "lsrc-abc123/my-subject/latest",
			expected:    "latest",
		},
		{
			name:        "invalid format",
			terraformId: "bad-id",
			expectErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractSchemaIdentifierFromTfId(tt.terraformId)
			if (err != nil) != tt.expectErr {
				t.Fatalf("extractSchemaIdentifierFromTfId(%q) error = %v, expectErr %v", tt.terraformId, err, tt.expectErr)
			}
			if err == nil && result != tt.expected {
				t.Errorf("got %q; want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractSubjectNameFromTfId(t *testing.T) {
	tests := []struct {
		name        string
		terraformId string
		expected    string
		expectErr   bool
	}{
		{
			name:        "simple subject",
			terraformId: "lsrc-abc123/my-subject/100042",
			expected:    "my-subject",
		},
		{
			name:        "subject with slashes",
			terraformId: "lsrc-abc123/com.example/my-topic-value/latest",
			expected:    "com.example/my-topic-value",
		},
		{
			name:        "invalid format",
			terraformId: "bad-id",
			expectErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractSubjectNameFromTfId(tt.terraformId)
			if (err != nil) != tt.expectErr {
				t.Fatalf("extractSubjectNameFromTfId(%q) error = %v, expectErr %v", tt.terraformId, err, tt.expectErr)
			}
			if err == nil && result != tt.expected {
				t.Errorf("got %q; want %q", result, tt.expected)
			}
		})
	}
}

func TestIsLatestSchema(t *testing.T) {
	tests := []struct {
		name             string
		schemaIdentifier string
		expected         bool
	}{
		{name: "latest", schemaIdentifier: "latest", expected: true},
		{name: "numeric", schemaIdentifier: "100042", expected: false},
		{name: "empty", schemaIdentifier: "", expected: false},
		{name: "LATEST uppercase", schemaIdentifier: "LATEST", expected: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLatestSchema(tt.schemaIdentifier)
			if result != tt.expected {
				t.Errorf("isLatestSchema(%q) = %v; want %v", tt.schemaIdentifier, result, tt.expected)
			}
		})
	}
}

func TestFindSchemaById(t *testing.T) {
	schema1 := schemaregistryv1.Schema{}
	schema1.SetId(100)
	schema1.SetSubject("my-subject")

	schema2 := schemaregistryv1.Schema{}
	schema2.SetId(200)
	schema2.SetSubject("other-subject")

	schema3 := schemaregistryv1.Schema{}
	schema3.SetId(100)
	schema3.SetSubject("other-subject")

	schemas := []schemaregistryv1.Schema{schema1, schema2, schema3}

	tests := []struct {
		name             string
		schemaIdentifier string
		subjectName      string
		expectFound      bool
		expectedId       int32
	}{
		{
			name:             "found by id and subject",
			schemaIdentifier: "100",
			subjectName:      "my-subject",
			expectFound:      true,
			expectedId:       100,
		},
		{
			name:             "same id different subject",
			schemaIdentifier: "100",
			subjectName:      "other-subject",
			expectFound:      true,
			expectedId:       100,
		},
		{
			name:             "not found - wrong id",
			schemaIdentifier: "999",
			subjectName:      "my-subject",
			expectFound:      false,
		},
		{
			name:             "not found - wrong subject",
			schemaIdentifier: "200",
			subjectName:      "my-subject",
			expectFound:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, found := findSchemaById(schemas, tt.schemaIdentifier, tt.subjectName)
			if found != tt.expectFound {
				t.Fatalf("findSchemaById() found = %v; want %v", found, tt.expectFound)
			}
			if found && result.GetId() != tt.expectedId {
				t.Errorf("findSchemaById() id = %d; want %d", result.GetId(), tt.expectedId)
			}
		})
	}
}

func TestBuildTfSchemaReferences(t *testing.T) {
	ref1 := schemaregistryv1.SchemaReference{}
	ref1.SetSubject("ref-subject-1")
	ref1.SetName("ref-name-1")
	ref1.SetVersion(1)

	ref2 := schemaregistryv1.SchemaReference{}
	ref2.SetSubject("ref-subject-2")
	ref2.SetName("ref-name-2")
	ref2.SetVersion(3)

	result := buildTfSchemaReferences([]schemaregistryv1.SchemaReference{ref1, ref2})
	if result == nil {
		t.Fatal("buildTfSchemaReferences returned nil")
	}
	if len(*result) != 2 {
		t.Fatalf("expected 2 references, got %d", len(*result))
	}
	if (*result)[0][paramSubjectName] != "ref-subject-1" {
		t.Errorf("first reference subject = %v; want ref-subject-1", (*result)[0][paramSubjectName])
	}
	if (*result)[1][paramName] != "ref-name-2" {
		t.Errorf("second reference name = %v; want ref-name-2", (*result)[1][paramName])
	}
	if (*result)[1][paramVersion] != int32(3) {
		t.Errorf("second reference version = %v; want 3", (*result)[1][paramVersion])
	}
}

func TestInferTypeFromString(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected interface{}
	}{
		{name: "integer", value: "42", expected: int64(42)},
		{name: "negative integer", value: "-10", expected: int64(-10)},
		{name: "zero", value: "0", expected: int64(0)},
		{name: "one", value: "1", expected: int64(1)},
		{name: "large LSN number", value: "9223372036854775807", expected: int64(9223372036854775807)},
		{name: "float with dot", value: "3.14", expected: float64(3.14)},
		{name: "float with exponent", value: "1e10", expected: float64(1e10)},
		{name: "boolean true", value: "true", expected: true},
		{name: "boolean false", value: "false", expected: false},
		{name: "plain string", value: "hello", expected: "hello"},
		{name: "empty string", value: "", expected: ""},
		{name: "string with spaces", value: "hello world", expected: "hello world"},
		{name: "uppercase TRUE is parsed as bool", value: "TRUE", expected: true},
		{name: "uppercase FALSE is parsed as bool", value: "FALSE", expected: false},
		{name: "mixed case True is parsed as bool", value: "True", expected: true},
		{name: "1.0 is float not int", value: "1.0", expected: float64(1.0)},
		{name: "negative float", value: "-3.14", expected: float64(-3.14)},
		{name: "scientific notation uppercase E", value: "2.5E3", expected: float64(2500)},
		{name: "integer overflow treated as string", value: "99999999999999999999", expected: "99999999999999999999"},
		{name: "leading zeros still parsed as int", value: "007", expected: int64(7)},
		{name: "string that looks numeric but has trailing space", value: "42 ", expected: "42 "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferTypeFromString(tt.value)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("inferTypeFromString(%q) = %v (%T); want %v (%T)", tt.value, result, result, tt.expected, tt.expected)
			}
		})
	}
}

func TestExtractNonsensitiveConfigs(t *testing.T) {
	tests := []struct {
		name     string
		configs  map[string]string
		expected map[string]string
	}{
		{
			name: "filters out sensitive configs (stars)",
			configs: map[string]string{
				"kafka.api.key":    "**********",
				"kafka.api.secret": "***",
				"topics":           "my-topic",
			},
			expected: map[string]string{
				"topics": "my-topic",
			},
		},
		{
			name: "filters out internal configs",
			configs: map[string]string{
				"config.internal.something": "value",
				"cloud.environment":         "prod",
				"connector.class":           "io.confluent.connect.s3.S3SinkConnector",
			},
			expected: map[string]string{
				"connector.class": "io.confluent.connect.s3.S3SinkConnector",
			},
		},
		{
			name: "single star is not sensitive",
			configs: map[string]string{
				"single.star": "*",
				"normal":      "value",
			},
			expected: map[string]string{
				"single.star": "*",
				"normal":      "value",
			},
		},
		{
			name:     "empty map",
			configs:  map[string]string{},
			expected: map[string]string{},
		},
		{
			name: "filters all known ignoredConnectorConfigs",
			configs: map[string]string{
				"cloud.environment":                      "prod",
				"cloud.provider":                         "aws",
				"connector.crn":                          "crn://confluent.cloud/...",
				"kafka.endpoint":                         "SASL://pkc-12345.us-east-1.aws.confluent.cloud:9092",
				"kafka.max.partition.validation.disable": "false",
				"kafka.region":                           "us-east-1",
				"connector.class":                        "io.confluent.connect.s3.S3SinkConnector",
			},
			expected: map[string]string{
				"connector.class": "io.confluent.connect.s3.S3SinkConnector",
			},
		},
		{
			name: "exactly two stars is sensitive",
			configs: map[string]string{
				"password": "**",
				"name":     "test",
			},
			expected: map[string]string{
				"name": "test",
			},
		},
		{
			name: "mixed sensitive, internal, and normal configs",
			configs: map[string]string{
				"kafka.api.key":          "***",
				"config.internal.offset": "100",
				"cloud.environment":      "dev",
				"topics":                 "orders",
				"output.data.format":     "JSON",
			},
			expected: map[string]string{
				"topics":             "orders",
				"output.data.format": "JSON",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractNonsensitiveConfigs(tt.configs)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("extractNonsensitiveConfigs() = %v; want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractRequiredStringValueFromMap(t *testing.T) {
	config := map[string]string{
		"name":            "my-connector",
		"connector.class": "io.confluent.connect.s3.S3SinkConnector",
	}

	tests := []struct {
		name       string
		key        string
		configName string
		expected   string
		expectErr  bool
	}{
		{
			name:       "existing key",
			key:        "name",
			configName: "connector config",
			expected:   "my-connector",
		},
		{
			name:       "another existing key",
			key:        "connector.class",
			configName: "connector config",
			expected:   "io.confluent.connect.s3.S3SinkConnector",
		},
		{
			name:       "missing key",
			key:        "nonexistent",
			configName: "connector config",
			expectErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractRequiredStringValueFromMap(config, tt.key, tt.configName)
			if (err != nil) != tt.expectErr {
				t.Fatalf("extractRequiredStringValueFromMap(%q) error = %v, expectErr %v", tt.key, err, tt.expectErr)
			}
			if err == nil && result != tt.expected {
				t.Errorf("got %q; want %q", result, tt.expected)
			}
		})
	}
}
