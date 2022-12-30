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

func schemaRegistryClusterModeResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: schemaRegistryClusterModeCreate,
		ReadContext:   schemaRegistryClusterModeRead,
		UpdateContext: schemaRegistryClusterModeUpdate,
		DeleteContext: schemaRegistryClusterModeDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schemaRegistryClusterModeImport,
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
			paramMode: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice(acceptedModes, false),
			},
		},
	}
}

func schemaRegistryClusterModeCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Schema Registry Cluster Mode: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Schema Registry Cluster Mode: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Schema Registry Cluster Mode: %s", createDescriptiveError(err))
	}
	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)

	if _, ok := d.GetOk(paramMode); ok {
		compatibilityLevel := d.Get(paramMode).(string)

		createModeRequest := sr.NewModeUpdateRequest()
		createModeRequest.SetMode(compatibilityLevel)
		createModeRequestJson, err := json.Marshal(createModeRequest)
		if err != nil {
			return diag.Errorf("error creating Schema Registry Cluster Mode: error marshaling %#v to json: %s", createModeRequest, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Creating new Schema Registry Cluster Mode: %s", createModeRequestJson))

		_, _, err = executeSchemaRegistryClusterModeUpdate(ctx, schemaRegistryRestClient, createModeRequest)

		if err != nil {
			return diag.Errorf("error creating Schema Registry Cluster Mode: %s", createDescriptiveError(err))
		}

		time.Sleep(schemaRegistryAPIWaitAfterCreateOrDelete)
	}

	schemaRegistryClusterModeId := createSchemaRegistryClusterModeId(schemaRegistryRestClient.clusterId)
	d.SetId(schemaRegistryClusterModeId)

	tflog.Debug(ctx, fmt.Sprintf("Finished creating Schema Registry Cluster Mode %q", d.Id()), map[string]interface{}{schemaRegistryClusterModeLoggingKey: d.Id()})

	return schemaRegistryClusterModeRead(ctx, d, meta)
}

func schemaRegistryClusterModeDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Schema Registry Cluster Mode %q", d.Id()), map[string]interface{}{schemaRegistryClusterModeLoggingKey: d.Id()})

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Schema Registry Cluster Mode %q", d.Id()), map[string]interface{}{schemaRegistryClusterModeLoggingKey: d.Id()})

	return nil
}

func schemaRegistryClusterModeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Schema Registry Cluster Mode %q", d.Id()), map[string]interface{}{schemaRegistryClusterModeLoggingKey: d.Id()})

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
	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)

	_, err = readSchemaRegistryClusterModeAndSetAttributes(ctx, d, schemaRegistryRestClient)
	if err != nil {
		return diag.Errorf("error reading Schema Registry Cluster Mode: %s", createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Schema Registry Cluster Mode %q", d.Id()), map[string]interface{}{schemaRegistryClusterModeLoggingKey: d.Id()})

	return nil
}

func createSchemaRegistryClusterModeId(clusterId string) string {
	return fmt.Sprintf("%s", clusterId)
}

func schemaRegistryClusterModeImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Schema Registry Cluster Mode %q", d.Id()), map[string]interface{}{schemaRegistryClusterModeLoggingKey: d.Id()})

	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Schema Registry Cluster Mode: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Schema Registry Cluster Mode: %s", createDescriptiveError(err))
	}

	clusterId := d.Id()

	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readSchemaRegistryClusterModeAndSetAttributes(ctx, d, schemaRegistryRestClient); err != nil {
		return nil, fmt.Errorf("error importing Schema Registry Cluster Mode %q: %s", d.Id(), createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Schema Registry Cluster Mode %q", d.Id()), map[string]interface{}{schemaRegistryClusterModeLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func readSchemaRegistryClusterModeAndSetAttributes(ctx context.Context, d *schema.ResourceData, c *SchemaRegistryRestClient) ([]*schema.ResourceData, error) {
	schemaRegistryClusterMode, resp, err := c.apiClient.ModesV1Api.GetTopLevelMode(c.apiContext(ctx)).Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Schema Registry Cluster Mode %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{schemaRegistryClusterModeLoggingKey: d.Id()})

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
		return nil, fmt.Errorf("error reading Schema Registry Cluster Mode %q: error marshaling %#v to json: %s", d.Id(), schemaRegistryClusterMode, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Schema Registry Cluster Mode %q: %s", d.Id(), schemaRegistryClusterModeJson), map[string]interface{}{schemaRegistryClusterModeLoggingKey: d.Id()})

	if err := d.Set(paramMode, schemaRegistryClusterMode.GetMode()); err != nil {
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

	d.SetId(createSchemaRegistryClusterModeId(c.clusterId))

	return []*schema.ResourceData{d}, nil
}

func schemaRegistryClusterModeUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramCredentials, paramMode) {
		return diag.Errorf("error updating Schema Registry Cluster Mode %q: only %q and %q blocks can be updated for Schema Registry Cluster Mode", d.Id(), paramCredentials, paramMode)
	}
	if d.HasChange(paramMode) {
		updatedMode := d.Get(paramMode).(string)
		updateModeRequest := sr.NewModeUpdateRequest()
		updateModeRequest.SetMode(updatedMode)
		restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Schema Registry Cluster Mode: %s", createDescriptiveError(err))
		}
		clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Schema Registry Cluster Mode: %s", createDescriptiveError(err))
		}
		clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Schema Registry Cluster Mode: %s", createDescriptiveError(err))
		}
		schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
		updateModeRequestJson, err := json.Marshal(updateModeRequest)
		if err != nil {
			return diag.Errorf("error updating Schema Registry Cluster Mode: error marshaling %#v to json: %s", updateModeRequest, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Updating Schema Registry Cluster Mode %q: %s", d.Id(), updateModeRequestJson), map[string]interface{}{kafkaClusterConfigLoggingKey: d.Id()})

		_, _, err = executeSchemaRegistryClusterModeUpdate(ctx, schemaRegistryRestClient, updateModeRequest)
		if err != nil {
			return diag.Errorf("error updating Schema Registry Cluster Mode: %s", createDescriptiveError(err))
		}
		time.Sleep(kafkaRestAPIWaitAfterCreate)
		tflog.Debug(ctx, fmt.Sprintf("Finished updating Schema Registry Cluster Mode %q", d.Id()), map[string]interface{}{kafkaClusterConfigLoggingKey: d.Id()})
	}
	return schemaRegistryClusterModeRead(ctx, d, meta)
}

func executeSchemaRegistryClusterModeUpdate(ctx context.Context, c *SchemaRegistryRestClient, requestData *sr.ModeUpdateRequest) (sr.ModeUpdateRequest, *http.Response, error) {
	return c.apiClient.ModesV1Api.UpdateTopLevelMode(c.apiContext(ctx)).ModeUpdateRequest(*requestData).Execute()
}
