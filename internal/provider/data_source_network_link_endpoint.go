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
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	paramNetworkLinkService = "network_link_service"
)

func networkLinkEndpointDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: networkLinkEndpointDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the network link endpoint, for example, `nle-a1b2c`.",
			},
			paramDisplayName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The display name of the network link endpoint.",
			},
			paramDescription: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramEnvironment:        environmentDataSourceSchema(),
			paramNetwork:            networkDataSourceSchema(),
			paramNetworkLinkService: networkLinkServiceDataSourceSchema(),
			paramResourceName: {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func networkLinkServiceDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
		Computed: true,
	}
}

func networkLinkEndpointDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	nleId := d.Get(paramId).(string)
	if nleId == "" {
		return diag.Errorf("error reading network link endpoint: network link endpoint id is missing")
	}

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	if environmentId == "" {
		return diag.Errorf("error reading network link endpoint: environment Id is missing")
	}

	tflog.Debug(ctx, fmt.Sprintf("Reading network link endpoint %q=%q", paramId, nleId), map[string]interface{}{networkLinkEndpointLoggingKey: nleId})

	c := meta.(*Client)
	request := c.netClient.NetworkLinkEndpointsNetworkingV1Api.GetNetworkingV1NetworkLinkEndpoint(c.netApiContext(ctx), nleId).Environment(environmentId)
	nle, resp, err := c.netClient.NetworkLinkEndpointsNetworkingV1Api.GetNetworkingV1NetworkLinkEndpointExecute(request)
	if err != nil {
		return diag.Errorf("error reading network link endpoint %q: %s", nleId, createDescriptiveError(err, resp))
	}
	nleJson, err := json.Marshal(nle)
	if err != nil {
		return diag.Errorf("error reading network link endpoint %q: error marshaling %#v to json: %s", nleId, nle, createDescriptiveError(err, resp))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched network link endpoint %q: %s", nleId, nleJson), map[string]interface{}{networkLinkEndpointLoggingKey: nleId})

	if _, err := setNetworkLinkEndpointAttributes(d, nle); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}
