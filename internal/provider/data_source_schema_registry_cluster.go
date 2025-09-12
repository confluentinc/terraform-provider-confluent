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
	v3 "github.com/confluentinc/ccloud-sdk-go-v2/srcm/v3"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net/http"
)

const (
	// The maximum allowable page size - 1 (to avoid off-by-one errors) when listing service accounts using SG V3 API
	// https://docs.confluent.io/cloud/current/api.html#operation/listSrcmV3Clusters
	listSchemaRegistryClustersPageSize = 99
)

const (
	paramPackage             = "package"
	billingPackageEssentials = "ESSENTIALS"
	billingPackageAdvanced   = "ADVANCED"
)

var acceptedBillingPackages = []string{billingPackageEssentials, billingPackageAdvanced}

func schemaRegistryClusterDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: schemaRegistryDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "The ID of the Schema Registry cluster, for example, `lsrc-755ogo`.",
			},
			// Similarly, paramEnvironment is required as well
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
				Deprecated:  `Please use the private_regional_rest_endpoints attribute instead, which supersedes the private_rest_endpoint attribute.`,
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
	}
}

func schemaRegistryDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	clusterId := d.Get(paramId).(string)
	displayName := d.Get(paramDisplayName).(string)

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	if clusterId != "" {
		return schemaRegistryDataSourceReadUsingId(ctx, d, meta, environmentId, clusterId)
	} else if displayName != "" {
		return schemaRegistryDataSourceReadUsingDisplayName(ctx, d, meta, environmentId, displayName)
	} else {
		// There is at most 1 SR cluster per Environment
		c := meta.(*Client)
		schemaRegistryClusters, err := loadSchemaRegistryClusters(ctx, c, environmentId)
		if err != nil {
			return diag.Errorf("error reading Schema Registry Clusters: %s", createDescriptiveError(err))
		}
		if len(schemaRegistryClusters) == 0 {
			return diag.Errorf("error reading Schema Registry Clusters: there are no SR clusters in %q environment", environmentId)
		}
		if len(schemaRegistryClusters) != 1 {
			return diag.Errorf("error reading Schema Registry Clusters: there are multiple SR clusters in %q environment. "+
				"Please specify %q or %q", environmentId, paramId, paramDisplayName)
		}
		clusterId = schemaRegistryClusters[0].GetId()
		return schemaRegistryDataSourceReadUsingId(ctx, d, meta, environmentId, clusterId)
	}
}

func schemaRegistryDataSourceReadUsingDisplayName(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, displayName string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading SchemaRegistry Cluster %q=%q", paramDisplayName, displayName))

	c := meta.(*Client)
	schemaRegistryClusters, err := loadSchemaRegistryClusters(ctx, c, environmentId)
	if err != nil {
		return diag.Errorf("error reading Schema Registry Cluster %q: %s", displayName, createDescriptiveError(err))
	}
	if orgHasMultipleSchemaRegistryClustersWithTargetDisplayName(schemaRegistryClusters, displayName) {
		return diag.Errorf("error reading Schema Registry Cluster: there are multiple SchemaRegistry Clusters with %q=%q", paramDisplayName, displayName)
	}

	for _, cluster := range schemaRegistryClusters {
		if cluster.Spec.GetDisplayName() == displayName {
			if _, err := setSchemaRegistryClusterAttributes(d, cluster); err != nil {
				return diag.FromErr(createDescriptiveError(err))
			}
			return nil
		}
	}

	return diag.Errorf("error reading Schema Registry Cluster: SchemaRegistry Cluster with %q=%q was not found", paramDisplayName, displayName)
}

