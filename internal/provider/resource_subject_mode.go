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
	paramMode = "mode"

	modeReadWrite        = "READWRITE"
	modeReadOnly         = "READONLY"
	modeReadOnlyOverride = "READONLY_OVERRIDE"
	modeImport           = "IMPORT"
)

var acceptedModes = []string{modeReadWrite, modeReadOnly, modeReadOnlyOverride, modeImport}

func subjectModeResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: subjectModeCreate,
		ReadContext:   subjectModeRead,
		UpdateContext: subjectModeUpdate,
		DeleteContext: subjectModeDelete,
		Importer: &schema.ResourceImporter{
			StateContext: subjectModeImport,
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
			paramMode: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice(acceptedModes, false),
			},
		},
	}
}

func subjectModeCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Subject Mode: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Subject Mode: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Subject Mode: %s", createDescriptiveError(err))
	}
	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	subjectName := d.Get(paramSubjectName).(string)

	if _, ok := d.GetOk(paramMode); ok {
		compatibilityLevel := d.Get(paramMode).(string)

		createModeRequest := sr.NewModeUpdateRequest()
		createModeRequest.SetMode(compatibilityLevel)
		createModeRequestJson, err := json.Marshal(createModeRequest)
		if err != nil {
			return diag.Errorf("error creating Subject Mode: error marshaling %#v to json: %s", createModeRequest, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Creating new Subject Mode: %s", createModeRequestJson))

		_, _, err = executeSubjectConfigModeUpdate(ctx, schemaRegistryRestClient, createModeRequest, subjectName)

		if err != nil {
			return diag.Errorf("error creating Subject Mode: %s", createDescriptiveError(err))
		}

		time.Sleep(schemaRegistryAPIWaitAfterCreateOrDelete)
	}

	subjectModeId := createSubjectModeId(schemaRegistryRestClient.clusterId, subjectName)
	d.SetId(subjectModeId)

	tflog.Debug(ctx, fmt.Sprintf("Finished creating Subject Mode %q", d.Id()), map[string]interface{}{subjectModeLoggingKey: d.Id()})

	return subjectModeRead(ctx, d, meta)
}

func subjectModeDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Subject Mode %q", d.Id()), map[string]interface{}{subjectModeLoggingKey: d.Id()})

	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Subject Mode: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Subject Mode: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Subject Mode: %s", createDescriptiveError(err))
	}
	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	subjectName := d.Get(paramSubjectName).(string)

	// Deletes the specified subject-level mode config and reverts to the global default.
	_, _, err = schemaRegistryRestClient.apiClient.ModesV1Api.DeleteSubjectMode(schemaRegistryRestClient.apiContext(ctx), subjectName).Execute()

	if err != nil {
		return diag.Errorf("error deleting Subject Mode %q: %s", d.Id(), createDescriptiveError(err))
	}

	time.Sleep(schemaRegistryAPIWaitAfterCreateOrDelete)

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Subject Mode %q", d.Id()), map[string]interface{}{subjectModeLoggingKey: d.Id()})

	return nil
}

func subjectModeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Subject Mode %q", d.Id()), map[string]interface{}{subjectModeLoggingKey: d.Id()})

	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Subject Mode: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Subject Mode: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Subject Mode: %s", createDescriptiveError(err))
	}
	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	subjectName := d.Get(paramSubjectName).(string)

	_, err = readSubjectModeAndSetAttributes(ctx, d, schemaRegistryRestClient, subjectName)
	if err != nil {
		return diag.Errorf("error reading Subject Mode: %s", createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Subject Mode %q", d.Id()), map[string]interface{}{subjectModeLoggingKey: d.Id()})

	return nil
}

func createSubjectModeId(clusterId, subjectName string) string {
	return fmt.Sprintf("%s/%s", clusterId, subjectName)
}

func subjectModeImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Subject Mode %q", d.Id()), map[string]interface{}{subjectModeLoggingKey: d.Id()})

	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Subject Mode: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Subject Mode: %s", createDescriptiveError(err))
	}

	clusterIDAndSubjectName := d.Id()
	parts := strings.Split(clusterIDAndSubjectName, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing Subject Mode: invalid format: expected '<SG cluster ID>/<subject name>'")
	}

	clusterId := parts[0]
	subjectName := parts[1]

	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readSubjectModeAndSetAttributes(ctx, d, schemaRegistryRestClient, subjectName); err != nil {
		return nil, fmt.Errorf("error importing Subject Mode %q: %s", d.Id(), createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Subject Mode %q", d.Id()), map[string]interface{}{subjectModeLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func readSubjectModeAndSetAttributes(ctx context.Context, d *schema.ResourceData, c *SchemaRegistryRestClient, subjectName string) ([]*schema.ResourceData, error) {
	subjectMode, resp, err := c.apiClient.ModesV1Api.GetMode(c.apiContext(ctx), subjectName).DefaultToGlobal(true).Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Subject Mode %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{subjectModeLoggingKey: d.Id()})

		isResourceNotFound := ResponseHasExpectedStatusCode(resp, http.StatusNotFound)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Subject Mode %q in TF state because Subject Mode could not be found on the server", d.Id()), map[string]interface{}{subjectModeLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	subjectModeJson, err := json.Marshal(subjectMode)
	if err != nil {
		return nil, fmt.Errorf("error reading Subject Mode %q: error marshaling %#v to json: %s", d.Id(), subjectMode, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Subject Mode %q: %s", d.Id(), subjectModeJson), map[string]interface{}{subjectModeLoggingKey: d.Id()})

	if err := d.Set(paramSubjectName, subjectName); err != nil {
		return nil, err
	}

	if err := d.Set(paramMode, subjectMode.GetMode()); err != nil {
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

	d.SetId(createSubjectModeId(c.clusterId, subjectName))

	return []*schema.ResourceData{d}, nil
}

func subjectModeUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramCredentials, paramMode) {
		return diag.Errorf("error updating Subject Mode %q: only %q and %q blocks can be updated for Subject Mode", d.Id(), paramCredentials, paramMode)
	}
	if d.HasChange(paramMode) {
		updatedMode := d.Get(paramMode).(string)
		updateModeRequest := sr.NewModeUpdateRequest()
		updateModeRequest.SetMode(updatedMode)
		restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Subject Mode: %s", createDescriptiveError(err))
		}
		clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Subject Mode: %s", createDescriptiveError(err))
		}
		clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Subject Mode: %s", createDescriptiveError(err))
		}
		schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
		subjectName := d.Get(paramSubjectName).(string)
		updateModeRequestJson, err := json.Marshal(updateModeRequest)
		if err != nil {
			return diag.Errorf("error updating Subject Mode: error marshaling %#v to json: %s", updateModeRequest, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Updating Subject Mode %q: %s", d.Id(), updateModeRequestJson), map[string]interface{}{kafkaClusterConfigLoggingKey: d.Id()})

		_, _, err = executeSubjectConfigModeUpdate(ctx, schemaRegistryRestClient, updateModeRequest, subjectName)
		if err != nil {
			return diag.Errorf("error updating Subject Mode: %s", createDescriptiveError(err))
		}
		time.Sleep(kafkaRestAPIWaitAfterCreate)
		tflog.Debug(ctx, fmt.Sprintf("Finished updating Subject Mode %q", d.Id()), map[string]interface{}{kafkaClusterConfigLoggingKey: d.Id()})
	}
	return subjectModeRead(ctx, d, meta)
}

func executeSubjectConfigModeUpdate(ctx context.Context, c *SchemaRegistryRestClient, requestData *sr.ModeUpdateRequest, subjectName string) (sr.ModeUpdateRequest, *http.Response, error) {
	return c.apiClient.ModesV1Api.UpdateMode(c.apiContext(ctx), subjectName).ModeUpdateRequest(*requestData).Execute()
}
