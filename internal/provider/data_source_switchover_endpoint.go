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

func switchoverEndpointDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: switchoverEndpointDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the switchover endpoint (e.g. `se-abc123`).",
			},
			paramDisplayName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A human-readable name for the switchover endpoint.",
			},
			paramSwitchoverPairId: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the switchover pair this endpoint is bound to.",
			},
			paramTarget: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the endpoint that is currently active.",
			},
			paramEndpoints: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The endpoint definitions, one per side.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						paramName: {
							Type:     schema.TypeString,
							Computed: true,
						},
						paramHostname: {
							Type:     schema.TypeString,
							Computed: true,
						},
						paramEndpointFilter: {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									paramType: {
										Type:     schema.TypeString,
										Computed: true,
									},
									paramNetworkId: {
										Type:     schema.TypeString,
										Computed: true,
									},
									paramAccessPoint: {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
			paramPhase: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The lifecycle phase of the switchover endpoint.",
			},
			paramEnvironment: environmentDataSourceSchema(),
		},
	}
}

func switchoverEndpointDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)
	switchoverEndpointId := d.Get(paramId).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	tflog.Debug(ctx, fmt.Sprintf("Reading switchover endpoint data source %q", switchoverEndpointId), map[string]interface{}{switchoverEndpointLoggingKey: switchoverEndpointId})

	req := c.switchoverV1Client.SwitchoverEndpointsSwitchoverV1Api.GetSwitchoverV1SwitchoverEndpoint(c.switchoverV1ApiContext(ctx), switchoverEndpointId).Environment(environmentId)
	endpoint, resp, err := req.Execute()
	if err != nil {
		return diag.Errorf("error reading switchover endpoint data source %q: %s", switchoverEndpointId, createDescriptiveError(err, resp))
	}

	if _, err := setSwitchoverEndpointAttributes(d, endpoint, environmentId); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading switchover endpoint data source %q", switchoverEndpointId), map[string]interface{}{switchoverEndpointLoggingKey: switchoverEndpointId})
	return nil
}
