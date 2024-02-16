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
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"regexp"
	"strconv"
)

func schemaDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: schemaDataSourceRead,
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
			paramSubjectName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The name of the Schema Registry Subject.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramFormat: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The format of the Schema.",
			},
			paramSchema: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The definition of the Schema.",
			},
			paramVersion: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The version number of the Schema.",
			},
			paramSchemaIdentifier: {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Globally unique identifier of the Schema returned for a creation request. It should be used to retrieve this schema from the schemas resource and is different from the schema’s version which is associated with the subject.",
			},
			paramSchemaReference: {
				Description: "The list of references to other Schemas.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						paramSubjectName: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the referenced Schema Registry Subject (for example, \"User\").",
						},
						paramName: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the Schema references (for example, \"io.confluent.kafka.example.User\"). For Avro, the reference name is the fully qualified schema name, for JSON Schema it is a URL, and for Protobuf, it is the name of another Protobuf file.",
						},
						paramVersion: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The version of the referenced Schema.",
						},
					},
				},
			},
			paramHardDelete: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Controls whether a schema should be soft or hard deleted. Set it to `true` if you want to hard delete a schema on destroy. Defaults to `false` (soft delete).",
			},
			paramRecreateOnUpdate: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Controls whether a schema should be recreated on update.",
			},
		},
	}
}

func schemaDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Schema %q", d.Id()), map[string]interface{}{schemaLoggingKey: d.Id()})

	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Schema: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Schema: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Schema: %s", createDescriptiveError(err))
	}
	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	subjectName := d.Get(paramSubjectName).(string)
	schemaIdentifier := d.Get(paramSchemaIdentifier).(int)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()

	if _, err := readSchemaRegistryConfigAndSetAttributes(ctx, d, schemaRegistryRestClient, subjectName, strconv.Itoa(schemaIdentifier)); err != nil {
		return diag.Errorf("error reading Schema: %s", createDescriptiveError(err))
	}
	srSchema, _, err := loadSchema(ctx, d, schemaRegistryRestClient, subjectName, strconv.Itoa(schemaIdentifier))
	if err != nil {
		return diag.Errorf("error reading Schema: %s", createDescriptiveError(err))
	}
	if err := d.Set(paramSchema, srSchema.GetSchema()); err != nil {
		return diag.Errorf("error reading Schema: %s", createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished reading Schema %q", d.Id()), map[string]interface{}{schemaLoggingKey: d.Id()})

	return nil
}

func schemaRegistryClusterBlockDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "The Schema Registry cluster ID (e.g., `lsrc-abc123`).",
				},
			},
		},
		Optional: true,
		MinItems: 1,
		MaxItems: 1,
	}
}
