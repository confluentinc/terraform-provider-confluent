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

const (
	paramEntityTypes = "entity_types"
)

func tagDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: tagDataSourceRead,
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
				Description:  "The name of the tag to be created.",
			},
			paramDescription: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The description of the tag to be created.",
			},
			paramEntityTypes: {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
				Description: "The entity types of the tag to be created.",
			},
			paramId: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The id of the tag to be created.",
			},
			paramVersion: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The version.",
			},
		},
	}
}

func tagDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractCatalogRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Tag: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Tag: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Tag: %s", createDescriptiveError(err))
	}
	tagName := d.Get(paramName).(string)

	tflog.Debug(ctx, fmt.Sprintf("Reading Tag %q", tagName), map[string]interface{}{tagLoggingKey: createTagId(clusterId, tagName)})

	return tagDataSourceReadUsingTagName(ctx, d, meta, restEndpoint, clusterId, clusterApiKey, clusterApiSecret, tagName)
}

func tagDataSourceReadUsingTagName(ctx context.Context, d *schema.ResourceData, meta interface{}, restEndpoint string, clusterId string, clusterApiKey string, clusterApiSecret string, tagName string) diag.Diagnostics {
	catalogRestClient := meta.(*Client).catalogRestClientFactory.CreateCatalogRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	request := catalogRestClient.dataCatalogApiClient.TypesV1Api.GetTagDefByName(catalogRestClient.dataCatalogApiContext(ctx), tagName)
	tag, _, err := request.Execute()
	tagId := createTagId(clusterId, tagName)

	if err != nil {
		return diag.Errorf("error reading Tag %q: %s", tagId, createDescriptiveError(err))
	}
	tagJson, err := json.Marshal(tag)
	if err != nil {
		return diag.Errorf("error reading Tag %q: error marshaling %#v to json: %s", tagId, tag, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Tag %q: %s", tagId, tagJson), map[string]interface{}{tagLoggingKey: tagId})

	if _, err := setTagAttributes(d, catalogRestClient, clusterId, tag); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}
