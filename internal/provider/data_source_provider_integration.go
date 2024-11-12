// Copyright 2021 Confluent Inc. All Rights Reserved.
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

	pi "github.com/confluentinc/ccloud-sdk-go-v2/provider-integration/v1"
)

func providerIntegrationDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: providerIntegrationDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute, not both
				ExactlyOneOf: []string{paramId, paramDisplayName},
				Description:  "The ID for provider integration.",
			},
			paramDisplayName: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute, not both
				ExactlyOneOf: []string{paramId, paramDisplayName},
				Description:  "The display name of provider integration.",
			},
			paramAws: awsProviderIntegrationDataSourceConfigSchema(),
			paramUsages: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The usages of provider integration.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			paramEnvironment: environmentDataSourceSchema(),
		},
	}
}

func awsProviderIntegrationDataSourceConfigSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramIamRoleUrn: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "IAM role ARN.",
				},
				paramExternalId: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "External ID for the AWS role.",
				},
				paramCustomerRoleArn: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The AWS customer's IAM role ARN.",
				},
			},
		},
		Computed:    true,
		Description: "Config objects represent AWS cloud provider specific configs.",
	}
}

func providerIntegrationDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	pimId := d.Get(paramId).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	displayName := d.Get(paramDisplayName).(string)

	if pimId != "" {
		return providerIntegrationDataSourceReadUsingId(ctx, d, meta, environmentId, pimId)
	} else if displayName != "" {
		return providerIntegrationDataSourceReadUsingDisplayName(ctx, d, meta, environmentId, displayName)
	} else {
		return diag.Errorf("error reading provider integration: exactly one of %q or %q must be specified but they're both empty", paramId, paramDisplayName)
	}
}

func providerIntegrationDataSourceReadUsingId(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, pimId string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading provider integration data source using Id %q", pimId), map[string]interface{}{providerIntegrationLoggingKey: d.Id()})
	c := meta.(*Client)
	pim, _, err := executeProviderIntegrationRead(c.piApiContext(ctx), c, environmentId, pimId)

	if err != nil {
		return diag.Errorf("error reading provider integration data source using Id %q: %s", pimId, createDescriptiveError(err))
	}
	pimJson, err := json.Marshal(pim)
	if err != nil {
		return diag.Errorf("error reading provider integration %q: error marshaling %#v to json: %s", pimId, pim, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched provider integration %q: %s", pimId, pimJson), map[string]interface{}{providerIntegrationLoggingKey: pimId})

	if _, err := setProviderIntegrationAttributes(d, pim); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading provider integration %q", pimId), map[string]interface{}{providerIntegrationLoggingKey: pimId})
	return nil
}

func providerIntegrationDataSourceReadUsingDisplayName(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, displayName string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading provider integration data source using display name %q", displayName))
	c := meta.(*Client)
	providerIntegrations, err := loadProviderIntegrations(ctx, c, environmentId)

	if err != nil {
		return diag.Errorf("error reading provider integration data source using display name %q: %s", displayName, createDescriptiveError(err))
	}
	if orgHasMultipleProviderIntegrationsWithTargetDisplayName(providerIntegrations, displayName) {
		return diag.Errorf("error reading provider integration: there are multiple provider integrations with %q=%q", paramDisplayName, displayName)
	}

	for _, providerIntegration := range providerIntegrations {
		if providerIntegration.GetDisplayName() == displayName {
			pimJson, err := json.Marshal(providerIntegration)
			if err != nil {
				return diag.Errorf("error reading provider integration using display name %q: error marshaling %#v to json: %s", displayName, providerIntegration, createDescriptiveError(err))
			}

			if _, err := setProviderIntegrationAttributes(d, providerIntegration); err != nil {
				tflog.Debug(ctx, fmt.Sprintf("Fetched provider integration using display name %q: %s", displayName, pimJson))
				return diag.FromErr(createDescriptiveError(err))
			}
			return nil
		}
	}

	return nil
}

func loadProviderIntegrations(ctx context.Context, c *Client, environmentId string) ([]pi.PimV1Integration, error) {
	providerIntegrations := make([]pi.PimV1Integration, 0)
	done := false
	pageToken := ""

	for !done {
		providerIntegrationsPageList, _, err := executeListProviderIntegrations(ctx, c, environmentId, pageToken)
		if err != nil {
			return nil, fmt.Errorf("error reading provider integrations list: %s", createDescriptiveError(err))
		}
		providerIntegrations = append(providerIntegrations, providerIntegrationsPageList.GetData()...)

		// nextPageUrlStringNullable is nil for the last page
		nextPageUrlStringNullable := providerIntegrationsPageList.GetMetadata().Next

		if nextPageUrlStringNullable.IsSet() {
			nextPageUrlString := *nextPageUrlStringNullable.Get()
			if nextPageUrlString == "" {
				done = true
			} else {
				pageToken, err = extractPageToken(nextPageUrlString)
				if err != nil {
					return nil, fmt.Errorf("error reading provider integration list: %s", createDescriptiveError(err))
				}
			}
		} else {
			done = true
		}
	}

	return providerIntegrations, nil
}

func orgHasMultipleProviderIntegrationsWithTargetDisplayName(providerIntegrations []pi.PimV1Integration, displayName string) bool {
	var counter = 0
	for _, providerIntegration := range providerIntegrations {
		if providerIntegration.GetDisplayName() == displayName {
			counter += 1
		}
	}
	return counter > 1
}
