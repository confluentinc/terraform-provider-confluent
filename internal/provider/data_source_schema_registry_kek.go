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
)

func schemaRegistryKekDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: schemaRegistryKekDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramSchemaRegistryCluster: schemaRegistryClusterBlockDataSourceSchema(),
			paramRestEndpoint: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The REST endpoint of the Schema Registry cluster, for example, `https://psrc-00000.us-central1.gcp.confluent.cloud:443`).",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^http"), "the REST endpoint must start with 'https://'"),
			},
			paramCredentials: credentialsSchema(),
			paramName: {
				Type:     schema.TypeString,
				Required: true,
			},
			paramKmsType: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramKmsKeyId: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramProperties: {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
			},
			paramDoc: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramShared: {
				Type:     schema.TypeBool,
				Computed: true,
			},
			paramHardDelete: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Controls whether a schema registry kek should be soft or hard deleted. Set it to `true` if you want to hard delete a schema registry kek on destroy. Defaults to `false` (soft delete).",
			},
		},
	}
}

func schemaRegistryKekDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if err := dataSourceCredentialBlockValidationWithOAuth(d, meta.(*Client).isOAuthEnabled); err != nil {
		return diag.Errorf("error reading Schema Registry KEK: %s", createDescriptiveError(err))
	}

	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Schema Registry KEK: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Schema Registry KEK: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Schema Registry KEK: %s", createDescriptiveError(err))
	}
	kekName := d.Get(paramName).(string)

	tflog.Debug(ctx, fmt.Sprintf("Reading Schema Registry KEK %q", kekName), map[string]interface{}{schemaRegistryKekKey: createKekId(clusterId, kekName)})

	return schemaRegistryKekDataSourceReadUsingKekName(ctx, d, meta, restEndpoint, clusterId, clusterApiKey, clusterApiSecret, kekName)
}

func schemaRegistryKekDataSourceReadUsingKekName(ctx context.Context, d *schema.ResourceData, meta interface{}, restEndpoint string, clusterId string, clusterApiKey string, clusterApiSecret string, kekName string) diag.Diagnostics {
	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet, meta.(*Client).oauthToken)
	request := schemaRegistryRestClient.apiClient.KeyEncryptionKeysV1Api.GetKek(schemaRegistryRestClient.apiContext(ctx), kekName)
	kek, resp, err := request.Execute()
	kekId := createKekId(clusterId, kekName)

	if err != nil {
		return diag.Errorf("error reading Schema Registry KEK %q: %s", kekId, createDescriptiveError(err, resp))
	}
	kekJson, err := json.Marshal(kek)
	if err != nil {
		return diag.Errorf("error reading Schema Registry KEK %q: error marshaling %#v to json: %s", kekId, kek, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Schema Registry KEK %q: %s", kekId, kekJson), map[string]interface{}{schemaRegistryKekKey: kekId})

	if _, err := setKekAttributes(d, clusterId, kek); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}
