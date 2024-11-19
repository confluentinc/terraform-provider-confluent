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
)

func subjectConfigDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: subjectCompatibilityLevelDataSourceRead,
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
			paramCompatibilityLevel: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramCompatibilityGroup: {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func subjectCompatibilityLevelDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Subject Compatibility Level %q", d.Id()), map[string]interface{}{subjectConfigLoggingKey: d.Id()})

	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Subject Compatibility Level: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Subject Compatibility Level: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Subject Compatibility Level: %s", createDescriptiveError(err))
	}
	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	subjectName := d.Get(paramSubjectName).(string)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()

	if _, err := readSubjectConfigAndSetAttributes(ctx, d, schemaRegistryRestClient, subjectName); err != nil {
		return diag.Errorf("error reading Subject Compatibility Level: %s", createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Subject Compatibility Level %q", d.Id()), map[string]interface{}{subjectConfigLoggingKey: d.Id()})

	return nil
}
