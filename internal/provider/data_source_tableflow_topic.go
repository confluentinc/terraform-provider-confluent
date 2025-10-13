// Copyright 2025 Confluent Inc. All Rights Reserved.
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
)

func tableflowTopicDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: tableflowTopicDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the Kafka topic for which Tableflow is enabled.",
			},
			paramEnableCompaction: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "This flag determines whether to enable compaction for the Tableflow enabled topic.",
			},
			paramEnablePartitioning: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "This flag determines whether to enable partitioning for the Tableflow enabled topic.",
			},
			paramSuspended: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Indicates whether the Tableflow should be suspended.",
			},
			paramRetentionMs: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Retention time in milliseconds for the Tableflow enabled topic.",
			},
			paramTableFormats: {
				Type:        schema.TypeSet,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "The supported table formats for the Tableflow-enabled topic.",
			},
			paramTablePath: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The current storage path where the data and metadata is stored for this table.",
			},
			paramRecordFailureStrategy: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The strategy to handle record failures in the Tableflow enabled topic during materialization.",
			},
			paramWriteMode: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Indicates the write mode of the tableflow topic.",
			},
			paramKafkaCluster:         requiredKafkaClusterDataSourceSchema(),
			paramEnvironment:          environmentDataSourceSchema(),
			paramCredentials:          credentialsSchema(),
			paramByobAws:              byobAwsDataSourceSchema(),
			paramManagedStorage:       managedStorageDataSourceSchema(),
			paramErrorHandlingSuspend: errorHandlingSuspendDataSourceSchema(),
			paramErrorHandlingSkip:    errorHandlingSkipDataSourceSchema(),
			paramErrorHandlingLog:     errorHandlingLogDataSourceSchema(),
		},
	}
}

func tableflowTopicDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if err := dataSourceCredentialBlockValidationWithOAuth(d, meta.(*Client).isOAuthEnabled); err != nil {
		return diag.Errorf("error reading Tableflow Topic: %s", createDescriptiveError(err))
	}

	tableflowTopicId := d.Get(paramDisplayName).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	clusterId := extractStringValueFromBlock(d, paramKafkaCluster, paramId)

	tflog.Debug(ctx, fmt.Sprintf("Reading Tableflow Topic %q=%q", paramId, tableflowTopicId), map[string]interface{}{tableflowTopicKey: tableflowTopicId})

	c := meta.(*Client)

	tableflowApiKey, tableflowApiSecret, err := extractTableflowApiKeyAndApiSecret(c, d, false)
	if err != nil {
		return diag.Errorf("error reading Tableflow Topic: %s", createDescriptiveError(err))
	}
	tableflowRestClient := c.tableflowRestClientFactory.CreateTableflowRestClient(tableflowApiKey, tableflowApiSecret, c.isTableflowMetadataSet, c.oauthToken, c.stsToken)

	req := tableflowRestClient.apiClient.TableflowTopicsTableflowV1Api.GetTableflowV1TableflowTopic(tableflowRestClient.apiContext(ctx), tableflowTopicId).Environment(environmentId).SpecKafkaCluster(clusterId)
	tableflowTopic, resp, err := req.Execute()
	if err != nil {
		return diag.Errorf("error reading Tableflow Topic %q: %s", tableflowTopicId, createDescriptiveError(err, resp))
	}
	tableflowTopicJson, err := json.Marshal(tableflowTopic)
	if err != nil {
		return diag.Errorf("error reading Tableflow Topic %q: error marshaling %#v to json: %s", tableflowTopicId, tableflowTopic, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Tableflow Topic %q: %s", tableflowTopicId, tableflowTopicJson), map[string]interface{}{tableflowTopicKey: tableflowTopicId})

	if _, err := setTableflowTopicAttributes(d, tableflowRestClient, tableflowTopic); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}

func byobAwsDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramBucketName: {
					Type:     schema.TypeString,
					Computed: true,
				},
				paramBucketRegion: {
					Type:     schema.TypeString,
					Computed: true,
				},
				paramProviderIntegrationId: {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
		Computed: true,
	}
}

func managedStorageDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		MaxItems: 0,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{},
		},
		Computed: true,
	}
}

func requiredKafkaClusterDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:     schema.TypeString,
					Required: true,
				},
			},
		},
		Required: true,
		MaxItems: 1,
	}
}

func errorHandlingSuspendDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		MaxItems: 0,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{},
		},
		Computed: true,
	}
}

func errorHandlingSkipDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		MaxItems: 0,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{},
		},
		Computed: true,
	}
}

func errorHandlingLogDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramTarget: {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
		Computed: true,
	}
}
