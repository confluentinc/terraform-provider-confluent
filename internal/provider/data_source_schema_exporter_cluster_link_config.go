// Copyright 2026 Confluent Inc. All Rights Reserved.
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
	"io"
	"net/http"
	"net/url"
	"regexp"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func schemaExporterClusterLinkConfigDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: schemaExporterClusterLinkConfigDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramSchemaRegistryCluster: schemaRegistryClusterBlockDataSourceSchema(),
			paramRestEndpoint: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "The REST endpoint of the Schema Registry cluster, for example, `https://psrc-00000.us-central1.gcp.confluent.cloud:443`.",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^http"), "the REST endpoint must start with 'https://'"),
			},
			paramCredentials: credentialsSchema(),
			paramName: {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "The name of the Schema Exporter, for example, `my-exporter`.",
			},
			paramConfigs: {
				Type:        schema.TypeMap,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "The derived Cluster Link configuration that replicates the Schema Exporter's subject/context translation.",
			},
		},
	}
}

func schemaExporterClusterLinkConfigDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if err := dataSourceCredentialBlockValidationWithOAuth(d, meta.(*Client).isOAuthEnabled); err != nil {
		return diag.Errorf("error reading Schema Exporter Cluster Link Config: %s", createDescriptiveError(err))
	}

	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Schema Exporter Cluster Link Config: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Schema Exporter Cluster Link Config: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Schema Exporter Cluster Link Config: %s", createDescriptiveError(err))
	}
	c := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet, meta.(*Client).oauthToken)

	name := d.Get(paramName).(string)
	id := createExporterId(clusterId, name)

	tflog.Debug(ctx, fmt.Sprintf("Reading Schema Exporter Cluster Link Config %q", id), map[string]interface{}{schemaExporterLoggingKey: id})

	configs, err := readSchemaExporterClusterLinkConfig(ctx, c, name)
	if err != nil {
		return diag.Errorf("error reading Schema Exporter Cluster Link Config %q: %s", id, createDescriptiveError(err))
	}

	if !c.isMetadataSetInProviderBlock {
		if err := setKafkaCredentials(c.clusterApiKey, c.clusterApiSecret, d, c.externalAccessToken != nil); err != nil {
			return diag.FromErr(createDescriptiveError(err))
		}
		if err := d.Set(paramRestEndpoint, c.restEndpoint); err != nil {
			return diag.FromErr(createDescriptiveError(err))
		}
		if err := setStringAttributeInListBlockOfSizeOne(paramSchemaRegistryCluster, paramId, c.clusterId, d); err != nil {
			return diag.FromErr(createDescriptiveError(err))
		}
	}

	if err := d.Set(paramName, name); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	if err := d.Set(paramConfigs, configs); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	d.SetId(id)

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Schema Exporter Cluster Link Config %q", id), map[string]interface{}{schemaExporterLoggingKey: id})

	return nil
}

// readSchemaExporterClusterLinkConfig performs an authenticated GET against the Schema Registry
// exporter cluster-link config endpoint, which is not yet exposed by the vendored SR SDK.
func readSchemaExporterClusterLinkConfig(ctx context.Context, c *SchemaRegistryRestClient, name string) (map[string]string, error) {
	requestUrl := fmt.Sprintf("%s/exporters/%s/config/clusterlink", c.restEndpoint, url.PathEscape(name))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.schemaregistry.v1+json")
	req.Header.Set("User-Agent", c.apiClient.GetConfig().UserAgent)

	if c.externalAccessToken != nil {
		// Refresh the OAuth token in place (mirrors apiContext's side effect); the returned context is unused here.
		_ = c.apiContext(ctx)
		req.Header.Set("Authorization", "Bearer "+c.externalAccessToken.AccessToken)
		req.Header.Set("confluent-identity-pool-id", c.externalAccessToken.IdentityPoolId)
		req.Header.Set("target-sr-cluster", c.clusterId)
	} else {
		req.SetBasicAuth(c.clusterApiKey, c.clusterApiSecret)
	}

	tflog.Debug(ctx, fmt.Sprintf("Reading Schema Exporter Cluster Link Config for exporter %q via %s", name, requestUrl))

	httpClient := c.apiClient.GetConfig().HTTPClient
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected HTTP status %d reading cluster-link config for exporter %q: %s", resp.StatusCode, name, string(body))
	}

	configs := map[string]string{}
	if err := json.NewDecoder(resp.Body).Decode(&configs); err != nil {
		return nil, fmt.Errorf("error decoding cluster-link config response for exporter %q: %w", name, err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Fetched Schema Exporter Cluster Link Config for exporter %q", name))

	return configs, nil
}
