// Copyright 2022 Confluent Inc. All Rights Reserved.
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
	"net/http"
)

const (
	// The maximum allowable page size - 1 (to avoid off-by-one errors) when listing service accounts using CMK V2 API
	// https://docs.confluent.io/cloud/current/api.html#operation/listNetworkingV1Networks
	listNetworksPageSize = 99
)

func networkDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: networkDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute, not both
				ExactlyOneOf: []string{paramId, paramDisplayName},
				Description:  "The ID of the Network, for example, `n-abc123`.",
			},
			// Similarly, paramEnvironment is required as well
			paramEnvironment: environmentDataSourceSchema(),
			paramDisplayName: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute, not both
				ExactlyOneOf: []string{paramId, paramDisplayName},
			},
			paramCloud: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramRegion: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramConnectionTypes: connectionTypesDataSourceSchema(),
			paramCidr: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramZones:     zonesDataSourceSchema(),
			paramDnsConfig: optionalDnsConfigDataSourceSchema(),
			paramResourceName: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramDnsDomain: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramZonalSubdomains: {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "The DNS subdomain for each zone. Present on networks that support PrivateLink. Keys are zones and values are DNS domains.",
			},
			paramAws:   awsNetworkSchema(),
			paramAzure: azureNetworkSchema(),
			paramGcp:   gcpNetworkSchema(),
		},
	}
}

func networkDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// ExactlyOneOf specified in the schema ensures one of paramId or paramDisplayName is specified.
	// The next step is to figure out which one exactly is set.
	networkId := d.Get(paramId).(string)
	displayName := d.Get(paramDisplayName).(string)

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	if networkId != "" {
		return networkDataSourceReadUsingId(ctx, d, meta, environmentId, networkId)
	} else if displayName != "" {
		return networkDataSourceReadUsingDisplayName(ctx, d, meta, environmentId, displayName)
	} else {
		return diag.Errorf("error reading Network: exactly one of %q or %q must be specified but they're both empty", paramId, paramDisplayName)
	}
}

func networkDataSourceReadUsingDisplayName(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, displayName string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Network %q=%q", paramDisplayName, displayName))

	c := meta.(*Client)
	networks, err := loadNetworks(ctx, c, environmentId)
	if err != nil {
		return diag.Errorf("error reading Network %q: %s", displayName, createDescriptiveError(err))
	}
	if orgHasMultipleNetworksWithTargetDisplayName(networks, displayName) {
		return diag.Errorf("error reading Network: there are multiple Networks with %q=%q", paramDisplayName, displayName)
	}

	for _, network := range networks {
		if network.Spec.GetDisplayName() == displayName {
			if _, err := setNetworkAttributes(d, network); err != nil {
				return diag.FromErr(createDescriptiveError(err))
			}
			return nil
		}
	}

	return diag.Errorf("error reading Network: Network with %q=%q was not found", paramDisplayName, displayName)
}

func networkDataSourceReadUsingId(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, networkId string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Network %q=%q", paramId, networkId), map[string]interface{}{networkLoggingKey: networkId})

	c := meta.(*Client)
	network, _, err := executeNetworkRead(c.netApiContext(ctx), c, environmentId, networkId)
	if err != nil {
		return diag.Errorf("error reading Network %q: %s", networkId, createDescriptiveError(err))
	}
	networkJson, err := json.Marshal(network)
	if err != nil {
		return diag.Errorf("error reading Network %q: error marshaling %#v to json: %s", networkId, network, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Network %q: %s", networkId, networkJson), map[string]interface{}{networkLoggingKey: networkId})

	if _, err := setNetworkAttributes(d, network); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}

func orgHasMultipleNetworksWithTargetDisplayName(clusters []net.NetworkingV1Network, displayName string) bool {
	var numberOfClustersWithTargetDisplayName = 0
	for _, cluster := range clusters {
		if cluster.Spec.GetDisplayName() == displayName {
			numberOfClustersWithTargetDisplayName += 1
		}
	}
	return numberOfClustersWithTargetDisplayName > 1
}

func loadNetworks(ctx context.Context, c *Client, environmentId string) ([]net.NetworkingV1Network, error) {
	networks := make([]net.NetworkingV1Network, 0)

	allNetworksAreCollected := false
	pageToken := ""
	for !allNetworksAreCollected {
		networksPageList, _, err := executeListNetworks(ctx, c, environmentId, pageToken)
		if err != nil {
			return nil, fmt.Errorf("error reading Networks: %s", createDescriptiveError(err))
		}
		networks = append(networks, networksPageList.GetData()...)

		// nextPageUrlStringNullable is nil for the last page
		nextPageUrlStringNullable := networksPageList.GetMetadata().Next

		if nextPageUrlStringNullable.IsSet() {
			nextPageUrlString := *nextPageUrlStringNullable.Get()
			if nextPageUrlString == "" {
				allNetworksAreCollected = true
			} else {
				pageToken, err = extractPageToken(nextPageUrlString)
				if err != nil {
					return nil, fmt.Errorf("error reading Networks: %s", createDescriptiveError(err))
				}
			}
		} else {
			allNetworksAreCollected = true
		}
	}
	return networks, nil
}

func executeListNetworks(ctx context.Context, c *Client, environmentId, pageToken string) (net.NetworkingV1NetworkList, *http.Response, error) {
	if pageToken != "" {
		return c.netClient.NetworksNetworkingV1Api.ListNetworkingV1Networks(c.netApiContext(ctx)).Environment(environmentId).PageSize(listNetworksPageSize).PageToken(pageToken).Execute()
	} else {
		return c.netClient.NetworksNetworkingV1Api.ListNetworkingV1Networks(c.netApiContext(ctx)).Environment(environmentId).PageSize(listNetworksPageSize).Execute()
	}
}

func connectionTypesDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Elem:     &schema.Schema{Type: schema.TypeString},
	}
}

func zonesDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Elem:     &schema.Schema{Type: schema.TypeString},
	}
}

func optionalDnsConfigDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		Computed:    true,
		Description: "Network DNS config. It applies only to the PRIVATELINK network connection type.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramResolution: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Network DNS resolution.",
				},
			},
		},
	}
}
