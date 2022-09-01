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
	// The maximum allowable page size - 1 (to avoid off-by-one errors) when listing private link accesses using Networking API
	// https://docs.confluent.io/cloud/current/api.html#operation/listNetworkingV1PrivateLinkAccesses
	listPrivateLinkAccessesPageSize = 99
)

func privateLinkAccessDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: privateLinkAccessDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute, not both
				ExactlyOneOf: []string{paramId, paramDisplayName},
				Description:  "The ID of the Private Link Access, for example, `pla-abc123`.",
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
			paramAws:   awsPlaDataSourceSchema(),
			paramAzure: azurePlaDataSourceSchema(),
			paramGcp:   gcpPlaDataSourceSchema(),
		},
	}
}

func privateLinkAccessDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// ExactlyOneOf specified in the schema ensures one of paramId or paramDisplayName is specified.
	// The next step is to figure out which one exactly is set.
	privateLinkAccessId := d.Get(paramId).(string)
	displayName := d.Get(paramDisplayName).(string)

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	if privateLinkAccessId != "" {
		return privateLinkAccessDataSourceReadUsingId(ctx, d, meta, environmentId, privateLinkAccessId)
	} else if displayName != "" {
		return privateLinkAccessDataSourceReadUsingDisplayName(ctx, d, meta, environmentId, displayName)
	} else {
		return diag.Errorf("error reading Private Link Access: exactly one of %q or %q must be specified but they're both empty", paramId, paramDisplayName)
	}
}

func privateLinkAccessDataSourceReadUsingDisplayName(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, displayName string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Private Link Access %q=%q", paramDisplayName, displayName))

	c := meta.(*Client)
	privateLinkAccesses, err := loadPrivateLinkAccesses(ctx, c, environmentId)
	if err != nil {
		return diag.Errorf("error reading Private Link Access %q: %s", displayName, createDescriptiveError(err))
	}
	if orgHasMultiplePrivateLinkAccessesWithTargetDisplayName(privateLinkAccesses, displayName) {
		return diag.Errorf("error reading Private Link Access: there are multiple Private Link Access with %q=%q", paramDisplayName, displayName)
	}

	for _, privateLinkAccess := range privateLinkAccesses {
		if privateLinkAccess.Spec.GetDisplayName() == displayName {
			if _, err := setPrivateLinkAccessAttributes(d, privateLinkAccess); err != nil {
				return diag.FromErr(createDescriptiveError(err))
			}
			return nil
		}
	}

	return diag.Errorf("error reading Private Link Access: Private Link Access with %q=%q was not found", paramDisplayName, displayName)
}

func privateLinkAccessDataSourceReadUsingId(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, privateLinkAccessId string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Private Link Access %q=%q", paramId, privateLinkAccessId), map[string]interface{}{privateLinkAccessLoggingKey: privateLinkAccessId})

	c := meta.(*Client)
	privateLinkAccess, _, err := executePrivateLinkAccessRead(c.netApiContext(ctx), c, environmentId, privateLinkAccessId)
	if err != nil {
		return diag.Errorf("error reading Private Link Access %q: %s", privateLinkAccessId, createDescriptiveError(err))
	}
	privateLinkAccessJson, err := json.Marshal(privateLinkAccess)
	if err != nil {
		return diag.Errorf("error reading Private Link Access %q: error marshaling %#v to json: %s", privateLinkAccessId, privateLinkAccess, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Private Link Access %q: %s", privateLinkAccessId, privateLinkAccessJson), map[string]interface{}{privateLinkAccessLoggingKey: privateLinkAccessId})

	if _, err := setPrivateLinkAccessAttributes(d, privateLinkAccess); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}

func orgHasMultiplePrivateLinkAccessesWithTargetDisplayName(privateLinkAccesses []net.NetworkingV1PrivateLinkAccess, displayName string) bool {
	var numberOfPrivateLinkAccessesWithTargetDisplayName = 0
	for _, privateLinkAccess := range privateLinkAccesses {
		if privateLinkAccess.Spec.GetDisplayName() == displayName {
			numberOfPrivateLinkAccessesWithTargetDisplayName += 1
		}
	}
	return numberOfPrivateLinkAccessesWithTargetDisplayName > 1
}

func loadPrivateLinkAccesses(ctx context.Context, c *Client, environmentId string) ([]net.NetworkingV1PrivateLinkAccess, error) {
	privateLinkAccesses := make([]net.NetworkingV1PrivateLinkAccess, 0)

	allPrivateLinkAccessesAreCollected := false
	pageToken := ""
	for !allPrivateLinkAccessesAreCollected {
		privateLinkAccessesPageList, _, err := executeListPrivateLinkAccesses(ctx, c, environmentId, pageToken)
		if err != nil {
			return nil, fmt.Errorf("error reading PrivateLinkAccesses: %s", createDescriptiveError(err))
		}
		privateLinkAccesses = append(privateLinkAccesses, privateLinkAccessesPageList.GetData()...)

		// nextPageUrlStringNullable is nil for the last page
		nextPageUrlStringNullable := privateLinkAccessesPageList.GetMetadata().Next

		if nextPageUrlStringNullable.IsSet() {
			nextPageUrlString := *nextPageUrlStringNullable.Get()
			if nextPageUrlString == "" {
				allPrivateLinkAccessesAreCollected = true
			} else {
				pageToken, err = extractPageToken(nextPageUrlString)
				if err != nil {
					return nil, fmt.Errorf("error reading PrivateLinkAccesses: %s", createDescriptiveError(err))
				}
			}
		} else {
			allPrivateLinkAccessesAreCollected = true
		}
	}
	return privateLinkAccesses, nil
}

func executeListPrivateLinkAccesses(ctx context.Context, c *Client, environmentId, pageToken string) (net.NetworkingV1PrivateLinkAccessList, *http.Response, error) {
	if pageToken != "" {
		return c.netClient.PrivateLinkAccessesNetworkingV1Api.ListNetworkingV1PrivateLinkAccesses(c.netApiContext(ctx)).Environment(environmentId).PageSize(listPrivateLinkAccessesPageSize).PageToken(pageToken).Execute()
	} else {
		return c.netClient.PrivateLinkAccessesNetworkingV1Api.ListNetworkingV1PrivateLinkAccesses(c.netApiContext(ctx)).Environment(environmentId).PageSize(listPrivateLinkAccessesPageSize).Execute()
	}
}

func awsPlaDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramAccount: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "AWS Account ID to allow for PrivateLink access. Find here (https://console.aws.amazon.com/billing/home?#/account) under My Account in your AWS Management Console.",
				},
			},
		},
	}
}

func azurePlaDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramSubscription: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Azure subscription to allow for PrivateLink access.",
				},
			},
		},
	}
}

func gcpPlaDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramProject: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "GCP project to allow for Private Service Connect access.",
				},
			},
		},
	}
}