func setSchemaRegistryClusterAttributes(d *schema.ResourceData, schemaRegistryCluster v3.SrcmV3Cluster) (*schema.ResourceData, error) {
	if err := d.Set(paramPackage, schemaRegistryCluster.Spec.GetPackage()); err != nil {
		return nil, err
	}

	// Set blocks
	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, schemaRegistryCluster.Spec.Environment.GetId(), d); err != nil {
		return nil, err
	}

	// Set computed attributes
	if err := d.Set(paramDisplayName, schemaRegistryCluster.Spec.GetDisplayName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramRestEndpoint, schemaRegistryCluster.Spec.GetHttpEndpoint()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramRestEndpointPrivate, schemaRegistryCluster.Spec.GetPrivateHttpEndpoint()); err != nil {
		return nil, createDescriptiveError(err)
	}
	config := schemaRegistryCluster.Spec.GetPrivateNetworkingConfig()
	if err := d.Set(paramRestEndpointPrivateRegional, config.GetRegionalEndpoints()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramCatalogEndpoint, schemaRegistryCluster.Spec.GetCatalogHttpEndpoint()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramApiVersion, schemaRegistryCluster.GetApiVersion()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramKind, schemaRegistryCluster.GetKind()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramResourceName, schemaRegistryCluster.Metadata.GetResourceName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	// Region now is a primitive string instead of list block of size = 1
	if err := d.Set(paramRegion, schemaRegistryCluster.Spec.GetRegion()); err != nil {
		return nil, err
	}
	if err := d.Set(paramCloud, schemaRegistryCluster.Spec.GetCloud()); err != nil {
		return nil, err
	}
	d.SetId(schemaRegistryCluster.GetId())
	return d, nil
}

func schemaRegistryDataSourceReadUsingId(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, clusterId string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading SchemaRegistry Cluster %q=%q", paramId, clusterId), map[string]interface{}{schemaRegistryClusterLoggingKey: clusterId})

	c := meta.(*Client)
	cluster, resp, err := executeSchemaRegistryClusterRead(c.srcmApiContext(ctx), c, environmentId, clusterId)
	if err != nil {
		return diag.Errorf("error reading Schema Registry Cluster %q: %s", clusterId, createDescriptiveError(err, resp))
	}
	clusterJson, err := json.Marshal(cluster)
	if err != nil {
		return diag.Errorf("error reading Schema Registry Cluster %q: error marshaling %#v to json: %s", clusterId, cluster, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched SchemaRegistry Cluster %q: %s", clusterId, clusterJson), map[string]interface{}{schemaRegistryClusterLoggingKey: clusterId})

	if _, err := setSchemaRegistryClusterAttributes(d, cluster); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}

func orgHasMultipleSchemaRegistryClustersWithTargetDisplayName(clusters []v3.SrcmV3Cluster, displayName string) bool {
	var numberOfClustersWithTargetDisplayName = 0
	for _, cluster := range clusters {
		if cluster.Spec.GetDisplayName() == displayName {
			numberOfClustersWithTargetDisplayName += 1
		}
	}
	return numberOfClustersWithTargetDisplayName > 1
}

func loadSchemaRegistryClusters(ctx context.Context, c *Client, environmentId string) ([]v3.SrcmV3Cluster, error) {
	clusters := make([]v3.SrcmV3Cluster, 0)

	allClustersAreCollected := false
	pageToken := ""
	for !allClustersAreCollected {
		clustersPageList, resp, err := executeListSchemaRegistryClusters(ctx, c, environmentId, pageToken)
		if err != nil {
			return nil, fmt.Errorf("error reading Schema Registry Clusters: %s", createDescriptiveError(err, resp))
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
					return nil, fmt.Errorf("error reading Schema Registry Clusters: %s", createDescriptiveError(err, resp))
				}
			}
		} else {
			allClustersAreCollected = true
		}
	}
	return clusters, nil
}

func executeListSchemaRegistryClusters(ctx context.Context, c *Client, environmentId, pageToken string) (v3.SrcmV3ClusterList, *http.Response, error) {
	if pageToken != "" {
		return c.srcmClient.ClustersSrcmV3Api.ListSrcmV3Clusters(c.srcmApiContext(ctx)).Environment(environmentId).PageSize(listSchemaRegistryClustersPageSize).PageToken(pageToken).Execute()
	} else {
		return c.srcmClient.ClustersSrcmV3Api.ListSrcmV3Clusters(c.srcmApiContext(ctx)).Environment(environmentId).PageSize(listSchemaRegistryClustersPageSize).Execute()
	}
}

func executeSchemaRegistryClusterRead(ctx context.Context, c *Client, environmentId string, schemaRegistryClusterId string) (v3.SrcmV3Cluster, *http.Response, error) {
	req := c.srcmClient.ClustersSrcmV3Api.GetSrcmV3Cluster(c.srcmApiContext(ctx), schemaRegistryClusterId).Environment(environmentId)
	return req.Execute()
}
