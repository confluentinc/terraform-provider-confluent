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
			paramGateway:                                gatewayDataSourceSchema(),
			paramAwsEgressPrivateLinkEndpoint:           awsEgressPrivateLinkEndpointDataSourceSchema(),
			paramAzureEgressPrivateLinkEndpoint:         azureEgressPrivateLinkEndpointDataSourceSchema(),
			paramGcpEgressPrivateServiceConnectEndpoint: gcpEgressPrivateServiceConnectEndpointDataSourceSchema(),
			paramAwsPrivateNetworkInterface:             awsPrivateNetworkInterfaceDataSourceSchema(),
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

func azureEgressPrivateLinkEndpointDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramPrivateLinkServiceResourceId: {
					Type:     schema.TypeString,
					Computed: true,
				},
				paramPrivateLinkSubresourceName: {
					Type:     schema.TypeString,
					Computed: true,
				},
				paramPrivateEndpointResourceId: {
					Type:     schema.TypeString,
					Computed: true,
				},
				paramPrivateEndpointDomain: {
					Type:     schema.TypeString,
					Computed: true,
				},
				paramPrivateEndpointIpAddress: {
					Type:     schema.TypeString,
					Computed: true,
				},
				paramPrivateEndpointCustomDnsConfigDomains: {
					Type:     schema.TypeList,
					Computed: true,
					Elem:     &schema.Schema{Type: schema.TypeString},
				},
			},
		},
		Computed: true,
	}
}

func gcpEgressPrivateServiceConnectEndpointDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramPrivateServiceConnectEndpointTarget: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: `URI of the service attachment for the published service that the Private Service Connect Endpoint connects to, or "ALL_GOOGLE_APIS" or "all-google-apis" for global Google APIs`,
				},
				paramPrivateServiceConnectEndpointConnectionId: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Connection ID of the Private Service Connect Endpoint that is connected to the endpoint target.",
				},
				paramPrivateServiceConnectEndpointIpAddress: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "IP address of the Private Service Connect Endpoint that is connected to the endpoint target.",
				},
				paramPrivateServiceConnectEndpointName: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Name of the Private Service Connect Endpoint that is connected to the endpoint target.",
				},
			},
		},
	}
}

func awsPrivateNetworkInterfaceDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramNetworkInterfaces: {
					Type:     schema.TypeSet,
					Computed: true,
					Elem:     &schema.Schema{Type: schema.TypeString},
				},
				paramAccount: {
					Type:     schema.TypeString,
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
