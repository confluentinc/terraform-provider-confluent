// Copyright 2024 Confluent Inc. All Rights Reserved.
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
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ipFilterDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: ipFilterDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:     schema.TypeString,
				Required: true,
			},
			paramFilterName: {
				Type:        schema.TypeString,
				Description: "A human readable name for an IP Filter.",
				Computed:    true,
			},
			paramResourceGroup: {
				Type:        schema.TypeString,
				Description: "Scope of resources covered by this IP filter.",
				Computed:    true,
			},
			paramResourceScope: {
				Type:        schema.TypeString,
				Description: "A CRN that specifies the scope of the ip filter, specifically the organization or environment.",
				Computed:    true,
			},
			paramOperationGroups: {
				Type:        schema.TypeSet,
				Description: "Scope of resources covered by this IP filter.",
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			paramIPGroups: {
				Type:        schema.TypeSet,
				Description: "A list of IP Filters.",
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func ipFilterDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ipFilterID := d.Get(paramId).(string)

	tflog.Debug(ctx, fmt.Sprintf("Reading IP Filter %q=%q", paramId, ipFilterID), map[string]interface{}{ipFilterLoggingKey: ipFilterID})

	c := meta.(*Client)
	request := c.iamIPClient.IPFiltersIamV2Api.GetIamV2IpFilter(c.iamIPApiContext(ctx), ipFilterID)
	ipFilter, _, err := c.iamIPClient.IPFiltersIamV2Api.GetIamV2IpFilterExecute(request)
	if err != nil {
		return diag.Errorf("error reading IP Filter %q: %s", ipFilterID, createDescriptiveError(err))
	}
	ipFilterJson, err := json.Marshal(ipFilter)
	if err != nil {
		return diag.Errorf("error reading IP Filter %q: error marshaling %#v to json: %s", ipFilterID, ipFilter, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched IP Filter %q: %s", ipFilterID, ipFilterJson), map[string]interface{}{ipFilterLoggingKey: ipFilterID})

	if _, err := setIPFilterAttributes(d, ipFilter); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}
