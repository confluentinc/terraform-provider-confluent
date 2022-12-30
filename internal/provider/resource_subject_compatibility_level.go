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

func subjectCompatibilityLevelResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: subjectCompatibilityLevelCreate,
		ReadContext:   subjectCompatibilityLevelRead,
		UpdateContext: subjectCompatibilityLevelUpdate,
		DeleteContext: subjectCompatibilityLevelDelete,
		Importer: &schema.ResourceImporter{
			StateContext: subjectCompatibilityLevelImport,
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

func subjectCompatibilityLevelCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Subject Compatibility Level: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Subject Compatibility Level: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Subject Compatibility Level: %s", createDescriptiveError(err))
	}
	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	subjectName := d.Get(paramSubjectName).(string)

	if _, ok := d.GetOk(paramCompatibilityLevel); ok {
		compatibilityLevel := d.Get(paramCompatibilityLevel).(string)

		createCompatibilityLevelRequest := sr.NewConfigUpdateRequest()
		createCompatibilityLevelRequest.SetCompatibility(compatibilityLevel)
		createCompatibilityLevelRequestJson, err := json.Marshal(createCompatibilityLevelRequest)
		if err != nil {
			return diag.Errorf("error creating Subject Compatibility Level: error marshaling %#v to json: %s", createCompatibilityLevelRequest, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Creating new Subject Compatibility Level: %s", createCompatibilityLevelRequestJson))

		_, _, err = executeSubjectConfigCompatibilityUpdate(ctx, schemaRegistryRestClient, createCompatibilityLevelRequest, subjectName)

		if err != nil {
			return diag.Errorf("error creating Subject Compatibility Level: %s", createDescriptiveError(err))
		}

		time.Sleep(schemaRegistryAPIWaitAfterCreateOrDelete)
	}

	subjectCompatibilityLevelId := createSubjectCompatibilityLevelId(schemaRegistryRestClient.clusterId, subjectName)
	d.SetId(subjectCompatibilityLevelId)

	tflog.Debug(ctx, fmt.Sprintf("Finished creating Subject Compatibility Level %q", d.Id()), map[string]interface{}{subjectCompatibilityLevelLoggingKey: d.Id()})

	return subjectCompatibilityLevelRead(ctx, d, meta)
}

func subjectCompatibilityLevelDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Subject Compatibility Level %q", d.Id()), map[string]interface{}{subjectCompatibilityLevelLoggingKey: d.Id()})

	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Subject Compatibility Level: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Subject Compatibility Level: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Subject Compatibility Level: %s", createDescriptiveError(err))
	}
	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	subjectName := d.Get(paramSubjectName).(string)

	// Deletes the specified subject-level compatibility level config and reverts to the global default.
	_, _, err = schemaRegistryRestClient.apiClient.ConfigV1Api.DeleteSubjectConfig(schemaRegistryRestClient.apiContext(ctx), subjectName).Execute()

	if err != nil {
		return diag.Errorf("error deleting Subject Compatibility Level %q: %s", d.Id(), createDescriptiveError(err))
	}

	time.Sleep(schemaRegistryAPIWaitAfterCreateOrDelete)

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Subject Compatibility Level %q", d.Id()), map[string]interface{}{subjectCompatibilityLevelLoggingKey: d.Id()})

	return nil
}

func subjectCompatibilityLevelRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Subject Compatibility Level %q", d.Id()), map[string]interface{}{subjectCompatibilityLevelLoggingKey: d.Id()})

	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Subject Compatibility Level: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Subject Compatibility Level: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Subject Compatibility Level: %s", createDescriptiveError(err))
	}
	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	subjectName := d.Get(paramSubjectName).(string)

	_, err = readSubjectCompatibilityLevelAndSetAttributes(ctx, d, schemaRegistryRestClient, subjectName)
	if err != nil {
		return diag.Errorf("error reading Subject Compatibility Level: %s", createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Subject Compatibility Level %q", d.Id()), map[string]interface{}{subjectCompatibilityLevelLoggingKey: d.Id()})

	return nil
}

func createSubjectCompatibilityLevelId(clusterId, subjectName string) string {
	return fmt.Sprintf("%s/%s", clusterId, subjectName)
}

func subjectCompatibilityLevelImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Subject Compatibility Level %q", d.Id()), map[string]interface{}{subjectCompatibilityLevelLoggingKey: d.Id()})

	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Subject Compatibility Level: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Subject Compatibility Level: %s", createDescriptiveError(err))
	}

	clusterIDAndSubjectName := d.Id()
	parts := strings.Split(clusterIDAndSubjectName, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing Subject Compatibility Level: invalid format: expected '<SG cluster ID>/<subject name>'")
	}

	clusterId := parts[0]
	subjectName := parts[1]

	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readSubjectCompatibilityLevelAndSetAttributes(ctx, d, schemaRegistryRestClient, subjectName); err != nil {
		return nil, fmt.Errorf("error importing Subject Compatibility Level %q: %s", d.Id(), createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Subject Compatibility Level %q", d.Id()), map[string]interface{}{subjectCompatibilityLevelLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func readSubjectCompatibilityLevelAndSetAttributes(ctx context.Context, d *schema.ResourceData, c *SchemaRegistryRestClient, subjectName string) ([]*schema.ResourceData, error) {
	subjectCompatibilityLevel, resp, err := c.apiClient.ConfigV1Api.GetSubjectLevelConfig(c.apiContext(ctx), subjectName).DefaultToGlobal(true).Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Subject Compatibility Level %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{subjectCompatibilityLevelLoggingKey: d.Id()})

		isResourceNotFound := ResponseHasExpectedStatusCode(resp, http.StatusNotFound)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Subject Compatibility Level %q in TF state because Subject Compatibility Level could not be found on the server", d.Id()), map[string]interface{}{subjectCompatibilityLevelLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	subjectCompatibilityLevelJson, err := json.Marshal(subjectCompatibilityLevel)
	if err != nil {
		return nil, fmt.Errorf("error reading Subject Compatibility Level %q: error marshaling %#v to json: %s", d.Id(), subjectCompatibilityLevel, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Subject Compatibility Level %q: %s", d.Id(), subjectCompatibilityLevelJson), map[string]interface{}{subjectCompatibilityLevelLoggingKey: d.Id()})

	if err := d.Set(paramSubjectName, subjectName); err != nil {
		return nil, err
	}

	if err := d.Set(paramCompatibilityLevel, subjectCompatibilityLevel.GetCompatibilityLevel()); err != nil {
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

	d.SetId(createSubjectCompatibilityLevelId(c.clusterId, subjectName))

	return []*schema.ResourceData{d}, nil
}

func subjectCompatibilityLevelUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramCredentials, paramCompatibilityLevel) {
		return diag.Errorf("error updating Subject Compatibility Level %q: only %q and %q blocks can be updated for Subject Compatibility Level", d.Id(), paramCredentials, paramCompatibilityLevel)
	}
	if d.HasChange(paramCompatibilityLevel) {
		updatedCompatibilityLevel := d.Get(paramCompatibilityLevel).(string)
		updateCompatibilityLevelRequest := sr.NewConfigUpdateRequest()
		updateCompatibilityLevelRequest.SetCompatibility(updatedCompatibilityLevel)
		restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Schema: %s", createDescriptiveError(err))
		}
		clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Subject Compatibility Level: %s", createDescriptiveError(err))
		}
		clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Schema: %s", createDescriptiveError(err))
		}
		schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
		subjectName := d.Get(paramSubjectName).(string)
		updateCompatibilityLevelRequestJson, err := json.Marshal(updateCompatibilityLevelRequest)
		if err != nil {
			return diag.Errorf("error updating Subject Compatibility Level: error marshaling %#v to json: %s", updateCompatibilityLevelRequest, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Updating Subject Compatibility Level %q: %s", d.Id(), updateCompatibilityLevelRequestJson), map[string]interface{}{kafkaClusterConfigLoggingKey: d.Id()})

		_, _, err = executeSubjectConfigCompatibilityUpdate(ctx, schemaRegistryRestClient, updateCompatibilityLevelRequest, subjectName)
		if err != nil {
			return diag.Errorf("error updating Subject Compatibility Level: %s", createDescriptiveError(err))
		}
		time.Sleep(kafkaRestAPIWaitAfterCreate)
		tflog.Debug(ctx, fmt.Sprintf("Finished updating Subject Compatibility Level %q", d.Id()), map[string]interface{}{kafkaClusterConfigLoggingKey: d.Id()})
	}
	return subjectCompatibilityLevelRead(ctx, d, meta)
}

func executeSubjectConfigCompatibilityUpdate(ctx context.Context, c *SchemaRegistryRestClient, requestData *sr.ConfigUpdateRequest, subjectName string) (sr.ConfigUpdateRequest, *http.Response, error) {
	return c.apiClient.ConfigV1Api.UpdateSubjectLevelConfig(c.apiContext(ctx), subjectName).ConfigUpdateRequest(*requestData).Execute()
}
