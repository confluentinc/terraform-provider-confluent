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
	paramAttributes           = "attributes"
	paramBusinessMetadataName = "business_metadata_name"
)

func businessMetadataBindingResource() *schema.Resource {
	return &schema.Resource{
		ReadContext:   businessMetadataBindingRead,
		CreateContext: businessMetadataBindingCreate,
		DeleteContext: businessMetadataBindingDelete,
		UpdateContext: businessMetadataBindingUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: businessMetadataBindingImport,
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
			paramBusinessMetadataName: {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9_\\s]*$"), "The name must not be empty and consist of a letter followed by a sequence of letter, number, space, or _ characters"),
				Description:  "The name of the business metadata to be applied.",
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
			paramAttributes: {
				Type:        schema.TypeMap,
				Optional:    true,
				Computed:    true,
				Description: "The attributes.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func businessMetadataBindingCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Business Metadata Binding: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Business Metadata Binding: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Business Metadata Binding: %s", createDescriptiveError(err))
	}

	businessMetadataName := d.Get(paramBusinessMetadataName).(string)
	entityName := d.Get(paramEntityName).(string)
	entityType := d.Get(paramEntityType).(string)
	attributes := d.Get(paramAttributes).(map[string]interface{})
	businessMetadataBindingId := createBusinessMetadataBindingId(clusterId, businessMetadataName, entityName, entityType)

	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateDataCatalogClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	businessMetadataBindingRequest := dc.BusinessMetadata{}
	businessMetadataBindingRequest.SetEntityName(entityName)
	businessMetadataBindingRequest.SetEntityType(entityType)
	businessMetadataBindingRequest.SetTypeName(businessMetadataName)
	businessMetadataBindingRequest.SetAttributes(attributes)

	// sleep 60 seconds to wait for entity (resource) to sync to SR
	// https://github.com/confluentinc/terraform-provider-confluent/issues/282 to resolve "error creating Business Metadata Binding 404 Not Found"
	time.Sleep(60 * time.Second)

	request := schemaRegistryRestClient.dataCatalogApiClient.EntityV1Api.CreateBusinessMetadata(schemaRegistryRestClient.dataCatalogApiContext(ctx))
	request = request.BusinessMetadata([]dc.BusinessMetadata{businessMetadataBindingRequest})

	createBusinessMetadataBindingRequestJson, err := json.Marshal(request)
	if err != nil {
		return diag.Errorf("error creating Business Metadata Binding: error marshaling %#v to json: %s", request, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Business Metadata Binding: %s", createBusinessMetadataBindingRequestJson))

	createdBusinessMetadataBinding, _, err := request.Execute()
	if err != nil {
		return diag.Errorf("error creating Business Metadata Binding %s", createDescriptiveError(err))
	}
	if len(createdBusinessMetadataBinding) == 0 {
		return diag.Errorf("error creating Business Metadata Binding %q: empty response", businessMetadataBindingId)
	}
	if createdBusinessMetadataBinding[0].Error != nil {
		return diag.Errorf("error creating Business Metadata Binding %q: %s", businessMetadataBindingId, createdBusinessMetadataBinding[0].Error.GetMessage())
	}
	d.SetId(businessMetadataBindingId)

	if err := waitForBusinessMetadataBindingToProvision(schemaRegistryRestClient.dataCatalogApiContext(ctx), schemaRegistryRestClient, businessMetadataBindingId, businessMetadataName, entityName, entityType); err != nil {
		return diag.Errorf("error waiting for Business Metadata Binding %q to provision: %s", businessMetadataBindingId, createDescriptiveError(err))
	}

	// https://github.com/confluentinc/terraform-provider-confluent/issues/282 to resolve "Root object was present, but now absent."
	time.Sleep(2 * dataCatalogAPIWaitAfterCreate)

	createdBusinessMetadataBindingJson, err := json.Marshal(createdBusinessMetadataBinding)
	if err != nil {
		return diag.Errorf("error creating Business Metadata Binding %q: error marshaling %#v to json: %s", businessMetadataBindingId, createdBusinessMetadataBinding, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Business Metadata Binding %q: %s", businessMetadataBindingId, createdBusinessMetadataBindingJson), map[string]interface{}{businessMetadataBindingLoggingKey: businessMetadataBindingId})
	return businessMetadataBindingRead(ctx, d, meta)
}

func businessMetadataBindingRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	businessMetadataBindingId := d.Id()

	tflog.Debug(ctx, fmt.Sprintf("Reading Business Metadata Binding %q=%q", paramId, businessMetadataBindingId), map[string]interface{}{businessMetadataBindingLoggingKey: businessMetadataBindingId})
	if _, err := readBusinessMetadataBindingAndSetAttributes(ctx, d, meta); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Business Metadata Binding %q: %s", businessMetadataBindingId, createDescriptiveError(err)))
	}

	return nil
}

func readBusinessMetadataBindingAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return nil, fmt.Errorf("error reading Business Metadata Binding: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return nil, fmt.Errorf("error reading Business Metadata Binding: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return nil, fmt.Errorf("error reading Business Metadata Binding: %s", createDescriptiveError(err))
	}

	businessMetadataName := d.Get(paramBusinessMetadataName).(string)
	entityName := d.Get(paramEntityName).(string)
	entityType := d.Get(paramEntityType).(string)
	businessMetadataBindingId := createBusinessMetadataBindingId(clusterId, businessMetadataName, entityName, entityType)

	tflog.Debug(ctx, fmt.Sprintf("Reading Business Metadata Binding %q=%q", paramId, businessMetadataBindingId), map[string]interface{}{businessMetadataBindingLoggingKey: businessMetadataBindingId})

	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateDataCatalogClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	request := schemaRegistryRestClient.dataCatalogApiClient.EntityV1Api.GetBusinessMetadata(schemaRegistryRestClient.dataCatalogApiContext(ctx), entityType, entityName)
	businessMetadataBindings, resp, err := request.Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Business Metadata Binding %q: %s", businessMetadataBindingId, createDescriptiveError(err)), map[string]interface{}{businessMetadataBindingLoggingKey: businessMetadataBindingId})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Business Metadata Binding %q in TF state because Business Metadata Binding could not be found on the server", businessMetadataBindingId), map[string]interface{}{businessMetadataBindingLoggingKey: businessMetadataBindingId})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}

	businessMetadataBinding, err := findBusinessMetadataBindingByBusinessMetadataName(businessMetadataBindings, businessMetadataName)
	if err != nil {
		if !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Business Metadata Binding %q in TF state because Business Metadata Binding could not be found on the server", businessMetadataBindingId), map[string]interface{}{businessMetadataBindingLoggingKey: businessMetadataBindingId})
			d.SetId("")
			return nil, nil
		}
		return nil, err
	}

	businessMetadataBindingJson, err := json.Marshal(businessMetadataBinding)
	if err != nil {
		return nil, fmt.Errorf("error reading Business Metadata Binding %q: error marshaling %#v to json: %s", businessMetadataBindingId, businessMetadataBindingJson, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Business Metadata Binding %q: %s", businessMetadataBindingId, businessMetadataBindingJson), map[string]interface{}{businessMetadataBindingLoggingKey: businessMetadataBindingId})

	if _, err := setBusinessMetadataBindingAttributes(d, clusterId, businessMetadataBinding); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Business Metadata Binding %q", businessMetadataBindingId), map[string]interface{}{businessMetadataBindingLoggingKey: businessMetadataBindingId})

	return []*schema.ResourceData{d}, nil
}

func findBusinessMetadataBindingByBusinessMetadataName(businessMetadataBindings []dc.BusinessMetadataResponse, businessMetadataName string) (dc.BusinessMetadataResponse, error) {
	for _, businessMetadataBinding := range businessMetadataBindings {
		if businessMetadataBinding.GetTypeName() == businessMetadataName {
			return businessMetadataBinding, nil
		}
	}

	return dc.BusinessMetadataResponse{}, fmt.Errorf(fmt.Sprintf("error reading Business Metadata Binding: couldn't find the Business Metadata binding: %s", businessMetadataName))
}

func businessMetadataBindingDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Business Metadata Binding: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Business Metadata Binding: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Business Metadata Binding: %s", createDescriptiveError(err))
	}

	businessMetadataName := d.Get(paramBusinessMetadataName).(string)
	entityName := d.Get(paramEntityName).(string)
	entityType := d.Get(paramEntityType).(string)
	businessMetadataBindingId := createBusinessMetadataBindingId(clusterId, businessMetadataName, entityName, entityType)

	tflog.Debug(ctx, fmt.Sprintf("Deleting Business Metadata Binding %q=%q", paramId, businessMetadataBindingId), map[string]interface{}{businessMetadataBindingLoggingKey: businessMetadataBindingId})

	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateDataCatalogClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	request := schemaRegistryRestClient.dataCatalogApiClient.EntityV1Api.DeleteBusinessMetadata(schemaRegistryRestClient.dataCatalogApiContext(ctx), entityType, entityName, businessMetadataName)
	_, serviceErr := request.Execute()
	if serviceErr != nil {
		return diag.Errorf("error deleting Business Metadata Binding %q: %s", businessMetadataBindingId, createDescriptiveError(serviceErr))
	}

	time.Sleep(time.Second)

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Business Metadata Binding %q", businessMetadataBindingId), map[string]interface{}{businessMetadataBindingLoggingKey: businessMetadataBindingId})

	return nil
}

func businessMetadataBindingUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramCredentials, paramAttributes, paramEntityName) {
		return diag.Errorf("error updating Business Metadata Binding %q: only %q, %q, %q attributes can be updated for Business Metadata Binding", d.Id(), paramCredentials, paramAttributes, paramEntityName)
	}

	if d.HasChange(paramAttributes) || d.HasChange(paramEntityName) {
		oldEntityNameObject, newEntityNameObject := d.GetChange(paramEntityName)
		oldEntityName := oldEntityNameObject.(string)
		newEntityName := newEntityNameObject.(string)
		entityType := d.Get(paramEntityType).(string)
		if !canUpdateEntityNameBusinessMetadata(entityType, oldEntityName, newEntityName) {
			return diag.Errorf("error updating business metadata Binding %q: schema_identifier in %q block can only be updated for business metadata Bindings if entity type is %q. The entity_name must be incremental and the cluster id must remain the same", d.Id(), paramEntityName, schemaEntityType)
		}
		restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Business Metadata Binding: %s", createDescriptiveError(err))
		}
		clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Business Metadata Binding: %s", createDescriptiveError(err))
		}
		clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Business Metadata Binding: %s", createDescriptiveError(err))
		}

		businessMetadataName := d.Get(paramBusinessMetadataName).(string)
		entityName := d.Get(paramEntityName).(string)
		attributes := d.Get(paramAttributes).(map[string]interface{})
		businessMetadataBindingId := createBusinessMetadataBindingId(clusterId, businessMetadataName, entityName, entityType)

		schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateDataCatalogClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
		businessMetadataBindingRequest := dc.BusinessMetadata{}
		businessMetadataBindingRequest.SetEntityName(entityName)
		businessMetadataBindingRequest.SetEntityType(entityType)
		businessMetadataBindingRequest.SetTypeName(businessMetadataName)
		businessMetadataBindingRequest.SetAttributes(attributes)

		request := schemaRegistryRestClient.dataCatalogApiClient.EntityV1Api.UpdateBusinessMetadata(schemaRegistryRestClient.dataCatalogApiContext(ctx))
		request = request.BusinessMetadata([]dc.BusinessMetadata{businessMetadataBindingRequest})

		updateBusinessMetadataBindingRequestJson, err := json.Marshal(request)
		if err != nil {
			return diag.Errorf("error updating Business Metadata Binding: error marshaling %#v to json: %s", request, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Updating new Business Metadata Binding: %s", updateBusinessMetadataBindingRequestJson))

		updatedBusinessMetadataBinding, _, err := request.Execute()
		if err != nil {
			return diag.Errorf("error updating Business Metadata Binding %s", createDescriptiveError(err))
		}
		if len(updatedBusinessMetadataBinding) == 0 {
			return diag.Errorf("error updating Business Metadata Binding %q: empty response", businessMetadataBindingId)
		}
		if updatedBusinessMetadataBinding[0].Error != nil {
			return diag.Errorf("error updating Business Metadata Binding %q: %s", businessMetadataBindingId, updatedBusinessMetadataBinding[0].Error.GetMessage())
		}
		d.SetId(businessMetadataBindingId)

		updatedBusinessMetadataBindingJson, err := json.Marshal(updatedBusinessMetadataBinding)
		if err != nil {
			return diag.Errorf("error updating Business Metadata Binding %q: error marshaling %#v to json: %s", businessMetadataBindingId, updatedBusinessMetadataBinding, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Finished updating Business Metadata Binding %q: %s", businessMetadataBindingId, updatedBusinessMetadataBindingJson), map[string]interface{}{businessMetadataBindingLoggingKey: businessMetadataBindingId})
	}
	return businessMetadataBindingRead(ctx, d, meta)
}

func businessMetadataBindingImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	businessMetadataBindingId := d.Id()
	if businessMetadataBindingId == "" {
		return nil, fmt.Errorf("error importing Business Metadata Binding: Business Metadata Binding id is missing")
	}

	parts := strings.Split(businessMetadataBindingId, "/")
	if len(parts) != 4 {
		return nil, fmt.Errorf("error importing Business Metadata Binding: invalid format: expected '<Schema Registry Cluster Id>/<Business Metadata Name>/<Entity Name>/<Entity Type>'")
	}
	d.Set(paramBusinessMetadataName, parts[1])
	d.Set(paramEntityName, parts[2])
	d.Set(paramEntityType, parts[3])

	tflog.Debug(ctx, fmt.Sprintf("Imporing Business Metadata Binding %q=%q", paramId, businessMetadataBindingId), map[string]interface{}{businessMetadataBindingLoggingKey: businessMetadataBindingId})
	d.MarkNewResource()
	if _, err := readBusinessMetadataBindingAndSetAttributes(ctx, d, meta); err != nil {
		return nil, fmt.Errorf("error importing Business Metadata Binding %q: %s", businessMetadataBindingId, createDescriptiveError(err))
	}

	return []*schema.ResourceData{d}, nil
}

func createBusinessMetadataBindingId(clusterId, businessMetadataName, entityName, entityType string) string {
	return fmt.Sprintf("%s/%s/%s/%s", clusterId, businessMetadataName, entityName, entityType)
}

func setBusinessMetadataBindingAttributes(d *schema.ResourceData, clusterId string, businessMetadataBinding dc.BusinessMetadataResponse) (*schema.ResourceData, error) {
	if err := d.Set(paramBusinessMetadataName, businessMetadataBinding.GetTypeName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramEntityName, businessMetadataBinding.GetEntityName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramEntityType, businessMetadataBinding.GetEntityType()); err != nil {
		return nil, err
	}
	if err := d.Set(paramAttributes, businessMetadataBinding.GetAttributes()); err != nil {
		return nil, err
	}
	d.SetId(createBusinessMetadataBindingId(clusterId, businessMetadataBinding.GetTypeName(), businessMetadataBinding.GetEntityName(), businessMetadataBinding.GetEntityType()))
	return d, nil
}
