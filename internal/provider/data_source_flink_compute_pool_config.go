// Copyright 2025 Confluent Inc. All Rights Reserved.
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

func computePoolConfigDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: computePoolConfigDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID Compute Pool Config.",
			},
			paramDefaultPoolEnabled: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether default compute pools are enabled for the organization.",
			},
			paramMaxCFU: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Maximum number of Confluent Flink Units (CFU).",
			},
			paramApiVersion: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramKind: {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func computePoolConfigDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if err := dataSourceCredentialBlockValidationWithOAuth(d, meta.(*Client).isOAuthEnabled); err != nil {
		return diag.Errorf("error reading Tag: %s", createDescriptiveError(err))
	}
	c := meta.(*Client)

	req := c.fcpmClient.OrgComputePoolConfigsFcpmV2Api.GetFcpmV2OrgComputePoolConfig(c.fcpmApiContext(ctx))
	computePoolConfig, resp, err := req.Execute()
	if err != nil {
		return diag.Errorf("error reading Compute Pool Config %q: %s", map[string]interface{}{computePoolLoggingConfigKey: d.Id()}, createDescriptiveError(err, resp))
	}
	computePoolConfigJson, err := json.Marshal(computePoolConfig)
	if err != nil {
		return diag.Errorf("error reading Compute Pool Config %q: error marshaling %#v to json: %s", map[string]interface{}{computePoolLoggingConfigKey: d.Id()}, computePoolConfig, createDescriptiveError(err))
	}
	d.SetId(computePoolConfig.GetOrganizationId())
	tflog.Debug(ctx, fmt.Sprintf("Fetched Compute Pool Config %q: %s", map[string]interface{}{computePoolLoggingConfigKey: d.Id()}, computePoolConfigJson))
	if _, err := setComputePoolConfigAttributes(d, computePoolConfig); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}
