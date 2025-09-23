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
	fcpm "github.com/confluentinc/ccloud-sdk-go-v2/flink/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net/http"
)

const (
	// The maximum allowable page size - 1 (to avoid off-by-one errors) when listing service accounts using CMK V2 API
	// https://docs.confluent.io/cloud/current/api.html#operation/listNetworkingV1ComputePools
	listComputePoolsPageSize = 99
)

func computePoolDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: computePoolDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute, not both
				ExactlyOneOf: []string{paramId, paramDisplayName},
				Description:  "The ID of the Compute Pool, for example, `lfcp-abc123`.",
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
			paramMaxCfu: {
				Type:     schema.TypeInt,
				Computed: true,
			},
			paramApiVersion: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramKind: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramResourceName: {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func computePoolDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// ExactlyOneOf specified in the schema ensures one of paramId or paramDisplayName is specified.
	// The next step is to figure out which one exactly is set.
	computePoolId := d.Get(paramId).(string)
	displayName := d.Get(paramDisplayName).(string)

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	if computePoolId != "" {
		return computePoolDataSourceReadUsingId(ctx, d, meta, environmentId, computePoolId)
	} else if displayName != "" {
		return computePoolDataSourceReadUsingDisplayName(ctx, d, meta, environmentId, displayName)
	} else {
		return diag.Errorf("error reading Flink Compute Pool: exactly one of %q or %q must be specified but they're both empty", paramId, paramDisplayName)
	}
}

func computePoolDataSourceReadUsingDisplayName(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, displayName string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Flink Compute Pool %q=%q", paramDisplayName, displayName))

	c := meta.(*Client)
	computePools, err := loadComputePools(ctx, c, environmentId)
	if err != nil {
		return diag.Errorf("error reading Flink Compute Pool %q: %s", displayName, createDescriptiveError(err))
	}
	if orgHasMultipleComputePoolsWithTargetDisplayName(computePools, displayName) {
		return diag.Errorf("error reading Flink Compute Pool: there are multiple ComputePools with %q=%q", paramDisplayName, displayName)
	}

	for _, computePool := range computePools {
		if computePool.Spec.GetDisplayName() == displayName {
			if _, err := setComputePoolAttributes(d, computePool); err != nil {
				return diag.FromErr(createDescriptiveError(err))
			}
			return nil
		}
	}

	return diag.Errorf("error reading Flink Compute Pool: Flink Compute Pool with %q=%q was not found", paramDisplayName, displayName)
}

func computePoolDataSourceReadUsingId(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, computePoolId string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Flink Compute Pool %q=%q", paramId, computePoolId), map[string]interface{}{computePoolLoggingKey: computePoolId})

	c := meta.(*Client)
	computePool, resp, err := executeComputePoolRead(c.fcpmApiContext(ctx), c, environmentId, computePoolId)
	if err != nil {
		return diag.Errorf("error reading Flink Compute Pool %q: %s", computePoolId, createDescriptiveError(err, resp))
	}
	computePoolJson, err := json.Marshal(computePool)
	if err != nil {
		return diag.Errorf("error reading Flink Compute Pool %q: error marshaling %#v to json: %s", computePoolId, computePool, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Flink Compute Pool %q: %s", computePoolId, computePoolJson), map[string]interface{}{computePoolLoggingKey: computePoolId})

	if _, err := setComputePoolAttributes(d, computePool); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}

func orgHasMultipleComputePoolsWithTargetDisplayName(clusters []fcpm.FcpmV2ComputePool, displayName string) bool {
	var numberOfComputePoolsWithTargetDisplayName = 0
	for _, cluster := range clusters {
		if cluster.Spec.GetDisplayName() == displayName {
			numberOfComputePoolsWithTargetDisplayName += 1
		}
	}
	return numberOfComputePoolsWithTargetDisplayName > 1
}

func loadComputePools(ctx context.Context, c *Client, environmentId string) ([]fcpm.FcpmV2ComputePool, error) {
	computePools := make([]fcpm.FcpmV2ComputePool, 0)

	allComputePoolsAreCollected := false
	pageToken := ""
	for !allComputePoolsAreCollected {
		computePoolsPageList, resp, err := executeListComputePools(ctx, c, environmentId, pageToken)
		if err != nil {
			return nil, fmt.Errorf("error reading ComputePools: %s", createDescriptiveError(err, resp))
		}
		computePools = append(computePools, computePoolsPageList.GetData()...)

		// nextPageUrlStringNullable is nil for the last page
		nextPageUrlStringNullable := computePoolsPageList.GetMetadata().Next

		if nextPageUrlStringNullable.IsSet() {
			nextPageUrlString := *nextPageUrlStringNullable.Get()
			if nextPageUrlString == "" {
				allComputePoolsAreCollected = true
			} else {
				pageToken, err = extractPageToken(nextPageUrlString)
				if err != nil {
					return nil, fmt.Errorf("error reading ComputePools: %s", createDescriptiveError(err, resp))
				}
			}
		} else {
			allComputePoolsAreCollected = true
		}
	}
	return computePools, nil
}

func executeListComputePools(ctx context.Context, c *Client, environmentId, pageToken string) (fcpm.FcpmV2ComputePoolList, *http.Response, error) {
	if pageToken != "" {
		return c.fcpmClient.ComputePoolsFcpmV2Api.ListFcpmV2ComputePools(c.fcpmApiContext(ctx)).Environment(environmentId).PageSize(listComputePoolsPageSize).PageToken(pageToken).Execute()
	} else {
		return c.fcpmClient.ComputePoolsFcpmV2Api.ListFcpmV2ComputePools(c.fcpmApiContext(ctx)).Environment(environmentId).PageSize(listComputePoolsPageSize).Execute()
	}
}

func standardComputePoolDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 0,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{},
		},
	}
}
