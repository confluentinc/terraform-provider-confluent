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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"regexp"
)

const (
	qualifiedName = "qualifiedName"
)

func catalogEntityAttributesResource() *schema.Resource {
	return &schema.Resource{
		ReadContext:   catalogEntityAttributesRead,
		CreateContext: catalogEntityAttributesCreate,
		DeleteContext: catalogEntityAttributesDelete,
		UpdateContext: catalogEntityAttributesUpdate,
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
			paramEntityName: {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "The qualified name of the entity.",
				ForceNew:     true,
			},
			paramEntityType: {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "The entity type.",
				ForceNew:     true,
			},
			paramAttributes: {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "The attributes.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
		CustomizeDiff: customdiff.Sequence(resourceCredentialBlockValidationWithOAuth),
	}
}

func catalogEntityAttributesCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractCatalogRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Entity Attributes: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Entity Attributes: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Entity Attributes: %s", createDescriptiveError(err))
	}

	entityName := d.Get(paramEntityName).(string)
	entityType := d.Get(paramEntityType).(string)
	attributes := d.Get(paramAttributes).(map[string]interface{})
	entityAttributesId := createEntityAttributesId(entityType, entityName)
	attributes[qualifiedName] = entityName

	catalogRestClient := meta.(*Client).catalogRestClientFactory.CreateCatalogRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet, meta.(*Client).oauthToken)
	entityRequest := dc.Entity{}
	entityRequest.SetTypeName(entityType)
	entityRequest.SetAttributes(attributes)
	entityWithExtInfo := dc.EntityWithExtInfo{}
	entityWithExtInfo.SetEntity(entityRequest)

	request := catalogRestClient.apiClient.EntityV1Api.PartialEntityUpdate(catalogRestClient.dataCatalogApiContext(ctx))
	request = request.EntityWithExtInfo(entityWithExtInfo)

	createEntityAttributesRequestJson, err := json.Marshal(request)
	if err != nil {
		return diag.Errorf("error creating Entity Attributes: error marshaling %#v to json: %s", request, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Entity Attributes: %s", createEntityAttributesRequestJson))

	createdEntityAttributes, resp, err := request.Execute()
	if err != nil {
		return diag.Errorf("error creating Entity Attributes %s", createDescriptiveError(err, resp))
	}

	d.SetId(entityAttributesId)

	createdEntityAttributesJson, err := json.Marshal(createdEntityAttributes)
	if err != nil {
		return diag.Errorf("error creating Entity Attributes %q: error marshaling %#v to json: %s", entityAttributesId, createdEntityAttributes, createDescriptiveError(err, resp))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Entity Attributes %q: %s", entityAttributesId, createdEntityAttributesJson), map[string]interface{}{entityAttributesLoggingKey: entityAttributesId})
	return catalogEntityAttributesRead(ctx, d, meta)
}

func catalogEntityAttributesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	entityAttributesId := d.Id()

	tflog.Debug(ctx, fmt.Sprintf("Reading Entity Attributes %q=%q", paramId, entityAttributesId), map[string]interface{}{entityAttributesLoggingKey: entityAttributesId})
	if _, err := readEntityAttributesAndSetAttributes(ctx, d, meta); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Entity Attributes %q: %s", entityAttributesId, createDescriptiveError(err)))
	}

	return nil
}

func readEntityAttributesAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	restEndpoint, err := extractCatalogRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return nil, fmt.Errorf("error reading Entity Attributes: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return nil, fmt.Errorf("error reading Entity Attributes: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return nil, fmt.Errorf("error reading Entity Attributes: %s", createDescriptiveError(err))
	}

	entityName := d.Get(paramEntityName).(string)
	entityType := d.Get(paramEntityType).(string)
	entityAttributesId := createEntityAttributesId(entityType, entityName)

	tflog.Debug(ctx, fmt.Sprintf("Reading Entity Attributes %q=%q", paramId, entityAttributesId), map[string]interface{}{entityAttributesLoggingKey: entityAttributesId})

	catalogRestClient := meta.(*Client).catalogRestClientFactory.CreateCatalogRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet, meta.(*Client).oauthToken)
	request := catalogRestClient.apiClient.EntityV1Api.GetByUniqueAttributes(catalogRestClient.dataCatalogApiContext(ctx), entityType, entityName)
	entity, resp, err := request.Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Entity Attributes %q: %s", entityAttributesId, createDescriptiveError(err, resp)), map[string]interface{}{entityAttributesLoggingKey: entityAttributesId})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Entity Attributes %q in TF state because Entity Attributes could not be found on the server", entityAttributesId), map[string]interface{}{entityAttributesLoggingKey: entityAttributesId})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}

	entityJson, err := json.Marshal(entity)
	if err != nil {
		return nil, fmt.Errorf("error reading Entity Attributes %q: error marshaling %#v to json: %s", entityAttributesId, entityJson, createDescriptiveError(err, resp))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Entity Attributes %q: %s", entityAttributesId, entityJson), map[string]interface{}{entityAttributesLoggingKey: entityAttributesId})

	if _, err := setEntityAttributesAttributes(d, entity.Entity); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Entity Attributes %q", entityAttributesId), map[string]interface{}{entityAttributesLoggingKey: entityAttributesId})

	return []*schema.ResourceData{d}, nil
}

func catalogEntityAttributesDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractCatalogRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Entity Attributes: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Entity Attributes: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Entity Attributes: %s", createDescriptiveError(err))
	}

	entityName := d.Get(paramEntityName).(string)
	entityType := d.Get(paramEntityType).(string)
	attributes := d.Get(paramAttributes).(map[string]interface{})
	resetAttributes(attributes)
	entityAttributesId := createEntityAttributesId(entityType, entityName)
	attributes[qualifiedName] = entityName

	catalogRestClient := meta.(*Client).catalogRestClientFactory.CreateCatalogRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet, meta.(*Client).oauthToken)
	entityRequest := dc.Entity{}
	entityRequest.SetTypeName(entityType)
	entityRequest.SetAttributes(attributes)
	entityWithExtInfo := dc.EntityWithExtInfo{}
	entityWithExtInfo.SetEntity(entityRequest)

	request := catalogRestClient.apiClient.EntityV1Api.PartialEntityUpdate(catalogRestClient.dataCatalogApiContext(ctx))
	request = request.EntityWithExtInfo(entityWithExtInfo)

	updateEntityAttributesRequestJson, err := json.Marshal(request)
	if err != nil {
		return diag.Errorf("error deleting Entity Attributes: error marshaling %#v to json: %s", request, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Deleting new Entity Attributes: %s", updateEntityAttributesRequestJson))

	_, resp, err := request.Execute()
	if err != nil {
		return diag.Errorf("error creating Entity Attributes %s", createDescriptiveError(err, resp))
	}

	tflog.Debug(ctx, fmt.Sprintf("Deleting Entity Attributes %q=%q", paramId, entityAttributesId), map[string]interface{}{entityAttributesLoggingKey: entityAttributesId})
	return nil
}

func catalogEntityAttributesUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramCredentials, paramAttributes) {
		return diag.Errorf("error updating Entity Attributes %q: only %q, %q attributes can be updated for Entity Attributes", d.Id(), paramCredentials, paramAttributes)
	}

	if d.HasChange(paramAttributes) {
		restEndpoint, err := extractCatalogRestEndpoint(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Entity Attributes: %s", createDescriptiveError(err))
		}
		clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Entity Attributes: %s", createDescriptiveError(err))
		}
		clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Entity Attributes: %s", createDescriptiveError(err))
		}

		entityName := d.Get(paramEntityName).(string)
		entityType := d.Get(paramEntityType).(string)
		attributes := d.Get(paramAttributes).(map[string]interface{})
		entityAttributesId := createEntityAttributesId(entityType, entityName)
		attributes[qualifiedName] = entityName

		catalogRestClient := meta.(*Client).catalogRestClientFactory.CreateCatalogRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet, meta.(*Client).oauthToken)
		entityRequest := dc.Entity{}
		entityRequest.SetTypeName(entityType)
		entityRequest.SetAttributes(attributes)
		entityWithExtInfo := dc.EntityWithExtInfo{}
		entityWithExtInfo.SetEntity(entityRequest)

		request := catalogRestClient.apiClient.EntityV1Api.PartialEntityUpdate(catalogRestClient.dataCatalogApiContext(ctx))
		request = request.EntityWithExtInfo(entityWithExtInfo)

		updateEntityAttributesRequestJson, err := json.Marshal(request)
		if err != nil {
			return diag.Errorf("error creating Entity Attributes: error marshaling %#v to json: %s", request, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Creating new Entity Attributes: %s", updateEntityAttributesRequestJson))

		updatedEntityAttributes, resp, err := request.Execute()
		if err != nil {
			return diag.Errorf("error creating Entity Attributes %s", createDescriptiveError(err, resp))
		}

		d.SetId(entityAttributesId)

		updatedEntityAttributesJson, err := json.Marshal(updatedEntityAttributes)
		if err != nil {
			return diag.Errorf("error updating Entity Attributes %q: error marshaling %#v to json: %s", entityAttributesId, updatedEntityAttributes, createDescriptiveError(err, resp))
		}
		tflog.Debug(ctx, fmt.Sprintf("Finished updating Entity Attributes %q: %s", entityAttributesId, updatedEntityAttributesJson), map[string]interface{}{entityAttributesLoggingKey: entityAttributesId})
	}
	return catalogEntityAttributesRead(ctx, d, meta)
}

func createEntityAttributesId(entityType, entityName string) string {
	return fmt.Sprintf("%s/%s", entityType, entityName)
}

func setEntityAttributesAttributes(d *schema.ResourceData, entity *dc.Entity) (*schema.ResourceData, error) {
	entityName := d.Get(paramEntityName).(string)
	if err := d.Set(paramEntityType, entity.GetTypeName()); err != nil {
		return nil, err
	}
	// The entity name returned from api response might be different from user input, so we stick with user's input value
	if err := d.Set(paramEntityName, entityName); err != nil {
		return nil, err
	}
	if err := d.Set(paramAttributes, filterAttributes(d, entity.GetAttributes())); err != nil {
		return nil, err
	}
	d.SetId(createEntityAttributesId(entity.GetTypeName(), entityName))
	return d, nil
}

func filterAttributes(d *schema.ResourceData, entityAttributes map[string]interface{}) map[string]interface{} {
	expectedAttributes := d.Get(paramAttributes).(map[string]interface{})
	newAttributes := make(map[string]interface{})
	for attributeName, _ := range expectedAttributes {
		if attributeName == qualifiedName {
			continue
		}

		if val, ok := entityAttributes[attributeName]; !ok {
			newAttributes[attributeName] = ""
		} else {
			newAttributes[attributeName] = val
		}
	}
	return newAttributes
}

func resetAttributes(entityAttributes map[string]interface{}) {
	for attributeName, _ := range entityAttributes {
		entityAttributes[attributeName] = ""
	}
}
