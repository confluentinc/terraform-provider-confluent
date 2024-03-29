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

func accessPointDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: accessPointDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the Access Point, for example, `ap-abc123`.",
			},
			paramEnvironment: environmentDataSourceSchema(),
			paramDisplayName: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramGateway:                      gatewayDataSourceSchema(),
			paramAwsEgressPrivateLinkEndpoint: awsEgressPrivateLinkEndpointDataSourceSchema(),
		},
	}
}

func awsEgressPrivateLinkEndpointDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramVpcEndpointServiceName: {
					Type:     schema.TypeString,
					Computed: true,
				},
				paramVpcEndpointId: {
					Type:     schema.TypeString,
					Computed: true,
				},
				paramVpcEndpointDnsName: {
					Type:     schema.TypeString,
					Computed: true,
				},
				paramEnableHighAvailability: {
					Type:     schema.TypeBool,
					Computed: true,
				},
			},
		},
		Computed: true,
	}
}

func accessPointDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	accessPointId := d.Get(paramId).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	tflog.Debug(ctx, fmt.Sprintf("Reading Access Point %q=%q", paramId, accessPointId), map[string]interface{}{accessPointKey: accessPointId})

	c := meta.(*Client)
	request := c.netAccessPointClient.AccessPointsNetworkingV1Api.GetNetworkingV1AccessPoint(c.netAPApiContext(ctx), accessPointId).Environment(environmentId)
	accessPoint, _, err := c.netAccessPointClient.AccessPointsNetworkingV1Api.GetNetworkingV1AccessPointExecute(request)
	if err != nil {
		return diag.Errorf("error reading Access Point %q: %s", accessPointId, createDescriptiveError(err))
	}
	accessPointJson, err := json.Marshal(accessPoint)
	if err != nil {
		return diag.Errorf("error reading Access Point %q: error marshaling %#v to json: %s", accessPointId, accessPoint, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Access Point %q: %s", accessPointId, accessPointJson), map[string]interface{}{accessPointKey: accessPointId})

	if _, err := setAccessPointAttributes(d, accessPoint); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}
