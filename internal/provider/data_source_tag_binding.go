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
	dc "github.com/confluentinc/ccloud-sdk-go-v2/data-catalog/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"regexp"
)

func tagBindingDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: tagBindingDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramSchemaRegistryCluster: schemaRegistryClusterBlockDataSourceSchema(),
			paramRestEndpoint: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The REST endpoint of the Schema Registry cluster, for example, `https://psrc-00000.us-central1.gcp.confluent.cloud:443`).",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^http"), "the REST endpoint must start with 'https://'"),
			},
			paramCredentials: credentialsSchema(),
			paramTagName: {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9_\\s]*$"), "The name must not be empty and consist of a letter followed by a sequence of letter, number, space, or _ characters"),
				Description:  "The name of the tag to be applied.",
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
				Description: "The id of the tag binding to be created.",
			},
		},
	}
}

func tagBindingDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if err := dataSourceCredentialBlockValidationWithOAuth(d, meta.(*Client).isOAuthEnabled); err != nil {
		return diag.Errorf("error reading Tag Binding: %s", createDescriptiveError(err))
	}

	restEndpoint, err := extractCatalogRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Tag Binding: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Tag Binding: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Tag Binding: %s", createDescriptiveError(err))
	}

	tagName := d.Get(paramTagName).(string)
	entityName := d.Get(paramEntityName).(string)
	entityType := d.Get(paramEntityType).(string)

	tagBindingId := createTagBindingId(clusterId, tagName, entityName, entityType)

	tflog.Debug(ctx, fmt.Sprintf("Reading Tag Binding %q=%q", paramId, tagBindingId), map[string]interface{}{tagBindingLoggingKey: tagBindingId})

	catalogRestClient := meta.(*Client).catalogRestClientFactory.CreateCatalogRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet, meta.(*Client).oauthToken)
	request := catalogRestClient.apiClient.EntityV1Api.GetTags(catalogRestClient.dataCatalogApiContext(ctx), entityType, entityName)
	tagBindings, resp, err := request.Execute()
	if err != nil {
		return diag.Errorf("error reading Tag Binding %q: %s", tagBindingId, createDescriptiveError(err, resp))
	}
	tagBinding, err := findTagBindingByTagName(tagBindings, tagName)
	if err != nil {
		return diag.Errorf("error reading Tag Binding %q: %s", tagBindingId, "The binding information cannot be found")
	}

	tagBindingJson, err := json.Marshal(tagBinding)
	if err != nil {
		return diag.Errorf("error reading Tag Binding %q: error marshaling %#v to json: %s", tagBindingId, tagBinding, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Tag Binding %q: %s", tagBindingId, tagBindingJson), map[string]interface{}{tagBindingLoggingKey: tagBindingId})

	if _, err := setTagBindingDataSourceAttributes(d, clusterId, tagBinding); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	return nil
}

func setTagBindingDataSourceAttributes(d *schema.ResourceData, clusterId string, tagBinding dc.TagResponse) (*schema.ResourceData, error) {
	if err := d.Set(paramTagName, tagBinding.GetTypeName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramEntityName, tagBinding.GetEntityName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramEntityType, tagBinding.GetEntityType()); err != nil {
		return nil, err
	}
	d.SetId(createTagBindingId(clusterId, tagBinding.GetTypeName(), tagBinding.GetEntityName(), tagBinding.GetEntityType()))
	return d, nil
}
