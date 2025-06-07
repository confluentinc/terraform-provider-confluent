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
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceIPGroup() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceIpGroupRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the IP group.",
			},
			paramGroupName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A human readable name for an IP Group.",
			},
			paramCIDRBlocks: {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
				Description: "A set of CIDR blocks to include in the IP group.",
			},
		},
	}
}

func dataSourceIpGroupRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Client)

	id := d.Get(paramId).(string)

	req := c.ipFilteringClient.IPGroupsIamV2Api.GetIamV2IpGroup(ctx, id)
	ipGroup, _, err := req.Execute()

	if err != nil {
		return diag.Errorf("error reading IP group %q: %s", id, createDescriptiveError(err))
	}

	if err := d.Set(paramGroupName, ipGroup.GetGroupName()); err != nil {
		return diag.Errorf("error reading IP group %q: %s", id, createDescriptiveError(err))
	}

	if err := d.Set(paramCIDRBlocks, ipGroup.GetCidrBlocks()); err != nil {
		return diag.Errorf("error reading IP group %q: %s", id, createDescriptiveError(err))
	}

	d.SetId(ipGroup.GetId())

	return nil
}
