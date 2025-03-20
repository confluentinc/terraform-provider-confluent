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
	v2 "github.com/confluentinc/ccloud-sdk-go-v2/org/v2"
	v3 "github.com/confluentinc/ccloud-sdk-go-v2/srcm/v3"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strconv"
	"time"
)

const (
	paramClusters = "clusters"
)

func schemaRegistryClustersDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: schemaRegistryClustersDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramEnvironment: environmentParameterSchema(),
			paramClusters:    schemaRegistryClustersSchema(),
		},
	}
}

func schemaRegistryClustersSchema() *schema.Schema {
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
					Optional: true,
				},
				paramRegion: {
					Type:        schema.TypeString,
					Description: "The cloud service provider region where the cluster is running.",
					Computed:    true,
				},
				paramPackage: {
					Type:        schema.TypeString,
					Description: "The billing package.",
					Computed:    true,
				},
				paramRestEndpoint: {
					Type:        schema.TypeString,
					Description: "The API endpoint of the Schema Registry Cluster.",
					Computed:    true,
				},
				paramRestEndpointPrivate: {
					Type:        schema.TypeString,
					Description: "The private API endpoint of the Schema Registry Cluster.",
					Computed:    true,
				},
				paramRestEndpointPrivateRegional: {
					Type:        schema.TypeMap,
					Description: "The private regional API endpoint of the Schema Registry Cluster.",
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Computed: true,
				},
				paramCatalogEndpoint: {
					Type:        schema.TypeString,
					Description: "The catalog endpoint of the Schema Registry Cluster.",
					Computed:    true,
				},
				paramApiVersion: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "API Version defines the schema version of this representation of a Schema Registry Cluster.",
				},
				paramKind: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Kind defines the object Schema Registry Cluster represents.",
				},
				paramResourceName: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The Confluent Resource Name of the Schema Registry Cluster.",
				},
				paramCloud: {
					Type:        schema.TypeString,
					Description: "The cloud service provider in which the cluster is running.",
					Computed:    true,
				},
			},
		},
	}
}

// Customer can optionally specify the environmentId to query the SR clusters under it
func environmentParameterSchema() *schema.Schema {
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
		Optional: true,
		Computed: true,
		MaxItems: 1,
	}
}

func schemaRegistryClustersDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	tflog.Debug(ctx, "Reading Schema Registry Clusters")

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	clusters, err := loadAllSRClusters(ctx, environmentId, meta)
	if err != nil {
		return err
	}

	if err := d.Set(paramClusters, []interface{}{}); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	result := make([]interface{}, len(clusters))
	for i, cluster := range clusters {
		result[i] = populateSRClusterResult(cluster)
	}

	if err := d.Set(paramClusters, result); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return nil
}

func loadAllSRClusters(ctx context.Context, environmentId string, meta interface{}) ([]v3.SrcmV3Cluster, diag.Diagnostics) {
	var clusters []v3.SrcmV3Cluster
	var environments []v2.OrgV2Environment
	client := meta.(*Client)

	if environmentId != "" {
		environments = []v2.OrgV2Environment{{Id: ptr(environmentId)}}
	} else {
		var err error
		environments, err = loadEnvironments(ctx, client)
		if err != nil {
			return nil, diag.FromErr(createDescriptiveError(err))
		}
	}

	for _, environment := range environments {
		schemaRegistryClusters, err := loadSchemaRegistryClusters(ctx, client, environment.GetId())
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Error reading Schema Registry Clusters in Environment %q: %s", environment.GetId(), createDescriptiveError(err)))
			return nil, diag.FromErr(createDescriptiveError(err))
		}
		schemaRegistryClustersJson, err := json.Marshal(schemaRegistryClusters)
		if err != nil {
			return nil, diag.Errorf("error reading Schema Registry Clusters in Environment %q: error marshaling %#v to json: %s", environment.GetId(), schemaRegistryClusters, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Fetched Schema Registry Clusters in Environment %q: %s", environment.GetId(), schemaRegistryClustersJson))

		for _, schemaRegistryCluster := range schemaRegistryClusters {
			clusters = append(clusters, schemaRegistryCluster)
		}
	}

	return clusters, nil
}

func populateSRClusterResult(schemaRegistryCluster v3.SrcmV3Cluster) map[string]interface{} {
	env := make([]interface{}, 1)
	env[0] = map[string]interface{}{
		paramId: schemaRegistryCluster.Spec.Environment.GetId(),
	}

	return map[string]interface{}{
		paramId:                          schemaRegistryCluster.GetId(),
		paramDisplayName:                 schemaRegistryCluster.Spec.GetDisplayName(),
		paramEnvironment:                 env,
		paramPackage:                     schemaRegistryCluster.Spec.GetPackage(),
		paramRegion:                      schemaRegistryCluster.Spec.GetRegion(),
		paramCloud:                       schemaRegistryCluster.Spec.GetCloud(),
		paramKind:                        schemaRegistryCluster.GetKind(),
		paramApiVersion:                  schemaRegistryCluster.GetApiVersion(),
		paramRestEndpoint:                schemaRegistryCluster.Spec.GetHttpEndpoint(),
		paramRestEndpointPrivate:         schemaRegistryCluster.Spec.GetPrivateHttpEndpoint(),
		paramRestEndpointPrivateRegional: schemaRegistryCluster.Spec.PrivateNetworkingConfig.GetRegionalEndpoints(),
		paramCatalogEndpoint:             schemaRegistryCluster.Spec.GetCatalogHttpEndpoint(),
		paramResourceName:                schemaRegistryCluster.Metadata.GetResourceName(),
	}
}
