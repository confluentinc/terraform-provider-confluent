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

	net "github.com/confluentinc/ccloud-sdk-go-v2/networking/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	paramAwsPeeringGateway             = "aws_peering_gateway"
	paramAwsEgressPrivateLinkGateway   = "aws_egress_private_link_gateway"
	paramAzureEgressPrivateLinkGateway = "azure_egress_private_link_gateway"
	paramAzurePeeringGateway           = "azure_peering_gateway"
	paramPrincipalArn                  = "principal_arn"
)

func gatewayDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: gatewayDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the Gateway, for example, `gw-abc123`.",
			},
			paramEnvironment: environmentDataSourceSchema(),
			paramDisplayName: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramAwsEgressPrivateLinkGateway:   awsEgressPrivateLinkGatewayDataSourceSchema(),
			paramAwsPeeringGateway:             awsPeeringGatewaySpecDataSourceSchema(),
			paramAzureEgressPrivateLinkGateway: azureEgressPrivateLinkGatewayDataSourceSchema(),
			paramAzurePeeringGateway:           azurePeeringGatewaySpecDataSourceSchema(),
		},
	}
}

func awsPeeringGatewaySpecDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramRegion: {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
		Computed: true,
	}
}

func awsEgressPrivateLinkGatewayDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramRegion: {
					Type:     schema.TypeString,
					Computed: true,
				},
				paramPrincipalArn: {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
		Computed: true,
	}
}

func azurePeeringGatewaySpecDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramRegion: {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
		Computed: true,
	}
}

func azureEgressPrivateLinkGatewayDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramRegion: {
					Type:     schema.TypeString,
					Computed: true,
				},
				paramSubscription: {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
		Computed: true,
	}
}

func gatewayDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	gatewayId := d.Get(paramId).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	tflog.Debug(ctx, fmt.Sprintf("Reading Gateway %q=%q", paramId, gatewayId), map[string]interface{}{gatewayKey: gatewayId})

	c := meta.(*Client)
	request := c.netClient.GatewaysNetworkingV1Api.GetNetworkingV1Gateway(c.netApiContext(ctx), gatewayId).Environment(environmentId)
	gateway, _, err := c.netClient.GatewaysNetworkingV1Api.GetNetworkingV1GatewayExecute(request)
	if err != nil {
		return diag.Errorf("error reading Gateway %q: %s", gatewayId, createDescriptiveError(err))
	}
	gatewayJson, err := json.Marshal(gateway)
	if err != nil {
		return diag.Errorf("error reading Gateway %q: error marshaling %#v to json: %s", gatewayId, gateway, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Gateway %q: %s", gatewayId, gatewayJson), map[string]interface{}{gatewayKey: gatewayId})

	if _, err := setGatewayAttributes(d, gateway); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}

func setGatewayAttributes(d *schema.ResourceData, gateway net.NetworkingV1Gateway) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, gateway.Spec.GetDisplayName()); err != nil {
		return nil, err
	}

	if gateway.Spec.Config.NetworkingV1AwsEgressPrivateLinkGatewaySpec != nil && gateway.Status.CloudGateway.NetworkingV1AwsEgressPrivateLinkGatewayStatus != nil {
		if err := d.Set(paramAwsEgressPrivateLinkGateway, []interface{}{map[string]interface{}{
			paramRegion:       gateway.Spec.Config.NetworkingV1AwsEgressPrivateLinkGatewaySpec.GetRegion(),
			paramPrincipalArn: gateway.Status.CloudGateway.NetworkingV1AwsEgressPrivateLinkGatewayStatus.GetPrincipalArn(),
		}}); err != nil {
			return nil, err
		}
	} else if gateway.Spec.Config.NetworkingV1AwsPeeringGatewaySpec != nil {
		if err := d.Set(paramAwsPeeringGateway, []interface{}{map[string]interface{}{
			paramRegion: gateway.Spec.Config.NetworkingV1AwsPeeringGatewaySpec.GetRegion(),
		}}); err != nil {
			return nil, err
		}
	} else if gateway.Spec.Config.NetworkingV1AzureEgressPrivateLinkGatewaySpec != nil && gateway.Status.CloudGateway.NetworkingV1AzureEgressPrivateLinkGatewayStatus != nil {
		if err := d.Set(paramAzureEgressPrivateLinkGateway, []interface{}{map[string]interface{}{
			paramRegion:       gateway.Spec.Config.NetworkingV1AzureEgressPrivateLinkGatewaySpec.GetRegion(),
			paramSubscription: gateway.Status.CloudGateway.NetworkingV1AzureEgressPrivateLinkGatewayStatus.GetSubscription(),
		}}); err != nil {
			return nil, err
		}
	} else if gateway.Spec.Config.NetworkingV1AzurePeeringGatewaySpec != nil {
		if err := d.Set(paramAzurePeeringGateway, []interface{}{map[string]interface{}{
			paramRegion: gateway.Spec.Config.NetworkingV1AzurePeeringGatewaySpec.GetRegion(),
		}}); err != nil {
			return nil, err
		}
	}

	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, gateway.Spec.Environment.GetId(), d); err != nil {
		return nil, err
	}
	d.SetId(gateway.GetId())
	return d, nil
}
