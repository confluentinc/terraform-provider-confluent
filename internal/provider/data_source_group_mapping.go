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
	v2 "github.com/confluentinc/ccloud-sdk-go-v2/sso/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net/http"
)

const (
	// The maximum allowable page size - 1 (to avoid off-by-one errors) when listing group mappings using IAM V2 API
	// https://docs.confluent.io/cloud/current/api.html#tag/Group-Mappings-(iamv2sso)/operation/listIamV2SsoGroupMappings
	listGroupMappingsPageSize = 99
)

func groupMappingDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: groupMappingDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute, not both
				ExactlyOneOf: []string{paramId, paramDisplayName},
			},
			paramDisplayName: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute, not both
				ExactlyOneOf: []string{paramId, paramDisplayName},
			},
			paramFilter: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramDescription: {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func groupMappingDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// ExactlyOneOf specified in the schema ensures one of paramId or paramDisplayName is specified.
	// The next step is to figure out which one exactly is set.

	groupMappingId := d.Get(paramId).(string)
	displayName := d.Get(paramDisplayName).(string)

	if groupMappingId != "" {
		return groupMappingDataSourceReadUsingId(ctx, d, meta, groupMappingId)
	} else if displayName != "" {
		return groupMappingDataSourceReadUsingDisplayName(ctx, d, meta, displayName)
	} else {
		return diag.Errorf("error reading Group Mapping: exactly one of %q or %q must be specified but they're both empty", paramId, paramDisplayName)
	}
}

func groupMappingDataSourceReadUsingDisplayName(ctx context.Context, d *schema.ResourceData, meta interface{}, displayName string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Group Mapping %q=%q", paramDisplayName, displayName))

	client := meta.(*Client)
	groupMappings, err := loadGroupMappings(ctx, client)
	if err != nil {
		return diag.Errorf("error reading Group Mapping %q: %s", displayName, createDescriptiveError(err))
	}
	if orgHasMultipleGroupMappingsWithTargetDisplayName(groupMappings, displayName) {
		return diag.Errorf("error reading Group Mapping: there are multiple Group Mappings with %q=%q", paramDisplayName, displayName)
	}
	for _, groupMapping := range groupMappings {
		if groupMapping.GetDisplayName() == displayName {
			if _, err := setGroupMappingAttributes(d, groupMapping); err != nil {
				return diag.FromErr(createDescriptiveError(err))
			}
			return nil
		}
	}

	return diag.Errorf("error reading Group Mapping: Group Mapping with %q=%q was not found", paramDisplayName, displayName)
}

func loadGroupMappings(ctx context.Context, c *Client) ([]v2.IamV2SsoGroupMapping, error) {
	groupMappings := make([]v2.IamV2SsoGroupMapping, 0)

	allGroupMappingsAreCollected := false
	pageToken := ""
	for !allGroupMappingsAreCollected {
		groupMappingPageList, _, err := executeListGroupMappings(ctx, c, pageToken)
		if err != nil {
			return nil, fmt.Errorf("error reading Group Mappings: %s", createDescriptiveError(err))
		}
		groupMappings = append(groupMappings, groupMappingPageList.GetData()...)

		// nextPageUrlStringNullable is nil for the last page
		nextPageUrlStringNullable := groupMappingPageList.GetMetadata().Next

		if nextPageUrlStringNullable.IsSet() {
			nextPageUrlString := *nextPageUrlStringNullable.Get()
			if nextPageUrlString == "" {
				allGroupMappingsAreCollected = true
			} else {
				pageToken, err = extractPageToken(nextPageUrlString)
				if err != nil {
					return nil, fmt.Errorf("error reading Group Mappings: %s", createDescriptiveError(err))
				}
			}
		} else {
			allGroupMappingsAreCollected = true
		}
	}
	return groupMappings, nil
}

func executeListGroupMappings(ctx context.Context, c *Client, pageToken string) (v2.IamV2SsoGroupMappingList, *http.Response, error) {
	if pageToken != "" {
		return c.ssoClient.GroupMappingsIamV2SsoApi.ListIamV2SsoGroupMappings(c.ssoApiContext(ctx)).PageSize(listGroupMappingsPageSize).PageToken(pageToken).Execute()
	} else {
		return c.ssoClient.GroupMappingsIamV2SsoApi.ListIamV2SsoGroupMappings(c.ssoApiContext(ctx)).PageSize(listGroupMappingsPageSize).Execute()
	}
}

func groupMappingDataSourceReadUsingId(ctx context.Context, d *schema.ResourceData, meta interface{}, groupMappingId string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Group Mapping %q=%q", paramId, groupMappingId), map[string]interface{}{groupMappingLoggingKey: groupMappingId})

	c := meta.(*Client)
	groupMapping, _, err := executeGroupMappingRead(c.ssoApiContext(ctx), c, groupMappingId)
	if err != nil {
		return diag.Errorf("error reading Group Mapping %q: %s", groupMappingId, createDescriptiveError(err))
	}
	groupMappingJson, err := json.Marshal(groupMapping)
	if err != nil {
		return diag.Errorf("error reading Group Mapping %q: error marshaling %#v to json: %s", groupMappingId, groupMapping, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Group Mapping %q: %s", groupMappingId, groupMappingJson), map[string]interface{}{groupMappingLoggingKey: groupMappingId})

	if _, err := setGroupMappingAttributes(d, groupMapping); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}

func orgHasMultipleGroupMappingsWithTargetDisplayName(groupMappings []v2.IamV2SsoGroupMapping, displayName string) bool {
	var numberOfGroupMappingsWithTargetDisplayName = 0
	for _, groupMapping := range groupMappings {
		if groupMapping.GetDisplayName() == displayName {
			numberOfGroupMappingsWithTargetDisplayName += 1
		}
	}
	return numberOfGroupMappingsWithTargetDisplayName > 1
}
