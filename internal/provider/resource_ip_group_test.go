// Copyright 2023 Confluent Inc. All Rights Reserved.
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

import "fmt"

const (
	ipGroupResourceScenarioName = "confluent_ip_group Resource Lifecycle"

	createIpGroupUrlPath      = "/iam/v2/ip-groups"
	readCreatedIpGroupUrlPath = "/iam/v2/ip-groups/ipg-12345"
)

func testAccReourceIpGroupConfig(mockServerUrl, resourceLabel, groupName, cidrBlock string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_ip_group" "%s" {
		group_name = "%s"
		cidr_blocks = ["%s"]
	}
	`, mockServerUrl, resourceLabel, groupName, cidrBlock)
}
