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
	v2 "github.com/confluentinc/ccloud-sdk-go-v2/identity-provider/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net/http"
)

const (
	// The maximum allowable page size - 1 (to avoid off-by-one errors) when listing identity pools
	listIdentityPoolsPageSize = 99
)

func identityPoolDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: identityPoolDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute, not both
				ExactlyOneOf: []string{paramId, paramDisplayName},
			},
			// Similarly, paramIdentityProvider is required as well
			paramIdentityProvider: identityProviderDataSourceSchema(),
			paramDisplayName: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute, not both
				ExactlyOneOf: []string{paramId, paramDisplayName},
			},
			paramDescription: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramIdentityClaim: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramFilter: {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func identityPoolDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// ExactlyOneOf specified in the schema ensures one of paramId or paramDisplayName is specified.
	// The next step is to figure out which one exactly is set.
	identityPoolId := d.Get(paramId).(string)
	displayName := d.Get(paramDisplayName).(string)

	identityProviderId := extractStringValueFromBlock(d, paramIdentityProvider, paramId)

	if identityPoolId != "" {
		return identityPoolDataSourceReadUsingId(ctx, d, meta, identityProviderId, identityPoolId)
	} else if displayName != "" {
		return identityPoolDataSourceReadUsingDisplayName(ctx, d, meta, identityProviderId, displayName)
	} else {
		return diag.Errorf("error reading Identity Pool: exactly one of %q or %q must be specified but they're both empty", paramId, paramDisplayName)
	}
}

func identityPoolDataSourceReadUsingDisplayName(ctx context.Context, d *schema.ResourceData, meta interface{}, identityProviderId, displayName string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Identity Pool %q=%q", paramDisplayName, displayName))

	c := meta.(*Client)
	identityPools, err := loadIdentityPools(ctx, c, identityProviderId)
	if err != nil {
		return diag.Errorf("error reading Identity Pool %q: %s", displayName, createDescriptiveError(err))
	}
	if orgHasMultipleIdentityPoolsWithTargetDisplayName(identityPools, displayName) {
		return diag.Errorf("error reading Identity Pool: there are multiple Identity Pools with %q=%q", paramDisplayName, displayName)
	}

	for _, identityPool := range identityPools {
		if identityPool.GetDisplayName() == displayName {
			if _, err := setIdentityPoolAttributes(d, identityPool, identityProviderId); err != nil {
				return diag.FromErr(createDescriptiveError(err))
			}
			return nil
		}
	}

	return diag.Errorf("error reading Identity Pool: Identity Pool with %q=%q was not found", paramDisplayName, displayName)
}

func identityPoolDataSourceReadUsingId(ctx context.Context, d *schema.ResourceData, meta interface{}, identityProviderId, identityPoolId string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Identity Pool %q=%q", paramId, identityPoolId), map[string]interface{}{identityPoolLoggingKey: identityPoolId})

	c := meta.(*Client)
	identityPool, _, err := executeIdentityPoolRead(c.oidcApiContext(ctx), c, identityPoolId, identityProviderId)
	if err != nil {
		return diag.Errorf("error reading Identity Pool %q: %s", identityPoolId, createDescriptiveError(err))
	}
	identityPoolJson, err := json.Marshal(identityPool)
	if err != nil {
		return diag.Errorf("error reading Identity Pool %q: error marshaling %#v to json: %s", identityPoolId, identityPool, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Identity Pool %q: %s", identityPoolId, identityPoolJson), map[string]interface{}{identityPoolLoggingKey: identityPoolId})

	if _, err := setIdentityPoolAttributes(d, identityPool, identityProviderId); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}

func orgHasMultipleIdentityPoolsWithTargetDisplayName(identityPools []v2.IamV2IdentityPool, displayName string) bool {
	var numberOfIdentityPoolsWithTargetDisplayName = 0
	for _, identityPool := range identityPools {
		if identityPool.GetDisplayName() == displayName {
			numberOfIdentityPoolsWithTargetDisplayName += 1
		}
	}
	return numberOfIdentityPoolsWithTargetDisplayName > 1
}

func loadIdentityPools(ctx context.Context, c *Client, identityProviderId string) ([]v2.IamV2IdentityPool, error) {
	identityPools := make([]v2.IamV2IdentityPool, 0)

	allIdentityPoolsAreCollected := false
	pageToken := ""
	for !allIdentityPoolsAreCollected {
		identityPoolsPageList, _, err := executeListIdentityPools(ctx, c, identityProviderId, pageToken)
		if err != nil {
			return nil, fmt.Errorf("error reading Identity Pools: %s", createDescriptiveError(err))
		}
		identityPools = append(identityPools, identityPoolsPageList.GetData()...)

		// nextPageUrlStringNullable is nil for the last page
		nextPageUrlStringNullable := identityPoolsPageList.GetMetadata().Next

		if nextPageUrlStringNullable.IsSet() {
			nextPageUrlString := *nextPageUrlStringNullable.Get()
			if nextPageUrlString == "" {
				allIdentityPoolsAreCollected = true
			} else {
				pageToken, err = extractPageToken(nextPageUrlString)
				if err != nil {
					return nil, fmt.Errorf("error reading Identity Pools: %s", createDescriptiveError(err))
				}
			}
		} else {
			allIdentityPoolsAreCollected = true
		}
	}
	return identityPools, nil
}

func executeListIdentityPools(ctx context.Context, c *Client, identityProviderId, pageToken string) (v2.IamV2IdentityPoolList, *http.Response, error) {
	if pageToken != "" {
		return c.oidcClient.IdentityPoolsIamV2Api.ListIamV2IdentityPools(c.oidcApiContext(ctx), identityProviderId).PageSize(listIdentityPoolsPageSize).PageToken(pageToken).Execute()
	} else {
		return c.oidcClient.IdentityPoolsIamV2Api.ListIamV2IdentityPools(c.oidcApiContext(ctx), identityProviderId).PageSize(listIdentityPoolsPageSize).Execute()
	}
}

// https://github.com/hashicorp/terraform-plugin-sdk/issues/155#issuecomment-489699737
////  alternative - https://github.com/hashicorp/terraform-plugin-sdk/issues/248#issuecomment-725013327
func identityProviderDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:     schema.TypeString,
					Required: true,
				},
			},
		},
		Required: true,
		MaxItems: 1,
	}
}
