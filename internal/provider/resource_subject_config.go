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
	"strings"
	"time"
)

const (
	paramCompatibilityLevel = "compatibility_level"

	compatibilityLevelBackward           = "BACKWARD"
	compatibilityLevelBackwardTransitive = "BACKWARD_TRANSITIVE"
	compatibilityLevelForward            = "FORWARD"
	compatibilityLevelForwardTransitive  = "FORWARD_TRANSITIVE"
	compatibilityLevelFull               = "FULL"
	compatibilityLevelFullTransitive     = "FULL_TRANSITIVE"
	compatibilityLevelNone               = "NONE"
)

var acceptedCompatibilityLevels = []string{compatibilityLevelBackward, compatibilityLevelBackwardTransitive,
	compatibilityLevelForward, compatibilityLevelForwardTransitive, compatibilityLevelFull, compatibilityLevelFullTransitive,
	compatibilityLevelNone}

func subjectConfigResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: subjectConfigCreate,
		ReadContext:   subjectConfigRead,
		UpdateContext: subjectConfigUpdate,
		DeleteContext: subjectConfigDelete,
		Importer: &schema.ResourceImporter{
			StateContext: subjectConfigImport,
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
			paramSubjectName: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The name of the Schema Registry Subject.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramCompatibilityLevel: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice(acceptedCompatibilityLevels, false),
			},
		},
	}
}

func subjectConfigCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Subject Config: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Subject Config: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Subject Config: %s", createDescriptiveError(err))
	}
	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	subjectName := d.Get(paramSubjectName).(string)

	if _, ok := d.GetOk(paramCompatibilityLevel); ok {
		compatibilityLevel := d.Get(paramCompatibilityLevel).(string)

		createConfigRequest := sr.NewConfigUpdateRequest()
		createConfigRequest.SetCompatibility(compatibilityLevel)
		createConfigRequestJson, err := json.Marshal(createConfigRequest)
		if err != nil {
			return diag.Errorf("error creating Subject Config: error marshaling %#v to json: %s", createConfigRequest, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Creating new Subject Config: %s", createConfigRequestJson))

		_, _, err = executeSubjectConfigUpdate(ctx, schemaRegistryRestClient, createConfigRequest, subjectName)

		if err != nil {
			return diag.Errorf("error creating Subject Config: %s", createDescriptiveError(err))
		}

		time.Sleep(schemaRegistryAPIWaitAfterCreateOrDelete)
	}

	subjectConfigId := createSubjectConfigId(schemaRegistryRestClient.clusterId, subjectName)
	d.SetId(subjectConfigId)

	tflog.Debug(ctx, fmt.Sprintf("Finished creating Subject Config %q", d.Id()), map[string]interface{}{subjectConfigLoggingKey: d.Id()})

	return subjectConfigRead(ctx, d, meta)
}

func subjectConfigDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Subject Config %q", d.Id()), map[string]interface{}{subjectConfigLoggingKey: d.Id()})

	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Subject Config: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Subject Config: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Subject Config: %s", createDescriptiveError(err))
	}
	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	subjectName := d.Get(paramSubjectName).(string)

	// Deletes the specified subject-level compatibility level config and reverts to the global default.
	_, _, err = schemaRegistryRestClient.apiClient.ConfigV1Api.DeleteSubjectConfig(schemaRegistryRestClient.apiContext(ctx), subjectName).Execute()

	if err != nil {
		return diag.Errorf("error deleting Subject Config %q: %s", d.Id(), createDescriptiveError(err))
	}

	time.Sleep(schemaRegistryAPIWaitAfterCreateOrDelete)

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Subject Config %q", d.Id()), map[string]interface{}{subjectConfigLoggingKey: d.Id()})

	return nil
}

func subjectConfigRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Subject Config %q", d.Id()), map[string]interface{}{subjectConfigLoggingKey: d.Id()})

	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Subject Config: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Subject Config: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Subject Config: %s", createDescriptiveError(err))
	}
	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	subjectName := d.Get(paramSubjectName).(string)

	_, err = readSubjectConfigAndSetAttributes(ctx, d, schemaRegistryRestClient, subjectName)
	if err != nil {
		return diag.Errorf("error reading Subject Config: %s", createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Subject Config %q", d.Id()), map[string]interface{}{subjectConfigLoggingKey: d.Id()})

	return nil
}

func createSubjectConfigId(clusterId, subjectName string) string {
	return fmt.Sprintf("%s/%s", clusterId, subjectName)
}

func subjectConfigImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Subject Config %q", d.Id()), map[string]interface{}{subjectConfigLoggingKey: d.Id()})

	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Subject Config: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Subject Config: %s", createDescriptiveError(err))
	}

	clusterIDAndSubjectName := d.Id()
	parts := strings.Split(clusterIDAndSubjectName, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing Subject Config: invalid format: expected '<SR cluster ID>/<subject name>'")
	}

	clusterId := parts[0]
	subjectName := parts[1]

	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readSubjectConfigAndSetAttributes(ctx, d, schemaRegistryRestClient, subjectName); err != nil {
		return nil, fmt.Errorf("error importing Subject Config %q: %s", d.Id(), createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Subject Config %q", d.Id()), map[string]interface{}{subjectConfigLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func readSubjectConfigAndSetAttributes(ctx context.Context, d *schema.ResourceData, c *SchemaRegistryRestClient, subjectName string) ([]*schema.ResourceData, error) {
	subjectConfig, resp, err := c.apiClient.ConfigV1Api.GetSubjectLevelConfig(c.apiContext(ctx), subjectName).DefaultToGlobal(true).Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Subject Config %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{subjectConfigLoggingKey: d.Id()})

		isResourceNotFound := ResponseHasExpectedStatusCode(resp, http.StatusNotFound)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Subject Config %q in TF state because Subject Config could not be found on the server", d.Id()), map[string]interface{}{subjectConfigLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	subjectConfigJson, err := json.Marshal(subjectConfig)
	if err != nil {
		return nil, fmt.Errorf("error reading Subject Config %q: error marshaling %#v to json: %s", d.Id(), subjectConfig, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Subject Config %q: %s", d.Id(), subjectConfigJson), map[string]interface{}{subjectConfigLoggingKey: d.Id()})

	if err := d.Set(paramSubjectName, subjectName); err != nil {
		return nil, err
	}

	if err := d.Set(paramCompatibilityLevel, subjectConfig.GetCompatibilityLevel()); err != nil {
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

	d.SetId(createSubjectConfigId(c.clusterId, subjectName))

	return []*schema.ResourceData{d}, nil
}

func subjectConfigUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramCredentials, paramCompatibilityLevel) {
		return diag.Errorf("error updating Subject Config %q: only %q and %q blocks can be updated for Subject Config", d.Id(), paramCredentials, paramCompatibilityLevel)
	}
	if d.HasChange(paramCompatibilityLevel) {
		updatedCompatibilityLevel := d.Get(paramCompatibilityLevel).(string)
		updateConfigRequest := sr.NewConfigUpdateRequest()
		updateConfigRequest.SetCompatibility(updatedCompatibilityLevel)
		restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Schema: %s", createDescriptiveError(err))
		}
		clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Subject Config: %s", createDescriptiveError(err))
		}
		clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Schema: %s", createDescriptiveError(err))
		}
		schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
		subjectName := d.Get(paramSubjectName).(string)
		updateConfigRequestJson, err := json.Marshal(updateConfigRequest)
		if err != nil {
			return diag.Errorf("error updating Subject Config: error marshaling %#v to json: %s", updateConfigRequest, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Updating Subject Config %q: %s", d.Id(), updateConfigRequestJson), map[string]interface{}{kafkaClusterConfigLoggingKey: d.Id()})

		_, _, err = executeSubjectConfigUpdate(ctx, schemaRegistryRestClient, updateConfigRequest, subjectName)
		if err != nil {
			return diag.Errorf("error updating Subject Config: %s", createDescriptiveError(err))
		}
		time.Sleep(kafkaRestAPIWaitAfterCreate)
		tflog.Debug(ctx, fmt.Sprintf("Finished updating Subject Config %q", d.Id()), map[string]interface{}{kafkaClusterConfigLoggingKey: d.Id()})
	}
	return subjectConfigRead(ctx, d, meta)
}

func executeSubjectConfigUpdate(ctx context.Context, c *SchemaRegistryRestClient, requestData *sr.ConfigUpdateRequest, subjectName string) (sr.ConfigUpdateRequest, *http.Response, error) {
	return c.apiClient.ConfigV1Api.UpdateSubjectLevelConfig(c.apiContext(ctx), subjectName).ConfigUpdateRequest(*requestData).Execute()
}
