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
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	paramSchemas                  = "schemas"
	paramSchemasFilter            = "filter"
	paramSchemasFilterDeleted     = "deleted"
	paramSchemasFilterLatestOnly  = "latest_only"
	paramSchemasFilterSubjectName = "subject_name"
)

func schemasDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: schemasDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramSchemaRegistryCluster: schemaRegistryClusterBlockDataSourceSchema(),
			paramRestEndpoint: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "The REST endpoint of the Schema Registry cluster, for example, `https://psrc-00000.us-central1.gcp.confluent.cloud:443`).",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^http"), "the REST endpoint must start with 'https://'"),
			},
			paramCredentials: credentialsSchema(),
			paramSchemasFilter: {
				MinItems:    0,
				MaxItems:    1,
				Optional:    true,
				Type:        schema.TypeList,
				Description: "Schema filters.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						paramSchemasFilterSubjectName: {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The name of the Schema Registry Subject.",
						},
						paramSchemasFilterLatestOnly: {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Whether to return soft deleted schemas.",
						},
						paramSchemasFilterDeleted: {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Whether to return latest schema versions only for each matching subject.",
						},
					},
				},
			},
			paramSchemas: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of schemas in Confluent",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						paramSubjectName: {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "The name of the schema subject.",
							ValidateFunc: validation.StringIsNotEmpty,
						},
						paramFormat: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The format of the schema.",
						},
						paramSchema: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The definition of the schema.",
						},
						paramVersion: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The version number of the schema.",
						},
						paramSchemaIdentifier: {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Globally unique identifier of the schema.",
						},
					},
				},
			},
		},
	}
}

func schemasDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Reading schemas")

	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading registry endpoint for schemas: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading cluster id for schemas: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading registry credentials for schemas: %s", createDescriptiveError(err))
	}
	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	getSchemasQuery := schemaRegistryRestClient.apiClient.SchemasV1Api.GetSchemas(schemaRegistryRestClient.apiContext(ctx))
	subjectName := d.Get(fmt.Sprintf("%s.0.%s", paramSchemasFilter, paramSchemasFilterSubjectName)).(string)
	latestOnly := d.Get(fmt.Sprintf("%s.0.%s", paramSchemasFilter, paramSchemasFilterLatestOnly)).(bool)
	deleted := d.Get(fmt.Sprintf("%s.0.%s", paramSchemasFilter, paramSchemasFilterDeleted)).(bool)

	schemas, _, err := getSchemasQuery.SubjectPrefix(subjectName).Deleted(deleted).LatestOnly(latestOnly).Execute()
	if err != nil {
		return diag.Errorf("error querying schemas: %s", createDescriptiveError(err))
	}

	result := make([]map[string]interface{}, len(schemas))
	for i, schema := range schemas {
		result[i] = map[string]interface{}{
			paramSubjectName:      schema.Subject,
			paramFormat:           schema.SchemaType,
			paramSchema:           schema.Schema,
			paramVersion:          schema.Version,
			paramSchemaIdentifier: schema.Id,
		}
	}

	if err := d.Set(paramSchemas, result); err != nil {
		return diag.FromErr(err)
	}

	// force this data to be refreshed in every Terraform apply
	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return nil
}
