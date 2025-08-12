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
	ccpm "github.com/confluentinc/ccloud-sdk-go-v2/ccpm/v1"
	dns "github.com/confluentinc/ccloud-sdk-go-v2/networking-dnsforwarder/v1"
	sr "github.com/confluentinc/ccloud-sdk-go-v2/schema-registry/v1"
	"reflect"
	"testing"

	apikeys "github.com/confluentinc/ccloud-sdk-go-v2/apikeys/v2"
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
		apiKey   apikeys.IamV2ApiKey
		expected bool
	}{
		{
			name: "SR API Key with api_version=srcm/v3",
			apiKey: apikeys.IamV2ApiKey{
				Spec: &apikeys.IamV2ApiKeySpec{
					Resource: &apikeys.ObjectReference{
						Kind:       apikeys.PtrString(schemaRegistryKind),
						ApiVersion: apikeys.PtrString(srcmV3ApiVersion),
					},
				},
			},
			expected: true,
		},
		{
			name: "SR API Key with api_version=srcm/v2",
			apiKey: apikeys.IamV2ApiKey{
				Spec: &apikeys.IamV2ApiKeySpec{
					Resource: &apikeys.ObjectReference{
						Kind:       apikeys.PtrString(schemaRegistryKind),
						ApiVersion: apikeys.PtrString(srcmV2ApiVersion),
					},
				},
			},
			expected: true,
		},
		{
			name: "Kafka API Key",
			apiKey: apikeys.IamV2ApiKey{
				Spec: &apikeys.IamV2ApiKeySpec{
					Resource: &apikeys.ObjectReference{
						Kind:       apikeys.PtrString(schemaRegistryKind),
						ApiVersion: apikeys.PtrString(cmkApiVersion),
					},
				},
			},
			expected: false,
		},
		{
			name: "Cloud API Key",
			apiKey: apikeys.IamV2ApiKey{
				Spec: &apikeys.IamV2ApiKeySpec{
					Resource: &apikeys.ObjectReference{
						Kind:       apikeys.PtrString("Cloud"),
						ApiVersion: apikeys.PtrString(iamApiVersion),
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
		map1Expected := map[string]dns.NetworkingV1ForwardViaGcpDnsZonesDomainMappings{
			"example": {Zone: dns.PtrString("zone1"), Project: dns.PtrString("project1")},
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
		map1Expected := map[string]dns.NetworkingV1ForwardViaGcpDnsZonesDomainMappings{
			"example": {Zone: dns.PtrString("zone1"), Project: dns.PtrString("project1")},
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
		map1Expected := map[string]dns.NetworkingV1ForwardViaGcpDnsZonesDomainMappings{
			"example": {Zone: dns.PtrString("zone1"), Project: dns.PtrString("project1")},
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
		map1Expected := map[string]dns.NetworkingV1ForwardViaGcpDnsZonesDomainMappings{
			"example": {Zone: dns.PtrString("zone1"), Project: dns.PtrString("project1")},
		}
		actual, _ := convertToStringObjectMap(map1)

		if reflect.DeepEqual(actual, map1Expected) {
			t.Fatalf("Unexpected error: expected %v, got %v", map1Expected, actual)
		}
	})
}

func TestBuildTfRules(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		domainRules := []sr.Rule{
			{
				Name:     sr.PtrString("ABC"),
				Disabled: sr.PtrBool(false),
				Doc:      sr.PtrString("Doc"),
				Expr:     sr.PtrString("EXPR"),
				Kind:     sr.PtrString("TRANSFORM"),
				Mode:     sr.PtrString("WRITEREAD"),
				Type:     sr.PtrString("ENCRYPT"),
				Tags: &[]string{
					"PII",
				},
				Params: &map[string]string{
					"encrypt.kek.name": "testkek2",
				},
				OnSuccess: sr.PtrString("NONE,NONE"),
				OnFailure: sr.PtrString("ERROR,ERROR"),
			},
		}
		migrationRules := []sr.Rule{
			{
				Name:     sr.PtrString("ABC"),
				Disabled: sr.PtrBool(false),
				Doc:      sr.PtrString("Doc"),
				Expr:     sr.PtrString("EXPR"),
				Kind:     sr.PtrString("TRANSFORM"),
				Mode:     sr.PtrString("WRITEREAD"),
				Type:     sr.PtrString("ENCRYPT"),
				Tags: &[]string{
					"PIIM",
				},
				Params: &map[string]string{
					"encrypt.kek.name": "testkekM",
				},
				OnSuccess: sr.PtrString("NONE,NONE"),
				OnFailure: sr.PtrString("ERROR,ERROR"),
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
		actual := buildTfRules(domainRules, migrationRules)

		if !reflect.DeepEqual(*actual, tfRuleSet) {
			t.Fatalf("Unexpected error: expected %v, got %v", tfRuleSet, *actual)
		}
	})

	t.Run("success, incomplete set", func(t *testing.T) {
		domainRules := []sr.Rule{
			{
				Name:     sr.PtrString("ABC"),
				Disabled: sr.PtrBool(false),
				Expr:     sr.PtrString("EXPR"),
				Kind:     sr.PtrString("TRANSFORM"),
				Mode:     sr.PtrString("WRITEREAD"),
				Type:     sr.PtrString("ENCRYPT"),
				Tags: &[]string{
					"PII",
				},
				Params: &map[string]string{
					"encrypt.kek.name": "testkek2",
				},
				OnSuccess: sr.PtrString("NONE,NONE"),
				OnFailure: sr.PtrString("ERROR,ERROR"),
			},
		}
		migrationRules := []sr.Rule{
			{
				Name:     sr.PtrString("ABC"),
				Disabled: sr.PtrBool(false),
				Expr:     sr.PtrString("EXPR"),
				Kind:     sr.PtrString("TRANSFORM"),
				Mode:     sr.PtrString("WRITEREAD"),
				Type:     sr.PtrString("ENCRYPT"),
				Tags: &[]string{
					"PIIM",
				},
				Params: &map[string]string{
					"encrypt.kek.name": "testkekM",
				},
				OnSuccess: sr.PtrString("NONE,NONE"),
				OnFailure: sr.PtrString("ERROR,ERROR"),
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
		actual := buildTfRules(domainRules, migrationRules)

		if !reflect.DeepEqual(*actual, tfRuleSet) {
			t.Fatalf("Unexpected error: expected %v, got %v", tfRuleSet, *actual)
		}
	})

	t.Run("success, without migration rules", func(t *testing.T) {
		domainRules := []sr.Rule{
			{
				Name:     sr.PtrString("ABC"),
				Disabled: sr.PtrBool(false),
				Expr:     sr.PtrString("EXPR"),
				Kind:     sr.PtrString("TRANSFORM"),
				Mode:     sr.PtrString("WRITEREAD"),
				Type:     sr.PtrString("ENCRYPT"),
				Tags: &[]string{
					"PII",
				},
				Params: &map[string]string{
					"encrypt.kek.name": "testkek2",
				},
				OnSuccess: sr.PtrString("NONE,NONE"),
				OnFailure: sr.PtrString("ERROR,ERROR"),
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
		actual := buildTfRules(domainRules, []sr.Rule{})

		if !reflect.DeepEqual(*actual, tfRuleSet) {
			t.Fatalf("Unexpected error: expected %v, got %v", tfRuleSet, *actual)
		}
	})

	t.Run("success, without domain rules", func(t *testing.T) {
		migrationRules := []sr.Rule{
			{
				Name:     sr.PtrString("ABC"),
				Disabled: sr.PtrBool(false),
				Expr:     sr.PtrString("EXPR"),
				Kind:     sr.PtrString("TRANSFORM"),
				Mode:     sr.PtrString("WRITEREAD"),
				Type:     sr.PtrString("ENCRYPT"),
				Tags: &[]string{
					"PII",
				},
				Params: &map[string]string{
					"encrypt.kek.name": "testkek2",
				},
				OnSuccess: sr.PtrString("NONE,NONE"),
				OnFailure: sr.PtrString("ERROR,ERROR"),
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
		actual := buildTfRules([]sr.Rule{}, migrationRules)

		if !reflect.DeepEqual(*actual, tfRuleSet) {
			t.Fatalf("Unexpected error: expected %v, got %v", tfRuleSet, *actual)
		}
	})

	t.Run("fail, inconsistent domain rules", func(t *testing.T) {
		migrationRules := []sr.Rule{
			{
				Name:     sr.PtrString("ABC"),
				Disabled: sr.PtrBool(false),
				Expr:     sr.PtrString("EXPR"),
				Kind:     sr.PtrString("TRANSFORM"),
				Mode:     sr.PtrString("WRITEREAD"),
				Type:     sr.PtrString("ENCRYPT"),
				Tags: &[]string{
					"PII",
				},
				Params: &map[string]string{
					"encrypt.kek.name": "testkek2",
				},
				OnSuccess: sr.PtrString("NONE,NONE"),
				OnFailure: sr.PtrString("ERROR,ERROR"),
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
		actual := buildTfRules(migrationRules, migrationRules)

		if reflect.DeepEqual(*actual, tfRuleSet) {
			t.Fatalf("Unexpected error: expected %v, got %v", tfRuleSet, *actual)
		}
	})
}

func TestBuildTfConnectorClasses(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		classes := []ccpm.CcpmV1ConnectorClass{
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
		classes := []ccpm.CcpmV1ConnectorClass{
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
		classes := []ccpm.CcpmV1ConnectorClass{
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
		classes := make([]ccpm.CcpmV1ConnectorClass, 2)

		class := ccpm.NewCcpmV1ConnectorClassWithDefaults()
		class.SetClassName("Class1")
		class.SetType("SOURCE")

		class2 := ccpm.NewCcpmV1ConnectorClassWithDefaults()
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
		classes := make([]ccpm.CcpmV1ConnectorClass, 2)

		class := ccpm.NewCcpmV1ConnectorClassWithDefaults()
		class.SetClassName("Class1")
		class.SetType("SOURCE")

		class2 := ccpm.NewCcpmV1ConnectorClassWithDefaults()
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
		classes := make([]ccpm.CcpmV1ConnectorClass, 2)

		class := ccpm.NewCcpmV1ConnectorClassWithDefaults()
		class.SetClassName("Class1")
		class.SetType("SOURCE")

		class2 := ccpm.NewCcpmV1ConnectorClassWithDefaults()
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
    tests := []struct{
        name string
        a    string
        b    string
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
