// Copyright 2024 Confluent Inc. All Rights Reserved.
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
	"io"
	"regexp"
	"strings"
)

const (
	paramKmsType                   = "kms_type"
	paramKmsKeyId                  = "kms_key_id"
	paramShared                    = "shared"
	paramDoc                       = "doc"
	paramHardDeleteKekDefaultValue = false
)

func schemaRegistryKekResource() *schema.Resource {
	return &schema.Resource{
		ReadContext:   schemaRegistryKekRead,
		CreateContext: schemaRegistryKekCreate,
		DeleteContext: schemaRegistryKekDelete,
		UpdateContext: schemaRegistryKekUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: schemaRegistryKekImport,
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
				ForceNew:     true,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramKmsType: {
				Type:         schema.TypeString,
				ForceNew:     true,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"aws-kms", "azure-kms", "gcp-kms"}, false),
			},
			paramKmsKeyId: {
				Type:         schema.TypeString,
				ForceNew:     true,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramProperties: {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
				Computed: true,
			},
			paramDoc: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			paramShared: {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			paramHardDelete: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     paramHardDeleteKekDefaultValue,
				Description: "Controls whether a kek should be soft or hard deleted. Set it to `true` if you want to hard delete a schema registry kek on destroy. Defaults to `false` (soft delete).",
			},
		},
	}
}

func schemaRegistryKekCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Schema Registry Kek: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Schema Registry Kek: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Schema Registry Kek: %s", createDescriptiveError(err))
	}

	kekName := d.Get(paramName).(string)
	kekId := createKekId(clusterId, kekName)

	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	kekRequest := sr.CreateKekRequest{}
	kekRequest.SetName(kekName)
	kekRequest.SetKmsType(d.Get(paramKmsType).(string))
	kekRequest.SetKmsKeyId(d.Get(paramKmsKeyId).(string))
	kekRequest.SetDoc(d.Get(paramDoc).(string))

	properties := convertToStringStringMap(d.Get(paramProperties).(map[string]interface{}))
	kekRequest.SetKmsProps(properties)

	if shared, ok := d.GetOk(paramShared); ok {
		kekRequest.SetShared(shared.(bool))
	}

	request := schemaRegistryRestClient.apiClient.KeyEncryptionKeysV1Api.CreateKek(schemaRegistryRestClient.apiContext(ctx))
	request = request.CreateKekRequest(kekRequest)

	createKekRequestJson, err := json.Marshal(request)
	if err != nil {
		return diag.Errorf("error creating Schema Registry Kek: error marshaling %#v to json: %s", request, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Schema Registry Kek: %s", createKekRequestJson))

	createdKek, resp, err := request.Execute()
	if err != nil {
		b, err := io.ReadAll(resp.Body)
		return diag.Errorf("error creating Schema Registry Kek %s, error message: %s", createDescriptiveError(err), string(b))
	}
	d.SetId(kekId)

	createdKekJson, err := json.Marshal(createdKek)
	if err != nil {
		return diag.Errorf("error creating Schema Registry Kek %q: error marshaling %#v to json: %s", kekId, createdKek, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Schema Registry Kek %q: %s", kekId, createdKekJson), map[string]interface{}{schemaRegistryKekKey: kekId})
	return schemaRegistryKekRead(ctx, d, meta)
}

func schemaRegistryKekRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	kekId := d.Id()

	tflog.Debug(ctx, fmt.Sprintf("Reading Schema Registry Kek %q=%q", paramId, kekId), map[string]interface{}{schemaRegistryKekKey: kekId})
	if _, err := readSchemaRegistryKekAndSetAttributes(ctx, d, meta); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Schema Registry Kek %q: %s", kekId, createDescriptiveError(err)))
	}

	return nil
}

func readSchemaRegistryKekAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return nil, fmt.Errorf("error reading Schema Registry Kek: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return nil, fmt.Errorf("error reading Schema Registry Kek: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return nil, fmt.Errorf("error reading Schema Registry Kek: %s", createDescriptiveError(err))
	}

	kekName := d.Get(paramName).(string)
	kekId := createKekId(clusterId, kekName)

	tflog.Debug(ctx, fmt.Sprintf("Reading Schema Registry Kek %q=%q", paramId, kekId), map[string]interface{}{schemaRegistryKekKey: kekId})

	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	request := schemaRegistryRestClient.apiClient.KeyEncryptionKeysV1Api.GetKek(schemaRegistryRestClient.apiContext(ctx), kekName)
	kek, resp, err := request.Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Schema Registry Kek %q: %s", kekId, createDescriptiveError(err)), map[string]interface{}{schemaRegistryKekKey: kekId})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Schema Registry Kek %q in TF state because Schema Registry Kek could not be found on the server", kekId), map[string]interface{}{schemaRegistryKekKey: kekId})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	kekJson, err := json.Marshal(kek)
	if err != nil {
		return nil, fmt.Errorf("error reading Schema Registry Kek %q: error marshaling %#v to json: %s", kekId, kekJson, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Schema Registry Kek %q: %s", kekId, kekJson), map[string]interface{}{schemaRegistryKekKey: kekId})

	if _, err := setKekAttributes(d, clusterId, kek); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Schema Registry Kek %q", kekId), map[string]interface{}{schemaRegistryKekKey: kekId})

	return []*schema.ResourceData{d}, nil
}

func schemaRegistryKekDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Schema Registry Kek: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Schema Registry Kek: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Schema Registry Kek: %s", createDescriptiveError(err))
	}

	kekName := d.Get(paramName).(string)
	kekId := createKekId(clusterId, kekName)

	tflog.Debug(ctx, fmt.Sprintf("Deleting Schema Registry Kek %q=%q", paramId, kekId), map[string]interface{}{schemaRegistryKekKey: kekId})

	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	isHardDeleteEnabled := d.Get(paramHardDelete).(bool)

	if isHardDeleteEnabled {
		// first soft delete the key
		request := schemaRegistryRestClient.apiClient.KeyEncryptionKeysV1Api.DeleteKek(schemaRegistryRestClient.apiContext(ctx), kekName)
		request = request.Permanent(false)
		_, serviceErr := request.Execute()
		if serviceErr != nil {
			return diag.Errorf("error soft deleting Schema Registry Kek %q: %s", kekId, createDescriptiveError(serviceErr))
		}
	}

	request := schemaRegistryRestClient.apiClient.KeyEncryptionKeysV1Api.DeleteKek(schemaRegistryRestClient.apiContext(ctx), kekName)
	request = request.Permanent(isHardDeleteEnabled)
	_, serviceErr := request.Execute()
	if serviceErr != nil {
		return diag.Errorf("error deleting Schema Registry Kek %q: %s", kekId, createDescriptiveError(serviceErr))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Schema Registry Kek %q", kekId), map[string]interface{}{schemaRegistryKekKey: kekId})

	return nil
}

func schemaRegistryKekUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramCredentials, paramProperties, paramDoc, paramShared, paramHardDelete) {
		return diag.Errorf("error updating Schema Registry Kek %q: only %q, %q, %q, %q, %q attributes can be updated for Schema Registry Kek", d.Id(), paramCredentials, paramProperties, paramDoc, paramShared, paramHardDelete)
	}

	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error updating Schema Registry Kek: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error updating Schema Registry Kek: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error updating Schema Registry Kek: %s", createDescriptiveError(err))
	}

	kekName := d.Get(paramName).(string)
	kekId := createKekId(clusterId, kekName)

	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	kekRequest := sr.UpdateKekRequest{}
	kekRequest.SetDoc(d.Get(paramDoc).(string))

	properties := convertToStringStringMap(d.Get(paramProperties).(map[string]interface{}))
	kekRequest.SetKmsProps(properties)

	if shared, ok := d.GetOk(paramShared); ok {
		kekRequest.SetShared(shared.(bool))
	}

	request := schemaRegistryRestClient.apiClient.KeyEncryptionKeysV1Api.PutKek(schemaRegistryRestClient.apiContext(ctx), kekName)
	request = request.UpdateKekRequest(kekRequest)

	updateKekRequestJson, err := json.Marshal(request)
	if err != nil {
		return diag.Errorf("error updating Schema Registry Kek: error marshaling %#v to json: %s", request, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating new Schema Registry Kek: %s", updateKekRequestJson))

	updatedKek, resp, err := request.Execute()
	if err != nil {
		b, err := io.ReadAll(resp.Body)
		return diag.Errorf("error updating Schema Registry Kek %s, error message: %s", createDescriptiveError(err), string(b))
	}

	updatedKekJson, err := json.Marshal(updatedKek)
	if err != nil {
		return diag.Errorf("error updating Schema Registry Kek %q: error marshaling %#v to json: %s", kekId, updatedKek, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Schema Registry Kek %q: %s", kekId, updatedKekJson), map[string]interface{}{schemaRegistryKekKey: kekId})
	return schemaRegistryKekRead(ctx, d, meta)
}

func schemaRegistryKekImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	kekId := d.Id()
	if kekId == "" {
		return nil, fmt.Errorf("error importing Schema Registry Kek: Schema Registry Kek id is missing")
	}

	parts := strings.Split(kekId, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing Schema Registry Kek: invalid format: expected '<Schema Registry Cluster Id>/<Schema Registry Kek Name>'")
	}
	d.Set(paramName, parts[1])

	tflog.Debug(ctx, fmt.Sprintf("Imporing Schema Registry Kek %q=%q", paramId, kekId), map[string]interface{}{schemaRegistryKekKey: kekId})
	d.MarkNewResource()
	if _, err := readSchemaRegistryKekAndSetAttributes(ctx, d, meta); err != nil {
		return nil, fmt.Errorf("error importing Schema Registry Kek %q: %s", kekId, createDescriptiveError(err))
	}

	return []*schema.ResourceData{d}, nil
}

func setKekAttributes(d *schema.ResourceData, clusterId string, kek sr.Kek) (*schema.ResourceData, error) {
	d.SetId(createKekId(clusterId, kek.GetName()))
	if err := d.Set(paramName, kek.GetName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramKmsType, kek.GetKmsType()); err != nil {
		return nil, err
	}
	if err := d.Set(paramKmsKeyId, kek.GetKmsKeyId()); err != nil {
		return nil, err
	}
	if err := d.Set(paramProperties, kek.GetKmsProps()); err != nil {
		return nil, err
	}
	if err := d.Set(paramDoc, kek.GetDoc()); err != nil {
		return nil, err
	}
	if err := d.Set(paramShared, kek.GetShared()); err != nil {
		return nil, err
	}

	// Explicitly set paramHardDelete to the default value if unset
	if _, ok := d.GetOk(paramHardDelete); !ok {
		if err := d.Set(paramHardDelete, paramHardDeleteKekDefaultValue); err != nil {
			return nil, createDescriptiveError(err)
		}
	}

	return d, nil
}

func createKekId(clusterId, keyName string) string {
	return fmt.Sprintf("%s/%s", clusterId, keyName)
}
