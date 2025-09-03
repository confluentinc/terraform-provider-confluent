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
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"regexp"
)

func schemaRegistryClusterModeDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: schemaRegistryClusterModeDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramSchemaRegistryCluster: schemaRegistryClusterBlockDataSourceSchema(),
			paramRestEndpoint: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "The REST endpoint of the Schema Registry cluster, for example, `https://psrc-00000.us-central1.gcp.confluent.cloud:443`).",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^http"), "the REST endpoint must start with 'https://'"),
			},
			paramCredentials: credentialsSchema(),
			paramMode: {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func schemaRegistryClusterModeDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Schema Registry Cluster Mode %q", d.Id()), map[string]interface{}{schemaRegistryClusterModeLoggingKey: d.Id()})

	if err := dataSourceCredentialBlockValidationWithOAuth(d, meta.(*Client).isOAuthEnabled); err != nil {
		return diag.Errorf("error reading Schema Registry Cluster Mode: %s", createDescriptiveError(err))
	}

	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Schema Registry Cluster Mode: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Schema Registry Cluster Mode: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Schema Registry Cluster Mode: %s", createDescriptiveError(err))
	}
	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet, meta.(*Client).oauthToken)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()

	if _, err := readSchemaRegistryClusterModeDataSourceAndSetAttributes(ctx, d, schemaRegistryRestClient); err != nil {
		return diag.Errorf("error reading Schema Registry Cluster Mode: %s", createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Schema Registry Cluster Mode %q", d.Id()), map[string]interface{}{schemaRegistryClusterModeLoggingKey: d.Id()})

	return nil
}

func readSchemaRegistryClusterModeDataSourceAndSetAttributes(ctx context.Context, d *schema.ResourceData, c *SchemaRegistryRestClient) ([]*schema.ResourceData, error) {
	schemaRegistryClusterMode, resp, err := c.apiClient.ModesV1Api.GetTopLevelMode(c.apiContext(ctx)).Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Schema Registry Cluster Mode %q: %s", d.Id(), createDescriptiveError(err, resp)), map[string]interface{}{schemaRegistryClusterModeLoggingKey: d.Id()})

		isResourceNotFound := ResponseHasExpectedStatusCode(resp, http.StatusNotFound)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Schema Registry Cluster Mode %q in TF state because Schema Registry Cluster Mode could not be found on the server", d.Id()), map[string]interface{}{schemaRegistryClusterModeLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	schemaRegistryClusterModeJson, err := json.Marshal(schemaRegistryClusterMode)
	if err != nil {
		return nil, fmt.Errorf("error reading Schema Registry Cluster Mode %q: error marshaling %#v to json: %s", d.Id(), schemaRegistryClusterMode, createDescriptiveError(err, resp))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Schema Registry Cluster Mode %q: %s", d.Id(), schemaRegistryClusterModeJson), map[string]interface{}{schemaRegistryClusterModeLoggingKey: d.Id()})

	if err := d.Set(paramMode, schemaRegistryClusterMode.GetMode()); err != nil {
		return nil, err
	}

	if !c.isMetadataSetInProviderBlock {
		if err := setKafkaCredentials(c.clusterApiKey, c.clusterApiSecret, d, c.externalAccessToken != nil); err != nil {
			return nil, err
		}
		if err := d.Set(paramRestEndpoint, c.restEndpoint); err != nil {
			return nil, err
		}
		if err := setStringAttributeInListBlockOfSizeOne(paramSchemaRegistryCluster, paramId, c.clusterId, d); err != nil {
			return nil, err
		}
	}

	d.SetId(createSchemaRegistryClusterModeId(c.clusterId))

	return []*schema.ResourceData{d}, nil
}
