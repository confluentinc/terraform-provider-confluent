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

func schemaRegistryClusterCompatibilityLevelResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: schemaRegistryClusterCompatibilityLevelCreate,
		ReadContext:   schemaRegistryClusterCompatibilityLevelRead,
		UpdateContext: schemaRegistryClusterCompatibilityLevelUpdate,
		DeleteContext: schemaRegistryClusterCompatibilityLevelDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schemaRegistryClusterCompatibilityLevelImport,
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

func schemaRegistryClusterCompatibilityLevelCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Schema Registry Cluster Compatibility Level: %s", createDescriptiveError(err))
	}
	clusterId := extractStringValueFromBlock(d, paramSchemaRegistryCluster, paramId)
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Schema Registry Cluster Compatibility Level: %s", createDescriptiveError(err))
	}
	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)

	if _, ok := d.GetOk(paramCompatibilityLevel); ok {
		compatibilityLevel := d.Get(paramCompatibilityLevel).(string)

		createModeRequest := sr.NewConfigUpdateRequest()
		createModeRequest.SetCompatibility(compatibilityLevel)
		createModeRequestJson, err := json.Marshal(createModeRequest)
		if err != nil {
			return diag.Errorf("error creating Schema Registry Cluster Compatibility Level: error marshaling %#v to json: %s", createModeRequest, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Creating new Schema Registry Cluster Compatibility Level: %s", createModeRequestJson))

		_, _, err = executeSchemaRegistryClusterCompatibilityLevelUpdate(ctx, schemaRegistryRestClient, createModeRequest)

		if err != nil {
			return diag.Errorf("error creating Schema Registry Cluster Compatibility Level: %s", createDescriptiveError(err))
		}

		time.Sleep(schemaRegistryAPIWaitAfterCreateOrDelete)
	}

	schemaRegistryClusterCompatibilityLevelId := createSchemaRegistryClusterCompatibilityLevelId(schemaRegistryRestClient.clusterId)
	d.SetId(schemaRegistryClusterCompatibilityLevelId)

	tflog.Debug(ctx, fmt.Sprintf("Finished creating Schema Registry Cluster Compatibility Level %q", d.Id()), map[string]interface{}{schemaRegistryClusterCompatibilityLevelLoggingKey: d.Id()})

	return schemaRegistryClusterCompatibilityLevelRead(ctx, d, meta)
}

func schemaRegistryClusterCompatibilityLevelDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Schema Registry Cluster Compatibility Level %q", d.Id()), map[string]interface{}{schemaRegistryClusterCompatibilityLevelLoggingKey: d.Id()})

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Schema Registry Cluster Compatibility Level %q", d.Id()), map[string]interface{}{schemaRegistryClusterCompatibilityLevelLoggingKey: d.Id()})

	return nil
}

func schemaRegistryClusterCompatibilityLevelRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Schema Registry Cluster Compatibility Level %q", d.Id()), map[string]interface{}{schemaRegistryClusterCompatibilityLevelLoggingKey: d.Id()})

	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Schema Registry Cluster Compatibility Level: %s", createDescriptiveError(err))
	}
	clusterId := extractStringValueFromBlock(d, paramSchemaRegistryCluster, paramId)
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Schema Registry Cluster Compatibility Level: %s", createDescriptiveError(err))
	}
	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)

	_, err = readSchemaRegistryClusterCompatibilityLevelAndSetAttributes(ctx, d, schemaRegistryRestClient)
	if err != nil {
		return diag.Errorf("error reading Schema Registry Cluster Compatibility Level: %s", createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Schema Registry Cluster Compatibility Level %q", d.Id()), map[string]interface{}{schemaRegistryClusterCompatibilityLevelLoggingKey: d.Id()})

	return nil
}

func createSchemaRegistryClusterCompatibilityLevelId(clusterId string) string {
	return fmt.Sprintf("%s", clusterId)
}

func schemaRegistryClusterCompatibilityLevelImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Schema Registry Cluster Compatibility Level %q", d.Id()), map[string]interface{}{schemaRegistryClusterCompatibilityLevelLoggingKey: d.Id()})

	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Schema Registry Cluster Compatibility Level: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Schema Registry Cluster Compatibility Level: %s", createDescriptiveError(err))
	}

	clusterId := d.Id()

	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readSchemaRegistryClusterCompatibilityLevelAndSetAttributes(ctx, d, schemaRegistryRestClient); err != nil {
		return nil, fmt.Errorf("error importing Schema Registry Cluster Compatibility Level %q: %s", d.Id(), createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Schema Registry Cluster Compatibility Level %q", d.Id()), map[string]interface{}{schemaRegistryClusterCompatibilityLevelLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func readSchemaRegistryClusterCompatibilityLevelAndSetAttributes(ctx context.Context, d *schema.ResourceData, c *SchemaRegistryRestClient) ([]*schema.ResourceData, error) {
	schemaRegistryClusterCompatibilityLevel, resp, err := c.apiClient.ConfigV1Api.GetTopLevelConfig(c.apiContext(ctx)).Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Schema Registry Cluster Compatibility Level %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{schemaRegistryClusterCompatibilityLevelLoggingKey: d.Id()})

		isResourceNotFound := ResponseHasExpectedStatusCode(resp, http.StatusNotFound)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Schema Registry Cluster Compatibility Level %q in TF state because Schema Registry Cluster Compatibility Level could not be found on the server", d.Id()), map[string]interface{}{schemaRegistryClusterCompatibilityLevelLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	schemaRegistryClusterCompatibilityLevelJson, err := json.Marshal(schemaRegistryClusterCompatibilityLevel)
	if err != nil {
		return nil, fmt.Errorf("error reading Schema Registry Cluster Compatibility Level %q: error marshaling %#v to json: %s", d.Id(), schemaRegistryClusterCompatibilityLevel, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Schema Registry Cluster Compatibility Level %q: %s", d.Id(), schemaRegistryClusterCompatibilityLevelJson), map[string]interface{}{schemaRegistryClusterCompatibilityLevelLoggingKey: d.Id()})

	if err := setStringAttributeInListBlockOfSizeOne(paramSchemaRegistryCluster, paramId, c.clusterId, d); err != nil {
		return nil, err
	}

	if err := d.Set(paramCompatibilityLevel, schemaRegistryClusterCompatibilityLevel.GetCompatibilityLevel()); err != nil {
		return nil, err
	}

	if !c.isMetadataSetInProviderBlock {
		if err := setKafkaCredentials(c.clusterApiKey, c.clusterApiSecret, d); err != nil {
			return nil, err
		}
		if err := d.Set(paramRestEndpoint, c.restEndpoint); err != nil {
			return nil, err
		}
	}

	d.SetId(createSchemaRegistryClusterCompatibilityLevelId(c.clusterId))

	return []*schema.ResourceData{d}, nil
}

func schemaRegistryClusterCompatibilityLevelUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramCredentials, paramCompatibilityLevel) {
		return diag.Errorf("error updating Schema Registry Cluster Compatibility Level %q: only %q and %q blocks can be updated for Schema Registry Cluster Compatibility Level", d.Id(), paramCredentials, paramCompatibilityLevel)
	}
	if d.HasChange(paramCompatibilityLevel) {
		updatedCompatibilityLevel := d.Get(paramCompatibilityLevel).(string)
		updateCompatibilityLevelRequest := sr.NewConfigUpdateRequest()
		updateCompatibilityLevelRequest.SetCompatibility(updatedCompatibilityLevel)
		restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Schema Registry Cluster Compatibility Level: %s", createDescriptiveError(err))
		}
		clusterId := extractStringValueFromBlock(d, paramSchemaRegistryCluster, paramId)
		clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Schema Registry Cluster Compatibility Level: %s", createDescriptiveError(err))
		}
		schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
		updateCompatibilityLevelRequestJson, err := json.Marshal(updateCompatibilityLevelRequest)
		if err != nil {
			return diag.Errorf("error updating Schema Registry Cluster Compatibility Level: error marshaling %#v to json: %s", updateCompatibilityLevelRequest, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Updating Schema Registry Cluster Compatibility Level %q: %s", d.Id(), updateCompatibilityLevelRequestJson), map[string]interface{}{kafkaClusterConfigLoggingKey: d.Id()})

		_, _, err = executeSchemaRegistryClusterCompatibilityLevelUpdate(ctx, schemaRegistryRestClient, updateCompatibilityLevelRequest)
		if err != nil {
			return diag.Errorf("error updating Schema Registry Cluster Compatibility Level: %s", createDescriptiveError(err))
		}
		time.Sleep(kafkaRestAPIWaitAfterCreate)
		tflog.Debug(ctx, fmt.Sprintf("Finished updating Schema Registry Cluster Compatibility Level %q", d.Id()), map[string]interface{}{kafkaClusterConfigLoggingKey: d.Id()})
	}
	return schemaRegistryClusterCompatibilityLevelRead(ctx, d, meta)
}

func executeSchemaRegistryClusterCompatibilityLevelUpdate(ctx context.Context, c *SchemaRegistryRestClient, requestData *sr.ConfigUpdateRequest) (sr.ConfigUpdateRequest, *http.Response, error) {
	return c.apiClient.ConfigV1Api.UpdateTopLevelConfig(c.apiContext(ctx)).ConfigUpdateRequest(*requestData).Execute()
}
