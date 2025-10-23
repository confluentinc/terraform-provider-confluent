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
	"fmt"
	v2 "github.com/confluentinc/ccloud-sdk-go-v2/cmk/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strconv"
	"time"
)

func kafkaClustersDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: kafkaClustersDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramEnvironment: environmentDataSourceSchema(),
			paramClusters:    kafkaClustersSchema(),
		},
	}
}

func kafkaClustersSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:     schema.TypeString,
					Computed: true,
				},
				paramEnvironment: environmentDataSourceSchema(),
				paramDisplayName: {
					Type:     schema.TypeString,
					Computed: true,
				},
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
		},
	}
}

func kafkaClustersDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Reading Kafka Clusters")

	c := meta.(*Client)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	clusters, err := loadKafkaClusters(ctx, c, environmentId)
	if err != nil {
		return diag.Errorf("error reading Kafka Clusters: %s", createDescriptiveError(err))
	}

	result := make([]interface{}, len(clusters))
	for i, cluster := range clusters {
		result[i], err = populateKafkaClusterResult(cluster)
		if err != nil {
			return diag.Errorf("error reading Kafka Clusters: %s", createDescriptiveError(err))
		}
	}
	if err := d.Set(paramClusters, result); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return nil
}

func populateKafkaClusterResult(cluster v2.CmkV2Cluster) (map[string]interface{}, error) {
	env := make([]interface{}, 1)
	env[0] = map[string]interface{}{
		paramId: cluster.Spec.Environment.GetId(),
	}

	network := make([]interface{}, 1)
	network[0] = map[string]interface{}{
		paramId: cluster.Spec.Network.GetId(),
	}

	byok := make([]interface{}, 1)
	byok[0] = map[string]interface{}{
		paramId: cluster.Spec.Byok.GetId(),
	}

	rbacCrn, err := clusterCrnToRbacClusterCrn(cluster.Metadata.GetResourceName())
	if err != nil {
		return nil, fmt.Errorf("error reading Kafka Clusters: could not construct %s", paramRbacCrn)
	}

	mp := map[string]interface{}{
		paramId:           cluster.GetId(),
		paramApiVersion:   cluster.GetApiVersion(),
		paramKind:         cluster.GetKind(),
		paramDisplayName:  cluster.Spec.GetDisplayName(),
		paramAvailability: cluster.Spec.GetAvailability(),
		paramCloud:        cluster.Spec.GetCloud(),
		paramRegion:       cluster.Spec.GetRegion(),
		// Reset all 5 cluster types since only one of these 5 should be set
		paramBasicCluster:      []interface{}{},
		paramStandardCluster:   []interface{}{},
		paramDedicatedCluster:  []interface{}{},
		paramEnterpriseCluster: []interface{}{},
		paramFreightCluster:    []interface{}{},

		paramBootStrapEndpoint: cluster.Spec.GetKafkaBootstrapEndpoint(),
		paramRestEndpoint:      cluster.Spec.GetHttpEndpoint(),
		paramRbacCrn:           rbacCrn,

		paramEnvironment:          env,
		paramNetwork:              network,
		paramConfluentCustomerKey: byok,

		paramEndpoints: constructEndpointsBlockValue(cluster.Spec.GetEndpoints()),
	}
	// Set a specific cluster type
	if cluster.Spec.Config.CmkV2Basic != nil {
		mp[paramBasicCluster] = []interface{}{make(map[string]string)}
	} else if cluster.Spec.Config.CmkV2Standard != nil {
		mp[paramStandardCluster] = []interface{}{make(map[string]string)}
	} else if cluster.Spec.Config.CmkV2Dedicated != nil {
		mp[paramDedicatedCluster] = []interface{}{map[string]interface{}{
			paramCku:           cluster.Status.GetCku(),
			paramEncryptionKey: cluster.Spec.Config.CmkV2Dedicated.GetEncryptionKey(),
			paramZones:         cluster.Spec.Config.CmkV2Dedicated.GetZones(),
		}}
	} else if cluster.Spec.Config.CmkV2Enterprise != nil {
		mp[paramEnterpriseCluster] = []interface{}{make(map[string]string)}
	} else if cluster.Spec.Config.CmkV2Freight != nil {
		mp[paramFreightCluster] = []interface{}{map[string]interface{}{
			paramZones: cluster.Spec.Config.CmkV2Freight.GetZones(),
		}}
	}
	return mp, nil
}
