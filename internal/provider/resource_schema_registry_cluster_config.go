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
	sr "github.com/confluentinc/ccloud-sdk-go-v2/schema-registry/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"regexp"
	"time"
)

func schemaRegistryClusterConfigResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: schemaRegistryClusterConfigCreate,
		ReadContext:   schemaRegistryClusterConfigRead,
		UpdateContext: schemaRegistryClusterConfigUpdate,
		DeleteContext: schemaRegistryClusterConfigDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schemaRegistryClusterConfigImport,
		},
		Schema: map[string]*schema.Schema{
			paramSchemaRegistryCluster: schemaRegistryClusterBlockSchema(),
			paramRestEndpoint: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "The REST endpoint of the Schema Registry cluster, for example, `https://psrc-00000.us-central1.gcp.confluent.cloud:443`).",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^http"), "the REST endpoint must start with 'https://'"),
			},
			paramCredentials: credentialsSchema(),
			paramCompatibilityLevel: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice(acceptedCompatibilityLevels, false),
			},
		},
	}
}

func schemaRegistryClusterConfigCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Schema Registry Cluster Config: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Schema Registry Cluster Config: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Schema Registry Cluster Config: %s", createDescriptiveError(err))
	}
	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)

	if _, ok := d.GetOk(paramCompatibilityLevel); ok {
		compatibilityLevel := d.Get(paramCompatibilityLevel).(string)

		createConfigRequest := sr.NewConfigUpdateRequest()
		createConfigRequest.SetCompatibility(compatibilityLevel)
		createModeRequestJson, err := json.Marshal(createConfigRequest)
		if err != nil {
			return diag.Errorf("error creating Schema Registry Cluster Config: error marshaling %#v to json: %s", createConfigRequest, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Creating new Schema Registry Cluster Config: %s", createModeRequestJson))

		_, _, err = executeSchemaRegistryClusterConfigUpdate(ctx, schemaRegistryRestClient, createConfigRequest)

		if err != nil {
			return diag.Errorf("error creating Schema Registry Cluster Config: %s", createDescriptiveError(err))
		}

		time.Sleep(schemaRegistryAPIWaitAfterCreateOrDelete)
	}

	schemaRegistryClusterConfigId := createSchemaRegistryClusterConfigId(schemaRegistryRestClient.clusterId)
	d.SetId(schemaRegistryClusterConfigId)

	tflog.Debug(ctx, fmt.Sprintf("Finished creating Schema Registry Cluster Config %q", d.Id()), map[string]interface{}{schemaRegistryClusterConfigLoggingKey: d.Id()})

	return schemaRegistryClusterConfigRead(ctx, d, meta)
}

func schemaRegistryClusterConfigDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Schema Registry Cluster Config %q", d.Id()), map[string]interface{}{schemaRegistryClusterConfigLoggingKey: d.Id()})

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Schema Registry Cluster Config %q", d.Id()), map[string]interface{}{schemaRegistryClusterConfigLoggingKey: d.Id()})

	return nil
}

func schemaRegistryClusterConfigRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Schema Registry Cluster Config %q", d.Id()), map[string]interface{}{schemaRegistryClusterConfigLoggingKey: d.Id()})

	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Schema Registry Cluster Config: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Schema Registry Cluster Config: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Schema Registry Cluster Config: %s", createDescriptiveError(err))
	}
	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)

	_, err = readSchemaRegistryClusterConfigAndSetAttributes(ctx, d, schemaRegistryRestClient)
	if err != nil {
		return diag.Errorf("error reading Schema Registry Cluster Config: %s", createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Schema Registry Cluster Config %q", d.Id()), map[string]interface{}{schemaRegistryClusterConfigLoggingKey: d.Id()})

	return nil
}

func createSchemaRegistryClusterConfigId(clusterId string) string {
	return fmt.Sprintf("%s", clusterId)
}

func schemaRegistryClusterConfigImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Schema Registry Cluster Config %q", d.Id()), map[string]interface{}{schemaRegistryClusterConfigLoggingKey: d.Id()})

	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Schema Registry Cluster Config: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Schema Registry Cluster Config: %s", createDescriptiveError(err))
	}

	clusterId := d.Id()

	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readSchemaRegistryClusterConfigAndSetAttributes(ctx, d, schemaRegistryRestClient); err != nil {
		return nil, fmt.Errorf("error importing Schema Registry Cluster Config %q: %s", d.Id(), createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Schema Registry Cluster Config %q", d.Id()), map[string]interface{}{schemaRegistryClusterConfigLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func readSchemaRegistryClusterConfigAndSetAttributes(ctx context.Context, d *schema.ResourceData, c *SchemaRegistryRestClient) ([]*schema.ResourceData, error) {
	schemaRegistryClusterConfig, resp, err := c.apiClient.ConfigV1Api.GetTopLevelConfig(c.apiContext(ctx)).Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Schema Registry Cluster Config %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{schemaRegistryClusterConfigLoggingKey: d.Id()})

		isResourceNotFound := ResponseHasExpectedStatusCode(resp, http.StatusNotFound)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Schema Registry Cluster Config %q in TF state because Schema Registry Cluster Config could not be found on the server", d.Id()), map[string]interface{}{schemaRegistryClusterConfigLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	schemaRegistryClusterConfigJson, err := json.Marshal(schemaRegistryClusterConfig)
	if err != nil {
		return nil, fmt.Errorf("error reading Schema Registry Cluster Config %q: error marshaling %#v to json: %s", d.Id(), schemaRegistryClusterConfig, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Schema Registry Cluster Config %q: %s", d.Id(), schemaRegistryClusterConfigJson), map[string]interface{}{schemaRegistryClusterConfigLoggingKey: d.Id()})

	if err := d.Set(paramCompatibilityLevel, schemaRegistryClusterConfig.GetCompatibilityLevel()); err != nil {
		return nil, err
	}

	if !c.isMetadataSetInProviderBlock {
		if err := setKafkaCredentials(c.clusterApiKey, c.clusterApiSecret, d); err != nil {
			return nil, err
		}
		if err := d.Set(paramRestEndpoint, c.restEndpoint); err != nil {
			return nil, err
		}
		if err := setStringAttributeInListBlockOfSizeOne(paramSchemaRegistryCluster, paramId, c.clusterId, d); err != nil {
			return nil, err
		}
	}

	d.SetId(createSchemaRegistryClusterConfigId(c.clusterId))

	return []*schema.ResourceData{d}, nil
}

func schemaRegistryClusterConfigUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramCredentials, paramCompatibilityLevel) {
		return diag.Errorf("error updating Schema Registry Cluster Config %q: only %q and %q blocks can be updated for Schema Registry Cluster Config", d.Id(), paramCredentials, paramCompatibilityLevel)
	}
	if d.HasChange(paramCompatibilityLevel) {
		updatedCompatibilityLevel := d.Get(paramCompatibilityLevel).(string)
		updateConfigRequest := sr.NewConfigUpdateRequest()
		updateConfigRequest.SetCompatibility(updatedCompatibilityLevel)
		restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Schema Registry Cluster Config: %s", createDescriptiveError(err))
		}
		clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Schema Registry Cluster Config: %s", createDescriptiveError(err))
		}
		clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Schema Registry Cluster Config: %s", createDescriptiveError(err))
		}
		schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
		updateCompatibilityLevelRequestJson, err := json.Marshal(updateConfigRequest)
		if err != nil {
			return diag.Errorf("error updating Schema Registry Cluster Config: error marshaling %#v to json: %s", updateConfigRequest, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Updating Schema Registry Cluster Config %q: %s", d.Id(), updateCompatibilityLevelRequestJson), map[string]interface{}{kafkaClusterConfigLoggingKey: d.Id()})

		_, _, err = executeSchemaRegistryClusterConfigUpdate(ctx, schemaRegistryRestClient, updateConfigRequest)
		if err != nil {
			return diag.Errorf("error updating Schema Registry Cluster Config: %s", createDescriptiveError(err))
		}
		time.Sleep(kafkaRestAPIWaitAfterCreate)
		tflog.Debug(ctx, fmt.Sprintf("Finished updating Schema Registry Cluster Config %q", d.Id()), map[string]interface{}{kafkaClusterConfigLoggingKey: d.Id()})
	}
	return schemaRegistryClusterConfigRead(ctx, d, meta)
}

func executeSchemaRegistryClusterConfigUpdate(ctx context.Context, c *SchemaRegistryRestClient, requestData *sr.ConfigUpdateRequest) (sr.ConfigUpdateRequest, *http.Response, error) {
	return c.apiClient.ConfigV1Api.UpdateTopLevelConfig(c.apiContext(ctx)).ConfigUpdateRequest(*requestData).Execute()
}
