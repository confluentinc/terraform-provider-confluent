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

func businessMetadataBindingDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: businessMetadataBindingDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramSchemaRegistryCluster: schemaRegistryClusterBlockDataSourceSchema(),
			paramRestEndpoint: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The REST endpoint of the Schema Registry cluster, for example, `https://psrc-00000.us-central1.gcp.confluent.cloud:443`).",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^http"), "the REST endpoint must start with 'https://'"),
			},
			paramCredentials: credentialsSchema(),
			paramBusinessMetadataName: {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9_\\s]*$"), "The name must not be empty and consist of a letter followed by a sequence of letter, number, space, or _ characters"),
				Description:  "The name of the business metadata.",
			},
			paramEntityName: {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "The qualified name of the entity.",
			},
			paramEntityType: {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "The entity type.",
			},
			paramId: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The id of the business metadata binding.",
			},
			paramAttributes: {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "The attributes.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func businessMetadataBindingDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractCatalogRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Business Metadata Binding: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Business Metadata Binding: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Business Metadata Binding: %s", createDescriptiveError(err))
	}

	businessMetadataName := d.Get(paramBusinessMetadataName).(string)
	entityName := d.Get(paramEntityName).(string)
	entityType := d.Get(paramEntityType).(string)
	businessMetadataBindingId := createBusinessMetadataBindingId(clusterId, businessMetadataName, entityName, entityType)

	tflog.Debug(ctx, fmt.Sprintf("Reading Business Metadata Binding %q=%q", paramId, businessMetadataBindingId), map[string]interface{}{businessMetadataBindingLoggingKey: businessMetadataBindingId})

	catalogRestClient := meta.(*Client).catalogRestClientFactory.CreateCatalogRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	request := catalogRestClient.apiClient.EntityV1Api.GetBusinessMetadata(catalogRestClient.dataCatalogApiContext(ctx), entityType, entityName)
	businessMetadataBindings, _, err := request.Execute()
	if err != nil {
		return diag.Errorf("error reading Business Metadata Binding %q: %s", businessMetadataBindingId, createDescriptiveError(err))
	}
	businessMetadataBinding, err := findBusinessMetadataBindingByBusinessMetadataName(businessMetadataBindings, businessMetadataName)
	if err != nil {
		return diag.Errorf("error reading Business Metadata Binding %q: %s", businessMetadataBindingId, "The binding information cannot be found")
	}

	businessMetadataBindingJson, err := json.Marshal(businessMetadataBinding)
	if err != nil {
		return diag.Errorf("error reading Business Metadata Binding %q: error marshaling %#v to json: %s", businessMetadataBindingId, businessMetadataBinding, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Business Metadata Binding %q: %s", businessMetadataBindingId, businessMetadataBindingJson), map[string]interface{}{businessMetadataBindingLoggingKey: businessMetadataBindingId})

	if _, err := setBusinessMetadataBindingAttributes(d, clusterId, businessMetadataBinding); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	return nil
}
