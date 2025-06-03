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

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	iamipfiltering "github.com/confluentinc/ccloud-sdk-go-v2/iam-ip-filtering/v2"
)

const (
	paramGroupName  = "group_name"
	paramCIDRBlocks = "cidr_blocks"
)

// ipGroupResource returns the schema.Resource for confluent_ip_group.
func ipGroupResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceIPGroupCreate,
		ReadContext:   resourceIPGroupRead,
		UpdateContext: resourceIPGroupUpdate,
		DeleteContext: resourceIPGroupDelete,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the IP group.",
			},
			paramGroupName: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "A human readable name for an IP Group.",
			},
			paramCIDRBlocks: {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Required:    true,
				Description: "A set of CIDR blocks to include in the IP group.",
			},
		},
	}
}

func resourceIPGroupCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Client)

	groupName := d.Get(paramGroupName).(string)
	cidrBlocks := convertToStringSlice(d.Get(paramCIDRBlocks).(*schema.Set).List())

	ipGroup := iamipfiltering.NewIamV2IpGroup()
	ipGroup.GroupName = &groupName
	ipGroup.CidrBlocks = &cidrBlocks

	req := c.ipFilteringClient.IPGroupsIamV2Api.CreateIamV2IpGroup(ctx).IamV2IpGroup(*ipGroup)
	createdIpGroup, _, err := req.Execute()

	if err != nil {
		return diag.Errorf("error creating ip group %s", createDescriptiveError(err))
	}

	setIPGroupAttributes(d, createdIpGroup)

	return nil
}

func resourceIPGroupRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Client)

	id := d.Get(paramId).(string)

	req := c.ipFilteringClient.IPGroupsIamV2Api.GetIamV2IpGroup(ctx, id)
	ipGroup, _, err := req.Execute()

	if err != nil {
		return diag.Errorf("error reading ip group %s", createDescriptiveError(err))
	}

	setIPGroupAttributes(d, ipGroup)

	return nil
}

func resourceIPGroupUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Client)

	id := d.Get(paramId).(string)
	groupName := d.Get(paramGroupName).(string)
	cidrBlocks := convertToStringSlice(d.Get(paramCIDRBlocks).(*schema.Set).List())

	ipGroup := iamipfiltering.NewIamV2IpGroup()
	ipGroup.GroupName = &groupName
	ipGroup.CidrBlocks = &cidrBlocks

	req := c.ipFilteringClient.IPGroupsIamV2Api.UpdateIamV2IpGroup(ctx, id).IamV2IpGroup(*ipGroup)
	updatedIpGroup, _, err := req.Execute()

	if err != nil {
		return diag.Errorf("error updating ip group %s", createDescriptiveError(err))
	}

	setIPGroupAttributes(d, updatedIpGroup)

	return nil
}

func resourceIPGroupDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Client)

	id := d.Get(paramId).(string)

	req := c.ipFilteringClient.IPGroupsIamV2Api.DeleteIamV2IpGroup(ctx, id)
	_, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting ip group %s", createDescriptiveError(err))
	}

	return nil
}

func setIPGroupAttributes(d *schema.ResourceData, ipGroup iamipfiltering.IamV2IpGroup) (*schema.ResourceData, error) {
	if err := d.Set(paramGroupName, ipGroup.GetGroupName()); err != nil {
		return nil, err
	}

	if err := d.Set(paramCIDRBlocks, ipGroup.GetCidrBlocks()); err != nil {
		return nil, err
	}

	d.SetId(ipGroup.GetId())

	return d, nil
}
