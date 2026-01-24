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
	"fmt"
	"net/http"
	"strconv"
	"time"

	netgw "github.com/confluentinc/ccloud-sdk-go-v2/networking-gateway/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	paramGateways    = "gateways"
	paramGatewayType = "gateway_type"
	paramPhase       = "phase"
)

const (
	// The maximum allowable page size - 1 (to avoid off-by-one errors) when listing using Networking Gateway API
	// https://github.com/confluentinc/api/blob/master/networking-gateway/minispec.yaml#L443
	listGatewaysPageSize = 99
)

func gatewaysDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: gatewaysDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramEnvironment: environmentDataSourceSchema(),
			paramFilter: {
				MaxItems:    1,
				Optional:    true,
				Type:        schema.TypeList,
				Description: "Filter the results by exact match for gateway properties.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						paramGatewayType: {
							Type:        schema.TypeList,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Optional:    true,
							Description: "Filter the results by exact match for gateway_type. Pass multiple times to see results matching any of the values. Valid values are: `AwsEgressPrivateLink`, `AwsIngressPrivateLink`, `AwsPeering`, `AwsPrivateNetworkInterface`, `AzureEgressPrivateLink`, `AzurePeering`, `GcpEgressPrivateServiceConnect`, `GcpPeering`.",
						},
						paramId: {
							Type:        schema.TypeList,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Optional:    true,
							Description: "Filter the results by exact match for id. Pass multiple times to see results matching any of the values.",
						},
						paramRegion: {
							Type:        schema.TypeList,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Optional:    true,
							Description: "Filter the results by exact match for spec.config.region. Pass multiple times to see results matching any of the values.",
						},
						paramDisplayName: {
							Type:        schema.TypeList,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Optional:    true,
							Description: "Filter the results by exact match for spec.display_name. Pass multiple times to see results matching any of the values.",
						},
						paramPhase: {
							Type:        schema.TypeList,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Optional:    true,
							Description: "Filter the results by exact match for status.phase. Pass multiple times to see results matching any of the values. Valid values are: `CREATED`, `PROVISIONING`, `READY`, `FAILED`, `DEPROVISIONING`, `EXPIRED`.",
						},
					},
				},
			},
			paramGateways: gatewaysSchema(),
		},
	}
}

func gatewaysSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The ID of the Gateway, for example, `gw-abc123`.",
				},
				paramDisplayName: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "A name for the Gateway.",
				},
				paramAwsEgressPrivateLinkGateway:           awsEgressPrivateLinkGatewayDataSourceSchema(),
				paramAwsIngressPrivateLinkGateway:          awsIngressPrivateLinkGatewayDataSourceSchema(),
				paramAwsPeeringGateway:                     awsPeeringGatewaySpecDataSourceSchema(),
				paramAwsPrivateNetworkInterfaceGateway:     awsPrivateNetworkInterfaceGatewayDataSourceSchema(),
				paramAzureEgressPrivateLinkGateway:         azureEgressPrivateLinkGatewayDataSourceSchema(),
				paramAzurePeeringGateway:                   azurePeeringGatewaySpecDataSourceSchema(),
				paramGcpEgressPrivateServiceConnectGateway: gcpEgressPrivateServiceConnectGatewayDataSourceSchema(),
				paramGcpPeeringGateway:                     gcpPeeringGatewaySpecDataSourceSchema(),
			},
		},
	}
}

func gatewaysDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Reading Gateways")

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	gatewayTypes := convertToStringSlice(d.Get(fmt.Sprintf("%s.0.%s", paramFilter, paramGatewayType)).([]interface{}))
	ids := convertToStringSlice(d.Get(fmt.Sprintf("%s.0.%s", paramFilter, paramId)).([]interface{}))
	regions := convertToStringSlice(d.Get(fmt.Sprintf("%s.0.%s", paramFilter, paramRegion)).([]interface{}))
	displayNames := convertToStringSlice(d.Get(fmt.Sprintf("%s.0.%s", paramFilter, paramDisplayName)).([]interface{}))
	phases := convertToStringSlice(d.Get(fmt.Sprintf("%s.0.%s", paramFilter, paramPhase)).([]interface{}))

	c := meta.(*Client)
	gateways, err := loadGateways(c.netGWApiContext(ctx), c, environmentId, gatewayTypes, ids, regions, displayNames, phases)
	if err != nil {
		return diag.Errorf("error reading Gateways: %s", createDescriptiveError(err))
	}

	result := make([]map[string]interface{}, len(gateways))
	for i, gateway := range gateways {
		gatewayMap := map[string]interface{}{
			paramId:          gateway.GetId(),
			paramDisplayName: gateway.Spec.GetDisplayName(),
		}

		// Set gateway type-specific attributes
		if gateway.Spec.GetConfig().NetworkingV1AwsEgressPrivateLinkGatewaySpec != nil && gateway.Status.GetCloudGateway().NetworkingV1AwsEgressPrivateLinkGatewayStatus != nil {
			gatewayMap[paramAwsEgressPrivateLinkGateway] = []interface{}{map[string]interface{}{
				paramRegion:       gateway.Spec.GetConfig().NetworkingV1AwsEgressPrivateLinkGatewaySpec.GetRegion(),
				paramPrincipalArn: gateway.Status.GetCloudGateway().NetworkingV1AwsEgressPrivateLinkGatewayStatus.GetPrincipalArn(),
			}}
		} else if gateway.Spec.GetConfig().NetworkingV1AwsIngressPrivateLinkGatewaySpec != nil && gateway.Status.GetCloudGateway().NetworkingV1AwsIngressPrivateLinkGatewayStatus != nil {
			gatewayMap[paramAwsIngressPrivateLinkGateway] = []interface{}{map[string]interface{}{
				paramRegion:                 gateway.Spec.GetConfig().NetworkingV1AwsIngressPrivateLinkGatewaySpec.GetRegion(),
				paramVpcEndpointServiceName: gateway.Status.GetCloudGateway().NetworkingV1AwsIngressPrivateLinkGatewayStatus.GetVpcEndpointServiceName(),
			}}
		} else if gateway.Spec.GetConfig().NetworkingV1AwsPeeringGatewaySpec != nil {
			gatewayMap[paramAwsPeeringGateway] = []interface{}{map[string]interface{}{
				paramRegion: gateway.Spec.GetConfig().NetworkingV1AwsPeeringGatewaySpec.GetRegion(),
			}}
		} else if gateway.Spec.GetConfig().NetworkingV1AwsPrivateNetworkInterfaceGatewaySpec != nil {
			zones := gateway.Spec.GetConfig().NetworkingV1AwsPrivateNetworkInterfaceGatewaySpec.GetZones()
			zonesList := make([]interface{}, len(zones))
			for j, zone := range zones {
				zonesList[j] = zone
			}
			account := ""
			if gateway.Status.GetCloudGateway().NetworkingV1AwsPrivateNetworkInterfaceGatewayStatus != nil {
				account = gateway.Status.CloudGateway.NetworkingV1AwsPrivateNetworkInterfaceGatewayStatus.GetAccount()
			}
			gatewayMap[paramAwsPrivateNetworkInterfaceGateway] = []interface{}{map[string]interface{}{
				paramRegion:  gateway.Spec.GetConfig().NetworkingV1AwsPrivateNetworkInterfaceGatewaySpec.GetRegion(),
				paramZones:   zonesList,
				paramAccount: account,
			}}
		} else if gateway.Spec.GetConfig().NetworkingV1AzureEgressPrivateLinkGatewaySpec != nil && gateway.Status.GetCloudGateway().NetworkingV1AzureEgressPrivateLinkGatewayStatus != nil {
			gatewayMap[paramAzureEgressPrivateLinkGateway] = []interface{}{map[string]interface{}{
				paramRegion:       gateway.Spec.GetConfig().NetworkingV1AzureEgressPrivateLinkGatewaySpec.GetRegion(),
				paramSubscription: gateway.Status.GetCloudGateway().NetworkingV1AzureEgressPrivateLinkGatewayStatus.GetSubscription(),
			}}
		} else if gateway.Spec.GetConfig().NetworkingV1AzurePeeringGatewaySpec != nil {
			gatewayMap[paramAzurePeeringGateway] = []interface{}{map[string]interface{}{
				paramRegion: gateway.Spec.GetConfig().NetworkingV1AzurePeeringGatewaySpec.GetRegion(),
			}}
		} else if gateway.Spec.GetConfig().NetworkingV1GcpEgressPrivateServiceConnectGatewaySpec != nil {
			project := ""
			if gateway.Status.GetCloudGateway().NetworkingV1GcpEgressPrivateServiceConnectGatewayStatus != nil {
				project = gateway.Status.CloudGateway.NetworkingV1GcpEgressPrivateServiceConnectGatewayStatus.GetProject()
			}
			gatewayMap[paramGcpEgressPrivateServiceConnectGateway] = []interface{}{map[string]interface{}{
				paramRegion:  gateway.Spec.GetConfig().NetworkingV1GcpEgressPrivateServiceConnectGatewaySpec.GetRegion(),
				paramProject: project,
			}}
		} else if gateway.Spec.GetConfig().NetworkingV1GcpPeeringGatewaySpec != nil {
			iamPrincipal := ""
			if gateway.Status.GetCloudGateway().NetworkingV1GcpPeeringGatewayStatus != nil {
				iamPrincipal = gateway.Status.CloudGateway.NetworkingV1GcpPeeringGatewayStatus.GetIamPrincipal()
			}
			gatewayMap[paramGcpPeeringGateway] = []interface{}{map[string]interface{}{
				paramRegion:       gateway.Spec.GetConfig().NetworkingV1GcpPeeringGatewaySpec.GetRegion(),
				paramIAMPrincipal: iamPrincipal,
			}}
		}

		result[i] = gatewayMap
	}

	if err := d.Set(paramGateways, result); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return nil
}

