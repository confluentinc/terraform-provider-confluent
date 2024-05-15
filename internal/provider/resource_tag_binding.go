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
	"strings"
	"time"
)

const (
	paramTagName     = "tag_name"
	paramEntityName  = "entity_name"
	paramEntityType  = "entity_type"
	schemaEntityType = "sr_schema"
	fieldEntityType  = "sr_field"
	recordEntityType = "sr_record"
)

func tagBindingResource() *schema.Resource {
	return &schema.Resource{
		ReadContext:   tagBindingRead,
		CreateContext: tagBindingCreate,
		DeleteContext: tagBindingDelete,
		UpdateContext: tagBindingUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: tagBindingImport,
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
			paramTagName: {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9_\\s]*$"), "The name must not be empty and consist of a letter followed by a sequence of letter, number, space, or _ characters"),
				Description:  "The name of the tag to be applied.",
				ForceNew:     true,
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
				ForceNew:     true,
			},
		},
	}
}

func tagBindingCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Tag Binding: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Tag Binding: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Tag Binding: %s", createDescriptiveError(err))
	}

	tagName := d.Get(paramTagName).(string)
	entityName := d.Get(paramEntityName).(string)
	entityType := d.Get(paramEntityType).(string)
	tagBindingId := createTagBindingId(clusterId, tagName, entityName, entityType)

	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateDataCatalogClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	tagBindingRequest := dc.Tag{}
	tagBindingRequest.SetEntityName(entityName)
	tagBindingRequest.SetEntityType(entityType)
	tagBindingRequest.SetTypeName(tagName)

	// sleep 60 seconds to wait for entity (resource) to sync to SR
	// https://github.com/confluentinc/terraform-provider-confluent/issues/282 to resolve "error creating Tag Binding 404 Not Found"
	time.Sleep(60 * time.Second)

	request := schemaRegistryRestClient.dataCatalogApiClient.EntityV1Api.CreateTags(schemaRegistryRestClient.dataCatalogApiContext(ctx))
	request = request.Tag([]dc.Tag{tagBindingRequest})

	createTagBindingRequestJson, err := json.Marshal(request)
	if err != nil {
		return diag.Errorf("error creating Tag Binding: error marshaling %#v to json: %s", request, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Tag Binding: %s", createTagBindingRequestJson))

	createdTagBinding, _, err := request.Execute()
	if err != nil {
		return diag.Errorf("error creating Tag Binding %s", createDescriptiveError(err))
	}
	if len(createdTagBinding) == 0 {
		return diag.Errorf("error creating Tag Binding %q: empty response", tagBindingId)
	}
	if createdTagBinding[0].Error != nil {
		return diag.Errorf("error creating Tag Binding %q: %s", tagBindingId, createdTagBinding[0].Error.GetMessage())
	}
	d.SetId(tagBindingId)

	if err := waitForTagBindingToProvision(schemaRegistryRestClient.dataCatalogApiContext(ctx), schemaRegistryRestClient, tagBindingId, tagName, entityName, entityType); err != nil {
		return diag.Errorf("error waiting for Tag Binding %q to provision: %s", tagBindingId, createDescriptiveError(err))
	}

	// https://github.com/confluentinc/terraform-provider-confluent/issues/282 to resolve "Root object was present, but now absent."
	time.Sleep(2 * dataCatalogAPIWaitAfterCreate)

	createdTagBindingJson, err := json.Marshal(createdTagBinding)
	if err != nil {
		return diag.Errorf("error creating Tag Binding %q: error marshaling %#v to json: %s", tagBindingId, createdTagBinding, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Tag Binding %q: %s", tagBindingId, createdTagBindingJson), map[string]interface{}{tagBindingLoggingKey: tagBindingId})
	return tagBindingRead(ctx, d, meta)
}

func tagBindingRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tagBindingId := d.Id()

	tflog.Debug(ctx, fmt.Sprintf("Reading Tag Binding %q=%q", paramId, tagBindingId), map[string]interface{}{tagBindingLoggingKey: tagBindingId})
	if _, err := readTagBindingAndSetAttributes(ctx, d, meta); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Tag Binding %q: %s", tagBindingId, createDescriptiveError(err)))
	}

	return nil
}

func readTagBindingAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return nil, fmt.Errorf("error reading Tag Binding: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return nil, fmt.Errorf("error reading Tag Binding: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return nil, fmt.Errorf("error reading Tag Binding: %s", createDescriptiveError(err))
	}

	tagName := d.Get(paramTagName).(string)
	entityName := d.Get(paramEntityName).(string)
	entityType := d.Get(paramEntityType).(string)
	tagBindingId := createTagBindingId(clusterId, tagName, entityName, entityType)

	tflog.Debug(ctx, fmt.Sprintf("Reading Tag Binding %q=%q", paramId, tagBindingId), map[string]interface{}{tagBindingLoggingKey: tagBindingId})

	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateDataCatalogClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	request := schemaRegistryRestClient.dataCatalogApiClient.EntityV1Api.GetTags(schemaRegistryRestClient.dataCatalogApiContext(ctx), entityType, entityName)
	tagBindings, resp, err := request.Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Tag Binding %q: %s", tagBindingId, createDescriptiveError(err)), map[string]interface{}{tagBindingLoggingKey: tagBindingId})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Tag Binding %q in TF state because Tag Binding could not be found on the server", tagBindingId), map[string]interface{}{tagBindingLoggingKey: tagBindingId})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}

	tagBinding, err := findTagBindingByTagName(tagBindings, tagName)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Removing Tag Binding %q in TF state because Tag Binding could not be found on the server", tagBindingId), map[string]interface{}{tagBindingLoggingKey: tagBindingId})
		d.SetId("")
		return nil, nil
	}

	tagBindingJson, err := json.Marshal(tagBinding)
	if err != nil {
		return nil, fmt.Errorf("error reading Tag Binding %q: error marshaling %#v to json: %s", tagBindingId, tagBindingJson, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Tag Binding %q: %s", tagBindingId, tagBindingJson), map[string]interface{}{tagBindingLoggingKey: tagBindingId})

	if _, err := setTagBindingAttributes(d, clusterId, tagBinding); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Tag Binding %q", tagBindingId), map[string]interface{}{tagBindingLoggingKey: tagBindingId})

	return []*schema.ResourceData{d}, nil
}

func findTagBindingByTagName(tagBindings []dc.TagResponse, tagName string) (dc.TagResponse, error) {
	for _, tagBinding := range tagBindings {
		if tagBinding.GetTypeName() == tagName {
			return tagBinding, nil
		}
	}

	return dc.TagResponse{}, fmt.Errorf("error reading Tag Binding: couldn't find the tag binding")
}

func tagBindingDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Tag Binding: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Tag Binding: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Tag Binding: %s", createDescriptiveError(err))
	}

	tagName := d.Get(paramTagName).(string)
	entityName := d.Get(paramEntityName).(string)
	entityType := d.Get(paramEntityType).(string)
	tagBindingId := createTagBindingId(clusterId, tagName, entityName, entityType)

	tflog.Debug(ctx, fmt.Sprintf("Deleting Tag Binding %q=%q", paramId, tagBindingId), map[string]interface{}{tagBindingLoggingKey: tagBindingId})

	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateDataCatalogClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	request := schemaRegistryRestClient.dataCatalogApiClient.EntityV1Api.DeleteTag(schemaRegistryRestClient.dataCatalogApiContext(ctx), entityType, entityName, tagName)
	_, serviceErr := request.Execute()
	if serviceErr != nil {
		return diag.Errorf("error deleting Tag Binding %q: %s", tagBindingId, createDescriptiveError(serviceErr))
	}

	time.Sleep(time.Second)

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Tag Binding %q", tagBindingId), map[string]interface{}{tagBindingLoggingKey: tagBindingId})

	return nil
}

func tagBindingImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tagBindingId := d.Id()
	if tagBindingId == "" {
		return nil, fmt.Errorf("error importing Tag Binding: TagBinding id is missing")
	}

	parts := strings.Split(tagBindingId, "/")
	if len(parts) != 4 {
		return nil, fmt.Errorf("error importing Tag Binding: invalid format: expected '<Schema Registry Cluster Id>/<Tag Name>/<Entity Name>/<Entity Type>'")
	}
	d.Set(paramTagName, parts[1])
	d.Set(paramEntityName, parts[2])
	d.Set(paramEntityType, parts[3])

	tflog.Debug(ctx, fmt.Sprintf("Imporing Tag Binding %q=%q", paramId, tagBindingId), map[string]interface{}{tagBindingLoggingKey: tagBindingId})
	d.MarkNewResource()
	if _, err := readTagBindingAndSetAttributes(ctx, d, meta); err != nil {
		return nil, fmt.Errorf("error importing Tag Binding %q: %s", tagBindingId, createDescriptiveError(err))
	}

	return []*schema.ResourceData{d}, nil
}

func tagBindingUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramCredentials, paramEntityName) {
		return diag.Errorf("error updating Tag Binding %q: only %q, %q blocks can be updated for Tag Bindings", d.Id(), paramCredentials, paramEntityName)
	}
	if d.HasChange(paramEntityName) {
		entityType := d.Get(paramEntityType).(string)
		oldEntityNameObject, newEntityNameObject := d.GetChange(paramEntityName)
		oldEntityName := oldEntityNameObject.(string)
		newEntityName := newEntityNameObject.(string)
		if !canUpdateEntityName(entityType, oldEntityName, newEntityName) {
			return diag.Errorf("error updating Tag Binding %q: schema_identifier in %q block can only be updated for Tag Bindings if entity type is %q, %q or %q", d.Id(), paramEntityName, schemaEntityType, recordEntityType, fieldEntityType)
		}
		// entity_name will be silently updated
	}
	return tagBindingRead(ctx, d, meta)
}

func createTagBindingId(clusterId, tagName, entityName, entityType string) string {
	return fmt.Sprintf("%s/%s/%s/%s", clusterId, tagName, entityName, entityType)
}

func setTagBindingAttributes(d *schema.ResourceData, clusterId string, tagBinding dc.TagResponse) (*schema.ResourceData, error) {
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
