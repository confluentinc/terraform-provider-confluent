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
	paramEnvironments = "environments"
	paramNetworks     = "networks"
	paramAccept       = "accept"
)

func networkLinkServiceDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: networkLinkServiceDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the network link service, for example, `nls-a1b2c`.",
			},
			paramDisplayName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The display name of the network link service.",
			},
			paramDescription: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramEnvironment: environmentDataSourceSchema(),
			paramNetwork:     networkDataSourceSchema(),
			paramResourceName: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramAccept: acceptSchema(),
		},
	}
}

func networkLinkServiceDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	nlsId := d.Get(paramId).(string)
	if nlsId == "" {
		return diag.Errorf("error reading network link service: network link service id is missing")
	}

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	if environmentId == "" {
		return diag.Errorf("error reading network link service: environment Id is missing")
	}

	tflog.Debug(ctx, fmt.Sprintf("Reading network link service %q=%q", paramId, nlsId), map[string]interface{}{networkLinkServiceLoggingKey: nlsId})

	c := meta.(*Client)
	request := c.netClient.NetworkLinkServicesNetworkingV1Api.GetNetworkingV1NetworkLinkService(c.netApiContext(ctx), nlsId).Environment(environmentId)
	nls, _, err := c.netClient.NetworkLinkServicesNetworkingV1Api.GetNetworkingV1NetworkLinkServiceExecute(request)
	if err != nil {
		return diag.Errorf("error reading network link service %q: %s", nlsId, createDescriptiveError(err))
	}
	nlsJson, err := json.Marshal(nls)
	if err != nil {
		return diag.Errorf("error reading network link service %q: error marshaling %#v to json: %s", nlsId, nls, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched network link service %q: %s", nlsId, nlsJson), map[string]interface{}{networkLinkServiceLoggingKey: nlsId})

	if _, err := setNetworkLinkServiceAttributes(d, nls); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}
