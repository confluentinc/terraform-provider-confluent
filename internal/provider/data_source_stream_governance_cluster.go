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
	v2 "github.com/confluentinc/ccloud-sdk-go-v2/stream-governance/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net/http"
)

const (
	// The maximum allowable page size - 1 (to avoid off-by-one errors) when listing service accounts using SG V2 API
	// https://docs.confluent.io/cloud/current/api.html#operation/listStreamGovernanceV2Clusters
	listStreamGovernanceClustersPageSize = 99
)

func streamGovernanceClusterDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: streamGovernanceDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute, not both
				ExactlyOneOf: []string{paramId, paramDisplayName},
				Description:  "The ID of the Stream Governance cluster, for example, `lsrc-755ogo`.",
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
			paramRegion: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						paramId: {
							Type:        schema.TypeString,
							Description: "The unique identifier for the Stream Governance Region.",
							Computed:    true,
						},
					},
				},
				Computed: true,
			},
			paramPackage: {
				Type:        schema.TypeString,
				Description: "The billing package.",
				Computed:    true,
			},
			paramHttpEndpoint: {
				Type:        schema.TypeString,
				Description: "The API endpoint of the Stream Governance Cluster.",
				Computed:    true,
			},
			paramApiVersion: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "API Version defines the schema version of this representation of a Stream Governance Cluster.",
			},
			paramKind: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Kind defines the object Stream Governance Cluster represents.",
			},
			paramResourceName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Confluent Resource Name of the Stream Governance Cluster.",
			},
		},
	}
}

func streamGovernanceDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// ExactlyOneOf specified in the schema ensures one of paramId or paramDisplayName is specified.
	// The next step is to figure out which one exactly is set.
	clusterId := d.Get(paramId).(string)
	displayName := d.Get(paramDisplayName).(string)

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	if clusterId != "" {
		return streamGovernanceDataSourceReadUsingId(ctx, d, meta, environmentId, clusterId)
	} else if displayName != "" {
		return streamGovernanceDataSourceReadUsingDisplayName(ctx, d, meta, environmentId, displayName)
	} else {
		return diag.Errorf("error reading Stream Governance Cluster: exactly one of %q or %q must be specified but they're both empty", paramId, paramDisplayName)
	}
}

func streamGovernanceDataSourceReadUsingDisplayName(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, displayName string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading StreamGovernance Cluster %q=%q", paramDisplayName, displayName))

	c := meta.(*Client)
	streamGovernanceClusters, err := loadStreamGovernanceClusters(ctx, c, environmentId)
	if err != nil {
		return diag.Errorf("error reading Stream Governance Cluster %q: %s", displayName, createDescriptiveError(err))
	}
	if orgHasMultipleStreamGovernanceClustersWithTargetDisplayName(streamGovernanceClusters, displayName) {
		return diag.Errorf("error reading Stream Governance Cluster: there are multiple StreamGovernance Clusters with %q=%q", paramDisplayName, displayName)
	}

	for _, cluster := range streamGovernanceClusters {
		if cluster.Spec.GetDisplayName() == displayName {
			if _, err := setStreamGovernanceClusterAttributes(d, cluster); err != nil {
				return diag.FromErr(createDescriptiveError(err))
			}
			return nil
		}
	}

	return diag.Errorf("error reading Stream Governance Cluster: StreamGovernance Cluster with %q=%q was not found", paramDisplayName, displayName)
}

func streamGovernanceDataSourceReadUsingId(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, clusterId string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading StreamGovernance Cluster %q=%q", paramId, clusterId), map[string]interface{}{streamGovernanceClusterLoggingKey: clusterId})

	c := meta.(*Client)
	cluster, _, err := executeStreamGovernanceClusterRead(c.sgApiContext(ctx), c, environmentId, clusterId)
	if err != nil {
		return diag.Errorf("error reading Stream Governance Cluster %q: %s", clusterId, createDescriptiveError(err))
	}
	clusterJson, err := json.Marshal(cluster)
	if err != nil {
		return diag.Errorf("error reading Stream Governance Cluster %q: error marshaling %#v to json: %s", clusterId, cluster, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched StreamGovernance Cluster %q: %s", clusterId, clusterJson), map[string]interface{}{streamGovernanceClusterLoggingKey: clusterId})

	if _, err := setStreamGovernanceClusterAttributes(d, cluster); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}

func orgHasMultipleStreamGovernanceClustersWithTargetDisplayName(clusters []v2.StreamGovernanceV2Cluster, displayName string) bool {
	var numberOfClustersWithTargetDisplayName = 0
	for _, cluster := range clusters {
		if cluster.Spec.GetDisplayName() == displayName {
			numberOfClustersWithTargetDisplayName += 1
		}
	}
	return numberOfClustersWithTargetDisplayName > 1
}

func loadStreamGovernanceClusters(ctx context.Context, c *Client, environmentId string) ([]v2.StreamGovernanceV2Cluster, error) {
	clusters := make([]v2.StreamGovernanceV2Cluster, 0)

	allClustersAreCollected := false
	pageToken := ""
	for !allClustersAreCollected {
		clustersPageList, _, err := executeListStreamGovernanceClusters(ctx, c, environmentId, pageToken)
		if err != nil {
			return nil, fmt.Errorf("error reading Stream Governance Clusters: %s", createDescriptiveError(err))
		}
		clusters = append(clusters, clustersPageList.GetData()...)

		// nextPageUrlStringNullable is nil for the last page
		nextPageUrlStringNullable := clustersPageList.GetMetadata().Next

		if nextPageUrlStringNullable.IsSet() {
			nextPageUrlString := *nextPageUrlStringNullable.Get()
			if nextPageUrlString == "" {
				allClustersAreCollected = true
			} else {
				pageToken, err = extractPageToken(nextPageUrlString)
				if err != nil {
					return nil, fmt.Errorf("error reading Stream Governance Clusters: %s", createDescriptiveError(err))
				}
			}
		} else {
			allClustersAreCollected = true
		}
	}
	return clusters, nil
}

func executeListStreamGovernanceClusters(ctx context.Context, c *Client, environmentId, pageToken string) (v2.StreamGovernanceV2ClusterList, *http.Response, error) {
	if pageToken != "" {
		return c.sgClient.ClustersStreamGovernanceV2Api.ListStreamGovernanceV2Clusters(c.sgApiContext(ctx)).Environment(environmentId).PageSize(listStreamGovernanceClustersPageSize).PageToken(pageToken).Execute()
	} else {
		return c.sgClient.ClustersStreamGovernanceV2Api.ListStreamGovernanceV2Clusters(c.sgApiContext(ctx)).Environment(environmentId).PageSize(listStreamGovernanceClustersPageSize).Execute()
	}
}
