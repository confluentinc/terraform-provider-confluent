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
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func switchoverPairDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: switchoverPairDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the switchover pair (e.g. `sw-abc123`).",
			},
			paramDisplayName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A human-readable name for the switchover pair.",
			},
			paramMembers: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The two clusters participating in this switchover pair.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						paramName: {
							Type:     schema.TypeString,
							Computed: true,
						},
						paramMemberId: {
							Type:     schema.TypeString,
							Computed: true,
						},
						paramEnvId: {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			paramActiveMember: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the member that is currently active.",
			},
			paramFailoverType: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The failover semantics most recently applied to this pair.",
			},
			paramPhase: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The lifecycle phase of the switchover pair.",
			},
			paramEnvironment: environmentDataSourceSchema(),
		},
	}
}

func switchoverPairDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)
	switchoverPairId := d.Get(paramId).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	tflog.Debug(ctx, fmt.Sprintf("Reading switchover pair data source %q", switchoverPairId), map[string]interface{}{switchoverPairLoggingKey: switchoverPairId})

	req := c.switchoverV1Client.SwitchoverPairsSwitchoverV1Api.GetSwitchoverV1SwitchoverPair(c.switchoverV1ApiContext(ctx), switchoverPairId).Environment(environmentId)
	pair, resp, err := req.Execute()
	if err != nil {
		return diag.Errorf("error reading switchover pair data source %q: %s", switchoverPairId, createDescriptiveError(err, resp))
	}

	if _, err := setSwitchoverPairAttributes(d, pair, environmentId); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading switchover pair data source %q", switchoverPairId), map[string]interface{}{switchoverPairLoggingKey: switchoverPairId})
	return nil
}
