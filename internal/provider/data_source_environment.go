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
	v2 "github.com/confluentinc/ccloud-sdk-go-v2/org/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net/http"
)

const (
	// The maximum allowable page size - 1 (to avoid off-by-one errors) when listing environments using ORG V2 API
	// https://docs.confluent.io/cloud/current/api.html#operation/listOrgV2Environments
	listEnvironmentsPageSize = 99
)

func environmentDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: environmentDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Description: "The ID of the Environment (e.g., `env-abc123`).",
				Computed:    true,
				Optional:    true,
				// A user should provide a value for either "id" or "display_name" attribute
				ExactlyOneOf: []string{paramId, paramDisplayName},
			},
			paramDisplayName: {
				Type:         schema.TypeString,
				Description:  "A human-readable name for the Environment.",
				Computed:     true,
				Optional:     true,
				ExactlyOneOf: []string{paramId, paramDisplayName},
			},
			paramResourceName: {
				Type:        schema.TypeString,
				Description: "The Confluent Resource Name of the Environment.",
				Computed:    true,
			},
		},
	}
}

func environmentDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// ExactlyOneOf specified in the schema ensures one of paramId or paramDisplayName is specified.
	// The next step is to figure out which one exactly is set.
	environmentId := d.Get(paramId).(string)
	displayName := d.Get(paramDisplayName).(string)

	if environmentId != "" {
		return environmentDataSourceReadUsingId(ctx, d, meta, environmentId)
	} else if displayName != "" {
		return environmentDataSourceReadUsingDisplayName(ctx, d, meta, displayName)
	} else {
		return diag.Errorf("error reading Environment: exactly one of %q or %q must be specified but they're both empty", paramId, paramDisplayName)
	}
}

func environmentDataSourceReadUsingDisplayName(ctx context.Context, d *schema.ResourceData, meta interface{}, displayName string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Environment %q=%q", paramDisplayName, displayName))

	c := meta.(*Client)
	environments, err := loadEnvironments(ctx, c)
	if err != nil {
		return diag.Errorf("error reading Environment %q: %s", displayName, createDescriptiveError(err))
	}
	if orgHasMultipleEnvsWithTargetDisplayName(environments, displayName) {
		return diag.Errorf("error reading Environment: there are multiple Environments with %q=%q", paramDisplayName, displayName)
	}

	for _, environment := range environments {
		if environment.GetDisplayName() == displayName {
			if _, err := setEnvironmentAttributes(d, environment); err != nil {
				return diag.FromErr(createDescriptiveError(err))
			}
			return nil
		}
	}

	return diag.Errorf("error reading Environment: Environment with %q=%q was not found", paramDisplayName, displayName)
}

func loadEnvironments(ctx context.Context, c *Client) ([]v2.OrgV2Environment, error) {
	environments := make([]v2.OrgV2Environment, 0)

	allEnvironmentsAreCollected := false
	pageToken := ""
	for !allEnvironmentsAreCollected {
		environmentPageList, _, err := executeListEnvironments(ctx, c, pageToken)
		if err != nil {
			return nil, fmt.Errorf("error reading Environments: %s", createDescriptiveError(err))
		}
		environments = append(environments, environmentPageList.GetData()...)

		// nextPageUrlStringNullable is nil for the last page
		nextPageUrlStringNullable := environmentPageList.GetMetadata().Next

		if nextPageUrlStringNullable.IsSet() {
			nextPageUrlString := *nextPageUrlStringNullable.Get()
			pageToken, err = extractPageToken(nextPageUrlString)
			if err != nil {
				return nil, fmt.Errorf("error reading Environments: %s", createDescriptiveError(err))
			}
		} else {
			allEnvironmentsAreCollected = true
		}
	}
	return environments, nil
}

func executeListEnvironments(ctx context.Context, c *Client, pageToken string) (v2.OrgV2EnvironmentList, *http.Response, error) {
	if pageToken != "" {
		return c.orgClient.EnvironmentsOrgV2Api.ListOrgV2Environments(c.orgApiContext(ctx)).PageSize(listEnvironmentsPageSize).PageToken(pageToken).Execute()
	} else {
		return c.orgClient.EnvironmentsOrgV2Api.ListOrgV2Environments(c.orgApiContext(ctx)).PageSize(listEnvironmentsPageSize).Execute()
	}
}

func environmentDataSourceReadUsingId(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Environment %q=%q", paramId, environmentId), map[string]interface{}{environmentLoggingKey: environmentId})

	c := meta.(*Client)
	environment, _, err := executeEnvironmentRead(c.orgApiContext(ctx), c, environmentId)
	if err != nil {
		return diag.Errorf("error reading Environment %q: %s", environmentId, createDescriptiveError(err))
	}
	environmentJson, err := json.Marshal(environment)
	if err != nil {
		return diag.Errorf("error reading Environment %q: error marshaling %#v to json: %s", environmentId, environment, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Environment %q: %s", environmentId, environmentJson), map[string]interface{}{environmentLoggingKey: environmentId})

	if _, err := setEnvironmentAttributes(d, environment); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}

func setEnvironmentAttributes(d *schema.ResourceData, environment v2.OrgV2Environment) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, environment.GetDisplayName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramResourceName, environment.Metadata.GetResourceName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	d.SetId(environment.GetId())
	return d, nil
}

func orgHasMultipleEnvsWithTargetDisplayName(environments []v2.OrgV2Environment, displayName string) bool {
	var numberOfEnvironmentsWithTargetDisplayName = 0
	for _, environment := range environments {
		if environment.GetDisplayName() == displayName {
			numberOfEnvironmentsWithTargetDisplayName += 1
		}
	}
	return numberOfEnvironmentsWithTargetDisplayName > 1
}
