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

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	paramFilterName      = "filter_name"
	paramResourceGroup   = "resource_group"
	paramResourceScope   = "resource_scope"
	paramOperationGroups = "operation_groups"
	paramIpGroupIds      = "ip_group_ids"
)

func ipFilterResource() *schema.Resource {
	return &schema.Resource{
		Create: ipFilterResourceCreate,
		Read:   ipFilterResourceRead,
		Update: ipFilterResourceUpdate,
		Delete: ipFilterResourceDelete,

		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the IP Filter.",
			},
			paramFilterName: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "A human readable name for an IP Filter.",
			},
			paramResourceGroup: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Scope of resources covered by this IP Filter. Available resource groups include 'management' and 'multiple'.",
			},
			paramResourceScope: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A CRN that specifies the scope of the ip filter, specifically the organization or environment. Without specifying this property, the ip filter would apply to the whole organization.",
			},
			paramOperationGroups: {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
				Description: "Scope of resources covered by this IP Filter. Resource group must be set to 'multiple' in order to use this property.",
			},
			paramIpGroupIds: {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Required:    true,
				Description: "A set of IP Group IDs to add to the IP Filter",
			},
		},
	}
}

func ipFilterResourceCreate(d *schema.ResourceData, m interface{}) error {
	// TODO: Implement create logic
	return ipFilterResourceRead(d, m)
}

func ipFilterResourceRead(d *schema.ResourceData, m interface{}) error {
	// TODO: Implement read logic
	return nil
}

func ipFilterResourceUpdate(d *schema.ResourceData, m interface{}) error {
	// TODO: Implement update logic
	return ipFilterResourceRead(d, m)
}

func ipFilterResourceDelete(d *schema.ResourceData, m interface{}) error {
	// TODO: Implement delete logic
	return nil
}
