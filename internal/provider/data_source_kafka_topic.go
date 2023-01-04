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
)

func kafkaTopicDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: kafkaTopicDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramKafkaCluster: optionalKafkaClusterBlockDataSourceSchema(),
			paramTopicName: {
				Type:     schema.TypeString,
				Required: true,
			},
			paramRestEndpoint: {
				Type:     schema.TypeString,
				Required: true,
			},
			paramCredentials: credentialsSchema(),
			paramPartitionsCount: {
				Type:     schema.TypeInt,
				Computed: true,
			},
			paramConfigs: {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
			},
		},
	}
}

func kafkaTopicDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Kafka Topic: %s", createDescriptiveError(err))
	}
	clusterId, err := extractKafkaClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Kafka Topic: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Kafka Topic: %s", createDescriptiveError(err))
	}
	kafkaRestClient := meta.(*Client).kafkaRestClientFactory.CreateKafkaRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isKafkaMetadataSet, meta.(*Client).isKafkaClusterIdSet)
	topicName := d.Get(paramTopicName).(string)
	tflog.Debug(ctx, fmt.Sprintf("Reading Kafka Topic %q", topicName))

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()

	if _, err := readTopicAndSetAttributes(ctx, d, kafkaRestClient, topicName); err != nil {
		return diag.Errorf("error reading Kafka Topic %q: %s", topicName, createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Kafka Topic %q", topicName))

	return nil
}

func optionalKafkaClusterBlockDataSourceSchema() *schema.Schema {
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
		Optional: true,
		MinItems: 1,
		MaxItems: 1,
	}
}
