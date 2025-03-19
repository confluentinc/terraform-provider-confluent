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
	v1 "github.com/confluentinc/ccloud-sdk-go-v2/networking/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net/http"
)

const (
	paramEnvironments = "environments"
	paramNetworks     = "networks"
	paramAccept       = "accept"
	// The maximum allowable page size - 1 (to avoid off-by-one errors) when listing service accounts using SG V3 API
	// https://docs.confluent.io/cloud/current/api.html#tag/Network-Link-Services-(networkingv1)/operation/listNetworkingV1NetworkLinkServices
	listNetworkLinkServicesPageSize = 99
)

func networkLinkServiceDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: networkLinkServiceDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute, not both
				ExactlyOneOf: []string{paramId, paramDisplayName},
				Description: "The ID of the network link service, for example, `nls-a1b2c`.",
			},
			paramDisplayName: {
				Type:        schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute, not both
				ExactlyOneOf: []string{paramId, paramDisplayName},
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
	// ExactlyOneOf specified in the schema ensures one of paramId or paramDisplayName is specified.
	// The next step is to figure out which one exactly is set.
	nlsId := d.Get(paramId).(string)
	displayName := d.Get(paramDisplayName).(string)

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	if environmentId == "" {
		return diag.Errorf("error reading network link service: environment Id is missing")
	}

	if nlsId != "" {
		return nlsDataSourceReadUsingId(ctx, d, meta, environmentId, nlsId)
	} else if displayName != "" {
		return nlsDataSourceReadUsingDisplayName(ctx, d, meta, environmentId, displayName)
	} else {
		return diag.Errorf("error reading network link service: exactly one of `id` or `display_name` must be specified but they're both empty")
	}
}

func nlsDataSourceReadUsingId(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId string, nlsId string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading network link service %q", nlsId), map[string]interface{}{networkLinkServiceLoggingKey: nlsId})

	c := meta.(*Client)
	nls, _, err := executeNlsRead(c.ssoApiContext(ctx), c, environmentId, nlsId)
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

func executeNlsRead(ctx context.Context, c *Client, environmentId string, nlsId string) (v1.NetworkingV1NetworkLinkService, *http.Response, error) {
	req := c.netClient.NetworkLinkServicesNetworkingV1Api.GetNetworkingV1NetworkLinkService(c.netApiContext(ctx), nlsId).Environment(environmentId)
	return req.Execute()
}

func nlsDataSourceReadUsingDisplayName(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, displayName string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading network link service data source using display name %q", displayName))
	c := meta.(*Client)
	networkLinks, err := loadNetworkLinkServices(ctx, c, environmentId)

	if err != nil {
		return diag.Errorf("error reading network link service data source using display name %q: %s", displayName, createDescriptiveError(err))
	}

	for _, networkLink := range networkLinks {
		if networkLink.Spec.GetDisplayName() == displayName {
			nlsJson, err := json.Marshal(networkLink)
			if err != nil {
				return diag.Errorf("error reading network link service using display name %q: error marshaling %#v to json: %s", displayName, networkLink, createDescriptiveError(err))
			}
			if _, err := setNetworkLinkServiceAttributes(d, networkLink); err != nil {
				tflog.Debug(ctx, fmt.Sprintf("Fetched Network Link Service using display name %q: %s", displayName, nlsJson))
				return diag.FromErr(createDescriptiveError(err))
			}
			return nil
		}
	}

	return nil
}

func loadNetworkLinkServices(ctx context.Context, c *Client, environmentId string) ([]v1.NetworkingV1NetworkLinkService, error) {
	networkLinks := make([]v1.NetworkingV1NetworkLinkService, 0)

	allNlsAreCollected := false
	pageToken := ""
	for !allNlsAreCollected {
		nlsList, _, err := executeListNetworkLinkServices(ctx, c, environmentId, pageToken)
		if err != nil {
			return nil, fmt.Errorf("error reading network link services: %s", createDescriptiveError(err))
		}
		networkLinks = append(networkLinks, nlsList.GetData()...)

		// nextPageUrlStringNullable is nil for the last page
		nextPageUrlStringNullable := nlsList.GetMetadata().Next

		if nextPageUrlStringNullable.IsSet() {
			nextPageUrlString := *nextPageUrlStringNullable.Get()
			if nextPageUrlString == "" {
				allNlsAreCollected = true
			} else {
				pageToken, err = extractPageToken(nextPageUrlString)
				if err != nil {
					return nil, fmt.Errorf("error reading network link services: %s", createDescriptiveError(err))
				}
			}
		} else {
			allNlsAreCollected = true
		}
	}
	return networkLinks, nil
}

func executeListNetworkLinkServices(ctx context.Context, c *Client, environmentId, pageToken string) (v1.NetworkingV1NetworkLinkServiceList, *http.Response, error) {
	if pageToken != "" {
		return c.netClient.NetworkLinkServicesNetworkingV1Api.ListNetworkingV1NetworkLinkServices(c.netApiContext(ctx)).Environment(environmentId).PageSize(listNetworkLinkServicesPageSize).PageToken(pageToken).Execute()
	} else {
		return c.netClient.NetworkLinkServicesNetworkingV1Api.ListNetworkingV1NetworkLinkServices(c.netApiContext(ctx)).Environment(environmentId).PageSize(listNetworkLinkServicesPageSize).Execute()
	}
}
