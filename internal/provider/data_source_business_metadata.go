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
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"regexp"
)

func businessMetadataDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: businessMetadataDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramSchemaRegistryCluster: schemaRegistryClusterBlockDataSourceSchema(),
			paramRestEndpoint: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The REST endpoint of the Schema Registry cluster, for example, `https://psrc-00000.us-central1.gcp.confluent.cloud:443`).",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^http"), "the REST endpoint must start with 'https://'"),
			},
			paramCredentials: credentialsSchema(),
			paramName: {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9_\\s]*$"), "The name must not be empty and consist of a letter followed by a sequence of letter, number, space, or _ characters"),
				Description:  "The name of the Business Metadata.",
			},
			paramDescription: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The description of the Business Metadata.",
			},
			paramId: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The id of the Business Metadata.",
			},
			paramVersion: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The version.",
			},
			paramAttributeDef: attributeDefsDataSourceSchema(),
		},
	}
}

func attributeDefsDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramName: {
					Type:     schema.TypeString,
					Computed: true,
				},
				paramType: {
					Type:     schema.TypeString,
					Computed: true,
				},
				paramIsOptional: {
					Type:     schema.TypeBool,
					Computed: true,
				},
				paramDefaultValue: {
					Type:     schema.TypeString,
					Computed: true,
				},
				paramDescription: {
					Type:     schema.TypeString,
					Computed: true,
				},
				paramOptions: {
					Type:     schema.TypeMap,
					Computed: true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
			},
		},
	}
}

func businessMetadataDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if err := dataSourceCredentialBlockValidationWithOAuth(d, meta.(*Client).isOAuthEnabled); err != nil {
		return diag.Errorf("error reading Business Metadata: %s", createDescriptiveError(err))
	}
	restEndpoint, err := extractCatalogRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Business Metadata: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Business Metadata: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Business Metadata: %s", createDescriptiveError(err))
	}
	businessMetadataName := d.Get(paramName).(string)

	tflog.Debug(ctx, fmt.Sprintf("Reading Business Metadata %q", businessMetadataName), map[string]interface{}{businessMetadataLoggingKey: createBusinessMetadataId(clusterId, businessMetadataName)})

	return businessMetadataDataSourceReadUsingName(ctx, d, meta, restEndpoint, clusterId, clusterApiKey, clusterApiSecret, businessMetadataName)
}

func businessMetadataDataSourceReadUsingName(ctx context.Context, d *schema.ResourceData, meta interface{}, restEndpoint string, clusterId string, clusterApiKey string, clusterApiSecret string, businessMetadataName string) diag.Diagnostics {
	catalogRestClient := meta.(*Client).catalogRestClientFactory.CreateCatalogRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet, meta.(*Client).oauthToken)
	request := catalogRestClient.apiClient.TypesV1Api.GetBusinessMetadataDefByName(catalogRestClient.dataCatalogApiContext(ctx), businessMetadataName)
	businessMetadata, resp, err := request.Execute()
	businessMetadataId := createBusinessMetadataId(clusterId, businessMetadataName)

	if err != nil {
		return diag.Errorf("error reading Business Metadata %q: %s", businessMetadataId, createDescriptiveError(err, resp))
	}
	businessMetadataJson, err := json.Marshal(businessMetadata)
	if err != nil {
		return diag.Errorf("error reading Business Metadata %q: error marshaling %#v to json: %s", businessMetadataId, businessMetadata, createDescriptiveError(err, resp))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Business Metadata %q: %s", businessMetadataId, businessMetadataJson), map[string]interface{}{businessMetadataLoggingKey: businessMetadataId})

	if _, err := setBusinessMetadataAttributes(d, clusterId, businessMetadata); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}
