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
	// The maximum allowable page size - 1 (to avoid off-by-one errors) when listing identity providers using IAM V2 API
	listIdentityProvidersPageSize = 99
)

func identityProviderDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: identityProviderDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute, not both
				ExactlyOneOf: []string{paramId, paramDisplayName},
				Description:  "The ID of the Identity Provider (e.g., `op-abc123`).",
			},
			paramDisplayName: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute, not both
				ExactlyOneOf: []string{paramId, paramDisplayName},
				Description:  "A name for the Identity Provider.",
			},
			paramDescription: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A description of the Identity Provider.",
			},
			paramIssuer: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A publicly reachable issuer URI for the Identity Provider.",
			},
			paramJwksUri: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A publicly reachable JWKS URI for the Identity Provider.",
			},
		},
	}
}

func identityProviderDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// ExactlyOneOf specified in the schema ensures one of paramId or paramDisplayName is specified.
	// The next step is to figure out which one exactly is set.

	identityProviderId := d.Get(paramId).(string)
	displayName := d.Get(paramDisplayName).(string)

	if identityProviderId != "" {
		return identityProviderDataSourceReadUsingId(ctx, d, meta, identityProviderId)
	} else if displayName != "" {
		return identityProviderDataSourceReadUsingDisplayName(ctx, d, meta, displayName)
	} else {
		return diag.Errorf("error reading Identity Provider: exactly one of %q or %q must be specified but they're both empty", paramId, paramDisplayName)
	}
}

func identityProviderDataSourceReadUsingDisplayName(ctx context.Context, d *schema.ResourceData, meta interface{}, displayName string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Identity Provider %q=%q", paramDisplayName, displayName))

	client := meta.(*Client)
	identityProviders, err := loadIdentityProviders(ctx, client)
	if err != nil {
		return diag.Errorf("error reading Identity Provider %q: %s", displayName, createDescriptiveError(err))
	}
	if orgHasMultipleIdentityProvidersWithTargetDisplayName(identityProviders, displayName) {
		return diag.Errorf("error reading Identity Provider: there are multiple Identity Providers with %q=%q", paramDisplayName, displayName)
	}
	for _, identityProvider := range identityProviders {
		if identityProvider.GetDisplayName() == displayName {
			if _, err := setIdentityProviderAttributes(d, identityProvider); err != nil {
				return diag.FromErr(createDescriptiveError(err))
			}
			return nil
		}
	}

	return diag.Errorf("error reading Identity Provider: Identity Provider with %q=%q was not found", paramDisplayName, displayName)
}

func loadIdentityProviders(ctx context.Context, c *Client) ([]v2.IamV2IdentityProvider, error) {
	identityProviders := make([]v2.IamV2IdentityProvider, 0)

	allIdentityProvidersAreCollected := false
	pageToken := ""
	for !allIdentityProvidersAreCollected {
		identityProviderPageList, _, err := executeListIdentityProviders(ctx, c, pageToken)
		if err != nil {
			return nil, fmt.Errorf("error reading Identity Providers: %s", createDescriptiveError(err))
		}
		identityProviders = append(identityProviders, identityProviderPageList.GetData()...)

		// nextPageUrlStringNullable is nil for the last page
		nextPageUrlStringNullable := identityProviderPageList.GetMetadata().Next

		if nextPageUrlStringNullable.IsSet() {
			nextPageUrlString := *nextPageUrlStringNullable.Get()
			if nextPageUrlString == "" {
				allIdentityProvidersAreCollected = true
			} else {
				pageToken, err = extractPageToken(nextPageUrlString)
				if err != nil {
					return nil, fmt.Errorf("error reading Identity Providers: %s", createDescriptiveError(err))
				}
			}
		} else {
			allIdentityProvidersAreCollected = true
		}
	}
	return identityProviders, nil
}

func executeListIdentityProviders(ctx context.Context, c *Client, pageToken string) (v2.IamV2IdentityProviderList, *http.Response, error) {
	if pageToken != "" {
		return c.oidcClient.IdentityProvidersIamV2Api.ListIamV2IdentityProviders(c.oidcApiContext(ctx)).PageSize(listIdentityProvidersPageSize).PageToken(pageToken).Execute()
	} else {
		return c.oidcClient.IdentityProvidersIamV2Api.ListIamV2IdentityProviders(c.oidcApiContext(ctx)).PageSize(listIdentityProvidersPageSize).Execute()
	}
}

func identityProviderDataSourceReadUsingId(ctx context.Context, d *schema.ResourceData, meta interface{}, identityProviderId string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Identity Provider %q=%q", paramId, identityProviderId), map[string]interface{}{identityProviderLoggingKey: identityProviderId})

	c := meta.(*Client)
	identityProvider, _, err := executeIdentityProviderRead(c.oidcApiContext(ctx), c, identityProviderId)
	if err != nil {
		return diag.Errorf("error reading Identity Provider %q: %s", identityProviderId, createDescriptiveError(err))
	}
	identityProviderJson, err := json.Marshal(identityProvider)
	if err != nil {
		return diag.Errorf("error reading Identity Provider %q: error marshaling %#v to json: %s", identityProviderId, identityProvider, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Identity Provider %q: %s", identityProviderId, identityProviderJson), map[string]interface{}{identityProviderLoggingKey: identityProviderId})

	if _, err := setIdentityProviderAttributes(d, identityProvider); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}

func orgHasMultipleIdentityProvidersWithTargetDisplayName(identityProviders []v2.IamV2IdentityProvider, displayName string) bool {
	var numberOfIdentityProvidersWithTargetDisplayName = 0
	for _, identityProvider := range identityProviders {
		if identityProvider.GetDisplayName() == displayName {
			numberOfIdentityProvidersWithTargetDisplayName += 1
		}
	}
	return numberOfIdentityProvidersWithTargetDisplayName > 1
}
