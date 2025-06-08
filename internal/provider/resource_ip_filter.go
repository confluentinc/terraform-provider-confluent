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

	iamipfiltering "github.com/confluentinc/ccloud-sdk-go-v2/iam-ip-filtering/v2"
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
		CreateContext: ipFilterResourceCreate,
		ReadContext:   ipFilterResourceRead,
		UpdateContext: ipFilterResourceUpdate,
		DeleteContext: ipFilterResourceDelete,

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

func ipFilterResourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Client)

	filterName := d.Get(paramFilterName).(string)
	resourceGroup := d.Get(paramResourceGroup).(string)
	resourceScope := d.Get(paramResourceScope).(string)
	operationGroups := convertToStringSlice(d.Get(paramOperationGroups).(*schema.Set).List())
	ipGroupIds := convertToStringSlice(d.Get(paramIpGroupIds).(*schema.Set).List())

	ipGroupIdGlobalObjectReferences := make([]iamipfiltering.GlobalObjectReference, len(ipGroupIds))
	for i, v := range ipGroupIds {
		ipGroupIdGlobalObjectReferences[i] = iamipfiltering.GlobalObjectReference{Id: v}
	}

	ipFilter := iamipfiltering.NewIamV2IpFilter()
	ipFilter.FilterName = &filterName
	ipFilter.ResourceGroup = &resourceGroup
	ipFilter.ResourceScope = &resourceScope
	ipFilter.OperationGroups = &operationGroups
	ipFilter.IpGroups = &ipGroupIdGlobalObjectReferences

	req := c.ipFilteringClient.IPFiltersIamV2Api.CreateIamV2IpFilter(ctx).IamV2IpFilter(*ipFilter)
	createdIpFilter, _, err := req.Execute()

	if err != nil {
		return diag.Errorf("error creating IP Filter %s", createDescriptiveError(err))
	}

	d.SetId(createdIpFilter.GetId())

	return ipFilterResourceRead(ctx, d, m)
}

func ipFilterResourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Client)

	ipFilterId := d.Get(paramId).(string)

	req := c.ipFilteringClient.IPFiltersIamV2Api.GetIamV2IpFilter(ctx, ipFilterId)
	ipFilter, _, err := req.Execute()

	if err != nil {
		return diag.Errorf("error reading IP Filter %s", createDescriptiveError(err))
	}

	if err := d.Set(paramFilterName, ipFilter.GetFilterName()); err != nil {
		return diag.Errorf("error reading IP Filter %s", createDescriptiveError(err))
	}
	if err := d.Set(paramResourceGroup, ipFilter.GetResourceGroup()); err != nil {
		return diag.Errorf("error reading IP Filter %s", createDescriptiveError(err))
	}
	if err := d.Set(paramResourceScope, ipFilter.GetResourceScope()); err != nil {
		return diag.Errorf("error reading IP Filter %s", createDescriptiveError(err))
	}
	if err := d.Set(paramOperationGroups, ipFilter.GetOperationGroups()); err != nil {
		return diag.Errorf("error reading IP Filter %s", createDescriptiveError(err))
	}
	//d.Set(paramIpGroupIds, ipFilter.GetIpGroups())

	return nil
}

func ipFilterResourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// TODO: Implement update logic
	return ipFilterResourceRead(ctx, d, m)
}

func ipFilterResourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// TODO: Implement delete logic
	return nil
}
