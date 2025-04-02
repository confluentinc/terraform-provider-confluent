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
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"regexp"
	"strconv"
)

func schemaRegistryDekDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: schemaRegistryDekDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramSchemaRegistryCluster: schemaRegistryClusterBlockDataSourceSchema(),
			paramRestEndpoint: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The REST endpoint of the Schema Registry cluster, for example, `https://psrc-00000.us-central1.gcp.confluent.cloud:443`).",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^http"), "the REST endpoint must start with 'https://'"),
			},
			paramCredentials: credentialsSchema(),
			paramKekName: {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramSubjectName: {
				Type:         schema.TypeString,
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
				Computed: true,
			},
			paramKeyMaterial: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramHardDelete: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Controls whether a schema registry dek should be soft or hard deleted. Set it to `true` if you want to hard delete a schema registry dek on destroy. Defaults to `false` (soft delete).",
			},
		},
	}
}

func schemaRegistryDekDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if err := dataSourceCredentialBlockValidationWithOAuth(d, meta.(*Client).isOAuthEnabled); err != nil {
		return diag.Errorf("error reading Schema Registry DEK: %s", createDescriptiveError(err))
	}

	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Schema Registry DEK: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Schema Registry DEK: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Schema Registry DEK: %s", createDescriptiveError(err))
	}
	kekName := d.Get(paramKekName).(string)
	subject := d.Get(paramSubjectName).(string)
	version := d.Get(paramVersion).(int)
	algorithm := d.Get(paramAlgorithm).(string)
	dekId := createDekId(clusterId, kekName, subject, algorithm, int32(version))

	tflog.Debug(ctx, fmt.Sprintf("Reading Schema Registry DEK %q", dekId), map[string]interface{}{schemaRegistryDekKey: dekId})

	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet, meta.(*Client).oauthToken)
	request := schemaRegistryRestClient.apiClient.DataEncryptionKeysV1Api.GetDekByVersion(schemaRegistryRestClient.apiContext(ctx), kekName, subject, strconv.Itoa(version))
	request = request.Algorithm(algorithm)
	dek, _, err := request.Execute()

	if err != nil {
		return diag.Errorf("error reading Schema Registry DEK %q: %s", dekId, createDescriptiveError(err))
	}
	dekJson, err := json.Marshal(dek)
	if err != nil {
		return diag.Errorf("error reading Schema Registry DEK %q: error marshaling %#v to json: %s", dekId, dek, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Schema Registry DEK %q: %s", dekId, dekJson), map[string]interface{}{schemaRegistryDekKey: dekId})

	if _, err := setDekAttributes(d, clusterId, dek); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}
