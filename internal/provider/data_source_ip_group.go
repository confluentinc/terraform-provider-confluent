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

func ipGroupDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: ipGroupDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:     schema.TypeString,
				Required: true,
			},
			paramGroupName: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramCidrBlocks: {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func ipGroupDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ipGroupID := d.Get(paramId).(string)

	tflog.Debug(ctx, fmt.Sprintf("Reading IP Group %q=%q", paramId, ipGroupID), map[string]interface{}{ipGroupLoggingKey: ipGroupID})

	c := meta.(*Client)
	request := c.iamIPClient.IPGroupsIamV2Api.GetIamV2IpGroup(c.iamIPApiContext(ctx), ipGroupID)
	ipGroup, resp, err := c.iamIPClient.IPGroupsIamV2Api.GetIamV2IpGroupExecute(request)
	if err != nil {
		return diag.Errorf("error reading IP Group %q: %s", ipGroupID, createDescriptiveError(err, resp))
	}
	ipGroupJson, err := json.Marshal(ipGroup)
	if err != nil {
		return diag.Errorf("error reading IP Group %q: error marshaling %#v to json: %s", ipGroupID, ipGroup, createDescriptiveError(err, resp))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched IP Group %q: %s", ipGroupID, ipGroupJson), map[string]interface{}{ipGroupLoggingKey: ipGroupID})

	if _, err := setIPGroupAttributes(d, ipGroup); err != nil {
		return diag.FromErr(createDescriptiveError(err, resp))
	}
	return nil
}
