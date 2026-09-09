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
	"regexp"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// requiredNetworkSchema is the shared Required + ForceNew `network { id }` block used by the
// networking resources that reference a Confluent-managed network — private_link_access,
// network_link_service, network_link_endpoint, peering, transit_gateway_attachment. It was
// historically defined in the hand-written resource_private_link_access.go; it moved here when that
// resource was migrated to cli-terraform-generator output, which references the helper via
// terraform.objectref_schema_overrides rather than emitting its own near-duplicate.
func requiredNetworkSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		MinItems:    1,
		MaxItems:    1,
		Required:    true,
		ForceNew:    true,
		Description: "Network represents a network (VPC) in Confluent Cloud. All Networks exist within Confluent-managed cloud provider accounts.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:         schema.TypeString,
					Required:     true,
					ForceNew:     true,
					Description:  "The unique identifier for the network.",
					ValidateFunc: validation.StringMatch(regexp.MustCompile("^(n-|nr-)"), "the network ID must start with 'n-' or 'nr-'"),
				},
			},
		},
	}
}
