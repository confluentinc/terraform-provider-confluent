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
	ccpmv1 "github.com/confluentinc/ccloud-sdk-go-v2/ccpm/v1"
)

// The generated resource_custom_connector_plugin_version.go inlines both directions of the
// connector_class conversion, so these three are no longer called by the resource. They are kept
// because utils_test.go tests buildConnectorClass and buildTfConnectorClasses directly, and the
// generated file carries a DO NOT EDIT banner — a regeneration would wipe them if they lived
// there. Same companion-file pattern as resource_certificate_authority_helpers.go and
// resource_environment_helpers.go.

func buildConnectorClass(connectorClass []interface{}) []ccpmv1.CcpmV1ConnectorClass {
	classes := make([]ccpmv1.CcpmV1ConnectorClass, len(connectorClass))
	for index, tfClass := range connectorClass {
		class := ccpmv1.NewCcpmV1ConnectorClassWithDefaults()
		tfClassMap := tfClass.(map[string]interface{})
		if className, exists := tfClassMap[paramConnectorClassName].(string); exists {
			class.SetClassName(className)
		}
		if classType, exists := tfClassMap[paramConnectorType].(string); exists {
			class.SetType(classType)
		}
		classes[index] = *class
	}
	return classes
}

func buildTfConnectorClasses(classes []ccpmv1.CcpmV1ConnectorClass) *[]map[string]interface{} {
	tfClasses := make([]map[string]interface{}, len(classes))
	for i, class := range classes {
		tfClasses[i] = *buildTfClasses(class)
	}
	return &tfClasses
}

func buildTfClasses(class ccpmv1.CcpmV1ConnectorClass) *map[string]interface{} {
	tfClass := make(map[string]interface{})
	tfClass[paramConnectorClassName] = class.GetClassName()
	tfClass[paramConnectorType] = class.GetType()
	return &tfClass
}
