// Copyright 2026 Confluent Inc. All Rights Reserved.
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

func rtceTopicDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: rtceTopicDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramTopicName: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The Kafka topic name containing the data for the RTCE topic.",
			},
			paramEnvironment: environmentDataSourceSchema(),
			paramCloud: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The cloud provider where the RTCE topic is deployed.",
			},
			paramDescription: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A model-readable description of the RTCE topic.",
			},
			paramKafkaCluster: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						paramId: {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Required:    true,
				MinItems:    1,
				MaxItems:    1,
				Description: "The Kafka cluster containing the topic to be materialized.",
			},
			paramRegion: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The cloud region where the RTCE topic is deployed.",
			},
			paramApiVersion: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "API Version defines the schema version of this representation of a resource.",
			},
			paramKind: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Kind defines the object this REST resource represents.",
			},
			paramResourceName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Confluent Resource Name of the resource.",
			},
		},
	}
}

func rtceTopicDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	topicName := d.Get(paramTopicName).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	kafkaClusterId := extractStringValueFromBlock(d, paramKafkaCluster, paramId)
	tflog.Debug(ctx, fmt.Sprintf("Reading RtceTopic %q=%q", paramTopicName, topicName), map[string]interface{}{rtceTopicLoggingKey: topicName})

	c := meta.(*Client)
	rtceTopic, resp, err := executeRtceTopicRead(c.rtceV1ApiContext(ctx), c, environmentId, kafkaClusterId, topicName)
	if err != nil {
		return diag.Errorf("error reading RtceTopic %q: %s", topicName, createDescriptiveError(err, resp))
	}
	rtceTopicJson, err := json.Marshal(rtceTopic)
	if err != nil {
		return diag.Errorf("error reading RtceTopic %q: error marshaling %#v to json: %s", topicName, rtceTopic, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched RtceTopic %q: %s", topicName, rtceTopicJson), map[string]interface{}{rtceTopicLoggingKey: topicName})

	if _, err := setRtceTopicAttributes(d, rtceTopic); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}
