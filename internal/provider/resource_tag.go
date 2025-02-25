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
	dataCatalogTimeout            = time.Minute
	dataCatalogExporterTimeout    = 10 * time.Minute
	dataCatalogAPIWaitAfterCreate = 30 * time.Second
)

var defaultEntityTypes = []string{"cf_entity"}

func tagResource() *schema.Resource {
	return &schema.Resource{
		ReadContext:   tagRead,
		CreateContext: tagCreate,
		UpdateContext: tagUpdate,
		DeleteContext: tagDelete,
		Importer: &schema.ResourceImporter{
			StateContext: tagImport,
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
			paramName: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9_\\s]*$"), "The name must not be empty and consist of a letter followed by a sequence of letter, number, space, or _ characters"),
				Description:  "The name of the tag to be created.",
			},
			paramDescription: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The description of the tag to be created.",
			},
			paramEntityTypes: {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
				Description: "The entity type of the tag to be created.",
			},
			paramVersion: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The version.",
			},
		},
	}
}

func tagCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	restEndpoint, err := extractCatalogRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Tag: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Tag: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Tag: %s", createDescriptiveError(err))
	}

	tagName := d.Get(paramName).(string)
	tagId := createTagId(clusterId, tagName)

	catalogRestClient := meta.(*Client).catalogRestClientFactory.CreateCatalogRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	tagRequest := dc.TagDef{}
	tagRequest.SetName(tagName)
	description := d.Get(paramDescription).(string)
	tagRequest.SetDescription(description)
	tagRequest.SetEntityTypes(defaultEntityTypes)

	request := catalogRestClient.apiClient.TypesV1Api.CreateTagDefs(catalogRestClient.dataCatalogApiContext(ctx))
	request = request.TagDef([]dc.TagDef{tagRequest})

	createTagRequestJson, err := json.Marshal(request)
	if err != nil {
		return diag.Errorf("error creating Tag: error marshaling %#v to json: %s", request, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Tag: %s", createTagRequestJson))

	createdTag, _, err := request.Execute()
	if err != nil {
		return diag.Errorf("error creating Tag %s", createDescriptiveError(err))
	}
	if len(createdTag) == 0 {
		return diag.Errorf("error creating Tag %q: empty response", tagId)
	}
	if createdTag[0].Error != nil {
		return diag.Errorf("error creating Tag %q: %s", tagId, createdTag[0].Error.GetMessage())
	}
	d.SetId(tagId)

	if err := waitForTagToProvision(catalogRestClient.dataCatalogApiContext(ctx), catalogRestClient, tagId, tagName); err != nil {
		return diag.Errorf("error waiting for Tag %q to provision: %s", tagId, createDescriptiveError(err))
	}

	// https://github.com/confluentinc/terraform-provider-confluent/issues/282
	SleepIfNotTestMode(dataCatalogAPIWaitAfterCreate, meta.(*Client).isAcceptanceTestMode)

	createdTagJson, err := json.Marshal(createdTag)
	if err != nil {
		return diag.Errorf("error creating Tag %q: error marshaling %#v to json: %s", tagId, createdTag, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Tag %q: %s", tagId, createdTagJson), map[string]any{tagLoggingKey: tagId})
	return tagRead(ctx, d, meta)
}

func tagRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Tag %q", d.Id()), map[string]any{tagLoggingKey: d.Id()})

	restEndpoint, err := extractCatalogRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Tag: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Tag: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Tag: %s", createDescriptiveError(err))
	}
	catalogRestClient := meta.(*Client).catalogRestClientFactory.CreateCatalogRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	tagName := d.Get(paramName).(string)

	_, err = readTagAndSetAttributes(ctx, d, catalogRestClient, tagName)
	if err != nil {
		return diag.Errorf("error reading Tag: %s", createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Tag %q", d.Id()), map[string]any{tagLoggingKey: d.Id()})

	return nil
}

func readTagAndSetAttributes(ctx context.Context, resourceData *schema.ResourceData, client *CatalogRestClient, tagName string) ([]*schema.ResourceData, error) {
	tagId := createTagId(client.clusterId, tagName)

	tflog.Debug(ctx, fmt.Sprintf("Reading Tag %q=%q", paramId, tagId), map[string]any{tagLoggingKey: tagId})

	request := client.apiClient.TypesV1Api.GetTagDefByName(client.dataCatalogApiContext(ctx), tagName)
	tag, resp, err := request.Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Tag %q: %s", tagId, createDescriptiveError(err)), map[string]any{tagLoggingKey: tagId})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !resourceData.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Tag %q in TF state because Tag could not be found on the server", tagId), map[string]any{tagLoggingKey: tagId})
			resourceData.SetId("")
			return nil, nil
		}

		return nil, err
	}
	tagJson, err := json.Marshal(tag)
	if err != nil {
		return nil, fmt.Errorf("error reading Tag %q: error marshaling %#v to json: %s", tagId, tagJson, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Tag %q: %s", tagId, tagJson), map[string]any{tagLoggingKey: tagId})

	if _, err := setTagAttributes(resourceData, client, client.clusterId, tag); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Tag %q", tagId), map[string]any{tagLoggingKey: tagId})

	return []*schema.ResourceData{resourceData}, nil
}

func tagDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	restEndpoint, err := extractCatalogRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Tag: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Tag: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Tag: %s", createDescriptiveError(err))
	}

	tagName := d.Get(paramName).(string)
	tagId := createTagId(clusterId, tagName)

	tflog.Debug(ctx, fmt.Sprintf("Deleting Tag %q=%q", paramId, tagId), map[string]any{tagLoggingKey: tagId})

	catalogRestClient := meta.(*Client).catalogRestClientFactory.CreateCatalogRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	request := catalogRestClient.apiClient.TypesV1Api.DeleteTagDef(catalogRestClient.dataCatalogApiContext(ctx), tagName)
	_, serviceErr := request.Execute()
	if serviceErr != nil {
		return diag.Errorf("error deleting Tag %q: %s", tagId, createDescriptiveError(serviceErr))
	}

	SleepIfNotTestMode(time.Second, meta.(*Client).isAcceptanceTestMode)
	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Tag %q", tagId), map[string]any{tagLoggingKey: tagId})

	return nil
}

func tagUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	if d.HasChangeExcept(paramDescription) {
		return diag.Errorf("error updating Tag %q: only %q attribute can be updated for Tag", d.Id(), paramDescription)
	}

	restEndpoint, err := extractCatalogRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error updating Tag: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error updating Tag: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error updating Tag: %s", createDescriptiveError(err))
	}

	tagName := d.Get(paramName).(string)
	tagId := createTagId(clusterId, tagName)

	catalogRestClient := meta.(*Client).catalogRestClientFactory.CreateCatalogRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	tagRequest := dc.TagDef{}
	tagRequest.SetName(tagName)
	description := d.Get(paramDescription).(string)
	tagRequest.SetDescription(description)

	request := catalogRestClient.apiClient.TypesV1Api.UpdateTagDefs(catalogRestClient.dataCatalogApiContext(ctx))
	request = request.TagDef([]dc.TagDef{tagRequest})

	updateTagRequestJson, err := json.Marshal(request)
	if err != nil {
		return diag.Errorf("error updating Tag: error marshaling %#v to json: %s", request, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating new Tag: %s", updateTagRequestJson))

	updatedTag, _, err := request.Execute()
	if err != nil {
		return diag.Errorf("error updating Tag %s", createDescriptiveError(err))
	}
	if len(updatedTag) == 0 {
		return diag.Errorf("error updating Tag %q: empty response", tagId)
	}
	if updatedTag[0].Error != nil {
		return diag.Errorf("error updating Tag %q: %s", tagId, updatedTag[0].Error.GetMessage())
	}
	d.SetId(tagId)

	updatedTagJson, err := json.Marshal(updatedTag)
	if err != nil {
		return diag.Errorf("error updating Tag %q: error marshaling %#v to json: %s", tagId, updatedTag, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Tag %q: %s", tagId, updatedTagJson), map[string]any{tagLoggingKey: tagId})
	return tagRead(ctx, d, meta)
}

func tagImport(ctx context.Context, d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Tag %q", d.Id()), map[string]any{tagLoggingKey: d.Id()})

	restEndpoint, err := extractCatalogRestEndpoint(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Tag: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Tag: %s", createDescriptiveError(err))
	}

	tagId := d.Id()
	if tagId == "" {
		return nil, fmt.Errorf("error importing Tag: invalid format: Tag import ID is missing")
	}

	parts := strings.Split(tagId, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing Tag: invalid format: expected '<Schema Registry Cluster Id>/<Tag Name>'")
	}

	clusterId := parts[0]
	tagName := parts[1]

	catalogRestClient := meta.(*Client).catalogRestClientFactory.CreateCatalogRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	_, err = readTagAndSetAttributes(ctx, d, catalogRestClient, tagName)
	if err != nil {
		return nil, fmt.Errorf("error importing Tag %q: %s", d.Id(), createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Tag %q", d.Id()), map[string]any{tagLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func createTagId(clusterId, tagName string) string {
	return fmt.Sprintf("%s/%s", clusterId, tagName)
}

func setTagAttributes(d *schema.ResourceData, c *CatalogRestClient, clusterId string, tag dc.TagDef) (*schema.ResourceData, error) {
	if err := d.Set(paramName, tag.GetName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramDescription, tag.GetDescription()); err != nil {
		return nil, err
	}
	if err := d.Set(paramEntityTypes, tag.GetEntityTypes()); err != nil {
		return nil, err
	}
	if err := d.Set(paramVersion, tag.GetVersion()); err != nil {
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

	d.SetId(createTagId(clusterId, tag.GetName()))
	return d, nil
}

func extractCatalogRestEndpoint(client *Client, d *schema.ResourceData, isImportOperation bool) (string, error) {
	if client.isCatalogRegistryMetadataSet {
		if client.catalogRestEndpoint != "" {
			return client.catalogRestEndpoint, nil
		}
	}
	if client.isSchemaRegistryMetadataSet {
		if client.schemaRegistryRestEndpoint != "" {
			return client.schemaRegistryRestEndpoint, nil
		}
	}
	if isImportOperation {
		restEndpoint := getEnv("CATALOG_REST_ENDPOINT", "")
		restEndpointOld := getEnv("SCHEMA_REGISTRY_REST_ENDPOINT", "")
		if restEndpoint != "" {
			return restEndpoint, nil
		} else if restEndpointOld != "" {
			return restEndpointOld, nil
		} else {
			return "", fmt.Errorf("one of provider.catalog_rest_endpoint (defaults to CATALOG_REST_ENDPOINT environment variable) or IMPORT_CATALOG_REST_ENDPOINT, or provider.schema_registry_rest_endpoint (defaults to SCHEMA_REGISTRY_REST_ENDPOINT environment variable) or IMPORT_SCHEMA_REGISTRY_REST_ENDPOINT environment variable must be set")
		}
	}
	restEndpoint := d.Get(paramRestEndpoint).(string)
	if restEndpoint != "" {
		return restEndpoint, nil
	}
	return "", fmt.Errorf("one of provider.catalog_rest_endpoint (defaults to CATALOG_REST_ENDPOINT environment variable) or provider.schema_registry_rest_endpoint (defaults to SCHEMA_REGISTRY_REST_ENDPOINT environment variable) or resource.rest_endpoint must be set")
}
