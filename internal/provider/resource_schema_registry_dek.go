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
	"regexp"
	"strconv"
	"strings"
)

const (
	paramKekName                   = "kek_name"
	paramAlgorithm                 = "algorithm"
	paramEncryptedKeyMaterial      = "encrypted_key_material"
	paramKeyMaterial               = "key_material"
	paramHardDeleteDekDefaultValue = true
)

var acceptedDekAlgorithm = []string{"AES128_GCM", "AES256_GCM", "AES256_SIV"}

func schemaRegistryDekResource() *schema.Resource {
	return &schema.Resource{
		ReadContext:   schemaRegistryDekRead,
		CreateContext: schemaRegistryDekCreate,
		DeleteContext: schemaRegistryDekDelete,
		UpdateContext: schemaRegistryDekUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: schemaRegistryDekImport,
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
			paramKekName: {
				Type:         schema.TypeString,
				ForceNew:     true,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramSubjectName: {
				Type:         schema.TypeString,
				ForceNew:     true,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramVersion: {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1,
			},
			paramAlgorithm: {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice(acceptedDekAlgorithm, false),
				Default:      "AES256_GCM",
			},
			paramEncryptedKeyMaterial: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			paramKeyMaterial: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramHardDelete: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     paramHardDeleteDekDefaultValue,
				Description: "Controls whether a dek should be soft or hard deleted. Set it to `true` if you want to hard delete a schema registry dek on destroy. Defaults to `false` (soft delete).",
			},
		},
	}
}

func schemaRegistryDekCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Schema Registry Dek: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Schema Registry Dek: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Schema Registry Dek: %s", createDescriptiveError(err))
	}

	kekName := d.Get(paramKekName).(string)
	subject := d.Get(paramSubjectName).(string)
	version := d.Get(paramVersion).(int)
	algorithm := d.Get(paramAlgorithm).(string)
	dekId := createDekId(clusterId, kekName, subject, algorithm, int32(version))

	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	dekRequest := sr.CreateDekRequest{}
	dekRequest.SetSubject(subject)
	dekRequest.SetVersion(int32(version))
	dekRequest.SetAlgorithm(algorithm)

	if encryptedKeyMaterial, ok := d.GetOk(paramEncryptedKeyMaterial); ok {
		dekRequest.SetEncryptedKeyMaterial(encryptedKeyMaterial.(string))
	}

	request := schemaRegistryRestClient.apiClient.DataEncryptionKeysV1Api.CreateDek(schemaRegistryRestClient.apiContext(ctx), kekName)
	request = request.CreateDekRequest(dekRequest)

	createDekRequestJson, err := json.Marshal(request)
	if err != nil {
		return diag.Errorf("error creating Schema Registry Dek: error marshaling %#v to json: %s", request, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Schema Registry Dek: %s", createDekRequestJson))

	createdDek, _, err := request.Execute()
	if err != nil {
		return diag.Errorf("error creating Schema Registry Dek %s", createDescriptiveError(err))
	}
	d.SetId(dekId)

	createdDekJson, err := json.Marshal(createdDek)
	if err != nil {
		return diag.Errorf("error creating Schema Registry Dek %q: error marshaling %#v to json: %s", dekId, createdDek, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Schema Registry Dek %q: %s", dekId, createdDekJson), map[string]interface{}{schemaRegistryDekKey: dekId})
	return schemaRegistryDekRead(ctx, d, meta)
}

func schemaRegistryDekRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	dekId := d.Id()

	tflog.Debug(ctx, fmt.Sprintf("Reading Schema Registry Dek %q=%q", paramId, dekId), map[string]interface{}{schemaRegistryDekKey: dekId})
	if _, err := readSchemaRegistryDekAndSetAttributes(ctx, d, meta); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Schema Registry Dek %q: %s", dekId, createDescriptiveError(err)))
	}

	return nil
}

func readSchemaRegistryDekAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return nil, fmt.Errorf("error reading Schema Registry Dek: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return nil, fmt.Errorf("error reading Schema Registry Dek: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return nil, fmt.Errorf("error reading Schema Registry Dek: %s", createDescriptiveError(err))
	}

	kekName := d.Get(paramKekName).(string)
	subject := d.Get(paramSubjectName).(string)
	version := d.Get(paramVersion).(int)
	algorithm := d.Get(paramAlgorithm).(string)
	dekId := createDekId(clusterId, kekName, subject, algorithm, int32(version))

	tflog.Debug(ctx, fmt.Sprintf("Reading Schema Registry Dek %q=%q", paramId, dekId), map[string]interface{}{schemaRegistryDekKey: dekId})

	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	request := schemaRegistryRestClient.apiClient.DataEncryptionKeysV1Api.GetDekByVersion(schemaRegistryRestClient.apiContext(ctx), kekName, subject, strconv.Itoa(version))
	request = request.Algorithm(algorithm)
	dek, resp, err := request.Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Schema Registry Dek %q: %s", dekId, createDescriptiveError(err)), map[string]interface{}{schemaRegistryDekKey: dekId})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Schema Registry Dek %q in TF state because Schema Registry Dek could not be found on the server", dekId), map[string]interface{}{schemaRegistryDekKey: dekId})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	dekJson, err := json.Marshal(dek)
	if err != nil {
		return nil, fmt.Errorf("error reading Schema Registry Dek %q: error marshaling %#v to json: %s", dekId, dekJson, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Schema Registry Dek %q: %s", dekId, dekJson), map[string]interface{}{schemaRegistryDekKey: dekId})

	if _, err := setDekAttributes(d, clusterId, dek); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Schema Registry Dek %q", dekId), map[string]interface{}{schemaRegistryDekKey: dekId})

	return []*schema.ResourceData{d}, nil
}

func schemaRegistryDekUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramCredentials, paramHardDelete) {
		return diag.Errorf("error updating Schema Registry Kek %q: only %q, %q attributes can be updated for Schema Registry Kek", d.Id(), paramCredentials, paramHardDelete)
	}

	return schemaRegistryDekRead(ctx, d, meta)
}

func schemaRegistryDekDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Schema Registry Dek: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Schema Registry Dek: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Schema Registry Dek: %s", createDescriptiveError(err))
	}

	kekName := d.Get(paramKekName).(string)
	subject := d.Get(paramSubjectName).(string)
	version := d.Get(paramVersion).(int)
	algorithm := d.Get(paramAlgorithm).(string)
	dekId := createDekId(clusterId, kekName, subject, algorithm, int32(version))

	tflog.Debug(ctx, fmt.Sprintf("Deleting Schema Registry Dek %q=%q", paramId, dekId), map[string]interface{}{schemaRegistryDekKey: dekId})

	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	isHardDeleteEnabled := d.Get(paramHardDelete).(bool)

	if isHardDeleteEnabled {
		// first soft delete the key
		request := schemaRegistryRestClient.apiClient.DataEncryptionKeysV1Api.DeleteDekVersion(schemaRegistryRestClient.apiContext(ctx), kekName, subject, strconv.Itoa(version))
		request = request.Algorithm(algorithm)
		request = request.Permanent(false)
		_, serviceErr := request.Execute()
		if serviceErr != nil {
			return diag.Errorf("error soft deleting Schema Registry Dek %q: %s", dekId, createDescriptiveError(serviceErr))
		}
	}

	request := schemaRegistryRestClient.apiClient.DataEncryptionKeysV1Api.DeleteDekVersion(schemaRegistryRestClient.apiContext(ctx), kekName, subject, strconv.Itoa(version))
	request = request.Algorithm(algorithm)
	request = request.Permanent(isHardDeleteEnabled)
	_, serviceErr := request.Execute()
	if serviceErr != nil {
		return diag.Errorf("error deleting Schema Registry Dek %q: %s", dekId, createDescriptiveError(serviceErr))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Schema Registry Dek %q", dekId), map[string]interface{}{schemaRegistryDekKey: dekId})

	return nil
}

func schemaRegistryDekImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	dekId := d.Id()
	if dekId == "" {
		return nil, fmt.Errorf("error importing Schema Registry Dek: Schema Registry Dek id is missing")
	}

	parts := strings.Split(dekId, "/")
	if len(parts) != 5 {
		return nil, fmt.Errorf("error importing Schema Registry Dek: invalid format: expected '<Schema Registry Cluster Id>/<Schema Registry Kek Name>/<Subject>/<Version>/<Algorithm>'")
	}
	d.Set(paramKekName, parts[1])
	d.Set(paramSubjectName, parts[2])
	if version, err := strconv.Atoi(parts[3]); err == nil {
		d.Set(paramVersion, version)
	}
	d.Set(paramAlgorithm, parts[4])

	tflog.Debug(ctx, fmt.Sprintf("Imporing Schema Registry Dek %q=%q", paramId, dekId), map[string]interface{}{schemaRegistryDekKey: dekId})
	d.MarkNewResource()
	if _, err := readSchemaRegistryDekAndSetAttributes(ctx, d, meta); err != nil {
		return nil, fmt.Errorf("error importing Schema Registry Dek %q: %s", dekId, createDescriptiveError(err))
	}

	return []*schema.ResourceData{d}, nil
}

func setDekAttributes(d *schema.ResourceData, clusterId string, dek sr.Dek) (*schema.ResourceData, error) {
	d.SetId(createDekId(clusterId, dek.GetKekName(), dek.GetSubject(), dek.GetAlgorithm(), dek.GetVersion()))
	if err := d.Set(paramKekName, dek.GetKekName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramSubjectName, dek.GetSubject()); err != nil {
		return nil, err
	}
	if err := d.Set(paramVersion, dek.GetVersion()); err != nil {
		return nil, err
	}
	if err := d.Set(paramAlgorithm, dek.GetAlgorithm()); err != nil {
		return nil, err
	}
	if err := d.Set(paramEncryptedKeyMaterial, dek.GetEncryptedKeyMaterial()); err != nil {
		return nil, err
	}
	if err := d.Set(paramKeyMaterial, dek.GetKeyMaterial()); err != nil {
		return nil, err
	}

	// Explicitly set paramHardDelete to the default value if unset
	if _, ok := d.GetOk(paramHardDelete); !ok {
		if err := d.Set(paramHardDelete, paramHardDeleteDekDefaultValue); err != nil {
			return nil, createDescriptiveError(err)
		}
	}

	return d, nil
}

func createDekId(clusterId, kekName, subject, algorithm string, version int32) string {
	return fmt.Sprintf("%s/%s/%s/%d/%s", clusterId, kekName, subject, version, algorithm)
}
