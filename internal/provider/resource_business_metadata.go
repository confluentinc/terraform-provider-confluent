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
	//paramCategory  = "category"
	//paramCreatedBy = "created_by"
	//paramUpdatedBy  = "updated_by"
	//paramCreateTime = "create_time"
	//paramUpdateTime  = "update_time"
	//paramTypeVersion = "type_version"

	paramAttributeDef = "attribute_definition"
	paramType         = "type"
	paramIsOptional   = "is_optional"
	paramDefaultValue = "default_value"
	paramOptions      = "options"
)

func businessMetadataResource() *schema.Resource {
	return &schema.Resource{
		ReadContext:   businessMetadataRead,
		CreateContext: businessMetadataCreate,
		DeleteContext: businessMetadataDelete,
		UpdateContext: businessMetadataUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: businessMetadataImport,
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
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9_\\s]*$"), "The name must not be empty and consist of a letter followed by a sequence of letter, number, space, or _ characters"),
				ForceNew:     true,
				Description:  "The name of the Business Metadata to be created.",
			},
			paramDescription: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The description of the Business Metadata to be created.",
			},
			paramVersion: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The version.",
			},
			paramAttributeDef: attributeDefsSchema(),
		},
	}
}

func attributeDefsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramName: {
					Type:     schema.TypeString,
					Required: true,
				},
				paramType: {
					Type:     schema.TypeString,
					Computed: true,
				},
				paramIsOptional: {
					Type:     schema.TypeBool,
					Optional: true,
					Computed: true,
				},
				paramDefaultValue: {
					Type:     schema.TypeString,
					Optional: true,
					Computed: true,
				},
				paramDescription: {
					Type:     schema.TypeString,
					Optional: true,
					Computed: true,
				},
				paramOptions: {
					Type:     schema.TypeMap,
					Computed: true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
			},
		},
	}
}

func businessMetadataCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Business Metadata: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Business Metadata: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Business Metadata: %s", createDescriptiveError(err))
	}
	businessMetadataName := d.Get(paramName).(string)
	businessMetadataId := createBusinessMetadataId(clusterId, businessMetadataName)

	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateDataCatalogClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	businessMetadataRequest := dc.BusinessMetadataDef{}
	businessMetadataRequest.SetName(businessMetadataName)
	description := d.Get(paramDescription).(string)
	businessMetadataRequest.SetDescription(description)
	attributeDefs := buildAttributeDefs(d.Get(paramAttributeDef).(*schema.Set).List())
	businessMetadataRequest.SetAttributeDefs(attributeDefs)

	request := schemaRegistryRestClient.dataCatalogApiClient.TypesV1Api.CreateBusinessMetadataDefs(schemaRegistryRestClient.dataCatalogApiContext(ctx))
	request = request.BusinessMetadataDef([]dc.BusinessMetadataDef{businessMetadataRequest})

	createBusinessMetadataRequestJson, err := json.Marshal(request)
	if err != nil {
		return diag.Errorf("error creating Business Metadata: error marshaling %#v to json: %s", request, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Business Metadata: %s", createBusinessMetadataRequestJson))

	createdBusinessMetadata, _, err := request.Execute()
	if err != nil {
		return diag.Errorf("error creating Business Metadata %s", createDescriptiveError(err))
	}
	if len(createdBusinessMetadata) == 0 {
		return diag.Errorf("error creating Business Metadata %q: empty response", businessMetadataId)
	}
	if createdBusinessMetadata[0].Error != nil {
		return diag.Errorf("error creating Business Metadata %q: %s", businessMetadataId, createdBusinessMetadata[0].Error.GetMessage())
	}
	d.SetId(businessMetadataId)

	if err := waitForBusinessMetadataToProvision(schemaRegistryRestClient.dataCatalogApiContext(ctx), schemaRegistryRestClient, businessMetadataId, businessMetadataName); err != nil {
		return diag.Errorf("error waiting for Business Metadata %q to provision: %s", businessMetadataId, createDescriptiveError(err))
	}

	createdBusinessMetadataJson, err := json.Marshal(createdBusinessMetadata)
	if err != nil {
		return diag.Errorf("error creating Business Metadata %q: error marshaling %#v to json: %s", businessMetadataId, createdBusinessMetadata, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Business Metadata %q: %s", businessMetadataId, createdBusinessMetadataJson), map[string]interface{}{businessMetadataLoggingKey: businessMetadataId})
	return businessMetadataRead(ctx, d, meta)
}

func businessMetadataRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	businessMetadataId := d.Id()

	tflog.Debug(ctx, fmt.Sprintf("Reading Business Metadata %q=%q", paramId, businessMetadataId), map[string]interface{}{businessMetadataLoggingKey: businessMetadataId})
	if _, err := readBusinessMetadataAndSetAttributes(ctx, d, meta); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Business Metadata %q: %s", businessMetadataId, createDescriptiveError(err)))
	}

	return nil
}

func readBusinessMetadataAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return nil, fmt.Errorf("error reading Business Metadata: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return nil, fmt.Errorf("error reading Business Metadata: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return nil, fmt.Errorf("error reading Business Metadata: %s", createDescriptiveError(err))
	}
	businessMetadataName := d.Get(paramName).(string)
	businessMetadataId := createBusinessMetadataId(clusterId, businessMetadataName)

	tflog.Debug(ctx, fmt.Sprintf("Reading Business Metadata %q=%q", paramId, businessMetadataId), map[string]interface{}{businessMetadataLoggingKey: businessMetadataId})

	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateDataCatalogClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	request := schemaRegistryRestClient.dataCatalogApiClient.TypesV1Api.GetBusinessMetadataDefByName(schemaRegistryRestClient.dataCatalogApiContext(ctx), businessMetadataName)
	businessMetadata, resp, err := request.Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Business Metadata %q: %s", businessMetadataId, createDescriptiveError(err)), map[string]interface{}{businessMetadataLoggingKey: businessMetadataId})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Business Metadata %q in TF state because Business Metadata could not be found on the server", businessMetadataId), map[string]interface{}{businessMetadataLoggingKey: businessMetadataId})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	businessMetadataJson, err := json.Marshal(businessMetadata)
	if err != nil {
		return nil, fmt.Errorf("error reading Business Metadata %q: error marshaling %#v to json: %s", businessMetadataId, businessMetadataJson, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Business Metadata %q: %s", businessMetadataId, businessMetadataJson), map[string]interface{}{businessMetadataLoggingKey: businessMetadataId})

	if _, err := setBusinessMetadataAttributes(d, clusterId, businessMetadata); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Business Metadata %q", businessMetadataId), map[string]interface{}{businessMetadataLoggingKey: businessMetadataId})

	return []*schema.ResourceData{d}, nil
}

func businessMetadataDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Business Metadata: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Business Metadata: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Business Metadata: %s", createDescriptiveError(err))
	}
	businessMetadataName := d.Get(paramName).(string)
	businessMetadataId := createBusinessMetadataId(clusterId, businessMetadataName)

	tflog.Debug(ctx, fmt.Sprintf("Deleting Business Metadata %q=%q", paramId, businessMetadataId), map[string]interface{}{businessMetadataLoggingKey: businessMetadataId})

	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateDataCatalogClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	request := schemaRegistryRestClient.dataCatalogApiClient.TypesV1Api.DeleteBusinessMetadataDef(schemaRegistryRestClient.dataCatalogApiContext(ctx), businessMetadataName)
	_, serviceErr := request.Execute()
	if serviceErr != nil {
		return diag.Errorf("error deleting Business Metadata %q: %s", businessMetadataId, createDescriptiveError(serviceErr))
	}

	time.Sleep(time.Second)

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Business Metadata %q", businessMetadataId), map[string]interface{}{businessMetadataLoggingKey: businessMetadataId})

	return nil
}

func businessMetadataUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramDescription, paramAttributeDef) {
		return diag.Errorf("error updating Business Metadata %q: only %q, %q attributes can be updated for Business Metadata", d.Id(), paramDescription, paramAttributeDef)
	}

	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error updating Business Metadata: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error updating Business Metadata: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error updating Business Metadata: %s", createDescriptiveError(err))
	}
	businessMetadataName := d.Get(paramName).(string)
	businessMetadataId := createBusinessMetadataId(clusterId, businessMetadataName)

	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateDataCatalogClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	businessMetadataRequest := dc.BusinessMetadataDef{}
	businessMetadataRequest.SetName(businessMetadataName)
	description := d.Get(paramDescription).(string)
	businessMetadataRequest.SetDescription(description)
	attributeDefs := buildAttributeDefs(d.Get(paramAttributeDef).(*schema.Set).List())
	businessMetadataRequest.SetAttributeDefs(attributeDefs)

	request := schemaRegistryRestClient.dataCatalogApiClient.TypesV1Api.UpdateBusinessMetadataDefs(schemaRegistryRestClient.dataCatalogApiContext(ctx))
	request = request.BusinessMetadataDef([]dc.BusinessMetadataDef{businessMetadataRequest})

	updateBusinessMetadataRequestJson, err := json.Marshal(request)
	if err != nil {
		return diag.Errorf("error updating Business Metadata: error marshaling %#v to json: %s", request, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating new businessMetadata: %s", updateBusinessMetadataRequestJson))

	updatedBusinessMetadata, _, err := request.Execute()
	if err != nil {
		return diag.Errorf("error updating Business Metadata %s", createDescriptiveError(err))
	}
	if len(updatedBusinessMetadata) == 0 {
		return diag.Errorf("error updating Business Metadata %q: empty response", businessMetadataId)
	}
	if updatedBusinessMetadata[0].Error != nil {
		return diag.Errorf("error updating Business Metadata %q: %s", businessMetadataId, updatedBusinessMetadata[0].Error.GetMessage())
	}
	d.SetId(businessMetadataId)

	updatedBusinessMetadataJson, err := json.Marshal(updatedBusinessMetadata)
	if err != nil {
		return diag.Errorf("error updating Business Metadata %q: error marshaling %#v to json: %s", businessMetadataId, updatedBusinessMetadata, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Business Metadata %q: %s", businessMetadataId, updatedBusinessMetadataJson), map[string]interface{}{businessMetadataLoggingKey: businessMetadataId})
	return businessMetadataRead(ctx, d, meta)
}

func businessMetadataImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	businessMetadataId := d.Get(paramId).(string)
	if businessMetadataId == "" {
		return nil, fmt.Errorf("error importing Business Metadata: Business Metadata id is missing")
	}

	parts := strings.Split(businessMetadataId, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing Business Metadata: invalid format: expected '<Schema Registry Cluster Id>/<Business Metadata Name>'")
	}
	d.Set(paramName, parts[1])

	tflog.Debug(ctx, fmt.Sprintf("Imporing Business Metadata %q=%q", paramId, businessMetadataId), map[string]interface{}{businessMetadataLoggingKey: businessMetadataId})
	d.MarkNewResource()
	if _, err := readBusinessMetadataAndSetAttributes(ctx, d, meta); err != nil {
		return nil, fmt.Errorf("error importing Business Metadata %q: %s", businessMetadataId, createDescriptiveError(err))
	}

	return []*schema.ResourceData{d}, nil
}

func buildAttributeDefs(tfAttributeDefs []interface{}) []dc.AttributeDef {
	attributeDefs := make([]dc.AttributeDef, len(tfAttributeDefs))
	for index, tfAttributeDef := range tfAttributeDefs {
		attributeDef := dc.NewAttributeDef()
		tfAttributeDefMap := tfAttributeDef.(map[string]interface{})
		if name, exists := tfAttributeDefMap[paramName].(string); exists {
			attributeDef.SetName(name)
		}
		attributeDef.SetTypeName("string")
		if isOptional, exists := tfAttributeDefMap[paramIsOptional].(bool); exists {
			attributeDef.SetIsOptional(isOptional)
		}
		if defaultValue, exists := tfAttributeDefMap[paramDefaultValue].(string); exists {
			attributeDef.SetDefaultValue(defaultValue)
		}
		if description, exists := tfAttributeDefMap[paramDescription].(string); exists {
			attributeDef.SetDescription(description)
		}
		attributeDefs[index] = *attributeDef
	}
	return attributeDefs
}

func buildTfAttributeDefs(attributeDefs []dc.AttributeDef) *[]map[string]interface{} {
	tfAttributeDefs := make([]map[string]interface{}, len(attributeDefs))
	for i, attributeDef := range attributeDefs {
		tfAttributeDef := make(map[string]interface{})
		tfAttributeDef[paramName] = attributeDef.GetName()
		tfAttributeDef[paramType] = attributeDef.GetTypeName()
		tfAttributeDef[paramIsOptional] = attributeDef.GetIsOptional()
		tfAttributeDef[paramDefaultValue] = attributeDef.GetDefaultValue()
		tfAttributeDef[paramDescription] = attributeDef.GetDescription()
		tfAttributeDef[paramOptions] = attributeDef.GetOptions()
		tfAttributeDefs[i] = tfAttributeDef
	}
	return &tfAttributeDefs
}

func createBusinessMetadataId(clusterId, businessMetadataName string) string {
	return fmt.Sprintf("%s/%s", clusterId, businessMetadataName)
}

func setBusinessMetadataAttributes(d *schema.ResourceData, clusterId string, businessMetadata dc.BusinessMetadataDef) (*schema.ResourceData, error) {
	if err := d.Set(paramName, businessMetadata.GetName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramDescription, businessMetadata.GetDescription()); err != nil {
		return nil, err
	}
	if err := d.Set(paramVersion, businessMetadata.GetVersion()); err != nil {
		return nil, err
	}
	if err := d.Set(paramAttributeDef, buildTfAttributeDefs(businessMetadata.GetAttributeDefs())); err != nil {
		return nil, err
	}
	d.SetId(createBusinessMetadataId(clusterId, businessMetadata.GetName()))
	return d, nil
}
