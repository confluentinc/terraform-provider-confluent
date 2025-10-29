// Copyright 2025 Confluent Inc. All Rights Reserved.
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
)

func catalogIntegrationDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: catalogIntegrationDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the Catalog Integration, for example, `tci-abc123`.",
			},
			paramDisplayName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the catalog integration.",
			},
			paramSuspended: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Indicates whether the Catalog Integration should be suspended.",
			},
			paramKafkaCluster: requiredKafkaClusterDataSourceSchema(),
			paramEnvironment:  environmentDataSourceSchema(),
			paramCredentials:  credentialsSchema(),
			paramAwsGlue:      awsGlueDataSourceSchema(),
			paramSnowflake:    snowflakeDataSourceSchema(),
			paramUnity:        unityDataSourceSchema(),
		},
	}
}

func catalogIntegrationDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if err := dataSourceCredentialBlockValidationWithOAuth(d, meta.(*Client).isOAuthEnabled); err != nil {
		return diag.Errorf("error reading Catalog Integration: %s", createDescriptiveError(err))
	}

	catalogIntegrationId := d.Get(paramId).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	clusterId := extractStringValueFromBlock(d, paramKafkaCluster, paramId)

	tflog.Debug(ctx, fmt.Sprintf("Reading Catalog Integration %q=%q", paramId, catalogIntegrationId), map[string]interface{}{catalogIntegrationKey: catalogIntegrationId})

	c := meta.(*Client)

	tableflowApiKey, tableflowApiSecret, err := extractTableflowApiKeyAndApiSecret(c, d, false)
	if err != nil {
		return diag.Errorf("error reading Catalog Integration: %s", createDescriptiveError(err))
	}
	tableflowRestClient := c.tableflowRestClientFactory.CreateTableflowRestClient(tableflowApiKey, tableflowApiSecret, c.isTableflowMetadataSet, c.oauthToken, c.stsToken)

	req := tableflowRestClient.apiClient.CatalogIntegrationsTableflowV1Api.GetTableflowV1CatalogIntegration(tableflowRestClient.apiContext(ctx), catalogIntegrationId).Environment(environmentId).SpecKafkaCluster(clusterId)
	catalogIntegration, resp, err := req.Execute()
	if err != nil {
		return diag.Errorf("error reading Catalog Integration %q: %s", catalogIntegrationId, createDescriptiveError(err, resp))
	}
	catalogIntegrationJson, err := json.Marshal(catalogIntegration)
	if err != nil {
		return diag.Errorf("error reading Catalog Integration %q: error marshaling %#v to json: %s", catalogIntegrationId, catalogIntegration, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Catalog Integration %q: %s", catalogIntegrationId, catalogIntegrationJson), map[string]interface{}{catalogIntegrationKey: catalogIntegrationId})

	if _, err := setCatalogIntegrationAttributes(d, tableflowRestClient, catalogIntegration); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}

func awsGlueDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramProviderIntegrationId: {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
		Computed: true,
	}
}

func snowflakeDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		MaxItems: 0,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramEndpoint: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The catalog integration connection endpoint for Snowflake Open Catalog.",
				},
				paramWarehouse: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Warehouse name of the Snowflake Open Catalog.",
				},
				paramAllowedScope: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Allowed scope of the Snowflake Open Catalog.",
				},
			},
		},
		Computed: true,
	}
}

func unityDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramWorkspaceEndpoint: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The Databricks workspace URL associated with the Unity Catalog.",
				},
				paramCatalogName: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The name of the catalog within Unity Catalog.",
				},
			},
		},
		Computed: true,
	}
}