func loadGateways(ctx context.Context, c *Client, environmentId string, gatewayTypes, ids, regions, displayNames, phases []string) ([]netgw.NetworkingV1Gateway, error) {
	gateways := make([]netgw.NetworkingV1Gateway, 0)

	allGatewaysAreCollected := false
	pageToken := ""
	for !allGatewaysAreCollected {
		gatewaysPageList, resp, err := executeListGateways(ctx, c, environmentId, gatewayTypes, ids, regions, displayNames, phases, pageToken)
		if err != nil {
			return nil, fmt.Errorf("error reading Gateways: %s", createDescriptiveError(err, resp))
		}
		gateways = append(gateways, gatewaysPageList.GetData()...)

		// nextPageUrlStringNullable is nil for the last page
		nextPageUrlStringNullable := gatewaysPageList.GetMetadata().Next

		if nextPageUrlStringNullable.IsSet() {
			nextPageUrlString := *nextPageUrlStringNullable.Get()
			if nextPageUrlString == "" {
				allGatewaysAreCollected = true
			} else {
				pageToken, err = extractPageToken(nextPageUrlString)
				if err != nil {
					return nil, fmt.Errorf("error reading Gateways: %s", createDescriptiveError(err, resp))
				}
			}
		} else {
			allGatewaysAreCollected = true
		}
	}
	return gateways, nil
}

func executeListGateways(ctx context.Context, c *Client, environmentId string, gatewayTypes, ids, regions, displayNames, phases []string, pageToken string) (netgw.NetworkingV1GatewayList, *http.Response, error) {
	request := c.netGatewayClient.GatewaysNetworkingV1Api.ListNetworkingV1Gateways(ctx).Environment(environmentId).PageSize(listGatewaysPageSize)

	if len(gatewayTypes) > 0 {
		request = request.GatewayType(gatewayTypes)
	}
	if len(ids) > 0 {
		request = request.Id(ids)
	}
	if len(regions) > 0 {
		request = request.SpecConfigRegion(regions)
	}
	if len(displayNames) > 0 {
		request = request.SpecDisplayName(displayNames)
	}
	if len(phases) > 0 {
		request = request.StatusPhase(phases)
	}
	if pageToken != "" {
		request = request.PageToken(pageToken)
	}

	return request.Execute()
}
