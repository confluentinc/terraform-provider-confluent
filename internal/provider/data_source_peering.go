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
	// The maximum allowable page size - 1 (to avoid off-by-one errors) when listing peerings using Networking API
	// https://docs.confluent.io/cloud/current/api.html#operation/listNetworkingV1Peerings
	listPeeringsPageSize = 99
)

func peeringDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: peeringDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute, not both
				ExactlyOneOf: []string{paramId, paramDisplayName},
				Description:  "The ID of the Peering, for example, `pla-abc123`.",
			},
			// Similarly, paramEnvironment is required as well
			paramEnvironment: environmentDataSourceSchema(),
			paramNetwork:     networkDataSourceSchema(),
			paramDisplayName: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute, not both
				ExactlyOneOf: []string{paramId, paramDisplayName},
			},
			paramAws:   awsPeeringDataSourceSchema(),
			paramAzure: azurePeeringDataSourceSchema(),
			paramGcp:   gcpPeeringDataSourceSchema(),
		},
	}
}

func peeringDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// ExactlyOneOf specified in the schema ensures one of paramId or paramDisplayName is specified.
	// The next step is to figure out which one exactly is set.
	peeringId := d.Get(paramId).(string)
	displayName := d.Get(paramDisplayName).(string)

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	if peeringId != "" {
		return peeringDataSourceReadUsingId(ctx, d, meta, environmentId, peeringId)
	} else if displayName != "" {
		return peeringDataSourceReadUsingDisplayName(ctx, d, meta, environmentId, displayName)
	} else {
		return diag.Errorf("error reading Peering: exactly one of %q or %q must be specified but they're both empty", paramId, paramDisplayName)
	}
}

func peeringDataSourceReadUsingDisplayName(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, displayName string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Peering %q=%q", paramDisplayName, displayName))

	c := meta.(*Client)
	peerings, err := loadPeerings(ctx, c, environmentId)
	if err != nil {
		return diag.Errorf("error reading Peering %q: %s", displayName, createDescriptiveError(err))
	}
	if orgHasMultiplePeeringsWithTargetDisplayName(peerings, displayName) {
		return diag.Errorf("error reading Peering: there are multiple Peering with %q=%q", paramDisplayName, displayName)
	}

	for _, peering := range peerings {
		if peering.Spec.GetDisplayName() == displayName {
			if _, err := setPeeringAttributes(d, peering); err != nil {
				return diag.FromErr(createDescriptiveError(err))
			}
			return nil
		}
	}

	return diag.Errorf("error reading Peering: Peering with %q=%q was not found", paramDisplayName, displayName)
}

func peeringDataSourceReadUsingId(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, peeringId string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Peering %q=%q", paramId, peeringId), map[string]interface{}{peeringLoggingKey: peeringId})

	c := meta.(*Client)
	peering, resp, err := executePeeringRead(c.netApiContext(ctx), c, environmentId, peeringId)
	if err != nil {
		return diag.Errorf("error reading Peering %q: %s", peeringId, createDescriptiveError(err, resp))
	}
	peeringJson, err := json.Marshal(peering)
	if err != nil {
		return diag.Errorf("error reading Peering %q: error marshaling %#v to json: %s", peeringId, peering, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Peering %q: %s", peeringId, peeringJson), map[string]interface{}{peeringLoggingKey: peeringId})

	if _, err := setPeeringAttributes(d, peering); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}

func orgHasMultiplePeeringsWithTargetDisplayName(peerings []net.NetworkingV1Peering, displayName string) bool {
	var numberOfPeeringsWithTargetDisplayName = 0
	for _, peering := range peerings {
		if peering.Spec.GetDisplayName() == displayName {
			numberOfPeeringsWithTargetDisplayName += 1
		}
	}
	return numberOfPeeringsWithTargetDisplayName > 1
}

func loadPeerings(ctx context.Context, c *Client, environmentId string) ([]net.NetworkingV1Peering, error) {
	peerings := make([]net.NetworkingV1Peering, 0)

	allPeeringsAreCollected := false
	pageToken := ""
	for !allPeeringsAreCollected {
		peeringsPageList, resp, err := executeListPeerings(ctx, c, environmentId, pageToken)
		if err != nil {
			return nil, fmt.Errorf("error reading Peerings: %s", createDescriptiveError(err, resp))
		}
		peerings = append(peerings, peeringsPageList.GetData()...)

		// nextPageUrlStringNullable is nil for the last page
		nextPageUrlStringNullable := peeringsPageList.GetMetadata().Next

		if nextPageUrlStringNullable.IsSet() {
			nextPageUrlString := *nextPageUrlStringNullable.Get()
			if nextPageUrlString == "" {
				allPeeringsAreCollected = true
			} else {
				pageToken, err = extractPageToken(nextPageUrlString)
				if err != nil {
					return nil, fmt.Errorf("error reading Peerings: %s", createDescriptiveError(err, resp))
				}
			}
		} else {
			allPeeringsAreCollected = true
		}
	}
	return peerings, nil
}

func executeListPeerings(ctx context.Context, c *Client, environmentId, pageToken string) (net.NetworkingV1PeeringList, *http.Response, error) {
	if pageToken != "" {
		return c.netClient.PeeringsNetworkingV1Api.ListNetworkingV1Peerings(c.netApiContext(ctx)).Environment(environmentId).PageSize(listPeeringsPageSize).PageToken(pageToken).Execute()
	} else {
		return c.netClient.PeeringsNetworkingV1Api.ListNetworkingV1Peerings(c.netApiContext(ctx)).Environment(environmentId).PageSize(listPeeringsPageSize).Execute()
	}
}

func networkDataSourceSchema() *schema.Schema {
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

func awsPeeringDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramAccount: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "AWS account for VPC to peer with the network.",
				},
				paramVpc: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The id of the AWS VPC to peer with.",
				},
				paramRoutes: {
					Type:        schema.TypeList,
					Computed:    true,
					Elem:        &schema.Schema{Type: schema.TypeString},
					Description: "List of routes for the peering.",
				},
				paramCustomerRegion: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Region of customer VPC.",
				},
			},
		},
	}
}

func azurePeeringDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramTenant: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Customer Azure tenant.",
				},
				paramVnet: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Customer VNet to peer with.",
				},
				paramCustomerRegion: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Region of customer VNet.",
				},
			},
		},
	}
}

func gcpPeeringDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramProject: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The name of the GCP project.",
				},
				paramVpcNetwork: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The name of the GCP VPC network to peer with.",
				},
				paramImportCustomRoutes: {
					Type:        schema.TypeBool,
					Computed:    true,
					Description: "Enable customer route import.",
				},
			},
		},
	}
}
