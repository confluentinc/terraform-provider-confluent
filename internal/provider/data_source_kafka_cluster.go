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
	v2 "github.com/confluentinc/ccloud-sdk-go-v2/cmk/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net/http"
)

const (
	// The maximum allowable page size - 1 (to avoid off-by-one errors) when listing service accounts using CMK V2 API
	// https://docs.confluent.io/cloud/current/api.html#operation/listCmkV2Clusters
	listKafkaClustersPageSize = 99
)

func kafkaDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: kafkaDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute, not both
				ExactlyOneOf: []string{paramId, paramDisplayName},
				Description:  "The ID of the Kafka cluster, for example, `lkc-abc123`.",
			},
			// Similarly, paramEnvironment is required as well
			paramEnvironment:          environmentDataSourceSchema(),
			paramNetwork:              optionalNetworkDataSourceSchema(),
			paramConfluentCustomerKey: optionalByokDataSourceSchema(),
			paramApiVersion: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramKind: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramDisplayName: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute, not both
				ExactlyOneOf: []string{paramId, paramDisplayName},
			},
			paramAvailability: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramCloud: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramRegion: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramBasicCluster:      basicClusterDataSourceSchema(),
			paramStandardCluster:   standardClusterDataSourceSchema(),
			paramDedicatedCluster:  dedicatedClusterDataSourceSchema(),
			paramEnterpriseCluster: enterpriseClusterDataSourceSchema(),
			paramFreightCluster:    freightClusterDataSourceSchema(),
			paramBootStrapEndpoint: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramRestEndpoint: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramRbacCrn: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramEndpoints: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						paramAccessPointID: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The access point ID (e.g., 'public', 'privatelink').",
						},
						paramBootStrapEndpoint: {
							Type:     schema.TypeString,
							Computed: true,
						},
						paramRestEndpoint: {
							Type:     schema.TypeString,
							Computed: true,
						},
						paramConnectionType: {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
				Computed:    true,
				Description: "A map of endpoints for connecting to the Kafka cluster, keyed by access_point_id. Access Point ID 'public' and 'privatelink' are reserved. These can be used for different network access methods or regions.",
			},
		},
	}
}

func kafkaDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// ExactlyOneOf specified in the schema ensures one of paramId or paramDisplayName is specified.
	// The next step is to figure out which one exactly is set.
	clusterId := d.Get(paramId).(string)
	displayName := d.Get(paramDisplayName).(string)

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	if clusterId != "" {
		return kafkaDataSourceReadUsingId(ctx, d, meta, environmentId, clusterId)
	} else if displayName != "" {
		return kafkaDataSourceReadUsingDisplayName(ctx, d, meta, environmentId, displayName)
	} else {
		return diag.Errorf("error reading Kafka Cluster: exactly one of %q or %q must be specified but they're both empty", paramId, paramDisplayName)
	}
}

func kafkaDataSourceReadUsingDisplayName(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, displayName string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Kafka Cluster %q=%q", paramDisplayName, displayName))

	c := meta.(*Client)
	kafkaClusters, err := loadKafkaClusters(ctx, c, environmentId)
	if err != nil {
		return diag.Errorf("error reading Kafka Cluster %q: %s", displayName, createDescriptiveError(err))
	}
	if orgHasMultipleKafkaClustersWithTargetDisplayName(kafkaClusters, displayName) {
		return diag.Errorf("error reading Kafka Cluster: there are multiple Kafka Clusters with %q=%q", paramDisplayName, displayName)
	}

	for _, cluster := range kafkaClusters {
		if cluster.Spec.GetDisplayName() == displayName {
			if _, err := setKafkaClusterAttributes(d, cluster); err != nil {
				return diag.FromErr(createDescriptiveError(err))
			}
			return nil
		}
	}

	return diag.Errorf("error reading Kafka Cluster: Kafka Cluster with %q=%q was not found", paramDisplayName, displayName)
}

func kafkaDataSourceReadUsingId(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, clusterId string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Kafka Cluster %q=%q", paramId, clusterId), map[string]interface{}{kafkaClusterLoggingKey: clusterId})

	c := meta.(*Client)
	cluster, resp, err := executeKafkaRead(c.cmkApiContext(ctx), c, environmentId, clusterId)
	if err != nil {
		return diag.Errorf("error reading Kafka Cluster %q: %s", clusterId, createDescriptiveError(err, resp))
	}
	clusterJson, err := json.Marshal(cluster)
	if err != nil {
		return diag.Errorf("error reading Kafka Cluster %q: error marshaling %#v to json: %s", clusterId, cluster, createDescriptiveError(err, resp))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Kafka Cluster %q: %s", clusterId, clusterJson), map[string]interface{}{kafkaClusterLoggingKey: clusterId})

	if _, err := setKafkaClusterAttributes(d, cluster); err != nil {
		return diag.FromErr(createDescriptiveError(err, resp))
	}
	return nil
}

func orgHasMultipleKafkaClustersWithTargetDisplayName(clusters []v2.CmkV2Cluster, displayName string) bool {
	var numberOfClustersWithTargetDisplayName = 0
	for _, cluster := range clusters {
		if cluster.Spec.GetDisplayName() == displayName {
			numberOfClustersWithTargetDisplayName += 1
		}
	}
	return numberOfClustersWithTargetDisplayName > 1
}

func loadKafkaClusters(ctx context.Context, c *Client, environmentId string) ([]v2.CmkV2Cluster, error) {
	clusters := make([]v2.CmkV2Cluster, 0)

	allClustersAreCollected := false
	pageToken := ""
	for !allClustersAreCollected {
		clustersPageList, resp, err := executeListKafkaClusters(ctx, c, environmentId, pageToken)
		if err != nil {
			return nil, fmt.Errorf("error reading Kafka Clusters: %s", createDescriptiveError(err, resp))
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
					return nil, fmt.Errorf("error reading Kafka Clusters: %s", createDescriptiveError(err, resp))
				}
			}
		} else {
			allClustersAreCollected = true
		}
	}
	return clusters, nil
}

func executeListKafkaClusters(ctx context.Context, c *Client, environmentId, pageToken string) (v2.CmkV2ClusterList, *http.Response, error) {
	if pageToken != "" {
		return c.cmkClient.ClustersCmkV2Api.ListCmkV2Clusters(c.cmkApiContext(ctx)).Environment(environmentId).PageSize(listKafkaClustersPageSize).PageToken(pageToken).Execute()
	} else {
		return c.cmkClient.ClustersCmkV2Api.ListCmkV2Clusters(c.cmkApiContext(ctx)).Environment(environmentId).PageSize(listKafkaClustersPageSize).Execute()
	}
}

func basicClusterDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 0,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{},
		},
	}
}

func standardClusterDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 0,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{},
		},
	}
}

func dedicatedClusterDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramCku: {
					Type:        schema.TypeInt,
					Computed:    true,
					Description: "The number of Confluent Kafka Units (CKUs) for Dedicated cluster types. MULTI_ZONE dedicated clusters must have at least two CKUs.",
				},
				paramEncryptionKey: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The ID of the encryption key that is used to encrypt the data in the Kafka cluster.",
				},
				paramZones: {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Computed:    true,
					Description: "The list of zones the cluster is in.",
				},
			},
		},
	}
}

func enterpriseClusterDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 0,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{},
		},
	}
}

func freightClusterDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 0,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramZones: {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Computed:    true,
					Description: "The list of zones the cluster is in.",
				},
			},
		},
	}
}
